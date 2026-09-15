/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
)

func TestOIDCProviderVerifiesTokenAndUsesUserInfo(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const kid = "oidc-key"
	var server *httptest.Server
	var tokenNonce = "expected-nonce"
	var codeVerifier string
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize", "token_endpoint": server.URL + "/token",
				"jwks_uri": server.URL + "/jwks", "userinfo_endpoint": server.URL + "/userinfo",
			})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: kid, Algorithm: string(jose.RS256), Use: "sig"}}})
		case "/token":
			_ = r.ParseForm()
			codeVerifier = r.Form.Get("code_verifier")
			token := signOIDCTestToken(t, key, kid, map[string]any{
				"iss": server.URL, "sub": "subject-1", "aud": "client-id", "exp": time.Now().Add(time.Minute).Unix(),
				"iat": time.Now().Add(-time.Second).Unix(), "nonce": tokenNonce,
			})
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "provider-token", "token_type": "bearer", "id_token": token})
		case "/userinfo":
			if got := r.Header.Get("Authorization"); got != "Bearer provider-token" {
				t.Errorf("Authorization = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub": "subject-1", "preferred_username": "alice", "email": "alice@example.com", "groups": []string{"engineering"}, "roles": []string{"operator"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := NewOIDCProvider(context.Background(), "sso", configauth.ProviderConfig{
		Type: configauth.ProviderTypeOIDC, DisplayName: "SSO", Issuer: server.URL, ClientID: "client-id", ClientSecret: "secret",
		RedirectURL: "https://admin.example/api/v1/auth/providers/sso/callback", PostLoginRedirectURL: "https://admin.example/admin/", Scopes: []string{"openid", "profile", "email"},
	}, server.Client())
	if err != nil {
		t.Fatalf("NewOIDCProvider() error = %v", err)
	}
	principal, err := provider.Authenticate(context.Background(), "valid-code", "pkce-verifier", tokenNonce)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if codeVerifier != "pkce-verifier" {
		t.Fatalf("code_verifier = %q", codeVerifier)
	}
	if principal.Subject != "sso:subject-1" || principal.Username != "alice" || principal.Email != "alice@example.com" || principal.AuthType != "oidc" || len(principal.Groups) != 1 || len(principal.Roles) != 1 {
		t.Fatalf("principal = %+v", principal)
	}
}

func TestOIDCProviderRejectsInvalidTokenClaims(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(claims map[string]any)
		wantError string
	}{
		{name: "issuer", mutate: func(c map[string]any) { c["iss"] = "https://wrong.example" }, wantError: "issuer"},
		{name: "audience", mutate: func(c map[string]any) { c["aud"] = "wrong" }, wantError: "audience"},
		{name: "missing authorized party", mutate: func(c map[string]any) {
			c["aud"] = []string{"client-id", "another-client"}
		}, wantError: "authorized party"},
		{name: "authorized party", mutate: func(c map[string]any) {
			c["aud"] = []string{"client-id", "another-client"}
			c["azp"] = "another-client"
		}, wantError: "authorized party"},
		{name: "expiration", mutate: func(c map[string]any) { c["exp"] = time.Now().Add(-time.Minute).Unix() }, wantError: "expired"},
		{name: "nonce", mutate: func(c map[string]any) { c["nonce"] = "wrong" }, wantError: "nonce"},
		{name: "subject", mutate: func(c map[string]any) { c["sub"] = "" }, wantError: "subject"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, cleanup := newOIDCTestProvider(t, tt.mutate, "subject-1")
			defer cleanup()
			_, err := provider.Authenticate(context.Background(), "code", "verifier", "expected-nonce")
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tt.wantError) {
				t.Fatalf("Authenticate() error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestOIDCProviderAcceptsMatchingAuthorizedPartyForMultipleAudiences(t *testing.T) {
	provider, cleanup := newOIDCTestProvider(t, func(claims map[string]any) {
		claims["aud"] = []string{"client-id", "another-client"}
		claims["azp"] = "client-id"
	}, "subject-1")
	defer cleanup()
	if _, err := provider.Authenticate(context.Background(), "code", "verifier", "expected-nonce"); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
}

func TestOIDCProviderAcceptsJWKWithoutOptionalAlgorithm(t *testing.T) {
	provider, cleanup := newOIDCTestProviderWithJWKAlgorithm(t, func(map[string]any) {}, "subject-1", "")
	defer cleanup()
	if _, err := provider.Authenticate(context.Background(), "code", "verifier", "expected-nonce"); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
}

func TestOIDCProviderRejectsJWKWithConflictingAlgorithm(t *testing.T) {
	provider, cleanup := newOIDCTestProviderWithJWKAlgorithm(t, func(map[string]any) {}, "subject-1", string(jose.RS512))
	defer cleanup()
	if _, err := provider.Authenticate(context.Background(), "code", "verifier", "expected-nonce"); err == nil || !strings.Contains(err.Error(), "RS256") {
		t.Fatalf("Authenticate() error = %v, want RS256 error", err)
	}
}

func TestOIDCProviderRejectsInsecureDiscoveredEndpoint(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize",
			"token_endpoint": "http://sso.example/token", "jwks_uri": server.URL + "/jwks",
		})
	}))
	defer server.Close()
	_, err := NewOIDCProvider(context.Background(), "sso", configauth.ProviderConfig{
		Issuer: server.URL, ClientID: "client-id", ClientSecret: "secret", RedirectURL: "https://admin.example/callback", Scopes: []string{"openid"},
	}, server.Client())
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("NewOIDCProvider() error = %v, want HTTPS error", err)
	}
}

func TestOIDCProviderUsesBoundedDefaultHTTPClient(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize",
			"token_endpoint": server.URL + "/token", "jwks_uri": server.URL + "/jwks",
		})
	}))
	defer server.Close()
	provider, err := NewOIDCProvider(context.Background(), "sso", configauth.ProviderConfig{
		Issuer: server.URL, ClientID: "client-id", ClientSecret: "secret", RedirectURL: "https://admin.example/callback", Scopes: []string{"openid"},
	}, nil)
	if err != nil {
		t.Fatalf("NewOIDCProvider() error = %v", err)
	}
	if timeout := provider.(*oidcProvider).httpClient.Timeout; timeout <= 0 {
		t.Fatalf("HTTP client timeout = %v, want a positive timeout", timeout)
	}
}

func TestOIDCProviderRejectsUserInfoSubjectMismatch(t *testing.T) {
	provider, cleanup := newOIDCTestProvider(t, func(map[string]any) {}, "other-subject")
	defer cleanup()
	_, err := provider.Authenticate(context.Background(), "code", "verifier", "expected-nonce")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "subject") {
		t.Fatalf("Authenticate() error = %v", err)
	}
}

func newOIDCTestProvider(t *testing.T, mutate func(map[string]any), userInfoSubject string) (Provider, func()) {
	return newOIDCTestProviderWithJWKAlgorithm(t, mutate, userInfoSubject, string(jose.RS256))
}

func newOIDCTestProviderWithJWKAlgorithm(t *testing.T, mutate func(map[string]any), userInfoSubject, algorithm string) (Provider, func()) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize", "token_endpoint": server.URL + "/token", "jwks_uri": server.URL + "/jwks", "userinfo_endpoint": server.URL + "/userinfo"})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "kid", Algorithm: algorithm, Use: "sig"}}})
		case "/token":
			claims := map[string]any{"iss": server.URL, "sub": "subject-1", "aud": "client-id", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Add(-time.Second).Unix(), "nonce": "expected-nonce"}
			mutate(claims)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "provider-token", "token_type": "bearer", "id_token": signOIDCTestToken(t, key, "kid", claims)})
		case "/userinfo":
			_ = json.NewEncoder(w).Encode(map[string]any{"sub": userInfoSubject, "preferred_username": "alice", "email": "alice@example.com"})
		}
	}))
	provider, err := NewOIDCProvider(context.Background(), "sso", configauth.ProviderConfig{Issuer: server.URL, ClientID: "client-id", ClientSecret: "secret", RedirectURL: "https://admin.example/callback", Scopes: []string{"openid"}}, server.Client())
	if err != nil {
		server.Close()
		t.Fatalf("NewOIDCProvider() error = %v", err)
	}
	return provider, server.Close
}

func signOIDCTestToken(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	options := (&jose.SignerOptions{}).WithType("JWT").WithHeader(jose.HeaderKey("kid"), kid)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, options)
	if err != nil {
		t.Fatal(err)
	}
	token, err := josejwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}
