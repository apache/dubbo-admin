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
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestContextSizeTracingIsLazyAndIndependentOfPayloadCapture(t *testing.T) {
	for _, sampled := range []bool{false, true} {
		name := "unsampled"
		sampler := sdktrace.NeverSample()
		if sampled {
			name = "sampled"
			sampler = sdktrace.AlwaysSample()
		}
		t.Run(name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sampler), sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
			manager := testManager()
			if err := manager.Register(NewTracingRegistration(provider.Tracer("test"), CaptureNone)); err != nil {
				t.Fatal(err)
			}
			sizeCalls, payloadCalls := 0, 0
			start := State{Event: EventModelCallStart, Model: "test/model"}.WithContextSize(func() ContextSize {
				sizeCalls++
				return ContextSize{Messages: 3, JSONBytes: 101, EstimatedTokens: 26}
			}).WithInputContent(func() any {
				payloadCalls++
				return "private message"
			})
			ctx := manager.Emit(context.Background(), start)
			manager.Emit(ctx, State{Event: EventModelCallEnd, Model: "test/model", InputTokens: 11, OutputTokens: 2, TotalTokens: 13})
			wantCalls := 0
			if sampled {
				wantCalls = 1
			}
			if sizeCalls != wantCalls || payloadCalls != 0 {
				t.Fatalf("size/payload evaluations = %d/%d, want %d/0", sizeCalls, payloadCalls, wantCalls)
			}
			spans := recorder.Ended()
			if !sampled {
				if len(spans) != 0 {
					t.Fatal("unsampled trace exported a span")
				}
				return
			}
			if len(spans) != 1 {
				t.Fatalf("spans = %d, want one model span", len(spans))
			}
			attrs := spans[0].Attributes()
			assertContextIntAttribute(t, attrs, "agent.context.message_count", 3)
			assertContextIntAttribute(t, attrs, "agent.context.json_bytes", 101)
			assertContextIntAttribute(t, attrs, "agent.context.estimated_tokens", 26)
			assertContextIntAttribute(t, attrs, "gen_ai.usage.input_tokens", 11)
			assertNoSpanAttribute(t, attrs, "gen_ai.input.messages")
			assertNoSpanAttribute(t, attrs, "gen_ai.output.messages")
			if _, ok := start.ContextSize(); !ok || sizeCalls != 1 {
				t.Fatal("observers did not share one measurement")
			}
		})
	}
}

func TestUnavailableContextSizeDoesNotReportZeroBytes(t *testing.T) {
	state := State{}.WithContextSize(func() ContextSize {
		return ContextSize{Messages: 1, JSONBytes: -1, EstimatedTokens: -1}
	})
	attrs := contextSizeAttributes(state)
	assertContextIntAttribute(t, attrs, "agent.context.message_count", 1)
	assertNoSpanAttribute(t, attrs, "agent.context.json_bytes")
	assertNoSpanAttribute(t, attrs, "agent.context.estimated_tokens")
}

func assertContextIntAttribute(t *testing.T, attrs []attribute.KeyValue, key string, want int64) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key {
			if attr.Value.Type() != attribute.INT64 || attr.Value.AsInt64() != want {
				t.Fatalf("attribute %s = %v, want integer %d", key, attr.Value, want)
			}
			return
		}
	}
	t.Fatalf("missing integer attribute %s", key)
}
