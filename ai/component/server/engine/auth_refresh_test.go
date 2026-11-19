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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Signal when a request has reached the select waiting for the shared refresh.
type refreshWaitContext struct {
	context.Context
	once    sync.Once
	waiting chan struct{}
}

func (c *refreshWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func awaitRefreshSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for refresh synchronization")
	}
}

func TestJWKSRefreshCancellationIsIsolated(t *testing.T) {
	for _, cancelInitiator := range []bool{true, false} {
		name := "waiter"
		if cancelInitiator {
			name = "initiator"
		}
		t.Run(name, func(t *testing.T) {
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}
			started, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			var upstreamCancelled atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 2 {
					close(started)
					select {
					case <-release:
					case <-r.Context().Done():
						upstreamCancelled.Store(true)
						return
					}
				}
				_ = json.NewEncoder(w).Encode(jwksForTest(map[string]*rsa.PublicKey{"known": &key.PublicKey}))
			}))
			defer server.Close()
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			defer unblock()
			v, err := NewJWTVerifier(JWTAuthConfig{JWKSURL: server.URL, Issuer: "admin", Audience: "ai"}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			router.Use(v.Middleware())
			router.GET("/api/v1/ai/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			token := signAITestToken(t, key, "known", "admin", "ai", "user", time.Now().Add(time.Hour), time.Now())
			expireJWKSWindow(v)
			startRequest := func() (context.CancelFunc, <-chan struct{}, <-chan struct{}, *httptest.ResponseRecorder) {
				ctx, cancel := context.WithCancel(context.Background())
				observed := &refreshWaitContext{Context: ctx, waiting: make(chan struct{})}
				req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/test", nil).WithContext(observed)
				req.Header.Set("Authorization", "Bearer "+token)
				resp := httptest.NewRecorder()
				done := make(chan struct{})
				go func() { router.ServeHTTP(resp, req); close(done) }()
				return cancel, observed.waiting, done, resp
			}
			cancelA, _, doneA, respA := startRequest()
			defer cancelA()
			awaitRefreshSignal(t, started)
			cancelB, waitingB, doneB, respB := startRequest()
			defer cancelB()
			awaitRefreshSignal(t, waitingB)
			cancelledDone, survivorDone := doneB, doneA
			cancelledResp, survivorResp := respB, respA
			if cancelInitiator {
				cancelA()
				cancelledDone, survivorDone = doneA, doneB
				cancelledResp, survivorResp = respA, respB
			} else {
				cancelB()
			}
			awaitRefreshSignal(t, cancelledDone)
			if cancelledResp.Code != http.StatusUnauthorized {
				t.Fatalf("cancelled request status = %d", cancelledResp.Code)
			}
			unblock()
			awaitRefreshSignal(t, survivorDone)
			if survivorResp.Code != http.StatusNoContent {
				t.Fatalf("surviving request status = %d; body = %s", survivorResp.Code, survivorResp.Body.String())
			}
			if upstreamCancelled.Load() {
				t.Fatal("request cancellation cancelled the shared fetch")
			}
			for i := 0; i < 3; i++ {
				assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+token, http.StatusNoContent)
			}
			if calls.Load() != 2 {
				t.Fatalf("JWKS calls = %d, want startup plus one shared refresh", calls.Load())
			}
		})
	}
}

func TestJWKSRefreshFailureGraceAndRecovery(t *testing.T) {
	for _, failure := range []string{"unavailable", "timeout", "invalid document"} {
		t.Run(failure, func(t *testing.T) {
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}
			var failing atomic.Bool
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if failing.Load() {
					switch failure {
					case "unavailable":
						w.WriteHeader(http.StatusServiceUnavailable)
					case "timeout":
						<-r.Context().Done()
					case "invalid document":
						_, _ = w.Write([]byte(`{"keys":[{"kty":"RSA"`))
					}
					return
				}
				_ = json.NewEncoder(w).Encode(jwksForTest(map[string]*rsa.PublicKey{"known": &key.PublicKey}))
			}))
			defer server.Close()
			client := server.Client()
			client.Timeout = 100 * time.Millisecond
			var clock atomic.Int64
			clock.Store(time.Now().UnixNano())
			v := &JWTVerifier{config: JWTAuthConfig{Issuer: "admin", Audience: "ai"}, client: NewJWKSClient(server.URL, client), now: func() time.Time { return time.Unix(0, clock.Load()) }}
			if err := v.refresh(context.Background()); err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			router.Use(v.Middleware())
			router.GET("/api/v1/ai/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			valid := signAITestToken(t, key, "known", "admin", "ai", "user", time.Now().Add(time.Hour), time.Now())
			unknown := signAITestToken(t, key, "unknown", "admin", "ai", "user", time.Now().Add(time.Hour), time.Now())
			expired := signAITestToken(t, key, "known", "admin", "ai", "user", time.Now().Add(-time.Minute), time.Now().Add(-time.Hour))
			failing.Store(true)
			clock.Add(int64(30 * time.Second))
			// The request that encounters the failure must also use the bounded fallback.
			assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+valid, http.StatusNoContent)
			assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+expired, http.StatusUnauthorized)
			var wg sync.WaitGroup
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+valid, http.StatusNoContent)
					assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+unknown, http.StatusUnauthorized)
				}()
			}
			wg.Wait()
			if calls.Load() != 2 {
				t.Fatalf("JWKS calls during backoff = %d, want 2", calls.Load())
			}
			// Another failure must not extend the original absolute expiry.
			clock.Add(int64(119 * time.Second))
			assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+valid, http.StatusNoContent)
			clock.Add(int64(time.Second))
			assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+valid, http.StatusUnauthorized)
			if calls.Load() != 3 {
				t.Fatalf("JWKS calls at hard expiry during backoff = %d, want 3", calls.Load())
			}
			failing.Store(false)
			// Five seconds after the last failure, recovery is allowed (not 30 seconds).
			clock.Add(int64(4 * time.Second))
			assertAIStatus(t, router, http.MethodGet, "/api/v1/ai/test", "Bearer "+valid, http.StatusNoContent)
			if calls.Load() != 4 {
				t.Fatalf("JWKS calls after recovery = %d, want 4", calls.Load())
			}
		})
	}
}

// An injected client may have no Timeout; the verifier must still bound its fetch.
type jwksRoundTripFunc func(*http.Request) (*http.Response, error)

func (f jwksRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestJWKSRefreshSetsIndependentDeadline(t *testing.T) {
	var hasDeadline bool
	var remaining time.Duration
	client := &http.Client{Transport: jwksRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		hasDeadline = ok
		remaining = time.Until(deadline)
		return nil, context.DeadlineExceeded
	})}
	v := &JWTVerifier{client: NewJWKSClient("http://jwks.test", client)}
	if err := v.refresh(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("refresh error = %v, want deadline exceeded", err)
	}
	if !hasDeadline || remaining <= 0 || remaining > 10*time.Second {
		t.Fatalf("shared fetch deadline: present=%v, remaining=%v", hasDeadline, remaining)
	}
	// An initial failure must never authenticate a caller without any trusted keys.
	token := jwt.New(jwt.SigningMethodRS256)
	token.Header["kid"] = "unknown"
	if _, err := v.keyForToken(context.Background(), token); err == nil {
		t.Fatal("accepted token without a successfully loaded JWKS")
	}
}
