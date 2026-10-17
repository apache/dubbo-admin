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

package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/config/app"
	consoleconfig "github.com/apache/dubbo-admin/pkg/config/console"
	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	consoleauth "github.com/apache/dubbo-admin/pkg/console/auth"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
)

func TestInitRouterSeparatesPublicAndProtectedAuthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	r.Use(consoleauth.SessionMiddleware())
	if err := InitRouter(r, routerTestContext{}); err != nil {
		t.Fatal(err)
	}

	assertRouterStatus(t, r, http.MethodGet, "/api/v1/auth/providers", nil, nil, http.StatusOK)
	assertRouterStatus(t, r, http.MethodPost, "/api/v1/auth/logout", nil, nil, http.StatusUnauthorized)

	loginBody := strings.NewReader("user=admin&password=secret")
	login := assertRouterStatus(t, r, http.MethodPost, "/api/v1/auth/login", loginBody, nil, http.StatusOK)
	if len(login.Result().Cookies()) == 0 {
		t.Fatal("login did not set a session cookie")
	}
	assertRouterStatus(t, r, http.MethodGet, "/api/v1/auth/userinfo", nil, login.Result().Cookies()[0], http.StatusOK)
}

type routerTestContext struct {
	consolectx.Context
}

func (routerTestContext) ResourceManager() manager.ResourceManager { return nil }
func (routerTestContext) CounterManager() counter.CounterManager   { return nil }
func (routerTestContext) LockManager() lock.Lock                   { return nil }
func (routerTestContext) AppContext() context.Context              { return context.Background() }
func (routerTestContext) Config() app.AdminConfig {
	return app.AdminConfig{Console: &consoleconfig.Config{Auth: &configauth.Config{
		Methods:        []string{configauth.MethodPassword},
		User:           "admin",
		Password:       "secret",
		ExpirationTime: 3600,
		Providers:      map[string]configauth.ProviderConfig{},
	}}}
}

func assertRouterStatus(
	t *testing.T,
	router http.Handler,
	method string,
	path string,
	body *strings.Reader,
	cookie *http.Cookie,
	want int,
) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != want {
		t.Fatalf("%s %s status = %d, want %d; body = %s", method, path, recorder.Code, want, recorder.Body.String())
	}
	return recorder
}
