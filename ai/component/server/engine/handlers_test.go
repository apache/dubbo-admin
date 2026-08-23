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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/component/server/engine/session"
	"dubbo-admin-ai/schema"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type captureAgent struct {
	ctx   context.Context
	input *schema.UserInput
}

func TestDiscardAgentOutputUnblocksDetachedProducer(t *testing.T) {
	channels := agent.NewChannels(1)
	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		for i := 0; i < 10; i++ {
			channels.Send(schema.NewStreamFeedback("discard"))
		}
		channels.Close()
	}()
	go discardAgentOutput(channels)

	select {
	case <-producerDone:
	case <-time.After(time.Second):
		t.Fatal("detached producer remained blocked on response channels")
	}
}

func (a *captureAgent) Interact(ctx context.Context, input *schema.UserInput, _ string) *agent.Channels {
	a.ctx = ctx
	a.input = input
	channels := agent.NewChannels(1)
	channels.SetTraceID("trace-test")
	channels.Close()
	return channels
}

type requestContextKey struct{}

func (a *captureAgent) GetMemory() *memory.HistoryMemory {
	return nil
}

func TestStreamChatContextContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(previousPropagator) })
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
			requestCtx, cancel := context.WithCancel(context.WithValue(request.Context(), requestContextKey{}, "trace-value"))
			defer cancel()
			request = request.WithContext(requestCtx)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("traceparent", "00-00000000000000000000000000000001-0000000000000002-01")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "text/event-stream" {
				t.Fatalf("Content-Type = %q, want text/event-stream", contentType)
			}
			if traceID := response.Header().Get("X-Trace-ID"); traceID != "trace-test" {
				t.Fatalf("X-Trace-ID = %q, want trace-test", traceID)
			}
			if capturedAgent.input == nil {
				t.Fatal("agent did not receive user input")
			}
			if capturedAgent.ctx == nil {
				t.Fatal("agent did not receive request context")
			}
			if got := capturedAgent.ctx.Value(requestContextKey{}); got != "trace-value" {
				t.Fatalf("request context value = %v, want trace-value", got)
			}
			if _, ok := capturedAgent.ctx.Deadline(); ok {
				t.Fatal("interaction context retained the request deadline")
			}
			if capturedAgent.ctx.Done() != nil {
				t.Fatal("interaction context retained request cancellation")
			}
			if spanContext := trace.SpanContextFromContext(capturedAgent.ctx); !spanContext.IsValid() || !spanContext.IsRemote() {
				t.Fatalf("agent received invalid inbound span context: %v", spanContext)
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

// panicOnTextWriter fails the first SSE content write so StreamChat panics
// after the interaction has already started.
type panicOnTextWriter struct {
	gin.ResponseWriter
	panicked bool
}

func (w *panicOnTextWriter) Write(data []byte) (int, error) {
	if !w.panicked && bytes.Contains(data, []byte("content_block_delta")) {
		w.panicked = true
		panic("injected write failure")
	}
	return w.ResponseWriter.Write(data)
}

func (w *panicOnTextWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// blockingAgent emits more feedback than the channel buffer holds, so the
// producer stays blocked until someone drains the channels.
type blockingAgent struct {
	channels     *agent.Channels
	producerDone chan struct{}
}

func (a *blockingAgent) Interact(_ context.Context, _ *schema.UserInput, _ string) *agent.Channels {
	a.channels = agent.NewChannels(1)
	a.producerDone = make(chan struct{})
	go func() {
		defer close(a.producerDone)
		for i := 0; i < 8; i++ {
			a.channels.Send(schema.NewStreamFeedback("blocked"))
		}
		a.channels.Close()
	}()
	return a.channels
}

func (a *blockingAgent) GetMemory() *memory.HistoryMemory { return nil }

func TestStreamChatDrainsChannelsOnPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	blocked := &blockingAgent{}
	sessionManager := session.NewManager()
	handler := NewAgentHandler(blocked, sessionManager)
	router := gin.New()
	router.POST("/api/v1/ai/chat/stream", func(c *gin.Context) {
		c.Writer = &panicOnTextWriter{ResponseWriter: c.Writer}
		handler.StreamChat(c)
	})

	body, err := json.Marshal(map[string]any{"message": "hello", "sessionID": "session_test"})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat/stream", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), request)

	if blocked.producerDone == nil {
		t.Fatal("agent interaction never started")
	}
	select {
	case <-blocked.producerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("panic recovery left the detached producer blocked on its channels")
	}
}
