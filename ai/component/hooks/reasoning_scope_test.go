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
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestReasoningPolicyLeavesToolArgumentsUntouched pins the scope of the
// reasoning policy. The arguments of a tool call are business data: a tool
// that uses the field names of a message must still be reported faithfully in
// the trace, because the trace is what a reader trusts when a call is
// questioned.
func TestReasoningPolicyLeavesToolArgumentsUntouched(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	manager := testManager()
	registration := NewTracingRegistration(provider.Tracer("test"), CaptureFull)
	registration.CaptureReasoning = CaptureNone
	if err := manager.Register(registration); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	businessArguments := `{"parts":[{"type":"reasoning","content":"business record"},{"type":"result","content":"keep"}]}`

	ctx := manager.Emit(context.Background(), State{
		Event:         EventToolCallStart,
		InteractionID: "interaction",
		SessionID:     "session",
		ToolName:      "reviewTool",
		ToolCallID:    "call-1",
	}.WithInputContent(func() any {
		return json.RawMessage(businessArguments)
	}))
	manager.Emit(ctx, State{
		Event:         EventToolCallEnd,
		InteractionID: "interaction",
		SessionID:     "session",
		ToolName:      "reviewTool",
		ToolCallID:    "call-1",
	})

	exported := exportedAttribute(t, recorder, "gen_ai.tool.call.arguments")
	if exported != businessArguments {
		t.Fatalf("tool arguments were rewritten:\n got: %s\nwant: %s", exported, businessArguments)
	}
}

// TestReasoningPolicyStillFiltersMessageDocuments pins the other half of the
// scope: a message document still loses its reasoning parts, so the capture
// policy keeps working for the content it was written for.
func TestReasoningPolicyStillFiltersMessageDocuments(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	manager := testManager()
	registration := NewTracingRegistration(provider.Tracer("test"), CaptureFull)
	registration.CaptureReasoning = CaptureNone
	if err := manager.Register(registration); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

	messages := `[{"role":"assistant","parts":[{"type":"reasoning","content":"thinking"},{"type":"text","content":"answer"}]}]`

	ctx := manager.Emit(context.Background(), State{
		Event:         EventModelCallStart,
		InteractionID: "interaction",
		SessionID:     "session",
		Model:         "review/model",
	}.WithInputContent(func() any {
		return json.RawMessage(messages)
	}))
	manager.Emit(ctx, State{
		Event:         EventModelCallEnd,
		InteractionID: "interaction",
		SessionID:     "session",
		Model:         "review/model",
	})

	exported := exportedAttribute(t, recorder, "gen_ai.input.messages")
	if exported == messages {
		t.Fatalf("message document still carries the reasoning part: %s", exported)
	}
	if strings.Contains(exported, "thinking") {
		t.Fatalf("exported message document still carries the reasoning content: %s", exported)
	}
	if !strings.Contains(exported, "answer") {
		t.Fatalf("exported message document lost the text part: %s", exported)
	}
}

// exportedAttribute returns the value of the first attribute with the given
// key on any ended span.
func exportedAttribute(t *testing.T, recorder *tracetest.SpanRecorder, key string) string {
	t.Helper()
	for _, span := range recorder.Ended() {
		for _, attr := range span.Attributes() {
			if string(attr.Key) == key {
				return attr.Value.AsString()
			}
		}
	}
	t.Fatalf("no span carried the attribute %q", key)
	return ""
}
