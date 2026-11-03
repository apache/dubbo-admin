/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 */
package react

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"dubbo-admin-ai/component/server/engine"
	"dubbo-admin-ai/security"
	memorystore "dubbo-admin-ai/store/memory"
	"github.com/golang-jwt/jwt/v5"
)

// TestMVPBrowserServer is an opt-in, loopback-only server for Cypress. It is
// compiled only into the test binary; diagnostics/login never enter production.
// AI HTTP handlers, JWT validation, Store, Compiler and Provider adapter are real.
func TestMVPBrowserServer(t *testing.T) {
	if os.Getenv("AI_MVP_BROWSER") != "1" {
		t.Skip("set AI_MVP_BROWSER=1 to serve the browser fixture")
	}
	provider := &providerFixture{summaries: true}
	unused, _, spec, g, _ := compilerAgentFixture(t, provider)
	unused.Stop()
	backend := memorystore.NewMemoryStore(200)
	spec.ContextWindowTurns = 2
	spec.MaxIterations = 1
	spec.ExplicitMemory.Enabled = true
	spec.ConversationSummary.Enabled = true
	spec.ConversationSummary.ContextTokens = 32768
	spec.ConversationSummary.TriggerHeadroomTurns = 1
	spec.ConversationSummary.MaxCalls = 2
	spec.ConversationSummary.RecoveryMaxCalls = 8
	agent, err := NewReActAgentWithStore(g, backend, spec, newToolTimeoutResolver(5, nil), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(agent.Stop)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const fixtureKeyID = "mvp-fixture"
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": fixtureKeyID,
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(jwks.Close)
	auth, err := engine.NewJWTVerifier(engine.JWTAuthConfig{JWKSURL: jwks.URL, Issuer: "mvp-fixture", Audience: "mvp-browser"}, jwks.Client())
	if err != nil {
		t.Fatal(err)
	}
	router, err := engine.NewRouter(agent, backend, backend, auth)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = router.Close() })
	router.SetMatchContextValidator(engine.ProjectCatalog{
		{ID: "project-a", TenantID: "browser", OwnerIDs: []string{"alice"}},
		{ID: "project-b", TenantID: "browser", OwnerIDs: []string{"alice"}},
		{ID: "project-bob", TenantID: "browser", OwnerIDs: []string{"bob"}},
	})
	// Serialize control mutations with complete chat requests. Fixture controls
	// never race mutable agent configuration or mutate an in-flight interaction.
	var mu sync.Mutex
	shutdown := make(chan struct{}, 1)
	// Mirror Admin's token issuance: the same issuer, audience, subject and
	// tenant that the real JWT verifier checks.
	signAccessToken := func(user, tenant string) (string, error) {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": "mvp-fixture", "aud": "mvp-browser", "sub": user, "tenant_id": tenant, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()})
		token.Header["kid"] = fixtureKeyID
		return token.SignedString(key)
	}
	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		// Admin's public auth configuration. The MVP entry runs the same
		// discovery step as main.ts, enabling accessTokenManager.
		if r.URL.Path == "/api/v1/auth/providers" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "Success", "message": "success",
				"data": map[string]any{"methods": []string{}, "accessTokenEnabled": true, "providers": []any{}},
			})
			return
		}
		if r.URL.Path == "/api/v1/ai/__test/login" {
			user, tenant := r.URL.Query().Get("user"), "browser"
			if user != "alice" && user != "bob" && user != "tenant-alice" {
				http.Error(w, "unknown test actor", 400)
				return
			}
			if user == "tenant-alice" {
				user, tenant = "alice", "other-browser"
			}
			signed, err := signAccessToken(user, tenant)
			if err != nil {
				http.Error(w, "fixture signing failed", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "mvp-fixture", Value: signed, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if cookie, err := r.Cookie("mvp-fixture"); err == nil && r.Header.Get("Authorization") == "" {
			r.Header.Set("Authorization", "Bearer "+cookie.Value)
		}
		access, authErr := auth.Authenticate(r)
		// Admin's token issuance. The browser authenticates with the fixture
		// session cookie here; the AI service only ever verifies the Bearer
		// token, so this exercises the production issuance path.
		if r.URL.Path == "/api/v1/auth/token" && r.Method == http.MethodPost {
			if authErr != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			signed, err := signAccessToken(access.Subject.UserID, access.Subject.TenantID)
			if err != nil {
				http.Error(w, "fixture signing failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "Success", "message": "success",
				"data": map[string]string{"accessToken": signed},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/ai/__test/") {
			if authErr != nil {
				http.Error(w, "unauthorized", 401)
				return
			}
			if r.URL.Path == "/api/v1/ai/__test/stop" && r.Method == "POST" {
				select {
				case shutdown <- struct{}{}:
				default:
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.URL.Path == "/api/v1/ai/__test/control" && r.Method == "POST" {
				var control struct {
					SummaryFailure bool    `json:"summaryFailure"`
					SummaryEnabled bool    `json:"summaryEnabled"`
					ContextTokens  int     `json:"contextTokens"`
					SummaryPadding int     `json:"summaryPadding"`
					Language       *string `json:"language"`
				}
				if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&control); err != nil {
					http.Error(w, "bad control", 400)
					return
				}
				agent.summaryService.Config.Enabled = control.SummaryEnabled
				agent.compilerSpec.ConversationSummary.Enabled = control.SummaryEnabled
				if control.Language != nil {
					agent.compilerSpec.TemplateVariables["Language"] = *control.Language
				}
				if control.ContextTokens >= 4096 {
					agent.compilerSpec.ContextTokens = control.ContextTokens
				}
				// Invalid model output is a deterministic auxiliary failure. It
				// exercises the service's actual fallback and cursor behavior.
				provider.mu.Lock()
				provider.summaries = !control.SummaryFailure
				provider.summaryPadding = control.SummaryPadding
				provider.mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.URL.Path == "/api/v1/ai/__test/state" {
				provider.mu.Lock()
				requests := append([]map[string]any(nil), provider.requests...)
				provider.mu.Unlock()
				state := map[string]any{"requests": requests}
				if session := r.URL.Query().Get("session"); session != "" {
					v, err := backend.GetSummary(context.Background(), access, session)
					if err == nil {
						state["summary"] = v
					}
					history, err := backend.AllMemory(context.Background(), access, session)
					if err == nil {
						state["history"] = history
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(state)
				return
			}
			http.NotFound(w, r)
			return
		}
		// Capture old injected values in a deterministic first model answer,
		// proving their later absence is dependency filtering, not lucky output.
		if r.URL.Path == "/api/v1/ai/chat/stream" {
			provider.mu.Lock()
			if len(provider.requests) == 0 {
				provider.first = `{"role":"assistant","content":"remembered-v1"}`
			}
			provider.mu.Unlock()
		}
		router.GetEngine().ServeHTTP(w, r)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:8880")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	t.Cleanup(func() { _ = server.Close() })
	t.Log("MVP browser fixture ready at http://127.0.0.1:8880")
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		t.Fatal(err)
	}
}

// Smoke-test the browser's exact JSON envelope against strict decoding even
// when the browser fixture is not enabled in ordinary CI.
func TestMVPChatRejectsUnsupportedPageContext(t *testing.T) {
	provider := &providerFixture{}
	agent, backend, _, _, _ := compilerAgentFixture(t, provider)
	auth, _ := security.NewStaticAuthenticator(testBrowserSubject())
	router, err := engine.NewRouter(agent, backend, backend, auth)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = router.Close() })
	request := httptest.NewRequest("POST", "/api/v1/ai/chat/stream", bytes.NewBufferString(`{"sessionID":"compiled","message":"hello","context":{"page":"home"}}`))
	w := httptest.NewRecorder()
	router.GetEngine().ServeHTTP(w, request)
	if w.Code != 400 {
		t.Fatal("unsupported context accepted", w.Code)
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if len(provider.requests) != 0 {
		t.Fatal("invalid input called model")
	}
}

func testBrowserSubject() security.Subject {
	return security.Subject{TenantID: "tenant", UserID: "owner", AuthMethod: "test"}
}
