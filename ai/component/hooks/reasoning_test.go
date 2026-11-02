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
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"dubbo-admin-ai/config"
	"dubbo-admin-ai/runtime"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const reasoningDocument = `[{"role":"assistant","parts":[{"type":"reasoning","content":"thinking about the answer"},{"type":"text","content":"the answer"}]}]`

type reasoningMessage struct {
	Role  string           `json:"role"`
	Parts []map[string]any `json:"parts"`
}

func decodeReasoningDocument(t *testing.T, document string) []reasoningMessage {
	t.Helper()
	var messages []reasoningMessage
	if err := json.Unmarshal([]byte(document), &messages); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", document, err)
	}
	return messages
}

func TestApplyReasoningPolicyRemovesReasoningParts(t *testing.T) {
	filtered := ApplyReasoningPolicy(reasoningDocument, CaptureNone)
	messages := decodeReasoningDocument(t, filtered)
	if len(messages) != 1 || len(messages[0].Parts) != 1 {
		t.Fatalf("filtered document = %s, want a single text part", filtered)
	}
	part := messages[0].Parts[0]
	if part["type"] != "text" || part["content"] != "the answer" {
		t.Fatalf("remaining part = %v, want the text part", part)
	}
	if strings.Contains(filtered, reasoningPartType) {
		t.Fatalf("filtered document still holds a reasoning part: %s", filtered)
	}
}

func TestApplyReasoningPolicyTruncatesReasoningContent(t *testing.T) {
	long := strings.Repeat("思考", 1000)
	document := `[{"role":"assistant","parts":[{"type":"reasoning","content":"` + long + `"},{"type":"text","content":"the answer"}]}]`
	filtered := ApplyReasoningPolicy(document, CaptureTruncated)
	messages := decodeReasoningDocument(t, filtered)
	if len(messages) != 1 || len(messages[0].Parts) != 2 {
		t.Fatalf("truncated document = %s, want the reasoning and the text part", filtered)
	}
	reasoning := messages[0].Parts[0]
	if reasoning["type"] != reasoningPartType {
		t.Fatalf("first part = %v, want a reasoning part", reasoning)
	}
	content, ok := reasoning["content"].(string)
	if !ok {
		t.Fatalf("reasoning content = %v, want a string", reasoning["content"])
	}
	if len([]rune(content)) > truncatedReasoningLimit {
		t.Fatalf("reasoning content length = %d, want at most %d", len([]rune(content)), truncatedReasoningLimit)
	}
	if len([]rune(content)) != truncatedReasoningLimit {
		t.Fatalf("reasoning content length = %d, want %d", len([]rune(content)), truncatedReasoningLimit)
	}
	if !strings.HasSuffix(content, "…") {
		t.Fatalf("truncated reasoning content = %q, want an ellipsis marker", content)
	}
	if messages[0].Parts[1]["type"] != "text" {
		t.Fatalf("second part = %v, want the text part", messages[0].Parts[1])
	}
}

func TestApplyReasoningPolicyKeepsFullReasoning(t *testing.T) {
	if got := ApplyReasoningPolicy(reasoningDocument, CaptureFull); got != reasoningDocument {
		t.Fatalf("document = %s, want it unchanged", got)
	}
}

func TestApplyReasoningPolicyRemovesEscapedReasoningParts(t *testing.T) {
	// The type discriminator may be escaped in the JSON document. The raw bytes
	// of the document then do not hold the word "reasoning", so a plain string
	// search would skip the document and the reasoning content would reach the
	// trace unchanged. The escape below decodes to "reasoning".
	escapedType := fmt.Sprintf("%c%c%04x", '\\', 'u', 'r') + "easoning"
	document := `[{"role":"assistant","parts":[{"type":"text","content":"the answer"},{"type":"` + escapedType + `","content":"the secret"}]}]`
	if strings.Contains(document, reasoningPartType) {
		t.Fatalf("the test document has to hide the type discriminator: %s", document)
	}
	filtered := ApplyReasoningPolicy(document, CaptureNone)
	messages := decodeReasoningDocument(t, filtered)
	if len(messages) != 1 || len(messages[0].Parts) != 1 {
		t.Fatalf("filtered document = %s, want a single text part", filtered)
	}
	if messages[0].Parts[0]["type"] != "text" {
		t.Fatalf("remaining part = %v, want the text part", messages[0].Parts[0])
	}
	if strings.Contains(filtered, "secret") {
		t.Fatalf("filtered document still holds the reasoning content: %s", filtered)
	}
}

func TestApplyReasoningPolicyKeepsInvalidDocument(t *testing.T) {
	for _, document := range []string{"", "not json", "[{", `{"role":"user"} trailing`} {
		if got := ApplyReasoningPolicy(document, CaptureNone); got != document {
			t.Fatalf("document = %q, want it unchanged", got)
		}
	}
	// A document without a reasoning part keeps its original representation.
	document := `[{"role":"user","parts":[{"type":"text","content":"hi"}]}]`
	if got := ApplyReasoningPolicy(document, CaptureNone); got != document {
		t.Fatalf("document = %s, want it unchanged", got)
	}
}

func TestApplyReasoningPolicyIgnoresDocumentShape(t *testing.T) {
	document := `[{"role":"user","parts":[{"type":"text","content":"hi"}]},{"role":"assistant","parts":[{"type":"reasoning","content":"thinking"}]}]`
	filtered := ApplyReasoningPolicy(document, CaptureNone)
	messages := decodeReasoningDocument(t, filtered)
	if len(messages) != 2 {
		t.Fatalf("filtered document = %s, want both messages", filtered)
	}
	if len(messages[1].Parts) != 0 {
		t.Fatalf("second message = %v, want no parts", messages[1])
	}
}

func TestCaptureReasoningConfiguration(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		c, err := componentFromHookYAML(t, "hooks: [{name: trace, type: tracing}]")
		if err != nil {
			t.Fatal(err)
		}
		entries, err := c.spec.normalizedHooks()
		if err != nil {
			t.Fatal(err)
		}
		if entries[0].tracing.CaptureReasoning != CaptureNone {
			t.Fatalf("capture_reasoning = %q, want %q", entries[0].tracing.CaptureReasoning, CaptureNone)
		}
	})

	t.Run("configured", func(t *testing.T) {
		c, err := componentFromHookYAML(t, "hooks: [{name: trace, type: tracing, config: {capture_reasoning: truncated}}]")
		if err != nil {
			t.Fatal(err)
		}
		entries, err := c.spec.normalizedHooks()
		if err != nil {
			t.Fatal(err)
		}
		if entries[0].tracing.CaptureReasoning != CaptureTruncated {
			t.Fatalf("capture_reasoning = %q, want %q", entries[0].tracing.CaptureReasoning, CaptureTruncated)
		}
	})

	t.Run("legacy", func(t *testing.T) {
		c, err := componentFromHookYAML(t, "tracing: {enabled: true, service_name: test, sample_ratio: 1, capture_content: none, capture_reasoning: full}")
		if err != nil {
			t.Fatal(err)
		}
		if c.spec.Tracing.CaptureReasoning != CaptureFull {
			t.Fatalf("capture_reasoning = %q, want %q", c.spec.Tracing.CaptureReasoning, CaptureFull)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := componentFromHookYAML(t, "hooks: [{name: trace, type: tracing, config: {capture_reasoning: raw}}]")
		if err == nil || !strings.Contains(err.Error(), "capture_reasoning must be one of") {
			t.Fatalf("error = %v, want a capture_reasoning validation error", err)
		}
	})

	t.Run("invalid legacy", func(t *testing.T) {
		_, err := componentFromHookYAML(t, "tracing: {enabled: true, service_name: test, sample_ratio: 1, capture_content: none, capture_reasoning: raw}")
		if err == nil || !strings.Contains(err.Error(), "capture_reasoning must be one of") {
			t.Fatalf("error = %v, want a capture_reasoning validation error", err)
		}
	})
}

func TestManagerAppliesCaptureReasoningPolicy(t *testing.T) {
	for _, tt := range []struct {
		name   string
		policy string
		want   bool
	}{
		{name: "none", policy: CaptureNone, want: false},
		{name: "truncated", policy: CaptureTruncated, want: true},
		{name: "full", policy: CaptureFull, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(nil)
			var captured string
			if err := manager.Register(Registration{
				Events:           []Event{EventModelCallEnd},
				Tools:            []string{AllTools},
				CaptureContent:   true,
				CaptureReasoning: tt.policy,
				Hook: func(ctx context.Context, state State) context.Context {
					captured = state.snapshotOutputContent()
					return ctx
				},
			}); err != nil {
				t.Fatal(err)
			}
			manager.Emit(context.Background(), State{Event: EventModelCallEnd, Output: reasoningDocument})
			if tt.want {
				if !strings.Contains(captured, reasoningPartType) {
					t.Fatalf("captured content = %s, want a reasoning part", captured)
				}
			} else if strings.Contains(captured, reasoningPartType) {
				t.Fatalf("captured content = %s, want no reasoning part", captured)
			}
			if manager.ReasoningPolicy() != tt.policy {
				t.Fatalf("ReasoningPolicy() = %q, want %q", manager.ReasoningPolicy(), tt.policy)
			}
		})
	}
}

func TestComponentRegistersCaptureReasoningPolicy(t *testing.T) {
	c, err := componentFromHookYAML(t, "hooks: [{name: trace, type: tracing, config: {sample_ratio: 0, capture_reasoning: truncated}}]")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Init(runtime.NewRuntime()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Stop(); err != nil {
			t.Errorf("Stop() error: %v", err)
		}
	})
	if policy := c.GetManager().ReasoningPolicy(); policy != CaptureTruncated {
		t.Fatalf("ReasoningPolicy() = %q, want %q", policy, CaptureTruncated)
	}
}

func TestManagerRejectsInvalidCaptureReasoningPolicy(t *testing.T) {
	manager := NewManager(nil)
	err := manager.Register(Registration{
		Events:           []Event{EventModelCallEnd},
		CaptureContent:   true,
		CaptureReasoning: "raw",
		Hook:             func(ctx context.Context, _ State) context.Context { return ctx },
	})
	if err == nil || !strings.Contains(err.Error(), "capture_reasoning must be one of") {
		t.Fatalf("error = %v, want a capture_reasoning validation error", err)
	}
}

func TestShippedHooksConfigAcceptsCaptureReasoning(t *testing.T) {
	schemaDir, err := filepath.Abs("../../schema/json")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCHEMA_DIR", schemaDir)
	loaded, err := config.NewLoader("config.yaml").LoadComponent("hooks.yaml")
	if err != nil {
		t.Fatalf("the shipped hooks configuration is not valid: %v", err)
	}
	component, err := HookFactory(&loaded.Spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := component.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestTracingHookAppliesCaptureReasoningPolicy(t *testing.T) {
	for _, tt := range []struct {
		name   string
		policy string
		want   bool
	}{
		{name: "none", policy: CaptureNone, want: false},
		{name: "truncated", policy: CaptureTruncated, want: true},
		{name: "full", policy: CaptureFull, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
			manager := NewManager(nil)
			registration := NewTracingRegistration(provider.Tracer("test"), CaptureFull)
			registration.CaptureReasoning = tt.policy
			if err := manager.Register(registration); err != nil {
				t.Fatalf("register tracing hook: %v", err)
			}
			interactionCtx := manager.Emit(context.Background(), State{Event: EventInteractionStart})
			modelCtx := manager.Emit(interactionCtx, State{Event: EventModelCallStart})
			manager.Emit(modelCtx, State{Event: EventModelCallEnd}.WithOutputContent(func() any {
				return json.RawMessage(reasoningDocument)
			}))

			spans := recorder.Ended()
			if len(spans) != 1 {
				t.Fatalf("ended spans = %d, want 1", len(spans))
			}
			messages := reasoningSpanAttribute(t, spans[0].Attributes(), "gen_ai.output.messages")
			if tt.want {
				if !strings.Contains(messages, reasoningPartType) {
					t.Fatalf("output messages = %s, want a reasoning part", messages)
				}
				return
			}
			if strings.Contains(messages, reasoningPartType) {
				t.Fatalf("output messages = %s, want no reasoning part", messages)
			}
			if !strings.Contains(messages, "the answer") {
				t.Fatalf("output messages = %s, want the text part", messages)
			}
		})
	}
}

func reasoningSpanAttribute(t *testing.T, attrs []attribute.KeyValue, key string) string {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key {
			return attr.Value.AsString()
		}
	}
	t.Fatalf("attribute %s not found in %v", key, attrs)
	return ""
}
