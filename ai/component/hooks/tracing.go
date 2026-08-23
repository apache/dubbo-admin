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
	"fmt"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const truncatedContentLimit = 4096

func TraceIDFromContext(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.TraceID().String()
}

// NewTracingRegistration returns a complete context-deriving tracing
// registration. Its lifecycle events are intentionally not configurable so a
// partial registration cannot leak a span or end an unrelated parent span.
func NewTracingRegistration(tracer trace.Tracer, captureContent string) Registration {
	return Registration{
		Events:         AllEvents(),
		Tools:          []string{AllTools},
		CaptureContent: captureContent != CaptureNone,
		DerivesContext: true,
		lazyContent:    true,
		Hook:           newTracingHook(tracer, captureContent),
	}
}

func newTracingHook(tracer trace.Tracer, captureContent string) Hook {
	return func(ctx context.Context, state State) context.Context {
		if eventStartsSpan(state.Event) {
			attrs := stateAttributes(state)
			var span trace.Span
			ctx, span = tracer.Start(ctx, spanName(state),
				trace.WithSpanKind(spanKind(state.Event)),
				trace.WithAttributes(attrs...),
			)
			if span.IsRecording() {
				span.SetAttributes(contentAttributes(state, captureContent, true)...)
			}
			return ctx
		}

		span := trace.SpanFromContext(ctx)
		span.SetAttributes(stateAttributes(state)...)
		if span.IsRecording() {
			span.SetAttributes(contentAttributes(state, captureContent, false)...)
		}
		if state.Error != "" {
			span.RecordError(errors.New(state.Error))
			span.SetStatus(codes.Error, state.Error)
		}
		span.End()
		return ctx
	}
}

func eventStartsSpan(event Event) bool {
	switch event {
	case EventInteractionStart, EventIterationStart, EventStageStart, EventModelCallStart, EventToolCallStart:
		return true
	default:
		return false
	}
}

func spanName(state State) string {
	switch state.Event {
	case EventInteractionStart:
		return "invoke_agent"
	case EventIterationStart:
		return fmt.Sprintf("agent.iteration %d", state.Iteration)
	case EventStageStart:
		return "agent.stage " + state.Stage
	case EventModelCallStart:
		return "chat " + modelName(state.Model)
	case EventToolCallStart:
		return "execute_tool " + state.ToolName
	default:
		return string(state.Event)
	}
}

func spanKind(event Event) trace.SpanKind {
	if event == EventModelCallStart {
		return trace.SpanKindClient
	}
	return trace.SpanKindInternal
}

func stateAttributes(state State) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 14)
	if state.InteractionID != "" {
		attrs = append(attrs, attribute.String("agent.interaction.id", state.InteractionID))
	}
	if state.SessionID != "" {
		attrs = append(attrs,
			attribute.String("gen_ai.conversation.id", state.SessionID),
			attribute.String("session.id", state.SessionID),
		)
	}
	if state.Iteration > 0 {
		attrs = append(attrs, attribute.Int("agent.iteration", state.Iteration))
	}
	if state.Stage != "" {
		attrs = append(attrs, attribute.String("agent.stage", state.Stage))
	}
	switch state.Event {
	case EventInteractionStart, EventInteractionEnd:
		attrs = append(attrs, attribute.String("gen_ai.operation.name", "invoke_agent"))
	case EventModelCallStart, EventModelCallEnd:
		attrs = append(attrs, attribute.String("gen_ai.operation.name", "chat"))
		if state.Model != "" {
			attrs = append(attrs, attribute.String("gen_ai.request.model", modelName(state.Model)))
		}
		if provider := providerName(state.Model); provider != "" {
			attrs = append(attrs, attribute.String("gen_ai.provider.name", provider))
		}
	case EventToolCallStart, EventToolCallEnd:
		attrs = append(attrs, attribute.String("gen_ai.operation.name", "execute_tool"))
	}
	if state.Event.toolCall() {
		if state.ToolName != "" {
			attrs = append(attrs, attribute.String("gen_ai.tool.name", state.ToolName))
		}
		if state.ToolCallID != "" {
			attrs = append(attrs, attribute.String("gen_ai.tool.call.id", state.ToolCallID))
		}
	}
	if state.Degraded {
		attrs = append(attrs, attribute.Bool("agent.degraded", true))
	}
	if state.FallbackUsed {
		attrs = append(attrs, attribute.Bool("agent.fallback.used", true))
	}
	if state.FallbackReason != "" {
		attrs = append(attrs, attribute.String("agent.fallback.reason", state.FallbackReason))
	}
	if state.ErrorType != "" {
		attrs = append(attrs, attribute.String("error.type", state.ErrorType))
	}
	if state.Event == EventModelCallEnd && state.InputTokens > 0 {
		attrs = append(attrs, attribute.Int("gen_ai.usage.input_tokens", state.InputTokens))
	}
	if state.Event == EventModelCallEnd && state.OutputTokens > 0 {
		attrs = append(attrs, attribute.Int("gen_ai.usage.output_tokens", state.OutputTokens))
	}
	if state.TotalTokens > 0 {
		attrs = append(attrs, attribute.Int("agent.usage.total_tokens", state.TotalTokens))
	}
	return attrs
}

func contentAttributes(state State, policy string, input bool) []attribute.KeyValue {
	if policy == CaptureNone {
		return nil
	}
	value, key := semanticContent(state, input)
	if value == "" {
		return nil
	}

	text := value
	if policy == CaptureTruncated && len(text) > truncatedContentLimit {
		text = truncateSemanticJSON(text, key)
	}
	return []attribute.KeyValue{attribute.String(key, text)}
}

func truncateSemanticJSON(text, key string) string {
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return strconv.Quote(strings.ToValidUTF8(text[:truncatedContentLimit], "") + "…")
	}
	value = truncateJSONStrings(value, 512)
	encoded, err := json.Marshal(value)
	if err == nil && len(encoded) <= truncatedContentLimit {
		return string(encoded)
	}
	if key == "gen_ai.input.messages" || key == "gen_ai.output.messages" {
		return "[]"
	}
	return "{}"
}

func truncateJSONStrings(value any, limit int) any {
	switch typed := value.(type) {
	case string:
		if len(typed) <= limit {
			return typed
		}
		return strings.ToValidUTF8(typed[:limit], "") + "…"
	case []any:
		for i := range typed {
			typed[i] = truncateJSONStrings(typed[i], limit)
		}
	case map[string]any:
		for key := range typed {
			typed[key] = truncateJSONStrings(typed[key], limit)
		}
	}
	return value
}

func semanticContent(state State, input bool) (string, string) {
	switch state.Event {
	case EventModelCallStart:
		if input {
			return state.snapshotInputContent(), "gen_ai.input.messages"
		}
	case EventModelCallEnd:
		if !input {
			return state.snapshotOutputContent(), "gen_ai.output.messages"
		}
	case EventToolCallStart:
		if input {
			return state.snapshotInputContent(), "gen_ai.tool.call.arguments"
		}
	case EventToolCallEnd:
		if !input && state.Error == "" && !state.Degraded {
			return state.snapshotOutputContent(), "gen_ai.tool.call.result"
		}
	}
	return "", ""
}

func providerName(model string) string {
	provider, _, ok := strings.Cut(model, "/")
	if !ok {
		return ""
	}
	return provider
}

func modelName(model string) string {
	_, name, ok := strings.Cut(model, "/")
	if ok {
		return name
	}
	return model
}
