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
	"testing"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/component/server/engine/session"
	"dubbo-admin-ai/schema"

	"github.com/gin-gonic/gin"
)

type captureAgent struct {
	input *schema.UserInput
}

func (a *captureAgent) Interact(input *schema.UserInput, _ string) *agent.Channels {
	a.input = input
	channels := agent.NewChannels(1)
	channels.Close()
	return channels
}

func (a *captureAgent) GetMemory() *memory.HistoryMemory {
	return nil
}

func TestStreamChatContextContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name        string
		withContext bool
	}{
		{name: "legacy request without context"},
		{name: "request with context", withContext: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capturedAgent := &captureAgent{}
			sessionManager := session.NewManager()
			handler := NewAgentHandler(capturedAgent, sessionManager)
			router := gin.New()
			router.POST("/api/v1/ai/chat/stream", handler.StreamChat)

			requestBody := map[string]any{
				"message":   "hello",
				"sessionID": "session_test",
			}
			if test.withContext {
				var pageContext any
				if err := json.Unmarshal(validContextJSON(t), &pageContext); err != nil {
					t.Fatalf("unmarshal fixture: %v", err)
				}
				requestBody["context"] = pageContext
			}
			body, err := json.Marshal(requestBody)
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}

			request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat/stream", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "text/event-stream" {
				t.Fatalf("Content-Type = %q, want text/event-stream", contentType)
			}
			if capturedAgent.input == nil {
				t.Fatal("agent did not receive user input")
			}
			if !test.withContext {
				if capturedAgent.input.Context != nil {
					t.Fatalf("legacy request context = %#v, want nil", capturedAgent.input.Context)
				}
				return
			}
			if capturedAgent.input.Context == nil {
				t.Fatal("agent did not receive page context")
			}
			if capturedAgent.input.Context.State.Filters["password"] != redactedContextValue {
				t.Fatalf("agent received unsanitized context: %#v", capturedAgent.input.Context.State.Filters)
			}
		})
	}
}
