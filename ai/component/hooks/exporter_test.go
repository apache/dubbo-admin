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
	"strings"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// recordingExporter keeps the spans that were exported.
type recordingExporter struct {
	spans []sdktrace.ReadOnlySpan
}

func (e *recordingExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *recordingExporter) Shutdown(_ context.Context) error { return nil }

// TestGenkitContentAttributesFollowCapturePolicy pins the scope of the
// capture policies over the whole trace.
//
// Genkit records the request and the response of a model call on the span of
// its own action. Those attributes never reach the Manager, so without the
// scrubbing exporter the policies would hold for the spans the hook owns and
// for nothing else — and the promise that no content attribute is exported at
// all would not hold.
func TestGenkitContentAttributesFollowCapturePolicy(t *testing.T) {
	for _, tc := range []struct {
		name             string
		captureContent   string
		captureReasoning string
		wantContent      bool
		wantReasoning    bool
	}{
		{name: "content none", captureContent: CaptureNone, captureReasoning: CaptureNone},
		{name: "content full reasoning none", captureContent: CaptureFull, captureReasoning: CaptureNone, wantContent: true},
		{name: "content full reasoning full", captureContent: CaptureFull, captureReasoning: CaptureFull, wantContent: true, wantReasoning: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exporter := &recordingExporter{}
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(&contentScrubbingExporter{
				inner:            exporter,
				captureContent:   tc.captureContent,
				captureReasoning: tc.captureReasoning,
			})))
			t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

			// The same global wiring the component does.
			otel.SetTracerProvider(provider)

			g := genkit.Init(context.Background())
			model := genkit.DefineModel(g, "reviewModel", &ai.ModelOptions{},
				func(_ context.Context, _ *ai.ModelRequest, _ ai.ModelStreamCallback) (*ai.ModelResponse, error) {
					return &ai.ModelResponse{
						Message: ai.NewMessage(ai.RoleModel, nil,
							ai.NewReasoningPart("REVIEW_REASONING_SENTINEL", nil),
							ai.NewTextPart("REVIEW_ANSWER_SENTINEL")),
					}, nil
				})
			if _, err := genkit.Generate(context.Background(), g,
				ai.WithModel(model),
				ai.WithMessages(ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart("REVIEW_INPUT_SENTINEL"))),
			); err != nil {
				t.Fatalf("generate: %v", err)
			}
			// The model span is ended by Genkit, so the exported batch already
			// holds the finished spans.
			var sawContent, sawReasoning, sawInput bool
			for _, span := range exporter.spans {
				for _, attr := range span.Attributes() {
					value := attr.Value.AsString()
					switch string(attr.Key) {
					case genkitInputAttribute, genkitOutputAttribute:
						sawContent = true
						if strings.Contains(value, "REVIEW_REASONING_SENTINEL") {
							sawReasoning = true
						}
						if strings.Contains(value, "REVIEW_INPUT_SENTINEL") {
							sawInput = true
						}
					}
				}
			}
			if tc.wantContent != sawContent {
				t.Fatalf("exported a Genkit content attribute = %v, want %v", sawContent, tc.wantContent)
			}
			if tc.wantReasoning != sawReasoning {
				t.Fatalf("exported Genkit reasoning content = %v, want %v", sawReasoning, tc.wantReasoning)
			}
			if tc.wantContent && !sawInput {
				t.Fatal("the exported Genkit content lost the prompt")
			}
		})
	}
}
