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

package engine

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTVerifierMiddlewareAndKeyRotation(t *testing.T) {
	key1, _ := rsa.GenerateKey(rand.Reader, 2048)
	key2, _ := rsa.GenerateKey(rand.Reader, 2048)
	keys := map[string]*rsa.PublicKey{"key-1": &key1.PublicKey}
	var mu sync.Mutex
	requests := 0
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		requests++
		body := jwksForTest(keys)
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer jwksServer.Close()

	verifier, err := NewJWTVerifier(JWTAuthConfig{JWKSURL: jwksServer.URL, Issuer: "dubbo-admin", Audience: "dubbo-admin-ai"}, jwksServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware(), verifier.Middleware())
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/api/v1/ai/test", func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok || claims.Subject == "" {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	valid := signAITestToken(t, key1, "key-1", "dubbo-admin", "dubbo-admin-ai", "user:1", time.Now().Add(time.Minute), time.Now())
	assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+valid, http.StatusNoContent)
	resp := assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "", http.StatusUnauthorized)
	if resp.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("WWW-Authenticate = %q", resp.Header().Get("WWW-Authenticate"))
	}
	assertAIStatus(t, router, http.MethodGet, "/health", "", http.StatusNoContent)
	assertAIStatus(t, router, http.MethodOptions, "/api/v1/ai/test", "", http.StatusNoContent)

	mu.Lock()
	keys["key-2"] = &key2.PublicKey
	mu.Unlock()
	rotated := signAITestToken(t, key2, "key-2", "dubbo-admin", "dubbo-admin-ai", "user:2", time.Now().Add(time.Minute), time.Now())
	assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+rotated, http.StatusUnauthorized)
	expireJWKSWindow(verifier)
	assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+rotated, http.StatusNoContent)
	mu.Lock()
	defer mu.Unlock()
	if requests != 2 {
		t.Fatalf("JWKS requests = %d, want initial load and unknown-kid refresh", requests)
	}
}

func expireJWKSWindow(v *JWTVerifier) {
	v.mu.Lock()
	v.nextRefresh = time.Time{}
	v.mu.Unlock()
}

func TestJWKSRefreshWindowAndFailure(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) > 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(jwksForTest(map[string]*rsa.PublicKey{"known": &key.PublicKey}))
	}))
	defer server.Close()
	v, err := NewJWTVerifier(JWTAuthConfig{JWKSURL: server.URL}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	for _, wantRequests := range []int32{1, 2} {
		if wantRequests == 2 {
			expireJWKSWindow(v)
		}
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				token := jwt.New(jwt.SigningMethodRS256)
				token.Header["kid"] = fmt.Sprintf("unknown-%d", i)
				if _, err := v.keyForToken(context.Background(), token); err == nil {
					t.Error("unknown kid accepted")
				}
			}(i)
		}
		wg.Wait()
		if got := requests.Load(); got != wantRequests {
			t.Fatalf("requests = %d, want %d", got, wantRequests)
		}
	}
	token := jwt.New(jwt.SigningMethodRS256)
	token.Header["kid"] = "known"
	if _, err := v.keyForToken(context.Background(), token); err == nil {
		t.Fatal("expired cached key accepted after refresh failure")
	}
}

func TestJWKSRefreshRequestCancellation(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer server.Close()
	v := &JWTVerifier{client: NewJWKSClient(server.URL, server.Client()), keys: map[string]*rsa.PublicKey{}}
	router := gin.New()
	router.Use(v.Middleware())
	router.GET("/api/v1/ai/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/test", nil).WithContext(ctx)
	// A parseable unverified header is sufficient to reach the key lookup.
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"unknown"}`))
	req.Header.Set("Authorization", "Bearer "+header+".e30.AA")
	done := make(chan struct{})
	recorder := httptest.NewRecorder()
	go func() { router.ServeHTTP(recorder, req); close(done) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("fetch not started")
	}
	// A concurrent waiter can cancel independently of the initiating request.
	waitCtx, cancelWait := context.WithCancel(context.Background())
	cancelWait()
	if err := v.refresh(waitCtx); err != context.Canceled {
		t.Fatalf("wait cancellation: %v", err)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("request cancellation did not stop refresh")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("upstream request not cancelled")
	}
	if recorder.Code != http.StatusUnauthorized || recorder.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("unexpected response: %d", recorder.Code)
	}
}

func TestJWTVerifierRejectsInvalidClaims(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jwksForTest(map[string]*rsa.PublicKey{"key-1": &key.PublicKey}))
	}))
	defer jwksServer.Close()
	verifier, err := NewJWTVerifier(JWTAuthConfig{JWKSURL: jwksServer.URL, Issuer: "dubbo-admin", Audience: "dubbo-admin-ai"}, jwksServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(verifier.Middleware())
	router.GET("/api/v1/ai/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	now := time.Now()
	tests := []struct{ name, token string }{
		{name: "malformed", token: "not-a-token"},
		{name: "expired", token: signAITestToken(t, key, "key-1", "dubbo-admin", "dubbo-admin-ai", "user", now.Add(-time.Minute), now.Add(-time.Hour))},
		{name: "wrong issuer", token: signAITestToken(t, key, "key-1", "wrong", "dubbo-admin-ai", "user", now.Add(time.Minute), now)},
		{name: "wrong audience", token: signAITestToken(t, key, "key-1", "dubbo-admin", "wrong", "user", now.Add(time.Minute), now)},
		{name: "future issued at", token: signAITestToken(t, key, "key-1", "dubbo-admin", "dubbo-admin-ai", "user", now.Add(time.Minute), now.Add(time.Minute))},
		{name: "empty subject", token: signAITestToken(t, key, "key-1", "dubbo-admin", "dubbo-admin-ai", "", now.Add(time.Minute), now)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+tt.token, http.StatusUnauthorized)
		})
	}
}

func TestJWKSExpiryRevokesRemovedKey(t *testing.T) {
	old, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var stage atomic.Int32
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		keys := map[string]*rsa.PublicKey{"new": &newKey.PublicKey}
		if stage.Load() == 0 {
			keys["old"] = &old.PublicKey
		}
		if stage.Load() == 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(jwksForTest(keys))
	}))
	defer server.Close()
	now := time.Now()
	v := &JWTVerifier{config: JWTAuthConfig{Issuer: "admin", Audience: "ai"}, client: NewJWKSClient(server.URL, server.Client()), now: func() time.Time { return now }}
	if err := v.refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(v.Middleware())
	r.GET("/api/v1/ai/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	oldToken := signAITestToken(t, old, "old", "admin", "ai", "user", time.Now().Add(time.Hour), time.Now())
	newToken := signAITestToken(t, newKey, "new", "admin", "ai", "user", time.Now().Add(time.Hour), time.Now())
	for _, token := range []string{oldToken, newToken} {
		assertAIStatus(t, r, http.MethodGet, "/api/v1/ai/test", "Bearer "+token, http.StatusNoContent)
	}
	stage.Store(1)
	now = now.Add(jwksRefreshInterval)
	assertAIStatus(t, r, http.MethodGet, "/api/v1/ai/test", "Bearer "+oldToken, http.StatusUnauthorized)
	assertAIStatus(t, r, http.MethodGet, "/api/v1/ai/test", "Bearer "+newToken, http.StatusNoContent)
	stage.Store(2)
	now = now.Add(jwksRefreshInterval)
	for i := 0; i < 5; i++ {
		assertAIStatus(t, r, http.MethodGet, "/api/v1/ai/test", "Bearer "+newToken, http.StatusUnauthorized)
	}
	if calls.Load() != 3 {
		t.Fatalf("JWKS calls = %d", calls.Load())
	}
	stage.Store(1)
	now = now.Add(jwksRefreshInterval)
	assertAIStatus(t, r, http.MethodGet, "/api/v1/ai/test", "Bearer "+newToken, http.StatusNoContent)
}

func TestJWKSRejectsAmbiguousAndInvalidKeys(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"duplicate", "modulus", "exponent", "empty"} {
		t.Run(kind, func(t *testing.T) {
			doc := jwksForTest(map[string]*rsa.PublicKey{"key": &key.PublicKey})
			items := doc["keys"].([]map[string]any)
			switch kind {
			case "duplicate":
				doc["keys"] = append(items, items[0])
			case "modulus":
				items[0]["n"] = "AQ"
			case "exponent":
				items[0]["e"] = "Ag"
			case "empty":
				doc["keys"] = []any{}
			}
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(doc) }))
			defer s.Close()
			if _, err := NewJWTVerifier(JWTAuthConfig{JWKSURL: s.URL}, s.Client()); err == nil {
				t.Fatal("invalid JWKS accepted")
			}
		})
	}
}

func assertAIStatus(t *testing.T, router http.Handler, method, path, authorization string, want int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != want {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, want, recorder.Body.String())
	}
	return recorder
}

func signAITestToken(t *testing.T, key *rsa.PrivateKey, kid, issuer, audience, subject string, expires, issuedAt time.Time) string {
	t.Helper()
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer: issuer, Subject: subject, Audience: jwt.ClaimStrings{audience}, ExpiresAt: jwt.NewNumericDate(expires), IssuedAt: jwt.NewNumericDate(issuedAt),
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func jwksForTest(keys map[string]*rsa.PublicKey) map[string]any {
	items := make([]map[string]any, 0, len(keys))
	for kid, key := range keys {
		items = append(items, map[string]any{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": kid,
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		})
	}
	return map[string]any{"keys": items}
}
