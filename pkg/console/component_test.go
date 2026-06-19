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

package console

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddlewareGatesRuleVersionsWithoutSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("secret"))))
	r.Use((&consoleWebServer{}).authMiddleware())
	r.GET("/api/v1/condition-rule/:ruleName/versions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/condition-rule/demo/versions", nil)
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"code":"Unauthorized","message":"no access, please login","data":null}`, recorder.Body.String())
}

// TestAuthMiddlewareGatesRollbackWithoutSession ensures rollback - a mutating
// endpoint - cannot be invoked without an authenticated session.
func TestAuthMiddlewareGatesRollbackWithoutSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("secret"))))
	r.Use((&consoleWebServer{}).authMiddleware())
	rollbackInvoked := false
	r.POST("/api/v1/condition-rule/:ruleName/versions/:versionId/rollback", func(c *gin.Context) {
		rollbackInvoked = true
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/condition-rule/demo/versions/123/rollback", nil)
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.False(t, rollbackInvoked, "rollback handler must not run for unauthenticated requests")
	require.JSONEq(t, `{"code":"Unauthorized","message":"no access, please login","data":null}`, recorder.Body.String())
}
