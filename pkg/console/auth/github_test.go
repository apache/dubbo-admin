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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	"golang.org/x/oauth2"
)

func TestGitHubProviderMapsPrimaryVerifiedEmailAndUsesPKCE(t *testing.T) {
	var tokenVerifier string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			tokenVerifier = r.Form.Get("code_verifier")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"provider-token","token_type":"bearer"}`))
		case "/user":
			if got := r.Header.Get("Authorization"); got != "Bearer provider-token" {
				t.Errorf("Authorization = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 123456, "login": "zhangsan", "email": ""})
		case "/user/emails":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"email": "other@example.com", "verified": true},
				{"email": "primary@example.com", "primary": true, "verified": true},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := newGitHubProvider("github", configauth.ProviderConfig{
		Type: configauth.ProviderTypeGitHub, DisplayName: "GitHub", ClientID: "id", ClientSecret: "secret",
		RedirectURL: "https://admin.example/api/v1/auth/providers/github/callback", PostLoginRedirectURL: "https://admin.example/admin/", Scopes: []string{"read:user", "user:email"},
	}, githubEndpoints{OAuth: oauth2.Endpoint{AuthURL: server.URL + "/authorize", TokenURL: server.URL + "/token"}, APIBaseURL: server.URL}, server.Client())

	principal, err := provider.Authenticate(context.Background(), "valid-code", "pkce-verifier", "")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if tokenVerifier != "pkce-verifier" {
		t.Fatalf("code_verifier = %q", tokenVerifier)
	}
	if principal.Subject != "github:123456" || principal.Username != "zhangsan" || principal.Email != "primary@example.com" || principal.AuthType != "oauth" || principal.Provider != "github" {
		t.Fatalf("principal = %+v", principal)
	}
}

func TestGitHubProviderRejectsInvalidUserResponses(t *testing.T) {
	tests := []struct {
		name     string
		userBody string
		wantErr  string
	}{
		{name: "missing id", userBody: `{"login":"octocat"}`, wantErr: "numeric id"},
		{name: "invalid json", userBody: `{`, wantErr: "decode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/token":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
				case "/user":
					_, _ = w.Write([]byte(tt.userBody))
				}
			}))
			defer server.Close()
			provider := newGitHubProvider("github", configauth.ProviderConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "https://admin.example/callback"}, githubEndpoints{OAuth: oauth2.Endpoint{TokenURL: server.URL + "/token"}, APIBaseURL: server.URL}, server.Client())
			_, err := provider.Authenticate(context.Background(), "code", "verifier", "")
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Fatalf("Authenticate() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestGitHubProviderSkipsEmailEndpointWithoutUserEmailScope(t *testing.T) {
	emailRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
		case "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 123, "login": "octocat", "email": ""})
		case "/user/emails":
			emailRequests++
			http.Error(w, "scope required", http.StatusForbidden)
		}
	}))
	defer server.Close()

	provider := newGitHubProvider("github", configauth.ProviderConfig{
		ClientID: "id", ClientSecret: "secret", RedirectURL: "https://admin.example/callback", Scopes: []string{"read:user"},
	}, githubEndpoints{OAuth: oauth2.Endpoint{TokenURL: server.URL + "/token"}, APIBaseURL: server.URL}, server.Client())
	principal, err := provider.Authenticate(context.Background(), "code", "verifier", "")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.Email != "" || emailRequests != 0 {
		t.Fatalf("email = %q, email endpoint requests = %d", principal.Email, emailRequests)
	}
}
