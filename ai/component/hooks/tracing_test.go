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

	"go.opentelemetry.io/otel/attribute"
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
	if spans[0].Name() != "chat model" || spans[1].Name() != "invoke_agent" {
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
			if totalTokens != 23 {
				t.Fatalf("final token count = %d, want 23", totalTokens)
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
