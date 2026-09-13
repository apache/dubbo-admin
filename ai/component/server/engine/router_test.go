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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddlewareAllowsTracePropagationHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodOptions, "/test", nil)
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
