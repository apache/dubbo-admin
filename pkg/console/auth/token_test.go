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
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"strings"
	"testing"
	"time"

	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
)

func TestTokenIssuerSignsPrincipalAndPublishesPublicJWKS(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	issuer, err := NewTokenIssuer(&configauth.AccessTokenConfig{
		Enabled: true, Issuer: "dubbo-admin", ActiveKeyID: "admin-key-1", TTL: 1800,
		Audiences: []string{"dubbo-admin-ai"}, Keys: []configauth.AccessTokenKey{{KeyID: "admin-key-1", PrivateKey: key}},
	})
	if err != nil {
		t.Fatal(err)
	}
	principal := Principal{Subject: "github:123", Username: "alice", Email: "alice@example.com", AuthType: "oauth", Provider: "github"}
	now := time.Unix(1_787_458_200, 0)
	response, err := issuer.Issue(principal, now)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := josejwt.ParseSigned(response.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		t.Fatal(err)
	}
	var claims AccessTokenClaims
	if err := parsed.Claims(&key.PublicKey, &claims); err != nil {
		t.Fatal(err)
	}
	if claims.Subject != principal.Subject || claims.Username != principal.Username || claims.Issuer != "dubbo-admin" || !claims.Audience.Contains("dubbo-admin-ai") {
		t.Fatalf("claims = %+v", claims)
	}
	if len(parsed.Headers) != 1 || parsed.Headers[0].KeyID != "admin-key-1" || parsed.Headers[0].Algorithm != string(jose.RS256) {
		t.Fatalf("headers = %+v", parsed.Headers)
	}
	raw, err := json.Marshal(issuer.JWKS())
	if err != nil {
		t.Fatal(err)
	}
	for _, privateField := range []string{`"d":`, `"p":`, `"q":`} {
		if strings.Contains(string(raw), privateField) {
			t.Fatalf("JWKS leaks private key field %s", privateField)
		}
	}
}

func TestTokenIssuerDisabled(t *testing.T) {
	issuer, err := NewTokenIssuer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if issuer.Enabled() {
		t.Fatal("disabled issuer reported enabled")
	}
	if _, err := issuer.Issue(LocalPrincipal("admin"), time.Now()); err == nil {
		t.Fatal("disabled issuer created a token")
	}
}

func TestTokenIssuerMultipleKeysAndRotation(t *testing.T) {
	old, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := []configauth.AccessTokenKey{{KeyID: "old", PrivateKey: old}, {KeyID: "new", PrivateKey: newKey}}
	for _, active := range []string{"old", "new"} {
		issuer, err := NewTokenIssuer(&configauth.AccessTokenConfig{Enabled: true, Issuer: "admin", Audiences: []string{"ai"}, TTL: 1800, ActiveKeyID: active, Keys: keys})
		if err != nil {
			t.Fatal(err)
		}
		response, err := issuer.Issue(LocalPrincipal("admin"), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := josejwt.ParseSigned(response.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Headers[0].KeyID != active {
			t.Fatal("wrong active key")
		}
		jwks := issuer.JWKS()
		if len(jwks.Keys) != 2 || jwks.Keys[0].KeyID != "new" || jwks.Keys[1].KeyID != "old" {
			t.Fatal("JWKS not complete or sorted")
		}
		for _, key := range jwks.Keys {
			if !key.IsPublic() {
				t.Fatal("private key published")
			}
		}
		var claims AccessTokenClaims
		if err := parsed.Claims(jwks.Key(active)[0].Key, &claims); err != nil {
			t.Fatal(err)
		}
	}
}
