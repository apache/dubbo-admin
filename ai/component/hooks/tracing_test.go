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

package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTracingHookBuildsNestedSpansAndRecordsErrors(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureNone)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	interactionCtx := manager.Emit(context.Background(), State{
		Event: EventInteractionStart, InteractionID: "interaction", SessionID: "session",
	})
	if TraceIDFromContext(interactionCtx) == "" {
		t.Fatal("interaction start did not create a trace ID")
	}
	modelCtx := manager.Emit(interactionCtx, State{
		Event: EventModelCallStart, InteractionID: "interaction", SessionID: "session", Model: "test/model",
	})
	modelErr := errors.New("model failed")
	manager.Emit(modelCtx, State{
		Event: EventModelCallEnd, InteractionID: "interaction", SessionID: "session", Model: "test/model", Error: modelErr.Error(),
	})
	manager.Emit(interactionCtx, State{
		Event: EventInteractionEnd, InteractionID: "interaction", SessionID: "session", InputTokens: 2, OutputTokens: 3, TotalTokens: 5,
	})

	spans := recorder.Ended()
	if len(spans) != 2 {
		t.Fatalf("ended spans = %d, want 2", len(spans))
	}
	if spans[0].Name() != "chat model" || spans[1].Name() != "invoke_agent "+AgentName {
		t.Fatalf("unexpected span order: %q, %q", spans[0].Name(), spans[1].Name())
	}
	if spans[0].Parent().SpanID() != spans[1].SpanContext().SpanID() {
		t.Fatal("model span is not a child of interaction span")
	}
	assertSpanAttribute(t, spans[0].Attributes(), "gen_ai.operation.name", "chat")
	assertSpanAttribute(t, spans[0].Attributes(), "gen_ai.provider.name", "test")
	assertSpanAttribute(t, spans[0].Attributes(), "gen_ai.request.model", "model")
	if spans[0].SpanKind() != trace.SpanKindClient {
		t.Fatalf("model span kind = %v, want CLIENT", spans[0].SpanKind())
	}
	assertSpanAttribute(t, spans[0].Attributes(), "gen_ai.conversation.id", "session")
	assertSpanAttribute(t, spans[0].Attributes(), "agent.interaction.id", "interaction")
	assertSpanAttribute(t, spans[1].Attributes(), "gen_ai.operation.name", "invoke_agent")
}

func TestTracingHookReportsGenAIAgentIdentity(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureNone)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	interactionCtx := manager.Emit(context.Background(), State{
		Event: EventInteractionStart, InteractionID: "interaction", SessionID: "session",
	})
	toolCtx := manager.Emit(interactionCtx, State{
		Event: EventToolCallStart, InteractionID: "interaction", SessionID: "session",
		ToolName: "lookup", ToolCallID: "call-lookup-1",
	})
	manager.Emit(toolCtx, State{
		Event: EventToolCallEnd, InteractionID: "interaction", SessionID: "session",
		ToolName: "lookup", ToolCallID: "call-lookup-1",
	})
	manager.Emit(interactionCtx, State{
		Event: EventInteractionEnd, InteractionID: "interaction", SessionID: "session",
	})

	spans := recorder.Ended()
	if len(spans) != 2 {
		t.Fatalf("ended spans = %d, want 2", len(spans))
	}
	tool, interaction := spans[0], spans[1]
	if tool.Name() != "execute_tool lookup" {
		t.Fatalf("tool span name = %q, want %q", tool.Name(), "execute_tool lookup")
	}
	if interaction.Name() != "invoke_agent "+AgentName {
		t.Fatalf("interaction span name = %q, want %q", interaction.Name(), "invoke_agent "+AgentName)
	}
	if interaction.SpanKind() != trace.SpanKindInternal {
		t.Fatalf("interaction span kind = %v, want INTERNAL", interaction.SpanKind())
	}
	for _, span := range []sdktrace.ReadOnlySpan{tool, interaction} {
		assertSpanAttribute(t, span.Attributes(), "gen_ai.agent.name", AgentName)
		// The semantic conventions forbid transient identifiers on the agent id.
		assertNoSpanAttribute(t, span.Attributes(), "gen_ai.agent.id")
		assertSpanAttribute(t, span.Attributes(), "gen_ai.conversation.id", "session")
	}
	assertSpanAttribute(t, tool.Attributes(), "gen_ai.operation.name", "execute_tool")
	assertSpanAttribute(t, tool.Attributes(), "gen_ai.tool.name", "lookup")
	assertSpanAttribute(t, tool.Attributes(), "gen_ai.tool.call.id", "call-lookup-1")
}

func TestTracingHookRecordsGenAIExceptionEvent(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureNone)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	modelCtx := manager.Emit(context.Background(), State{
		Event: EventModelCallStart, InteractionID: "interaction", SessionID: "session", Model: "test/model",
	})
	failed := State{
		Event: EventModelCallError, InteractionID: "interaction", SessionID: "session", Model: "test/model",
		Error: "model failed", ErrorType: "example.ModelError",
	}
	manager.Emit(modelCtx, failed)
	failed.Event = EventModelCallEnd
	manager.Emit(modelCtx, failed)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	event := findSpanEvent(t, spans[0], "gen_ai.client.operation.exception")
	assertEventAttribute(t, event, "exception.type", "example.ModelError")
	assertEventAttribute(t, event, "exception.message", "model failed")
	assertSpanAttribute(t, spans[0].Attributes(), "error.type", "example.ModelError")
	if spans[0].Status().Code != codes.Error {
		t.Fatalf("span status = %v, want ERROR", spans[0].Status().Code)
	}
	if countSpanEvents(spans[0], "gen_ai.client.operation.exception") != 1 {
		t.Fatalf("exception events = %d, want 1", countSpanEvents(spans[0], "gen_ai.client.operation.exception"))
	}
	// The standard error event stays in place: the hook records it for every
	// event that carries the error.
	if countSpanEvents(spans[0], "exception") == 0 {
		t.Fatal("the standard error event is missing")
	}
}

func TestTracingHookFallsBackToOtherErrorType(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureNone)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	toolCtx := manager.Emit(context.Background(), State{
		Event: EventToolCallStart, ToolName: "lookup", ToolCallID: "call-lookup-1",
	})
	manager.Emit(toolCtx, State{
		Event: EventToolCallEnd, ToolName: "lookup", ToolCallID: "call-lookup-1",
		Error: "tool failed", Degraded: true,
	})

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	assertSpanAttribute(t, spans[0].Attributes(), "error.type", "_OTHER")
	event := findSpanEvent(t, spans[0], "gen_ai.client.operation.exception")
	assertEventAttribute(t, event, "exception.type", "_OTHER")
	assertEventAttribute(t, event, "exception.message", "tool failed")
}

func TestTracingHookOmitsGenAITotalTokenAttribute(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureNone)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	modelCtx := manager.Emit(context.Background(), State{
		Event: EventModelCallStart, Model: "test/model",
	})
	manager.Emit(modelCtx, State{
		Event: EventModelCallEnd, Model: "test/model",
		InputTokens: 2, OutputTokens: 3, TotalTokens: 5,
	})

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	assertSpanIntAttribute(t, spans[0].Attributes(), "gen_ai.usage.input_tokens", 2)
	assertSpanIntAttribute(t, spans[0].Attributes(), "gen_ai.usage.output_tokens", 3)
	assertNoSpanAttribute(t, spans[0].Attributes(), "gen_ai.usage.total_tokens")
	// The semantic conventions define no total token attribute. The total is
	// derived from the input and output tokens, so no private total is emitted.
	assertNoSpanAttribute(t, spans[0].Attributes(), "agent.usage.total_tokens")
	assertNoSpanAttribute(t, spans[0].Attributes(), "gen_ai.usage.total_tokens")
}

func TestTracingHookOmitsProviderForUnqualifiedModel(t *testing.T) {
	unqualified := stateAttributes(State{Event: EventModelCallStart, Model: "gpt-4o"})
	assertNoSpanAttribute(t, unqualified, "gen_ai.provider.name")
	assertSpanAttribute(t, unqualified, "gen_ai.request.model", "gpt-4o")

	qualified := stateAttributes(State{Event: EventModelCallStart, Model: "openai/gpt-4o"})
	assertSpanAttribute(t, qualified, "gen_ai.provider.name", "openai")
	assertSpanAttribute(t, qualified, "gen_ai.request.model", "gpt-4o")

	nameless := stateAttributes(State{Event: EventModelCallStart, Model: "openai/"})
	assertNoSpanAttribute(t, nameless, "gen_ai.request.model")
}

func TestTracingHookMapsProviderToWellKnownValue(t *testing.T) {
	cases := []struct {
		name     string
		model    string
		provider string
	}{
		{"gemini maps to the well-known gcp.gemini value", "gemini/gemini-pro", "gcp.gemini"},
		{"openai is already a well-known value", "openai/gpt-4o", "openai"},
		{"dashscope is a custom value", "dashscope/qwen-max", "dashscope"},
		{"siliconflow is a custom value", "siliconflow/gpt-4", "siliconflow"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := stateAttributes(State{Event: EventModelCallStart, Model: tc.model})
			assertSpanAttribute(t, attrs, "gen_ai.provider.name", tc.provider)
		})
	}
}

func TestTracingHookKeepsCustomProviderNameVerbatim(t *testing.T) {
	// The semantic conventions allow a custom value for every provider that
	// is not one of the well-known values, so an unmapped provider is neither
	// dropped nor rewritten.
	for _, model := range []string{"dashscope/qwen-max", "siliconflow/gpt-4"} {
		t.Run(model, func(t *testing.T) {
			attrs := stateAttributes(State{Event: EventModelCallStart, Model: model})
			provider, _, _ := strings.Cut(model, "/")
			assertSpanAttribute(t, attrs, "gen_ai.provider.name", provider)
			if _, known := wellKnownGenAIProviders[provider]; known {
				t.Fatalf("provider %q is unexpectedly a well-known value", provider)
			}
		})
	}
}

func TestTracingHookPassesModelNameWithSlashesVerbatim(t *testing.T) {
	// Only the first separator splits the identifier: the model name keeps
	// every remaining slash.
	attrs := stateAttributes(State{Event: EventModelCallStart, Model: "vendor/org/model-name"})
	assertSpanAttribute(t, attrs, "gen_ai.provider.name", "vendor")
	assertSpanAttribute(t, attrs, "gen_ai.request.model", "org/model-name")
}

func TestTracingHookOmitsProviderForModelWithoutSeparator(t *testing.T) {
	attrs := stateAttributes(State{Event: EventModelCallStart, Model: "gpt-4o"})
	assertNoSpanAttribute(t, attrs, "gen_ai.provider.name")
	assertSpanAttribute(t, attrs, "gen_ai.request.model", "gpt-4o")
}

func TestWellKnownGenAIProvidersMatchSemanticConventions(t *testing.T) {
	want := []string{
		"anthropic", "aws.bedrock", "azure.ai.inference", "azure.ai.openai",
		"cohere", "deepseek", "gcp.gemini", "gcp.gen_ai", "gcp.vertex_ai",
		"groq", "ibm.watsonx.ai", "mistral_ai", "moonshot_ai", "openai",
		"perplexity", "x_ai",
	}
	if len(wellKnownGenAIProviders) != len(want) {
		t.Fatalf("well-known providers = %d, want %d", len(wellKnownGenAIProviders), len(want))
	}
	for _, value := range want {
		if _, known := wellKnownGenAIProviders[value]; !known {
			t.Errorf("well-known provider %q is missing from the reference set", value)
		}
	}
	// Every alias must resolve to a well-known value of the conventions.
	for alias, canonical := range genAIProviderAliases {
		if _, known := wellKnownGenAIProviders[canonical]; !known {
			t.Errorf("alias %q resolves to %q, which is not a well-known value", alias, canonical)
		}
	}
}

func TestTracingHookDoesNotMaterializeContentForUnsampledSpan(t *testing.T) {
	provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample()))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureFull)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	calls := 0
	start := State{Event: EventModelCallStart, Model: "test/model"}.WithInputContent(func() any {
		calls++
		return map[string]string{"message": "secret"}
	})
	ctx := manager.Emit(context.Background(), start)
	manager.Emit(ctx, State{Event: EventModelCallEnd, Model: "test/model"})
	if calls != 0 {
		t.Fatalf("unsampled span materialized content %d times, want 0", calls)
	}
}

func TestTracingHookMaterializesContentForRecordingSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureFull)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	calls := 0
	start := State{Event: EventModelCallStart, Model: "test/model"}.WithInputContent(func() any {
		calls++
		return []map[string]any{{"role": "user", "parts": []any{}}}
	})
	ctx := manager.Emit(context.Background(), start)
	manager.Emit(ctx, State{Event: EventModelCallEnd, Model: "test/model"})
	if calls != 1 {
		t.Fatalf("recording span materialized content %d times, want 1", calls)
	}
	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	assertSpanAttribute(t, spans[0].Attributes(), "gen_ai.input.messages", `[{"parts":[],"role":"user"}]`)
}

func TestStateAttributesClassifyToolAndStageEvents(t *testing.T) {
	toolAttrs := stateAttributes(State{
		Event: EventToolCallStart, SessionID: "session", InteractionID: "interaction",
		Model: "test/model", ToolName: "lookup", ToolCallID: "call-lookup-1",
	})
	assertSpanAttribute(t, toolAttrs, "gen_ai.operation.name", "execute_tool")
	assertSpanAttribute(t, toolAttrs, "gen_ai.tool.name", "lookup")
	assertSpanAttribute(t, toolAttrs, "gen_ai.tool.call.id", "call-lookup-1")
	assertNoSpanAttribute(t, toolAttrs, "gen_ai.request.model")

	stageAttrs := stateAttributes(State{
		Event: EventStageStart, SessionID: "session", InteractionID: "interaction", Model: "test/model",
	})
	assertNoSpanAttribute(t, stageAttrs, "gen_ai.operation.name")
	assertNoSpanAttribute(t, stageAttrs, "gen_ai.request.model")
}

func TestTracingHookContentPolicy(t *testing.T) {
	state := State{Event: EventModelCallStart, Input: SnapshotContent(map[string]string{"message": "secret"})}
	if attrs := contentAttributes(state, CaptureNone, true); len(attrs) != 0 {
		t.Fatalf("CaptureNone attributes = %v, want none", attrs)
	}
	state.Input = SnapshotContent([]map[string]any{{
		"role":  "user",
		"parts": []map[string]any{{"type": "text", "content": strings.Repeat("x", truncatedContentLimit+100)}},
	}})
	attrs := contentAttributes(state, CaptureTruncated, true)
	if len(attrs) != 1 || len(attrs[0].Value.AsString()) > truncatedContentLimit+4 {
		t.Fatalf("truncated attribute length = %d", len(attrs[0].Value.AsString()))
	}
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(attrs[0].Value.AsString()), &decoded); err != nil {
		t.Fatalf("truncated semantic content is not valid JSON: %v", err)
	}
}

func TestTracingHookTruncatedConversationKeepsItsFirstTurns(t *testing.T) {
	// A long conversation has to survive the truncation as a shorter
	// conversation. Dropping the whole document would leave an empty list,
	// and a backend reads an empty list as "there was no conversation".
	messages := make([]map[string]any, 0, 32)
	for i := 0; i < 32; i++ {
		messages = append(messages, map[string]any{
			"role":  "user",
			"parts": []map[string]any{{"type": "text", "content": strings.Repeat("x", truncatedJSONStringLimit+100)}},
		})
	}
	state := State{Event: EventModelCallStart, Input: SnapshotContent(messages)}
	attrs := contentAttributes(state, CaptureTruncated, true)
	if len(attrs) != 1 {
		t.Fatalf("truncated attributes = %v, want one", attrs)
	}
	value := attrs[0].Value.AsString()
	if len(value) > truncatedContentLimit {
		t.Fatalf("truncated attribute length = %d, want at most %d", len(value), truncatedContentLimit)
	}
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		t.Fatalf("truncated semantic content is not valid JSON: %v", err)
	}
	if len(decoded) == 0 {
		t.Fatalf("truncated conversation is empty, want the first turns to survive: %s", value)
	}
	if len(decoded) >= len(messages) {
		t.Fatalf("truncated conversation kept %d turns, want fewer than %d", len(decoded), len(messages))
	}
}

func TestFailedToolCallOmitsStandardResultContent(t *testing.T) {
	state := State{
		Event: EventToolCallEnd, Degraded: true, Error: "boom",
		Output: SnapshotContent(map[string]string{"summary": "tool failed: boom"}),
	}
	if attrs := contentAttributes(state, CaptureFull, false); len(attrs) != 0 {
		t.Fatalf("failed tool result attributes = %v, want none", attrs)
	}
}

func TestStateAttributesExposeFallbackAndErrorType(t *testing.T) {
	for _, event := range []Event{EventStageEnd, EventModelCallEnd} {
		t.Run(string(event), func(t *testing.T) {
			attrs := stateAttributes(State{
				Event:          event,
				FallbackUsed:   true,
				FallbackReason: FallbackReasonParseError,
				ErrorType:      "example.ParseError",
			})
			assertSpanBoolAttribute(t, attrs, "agent.fallback.used", true)
			assertSpanAttribute(t, attrs, "agent.fallback.reason", FallbackReasonParseError)
			assertSpanAttribute(t, attrs, "error.type", "example.ParseError")
		})
	}
}

func assertSpanAttribute(t *testing.T, attrs []attribute.KeyValue, key, want string) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key {
			if got := attr.Value.AsString(); got != want {
				t.Fatalf("attribute %s = %q, want %q", key, got, want)
			}
			return
		}
	}
	t.Fatalf("attribute %s not found in %v", key, attrs)
}

func assertNoSpanAttribute(t *testing.T, attrs []attribute.KeyValue, key string) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key {
			t.Fatalf("unexpected attribute %s in %v", key, attrs)
		}
	}
}

func assertSpanBoolAttribute(t *testing.T, attrs []attribute.KeyValue, key string, want bool) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key {
			if got := attr.Value.AsBool(); got != want {
				t.Fatalf("attribute %s = %v, want %v", key, got, want)
			}
			return
		}
	}
	t.Fatalf("attribute %s not found in %v", key, attrs)
}

func assertSpanIntAttribute(t *testing.T, attrs []attribute.KeyValue, key string, want int64) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key {
			if got := attr.Value.AsInt64(); got != want {
				t.Fatalf("attribute %s = %d, want %d", key, got, want)
			}
			return
		}
	}
	t.Fatalf("attribute %s not found in %v", key, attrs)
}

func findSpanEvent(t *testing.T, span sdktrace.ReadOnlySpan, name string) sdktrace.Event {
	t.Helper()
	for _, event := range span.Events() {
		if event.Name == name {
			return event
		}
	}
	t.Fatalf("span %q is missing the %q event", span.Name(), name)
	return sdktrace.Event{}
}

func countSpanEvents(span sdktrace.ReadOnlySpan, name string) int {
	count := 0
	for _, event := range span.Events() {
		if event.Name == name {
			count++
		}
	}
	return count
}

func assertEventAttribute(t *testing.T, event sdktrace.Event, key, want string) {
	t.Helper()
	for _, attr := range event.Attributes {
		if string(attr.Key) == key {
			if got := attr.Value.AsString(); got != want {
				t.Fatalf("event %s attribute %s = %q, want %q", event.Name, key, got, want)
			}
			return
		}
	}
	t.Fatalf("event %s is missing attribute %s in %v", event.Name, key, event.Attributes)
}

func TestTracingSpecializedEventsKeepSpanOpenUntilEnd(t *testing.T) {
	cases := []struct {
		name    string
		start   Event
		special Event
		end     Event
	}{
		{"interaction error", EventInteractionStart, EventInteractionError, EventInteractionEnd},
		{"interaction cancel", EventInteractionStart, EventInteractionCancel, EventInteractionEnd},
		{"interaction degrade", EventInteractionStart, EventInteractionDegrade, EventInteractionEnd},
		{"stage error", EventStageStart, EventStageError, EventStageEnd},
		{"model error", EventModelCallStart, EventModelCallError, EventModelCallEnd},
		{"tool error", EventToolCallStart, EventToolCallError, EventToolCallEnd},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
			manager := testManager()
			if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureNone)); err != nil {
				t.Fatal(err)
			}
			state := State{Event: tt.start, Model: "test/model", ToolName: "test_tool"}
			ctx := manager.Emit(context.Background(), state)
			state.Event = tt.special
			state.Error = "operation failed"
			manager.Emit(ctx, state)
			if len(recorder.Ended()) != 0 {
				t.Fatal("specialized event ended the span before its end event")
			}
			if !trace.SpanFromContext(ctx).IsRecording() {
				t.Fatal("span stopped recording early")
			}
			state.Event = tt.end
			state.TotalTokens = 23
			manager.Emit(ctx, state)
			spans := recorder.Ended()
			if len(spans) != 1 {
				t.Fatalf("ended spans = %d, want 1", len(spans))
			}
			var totalTokens int64
			for _, attr := range spans[0].Attributes() {
				if attr.Key == "agent.usage.total_tokens" {
					totalTokens = attr.Value.AsInt64()
				}
			}
			if totalTokens != 0 {
				t.Fatalf("private total token attribute = %d, want none", totalTokens)
			}
			found := false
			for _, event := range spans[0].Events() {
				if event.Name == string(tt.special) {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing %s span event", tt.special)
			}
		})
	}
}

func TestTracingLargeMultibyteContent(t *testing.T) {
	// Reuse one 10 MiB payload across cases; providers expose it only on demand.
	const payloadBytes = 10 * 1024 * 1024
	payload := strings.Repeat("界🙂", payloadBytes/len("界🙂")+1)
	for _, tc := range []struct {
		name    string
		policy  string
		sampled bool
	}{
		{"none", CaptureNone, true},
		{"truncated", CaptureTruncated, true},
		{"unsampled truncated", CaptureTruncated, false},
		{"unsampled full", CaptureFull, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sampler := sdktrace.AlwaysSample()
			if !tc.sampled {
				sampler = sdktrace.NeverSample()
			}
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sampler), sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
			manager := testManager()
			if err := manager.Register(NewTracingRegistration(provider.Tracer("large-payload"), tc.policy)); err != nil {
				t.Fatal(err)
			}
			inputCalls, outputCalls := 0, 0
			messages := func(role string) any {
				return []map[string]any{{"role": role, "parts": []map[string]string{{"type": "text", "content": payload}}}}
			}
			start := State{Event: EventModelCallStart, Model: "test/model"}.WithInputContent(func() any {
				inputCalls++
				return messages("user")
			})
			end := State{Event: EventModelCallEnd, Model: "test/model"}.WithOutputContent(func() any {
				outputCalls++
				return messages("assistant")
			})
			ctx := manager.Emit(context.Background(), start)
			manager.Emit(ctx, end)
			wantCalls := 0
			if tc.sampled && tc.policy != CaptureNone {
				wantCalls = 1
			}
			if inputCalls != wantCalls || outputCalls != wantCalls {
				t.Fatalf("content provider calls = %d/%d, want %d each", inputCalls, outputCalls, wantCalls)
			}
			spans := recorder.Ended()
			if !tc.sampled {
				if len(spans) != 0 {
					t.Fatalf("unsampled ended spans = %d, want 0", len(spans))
				}
				return
			}
			if len(spans) != 1 {
				t.Fatalf("ended spans = %d, want 1", len(spans))
			}
			for _, key := range []string{"gen_ai.input.messages", "gen_ai.output.messages"} {
				var value string
				found := false
				for _, attr := range spans[0].Attributes() {
					if string(attr.Key) == key {
						value, found = attr.Value.AsString(), true
						break
					}
				}
				if tc.policy == CaptureNone {
					if found {
						t.Fatalf("capture none emitted %s (%d bytes)", key, len(value))
					}
					continue
				}
				if !found || len(value) > truncatedContentLimit || !utf8.ValidString(value) || !json.Valid([]byte(value)) {
					t.Fatalf("%s: found=%v, bytes=%d, valid UTF-8=%v, valid JSON=%v", key, found, len(value), utf8.ValidString(value), json.Valid([]byte(value)))
				}
				var decoded []struct{ Parts []struct{ Content string } }
				if err := json.Unmarshal([]byte(value), &decoded); err != nil {
					t.Fatal(err)
				}
				if len(decoded) != 1 || len(decoded[0].Parts) != 1 || !strings.HasSuffix(decoded[0].Parts[0].Content, "…") {
					t.Fatalf("%s did not preserve a truncated message and its text part", key)
				}
			}
		})
	}
}

// The agent span is the trace root, and backends such as Langfuse take the
// trace input and output from it. Without the conversation and the final
// answer on that span the trace renders as empty.
func TestTracingHookPutsConversationAndAnswerOnAgentSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureFull)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	interactionCtx := manager.Emit(context.Background(), State{
		Event: EventInteractionStart, InteractionID: "interaction", SessionID: "session",
	})
	endState := State{
		Event: EventInteractionEnd, InteractionID: "interaction", SessionID: "session",
	}
	endState = endState.WithInputContent(func() any {
		return []map[string]any{{"role": "user", "parts": []map[string]any{{"type": "text", "content": "查一下 order-service"}}}}
	})
	endState = endState.WithOutputContent(func() any {
		return []map[string]any{{"role": "assistant", "parts": []map[string]any{{"type": "text", "content": "服务正常"}}}}
	})
	manager.Emit(interactionCtx, endState)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	if spans[0].Name() != "invoke_agent "+AgentName {
		t.Fatalf("span name = %q, want the agent span", spans[0].Name())
	}
	assertSpanAttribute(t, spans[0].Attributes(), "gen_ai.input.messages",
		`[{"parts":[{"content":"查一下 order-service","type":"text"}],"role":"user"}]`)
	assertSpanAttribute(t, spans[0].Attributes(), "gen_ai.output.messages",
		`[{"parts":[{"content":"服务正常","type":"text"}],"role":"assistant"}]`)
}

// The default policy keeps the agent span empty, like every other span.
func TestTracingHookOmitsAgentSpanContentWhenCaptureIsNone(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureNone)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	interactionCtx := manager.Emit(context.Background(), State{
		Event: EventInteractionStart, InteractionID: "interaction", SessionID: "session",
	})
	endState := State{
		Event: EventInteractionEnd, InteractionID: "interaction", SessionID: "session",
	}
	endState = endState.WithInputContent(func() any {
		return []map[string]any{{"role": "user", "parts": []map[string]any{{"type": "text", "content": "查一下 order-service"}}}}
	})
	endState = endState.WithOutputContent(func() any {
		return []map[string]any{{"role": "assistant", "parts": []map[string]any{{"type": "text", "content": "服务正常"}}}}
	})
	manager.Emit(interactionCtx, endState)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	for _, attr := range spans[0].Attributes() {
		switch string(attr.Key) {
		case "gen_ai.input.messages", "gen_ai.output.messages":
			t.Fatalf("attribute %s leaked under the none policy", attr.Key)
		}
	}
}
