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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dubbo-admin-ai/security"
	memorystore "dubbo-admin-ai/store/memory"

	"github.com/gin-gonic/gin"
)

type rejectingAuthenticator struct{}

func (rejectingAuthenticator) Authenticate(*http.Request) (security.AccessContext, error) {
	return security.AccessContext{}, security.ErrAuthenticationRequired
}

type systemAuthenticator struct{}

func (systemAuthenticator) Authenticate(*http.Request) (security.AccessContext, error) {
	return security.SystemAccessContext(), nil
}

func TestCORSMiddlewareAllowsTracePropagationHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware([]string{"http://trusted.example"}))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodOptions, "/test", nil)
	request.Header.Set("Origin", "http://trusted.example")
	request.Header.Set("Access-Control-Request-Headers", "traceparent,tracestate,baggage")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", response.Code, http.StatusNoContent)
	}
	allowed := strings.ToLower(response.Header().Get("Access-Control-Allow-Headers"))
	for _, header := range []string{"traceparent", "tracestate", "baggage"} {
		if !strings.Contains(allowed, header) {
			t.Fatalf("Access-Control-Allow-Headers = %q, missing %q", allowed, header)
		}
	}
	if exposed := response.Header().Get("Access-Control-Expose-Headers"); exposed != "X-Trace-ID" {
		t.Fatalf("Access-Control-Expose-Headers = %q, want X-Trace-ID", exposed)
	}
}

func TestSessionRoutesIgnoreSpoofedIdentityAndIsolateOwners(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := memorystore.NewMemoryStore()
	ownerAuth, err := security.NewStaticAuthenticator(security.Subject{
		TenantID: "tenant-a", UserID: "owner-a", AuthMethod: security.AuthMethodStatic,
	})
	if err != nil {
		t.Fatal(err)
	}
	ownerRouter, err := NewRouter(nil, store, store, ownerAuth)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ownerRouter.Close() })

	create := httptest.NewRequest(http.MethodPost, "/api/v1/ai/sessions", nil)
	create.Header.Set("X-Tenant-ID", "attacker-tenant")
	create.Header.Set("X-User-ID", "attacker-user")
	createdResponse := httptest.NewRecorder()
	ownerRouter.GetEngine().ServeHTTP(createdResponse, create)
	if createdResponse.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", createdResponse.Code, createdResponse.Body.String())
	}
	var created struct {
		Data struct {
			SessionID string `json:"session_id"`
			TenantID  string `json:"tenant_id"`
			OwnerID   string `json:"owner_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Data.TenantID != "tenant-a" || created.Data.OwnerID != "owner-a" {
		t.Fatalf("created identity = %s/%s", created.Data.TenantID, created.Data.OwnerID)
	}

	otherAuth, err := security.NewStaticAuthenticator(security.Subject{
		TenantID: "tenant-a", UserID: "owner-b", AuthMethod: security.AuthMethodStatic,
	})
	if err != nil {
		t.Fatal(err)
	}
	otherRouter, err := NewRouter(nil, store, store, otherAuth)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = otherRouter.Close() })
	get := httptest.NewRequest(http.MethodGet, "/api/v1/ai/sessions/"+created.Data.SessionID, nil)
	getResponse := httptest.NewRecorder()
	otherRouter.GetEngine().ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusNotFound {
		t.Fatalf("cross-owner status = %d, body = %s", getResponse.Code, getResponse.Body.String())
	}
	var denied Response
	if err := json.Unmarshal(getResponse.Body.Bytes(), &denied); err != nil {
		t.Fatal(err)
	}
	if denied.Code != "RESOURCE_NOT_ACCESSIBLE" {
		t.Fatalf("cross-owner code = %q", denied.Code)
	}

	list := httptest.NewRequest(http.MethodGet, "/api/v1/ai/sessions", nil)
	listResponse := httptest.NewRecorder()
	otherRouter.GetEngine().ServeHTTP(listResponse, list)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), `"total":0`) {
		t.Fatalf("cross-owner list status = %d, body = %s", listResponse.Code, listResponse.Body.String())
	}

	chatBody := []byte(`{"message":"hello","sessionID":"` + created.Data.SessionID + `"}`)
	chat := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat/stream", bytes.NewReader(chatBody))
	chat.Header.Set("Content-Type", "application/json")
	chatResponse := httptest.NewRecorder()
	otherRouter.GetEngine().ServeHTTP(chatResponse, chat)
	if chatResponse.Code != http.StatusBadRequest || !strings.Contains(chatResponse.Body.String(), "RESOURCE_NOT_ACCESSIBLE") {
		t.Fatalf("cross-owner chat status = %d, body = %s", chatResponse.Code, chatResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/ai/sessions/"+created.Data.SessionID, nil)
	deleteResponse := httptest.NewRecorder()
	otherRouter.GetEngine().ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNotFound || !strings.Contains(deleteResponse.Body.String(), "RESOURCE_NOT_ACCESSIBLE") {
		t.Fatalf("cross-owner delete status = %d, body = %s", deleteResponse.Code, deleteResponse.Body.String())
	}
	if _, err := store.Get(t.Context(), security.MustAccessContext(security.Subject{
		TenantID: "tenant-a", UserID: "owner-a", AuthMethod: security.AuthMethodStatic,
	}), created.Data.SessionID); err != nil {
		t.Fatalf("cross-owner delete modified owner session: %v", err)
	}
}

func TestAuthenticationMiddlewareRejectsMissingTrustedIdentity(t *testing.T) {
	store := memorystore.NewMemoryStore()
	router, err := NewRouter(nil, store, store, rejectingAuthenticator{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = router.Close() })
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ai/sessions", nil)
	response := httptest.NewRecorder()
	router.GetEngine().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var denied Response
	if err := json.Unmarshal(response.Body.Bytes(), &denied); err != nil {
		t.Fatal(err)
	}
	if denied.Code != "AUTHENTICATION_REQUIRED" {
		t.Fatalf("code = %q", denied.Code)
	}
}

func TestNewRouterRequiresExplicitAuthenticator(t *testing.T) {
	store := memorystore.NewMemoryStore()
	if _, err := NewRouter(nil, store, store, nil); err == nil {
		t.Fatal("NewRouter() succeeded without an authenticator")
	}
}

func TestAuthenticationMiddlewareRejectsSystemAccess(t *testing.T) {
	store := memorystore.NewMemoryStore()
	router, err := NewRouter(nil, store, store, systemAuthenticator{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = router.Close() })

	request := httptest.NewRequest(http.MethodGet, "/api/v1/ai/sessions", nil)
	response := httptest.NewRecorder()
	router.GetEngine().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
