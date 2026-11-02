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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/server/engine/session"
	"dubbo-admin-ai/component/server/engine/sse"
	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"
	"dubbo-admin-ai/testutils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type captureAgent struct {
	ctx   context.Context
	input *schema.UserInput
}

type failingAgent struct{ err error }

type panicAgent struct{}

func (*panicAgent) Interact(context.Context, *schema.UserInput, string) *agent.Channels {
	panic("injected agent panic")
}

type nilChannelsAgent struct{}

func (*nilChannelsAgent) Interact(context.Context, *schema.UserInput, string) *agent.Channels {
	return nil
}

type invalidChannelsAgent struct{}

func (*invalidChannelsAgent) Interact(context.Context, *schema.UserInput, string) *agent.Channels {
	return &agent.Channels{}
}

type touchFailStore struct {
	conversationstore.BackendStore
	err error
}

func (s *touchFailStore) Touch(context.Context, security.AccessContext, string, time.Time, ...time.Duration) error {
	return s.err
}

func (a *failingAgent) Interact(context.Context, *schema.UserInput, string) *agent.Channels {
	channels := agent.NewChannels(1)
	channels.ErrorChan <- a.err
	return channels
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

func TestStreamChatContextContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(previousPropagator) })
	capturedAgent := &captureAgent{}
	store := memorystore.NewMemoryStore(2)
	sessionManager := session.NewManager(store, store)
	t.Cleanup(func() { _ = sessionManager.Close() })
	access := testutils.TestAccess()
	created, err := sessionManager.CreateSession(context.Background(), access)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAgentHandler(capturedAgent, sessionManager)
	router := gin.New()
	router.POST("/api/v1/ai/chat/stream", handler.StreamChat)

	requestBody := map[string]any{
		"message":   "hello",
		"sessionID": created.ID,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat/stream", bytes.NewReader(body))
	requestCtx, cancel := context.WithCancel(context.WithValue(request.Context(), requestContextKey{}, "trace-value"))
	defer cancel()
	request = request.WithContext(testutils.ContextWithTestAccess(requestCtx))
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
}

func TestStreamChatRequiresAndEnforcesIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := memorystore.NewMemoryStore(2)
	manager := session.NewManager(store, store)
	t.Cleanup(func() { _ = manager.Close() })
	access := testutils.TestAccess()
	created, err := manager.CreateSession(context.Background(), access)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAgentHandler(&captureAgent{}, manager, store)
	router := gin.New()
	router.POST("/chat", handler.StreamChat)
	body := []byte(`{"message":"hello","sessionID":"` + created.ID + `"}`)

	send := func(key string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
		request = request.WithContext(testutils.ContextWithTestAccess(request.Context()))
		request.Header.Set("Content-Type", "application/json")
		if key != "" {
			request.Header.Set("Idempotency-Key", key)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	if response := send(""); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "IDEMPOTENCY_KEY_REQUIRED") {
		t.Fatalf("missing key response = %d %s", response.Code, response.Body.String())
	}
	first := send("request-1")
	if first.Code != http.StatusOK || first.Header().Get("X-Interaction-ID") == "" {
		t.Fatalf("first response = %d %s, interaction=%q", first.Code, first.Body.String(), first.Header().Get("X-Interaction-ID"))
	}
	retry := send("request-1")
	if retry.Code != http.StatusConflict || !strings.Contains(retry.Body.String(), "INTERACTION_IN_PROGRESS") {
		t.Fatalf("retry response = %d %s", retry.Code, retry.Body.String())
	}
	concurrent := send("request-2")
	if concurrent.Code != http.StatusConflict || !strings.Contains(concurrent.Body.String(), "SESSION_BUSY") {
		t.Fatalf("concurrent response = %d %s", concurrent.Code, concurrent.Body.String())
	}
}

func TestStreamChatReleasesPreparedInteractionWhenTouchFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	backend := memorystore.NewMemoryStore(2)
	store := &touchFailStore{BackendStore: backend, err: errors.New("touch failed")}
	manager := session.NewManager(store, store)
	t.Cleanup(func() { _ = manager.Close() })
	access := testutils.TestAccess()
	created := testutils.TestSession("touch-failure-session", time.Now())
	if err := backend.Create(context.Background(), access, created); err != nil {
		t.Fatal(err)
	}
	handler := NewAgentHandler(&captureAgent{}, manager, backend)
	router := gin.New()
	router.POST("/chat", handler.StreamChat)
	body := []byte(`{"message":"hello","sessionID":"` + created.ID + `"}`)
	request := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
	request = request.WithContext(testutils.ContextWithTestAccess(request.Context()))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "touch-failure")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	retried, err := backend.PrepareInteraction(context.Background(), access, created.ID, "touch-failure", hashChatRequest(ChatRequest{Message: "hello", SessionID: created.ID}), time.Minute)
	if err != nil {
		t.Fatalf("PrepareInteraction() after Touch failure error = %v", err)
	}
	if err := backend.FailInteraction(context.Background(), access, created.ID, retried.InteractionID, retried.TurnID); err != nil {
		t.Fatal(err)
	}
}

func TestStreamChatReleasesPreparedInteractionWhenSSEInitializationFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	backend := memorystore.NewMemoryStore(2)
	manager := session.NewManager(backend, backend)
	t.Cleanup(func() { _ = manager.Close() })
	access := testutils.TestAccess()
	created, err := manager.CreateSession(context.Background(), access)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAgentHandler(&captureAgent{}, manager, backend)
	handler.newStreamWriter = func(*gin.Context) (*sse.StreamWriter, error) {
		return nil, errors.New("streaming unsupported")
	}
	router := gin.New()
	router.POST("/chat", handler.StreamChat)
	body := []byte(`{"message":"hello","sessionID":"` + created.ID + `"}`)
	request := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
	request = request.WithContext(testutils.ContextWithTestAccess(request.Context()))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "sse-failure")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	retried, err := backend.PrepareInteraction(context.Background(), access, created.ID, "sse-failure", hashChatRequest(ChatRequest{Message: "hello", SessionID: created.ID}), time.Minute)
	if err != nil {
		t.Fatalf("PrepareInteraction() after SSE failure error = %v", err)
	}
	if err := backend.FailInteraction(context.Background(), access, created.ID, retried.InteractionID, retried.TurnID); err != nil {
		t.Fatal(err)
	}
}

func TestStreamChatRetainsInteractionOwnershipOnSynchronousAgentFailure(t *testing.T) {
	tests := []struct {
		name  string
		agent agent.Agent
	}{
		{name: "panic", agent: &panicAgent{}},
		{name: "nil channels", agent: &nilChannelsAgent{}},
		{name: "invalid channels", agent: &invalidChannelsAgent{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			backend := memorystore.NewMemoryStore(2)
			manager := session.NewManager(backend, backend)
			t.Cleanup(func() { _ = manager.Close() })
			access := testutils.TestAccess()
			created, err := manager.CreateSession(context.Background(), access)
			if err != nil {
				t.Fatal(err)
			}

			handler := NewAgentHandler(tt.agent, manager, backend)
			router := gin.New()
			router.POST("/chat", handler.StreamChat)
			requestBody := ChatRequest{Message: "hello", SessionID: created.ID}
			body, err := json.Marshal(requestBody)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
			request = request.WithContext(testutils.ContextWithTestAccess(request.Context()))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", "sync-agent-failure")
			router.ServeHTTP(httptest.NewRecorder(), request)

			retried, err := backend.PrepareInteraction(context.Background(), access, created.ID, "sync-agent-failure", hashChatRequest(requestBody), time.Minute)
			if err != nil {
				t.Fatalf("PrepareInteraction() after synchronous Agent failure error = %v", err)
			}
			if err := backend.FailInteraction(context.Background(), access, created.ID, retried.InteractionID, retried.TurnID); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStreamChatDoesNotExposeAgentError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "provider-secret-value"
	store := memorystore.NewMemoryStore(2)
	manager := session.NewManager(store, store)
	t.Cleanup(func() { _ = manager.Close() })
	access := testutils.TestAccess()
	created, err := manager.CreateSession(context.Background(), access)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAgentHandler(&failingAgent{err: errors.New("upstream token=" + secret)}, manager)
	router := gin.New()
	router.POST("/chat", handler.StreamChat)
	body, err := json.Marshal(map[string]any{"message": "hello", "sessionID": created.ID})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(body))
	request = request.WithContext(testutils.ContextWithTestAccess(request.Context()))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if strings.Contains(response.Body.String(), secret) || !strings.Contains(response.Body.String(), "Agent interaction failed") {
		t.Fatalf("unsafe SSE error response: %s", response.Body.String())
	}
}

func TestStreamChatDoesNotExposeJSONBindingError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAgentHandler(nil, nil)
	router := gin.New()
	router.POST("/chat", handler.StreamChat)
	request := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"message":`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), clientInvalidRequest) || strings.Contains(response.Body.String(), "unexpected EOF") {
		t.Fatalf("unsafe binding error response: %s", response.Body.String())
	}
}

func TestDrainDoesNotExposeAgentError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodGet, "/chat", nil)
	writer, err := sse.NewStreamWriter(c)
	if err != nil {
		t.Fatal(err)
	}
	stream := sse.NewStreamHandler(writer, "session")
	channels := agent.NewChannels(1)
	const secret = "drain-provider-secret"
	channels.ErrorChan <- errors.New("provider token=" + secret)
	channels.Close()

	NewAgentHandler(nil, nil).drainAndFinish(stream, channels, "session")
	if strings.Contains(response.Body.String(), secret) || !strings.Contains(response.Body.String(), clientAgentError) {
		t.Fatalf("unsafe drain error response: %s", response.Body.String())
	}
}

type failMessageDeltaWriter struct {
	gin.ResponseWriter
	failed bool
}

func (w *failMessageDeltaWriter) Write(data []byte) (int, error) {
	if !w.failed && bytes.Contains(data, []byte("message_delta")) {
		w.failed = true
		return 0, errors.New("provider token=finish-provider-secret")
	}
	return w.ResponseWriter.Write(data)
}

func (w *failMessageDeltaWriter) WriteString(value string) (int, error) {
	return w.Write([]byte(value))
}

func TestMessageDeltaDoesNotExposeFinishError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodGet, "/chat", nil)
	c.Writer = &failMessageDeltaWriter{ResponseWriter: c.Writer}
	writer, err := sse.NewStreamWriter(c)
	if err != nil {
		t.Fatal(err)
	}
	stream := sse.NewStreamHandler(writer, "session")
	NewAgentHandler(nil, nil).MessageDelta(stream, schema.Observation{})
	if strings.Contains(response.Body.String(), "finish-provider-secret") || !strings.Contains(response.Body.String(), clientFinishStreamError) {
		t.Fatalf("unsafe finish error response: %s", response.Body.String())
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

func TestStreamChatDrainsChannelsOnPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	blocked := &blockingAgent{}
	store := memorystore.NewMemoryStore(2)
	sessionManager := session.NewManager(store, store)
	t.Cleanup(func() { _ = sessionManager.Close() })
	access := testutils.TestAccess()
	created, err := sessionManager.CreateSession(context.Background(), access)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAgentHandler(blocked, sessionManager)
	router := gin.New()
	router.POST("/api/v1/ai/chat/stream", func(c *gin.Context) {
		c.Writer = &panicOnTextWriter{ResponseWriter: c.Writer}
		handler.StreamChat(c)
	})

	body, err := json.Marshal(map[string]any{"message": "hello", "sessionID": created.ID})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat/stream", bytes.NewReader(body))
	request = request.WithContext(testutils.ContextWithTestAccess(request.Context()))
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
