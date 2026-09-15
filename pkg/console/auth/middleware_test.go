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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestSessionMiddlewareAndRequireLoginHaveSeparateResponsibilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	r.Use(SessionMiddleware())

	r.GET("/public", func(c *gin.Context) {
		if _, ok := PrincipalFromContext(c); ok {
			t.Fatal("anonymous request unexpectedly has a principal")
		}
		c.Status(http.StatusNoContent)
	})
	r.POST("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		if err := PutPrincipal(session, LocalPrincipal("admin")); err != nil {
			t.Fatal(err)
		}
		if err := session.Save(); err != nil {
			t.Fatal(err)
		}
		c.Status(http.StatusNoContent)
	})
	protected := r.Group("/protected")
	protected.Use(RequireLogin())
	protected.GET("", func(c *gin.Context) {
		principal, ok := PrincipalFromContext(c)
		if !ok || principal.Username != "admin" {
			t.Fatalf("principal = %+v, found = %v", principal, ok)
		}
		c.Status(http.StatusNoContent)
	})

	assertMiddlewareStatus(t, r, http.MethodGet, "/public", nil, http.StatusNoContent)
	assertMiddlewareStatus(t, r, http.MethodGet, "/protected", nil, http.StatusUnauthorized)

	login := httptest.NewRecorder()
	r.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/login", nil))
	if login.Code != http.StatusNoContent || len(login.Result().Cookies()) == 0 {
		t.Fatalf("login status = %d, cookies = %v", login.Code, login.Result().Cookies())
	}
	assertMiddlewareStatus(t, r, http.MethodGet, "/protected", login.Result().Cookies()[0], http.StatusNoContent)
}

func assertMiddlewareStatus(t *testing.T, router http.Handler, method, path string, cookie *http.Cookie, want int) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != want {
		t.Fatalf("%s %s status = %d, want %d; body = %s", method, path, recorder.Code, want, recorder.Body.String())
	}
}
