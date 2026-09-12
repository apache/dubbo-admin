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

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	consoleauth "github.com/apache/dubbo-admin/pkg/console/auth"
)

type handlerProvider struct {
	authenticated int
}

func (p *handlerProvider) ID() string                   { return "github" }
func (p *handlerProvider) DisplayName() string          { return "GitHub" }
func (p *handlerProvider) IconURL() string              { return "/admin/auth-providers/github.svg" }
func (p *handlerProvider) NeedsNonce() bool             { return false }
func (p *handlerProvider) PostLoginRedirectURL() string { return "https://admin.example/admin/" }
func (p *handlerProvider) AuthorizationURL(tx consoleauth.OAuthTransaction) string {
	return "https://provider.example/authorize?state=" + url.QueryEscape(tx.State)
}
func (p *handlerProvider) Authenticate(_ context.Context, _, _, _ string) (consoleauth.Principal, error) {
	p.authenticated++
	return consoleauth.Principal{Subject: "github:123", Username: "octocat", Groups: []string{}, Roles: []string{}, AuthType: "oauth", Provider: "github"}, nil
}

func TestAuthHandlerPasswordLoginAndUserInfo(t *testing.T) {
	cfg := &configauth.Config{Methods: []string{configauth.MethodPassword}, User: "admin", Password: "secret", ExpirationTime: 3600}
	service, _ := consoleauth.NewServiceFromProviders()
	handler := newAuthHandler(cfg, service)
	router := authTestRouter(handler)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("user=admin&password=secret"))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginResp.Code, loginResp.Body.String())
	}
	cookie := loginResp.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.MaxAge != 3600 {
		t.Fatalf("session cookie = %+v", cookie)
	}

	userinfo := doAuthRequest(router, http.MethodGet, "/api/v1/auth/userinfo", cookie)
	if userinfo.Code != http.StatusOK || !strings.Contains(userinfo.Body.String(), `"subject":"local:admin"`) || !strings.Contains(userinfo.Body.String(), `"authType":"password"`) {
		t.Fatalf("userinfo status = %d, body = %s", userinfo.Code, userinfo.Body.String())
	}
}

func TestAuthHandlerProviderListDoesNotLeakConfiguration(t *testing.T) {
	cfg := &configauth.Config{Methods: []string{configauth.MethodPassword}, Providers: map[string]configauth.ProviderConfig{
		"github": {Type: configauth.ProviderTypeGitHub, DisplayName: "GitHub", IconURL: "/admin/auth-providers/github.svg", ClientID: "client", ClientSecret: "top-secret", RedirectURL: "https://admin.example/api/v1/auth/providers/github/callback", PostLoginRedirectURL: "https://admin.example/admin/", Scopes: []string{"read:user"}},
	}}
	service, err := consoleauth.NewService(context.Background(), cfg.Providers, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := newAuthHandler(cfg, service)
	router := authTestRouter(handler)
	resp := doAuthRequest(router, http.MethodGet, "/api/v1/auth/providers", nil)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"id":"github"`) {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), `"iconUrl":"/admin/auth-providers/github.svg"`) {
		t.Fatalf("provider response missing icon URL: %s", resp.Body.String())
	}
	for _, secret := range []string{"top-secret", "client", "redirectUrl", "scopes", "issuer"} {
		if strings.Contains(resp.Body.String(), secret) {
			t.Fatalf("provider response leaked %q: %s", secret, resp.Body.String())
		}
	}
}

func TestAuthHandlerPasswordOnlyProviderListUsesEmptyArray(t *testing.T) {
	service, err := consoleauth.NewServiceFromProviders()
	if err != nil {
		t.Fatal(err)
	}
	handler := newAuthHandler(
		&configauth.Config{Methods: []string{configauth.MethodPassword}},
		service,
	)
	resp := doAuthRequest(authTestRouter(handler), http.MethodGet, "/api/v1/auth/providers", nil)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"providers":[]`) {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
}

func TestAuthHandlerProviderOnlyMethodListUsesEmptyArray(t *testing.T) {
	service, err := consoleauth.NewServiceFromProviders()
	if err != nil {
		t.Fatal(err)
	}
	handler := newAuthHandler(&configauth.Config{Methods: []string{}}, service)
	resp := doAuthRequest(authTestRouter(handler), http.MethodGet, "/api/v1/auth/providers", nil)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"methods":[]`) {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
}

func TestAuthHandlerOAuthCallbackConsumesTransaction(t *testing.T) {
	provider := &handlerProvider{}
	service, _ := consoleauth.NewServiceFromProviders(provider)
	cfg := &configauth.Config{Providers: map[string]configauth.ProviderConfig{"github": {}}}
	handler := newAuthHandler(cfg, service)
	router := authTestRouter(handler)

	login := doAuthRequest(router, http.MethodGet, "/api/v1/auth/providers/github/login", nil)
	if login.Code != http.StatusFound {
		t.Fatalf("login status = %d, body = %s", login.Code, login.Body.String())
	}
	cookie := login.Result().Cookies()[0]
	redirect, _ := url.Parse(login.Header().Get("Location"))
	callbackURL := "/api/v1/auth/providers/github/callback?code=valid&state=" + url.QueryEscape(redirect.Query().Get("state"))
	callback := doAuthRequest(router, http.MethodGet, callbackURL, cookie)
	if callback.Code != http.StatusFound || callback.Header().Get("Location") != "https://admin.example/admin/" || provider.authenticated != 1 {
		t.Fatalf("callback status = %d location = %q calls = %d body = %s", callback.Code, callback.Header().Get("Location"), provider.authenticated, callback.Body.String())
	}
	// Replaying the original cookie must fail even though it still contains the transaction.
	replay := doAuthRequest(router, http.MethodGet, callbackURL, cookie)
	if replay.Code != http.StatusBadRequest || provider.authenticated != 1 {
		t.Fatalf("replay status = %d calls = %d body = %s", replay.Code, provider.authenticated, replay.Body.String())
	}
}

func TestWriteProviderErrorMapsFullTransactionStoreToTooManyRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	writeProviderError(ctx, consoleauth.ErrTransactionStoreFull)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
	}
}

func authTestRouter(authHandler *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	store.Options(sessions.Options{Path: "/", MaxAge: 3600, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	r.Use(sessions.Sessions("session", store))
	r.Use(consoleauth.SessionMiddleware())
	auth := r.Group("/api/v1/auth")
	auth.POST("/login", authHandler.Login)
	auth.POST("/logout", authHandler.Logout)
	auth.GET("/providers", authHandler.Providers)
	auth.GET("/providers/:provider/login", authHandler.ProviderLogin)
	auth.GET("/providers/:provider/callback", authHandler.ProviderCallback)
	auth.GET("/userinfo", authHandler.UserInfo)
	return r
}

func doAuthRequest(router http.Handler, method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}
