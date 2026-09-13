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

package sse

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSSEHandlerTextStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stream", func(c *gin.Context) {
		writer, err := NewStreamWriter(c)
		if err != nil {
			t.Errorf("NewStreamWriter() error = %v", err)
			return
		}
		handler := NewStreamHandler(writer, "session")
		if err := handler.HandleText("hello", 0); err != nil {
			t.Errorf("HandleText() error = %v", err)
		}
		if err := handler.HandleContentBlockStop(0); err != nil {
			t.Errorf("HandleContentBlockStop() error = %v", err)
		}
		if err := handler.FinishStream(); err != nil {
			t.Errorf("FinishStream() error = %v", err)
		}
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/stream", nil)
	router.ServeHTTP(recorder, request)

	body := recorder.Body.String()
	for _, event := range []string{"message_start", "content_block_start", "content_block_delta", "content_block_stop", "message_stop"} {
		if !strings.Contains(body, "event: "+event) {
			t.Fatalf("SSE body = %q, missing event %q", body, event)
		}
	}
	if recorder.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", recorder.Header().Get("Content-Type"))
	}
}

func TestStreamWriterRejectsCanceledContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	router := gin.New()
	router.GET("/stream", func(c *gin.Context) {
		c.Request = c.Request.WithContext(ctx)
		writer, err := NewStreamWriter(c)
		if err != nil {
			t.Errorf("NewStreamWriter() error = %v", err)
			return
		}
		if err := writer.WriteMessageStop(); err == nil {
			t.Error("WriteMessageStop() succeeded, want context cancellation error")
		}
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/stream", nil))
}
