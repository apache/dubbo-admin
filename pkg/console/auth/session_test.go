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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestPrincipalSessionRoundTripUsesJSONString(t *testing.T) {
	principal := Principal{
		Subject: "github:123", Username: "octocat", Email: "octo@example.com",
		Groups: []string{"engineering"}, Roles: []string{"admin"}, AuthType: "oauth", Provider: "github",
	}
	var stored any
	got := exerciseSession(t, func(c *gin.Context) {
		session := sessions.Default(c)
		if err := PutPrincipal(session, principal); err != nil {
			t.Fatalf("PutPrincipal() error = %v", err)
		}
		stored = session.Get(PrincipalSessionKey)
		parsed, err := PrincipalFromSession(session)
		if err != nil {
			t.Fatalf("PrincipalFromSession() error = %v", err)
		}
		c.JSON(http.StatusOK, parsed)
	})
	if _, ok := stored.(string); !ok {
		t.Fatalf("stored principal type = %T, want string", stored)
	}
	if got.Subject != principal.Subject || got.Username != principal.Username || len(got.Groups) != 1 {
		t.Fatalf("round-trip principal = %+v", got)
	}
}

func TestPrincipalFromSessionMigratesLegacyUser(t *testing.T) {
	got := exerciseSession(t, func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(LegacyUserSessionKey, "admin")
		principal, err := PrincipalFromSession(session)
		if err != nil {
			t.Fatalf("PrincipalFromSession() error = %v", err)
		}
		c.JSON(http.StatusOK, principal)
	})
	if got.Subject != "local:admin" || got.Username != "admin" || got.AuthType != "password" || got.Provider != "local" || len(got.Groups) != 0 || len(got.Roles) != 0 {
		t.Fatalf("legacy principal = %+v", got)
	}
}

func TestOAuthTransactionIsConsumed(t *testing.T) {
	exerciseSession(t, func(c *gin.Context) {
		session := sessions.Default(c)
		want := OAuthTransaction{ProviderID: "github", State: "state", CodeVerifier: "verifier", Nonce: "nonce"}
		if err := PutOAuthTransaction(session, want); err != nil {
			t.Fatalf("PutOAuthTransaction() error = %v", err)
		}
		got, err := ConsumeOAuthTransaction(session)
		if err != nil || got.ProviderID != want.ProviderID || got.State != want.State || got.CodeVerifier != "" || got.Nonce != "" {
			t.Fatalf("ConsumeOAuthTransaction() = %+v, %v; want only provider ID and state", got, err)
		}
		if _, err := ConsumeOAuthTransaction(session); err == nil {
			t.Fatal("second ConsumeOAuthTransaction() succeeded, want replay rejection")
		}
		c.JSON(http.StatusOK, Principal{})
	})
}

func exerciseSession(t *testing.T, handler gin.HandlerFunc) Principal {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	r.GET("/", handler)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var principal Principal
	if err := json.Unmarshal(recorder.Body.Bytes(), &principal); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return principal
}
