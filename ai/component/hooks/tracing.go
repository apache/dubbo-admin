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
	"runtime"
	"strconv"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// truncatedContentLimit is the size a captured content attribute may
	// reach. A longer document is shortened before it is attached to a span.
	truncatedContentLimit = 4096
	// truncatedJSONStringLimit is the size of a single string inside a
	// captured content document.
	truncatedJSONStringLimit = 512
)

// AgentName is the human-readable name of the Dubbo Admin SRE agent. The GenAI
// semantic conventions ask for gen_ai.agent.name on the agent-scoped spans
// (invoke_agent and execute_tool) and shape the invoke_agent span name as
// "invoke_agent {gen_ai.agent.name}".
const AgentName = "dubbo-admin-sre-agent"

// genAIExceptionEvent is the GenAI semantic convention event name for an
// exception raised during a generative AI client operation.
const genAIExceptionEvent = "gen_ai.client.operation.exception"

// errorTypeOther is the well-known fallback value of error.type used when the
// instrumentation cannot classify the failure.
const errorTypeOther = "_OTHER"

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
	exceptions := &exceptionRecorder{}
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
			// The agent span is closed by the interaction end, which carries
			// both the conversation and the final answer. Backends that read
			// the trace root (Langfuse does) take the trace input and output
			// from this span, so the input has to be emitted here too.
			span.SetAttributes(contentAttributes(state, captureContent, true)...)
		}
		if state.Error != "" {
			span.RecordError(errors.New(state.Error))
			span.SetStatus(codes.Error, state.Error)
			recordGenAIException(span, state, exceptions)
		}
		switch state.Event {
		case EventInteractionEnd, EventIterationEnd, EventStageEnd, EventModelCallEnd, EventToolCallEnd:
			span.End()
			exceptions.release(span)
		default:
			span.AddEvent(string(state.Event))
		}
		return ctx
	}
}

// exceptionRecorder tracks the spans that already carry a GenAI exception
// event. The agent emits a dedicated error event and the matching end event
// for a failing operation, and both carry the error, so the recorder keeps the
// exception reported exactly once per failing operation.
type exceptionRecorder struct {
	mu    sync.Mutex
	spans map[spanKey]struct{}
}

// spanKey identifies a span across traces so recycled span identifiers never
// collide.
type spanKey struct {
	traceID trace.TraceID
	spanID  trace.SpanID
}

func keyOf(span trace.Span) spanKey {
	spanContext := span.SpanContext()
	return spanKey{traceID: spanContext.TraceID(), spanID: spanContext.SpanID()}
}

// claim reports whether the caller owns recording the exception event for the
// span. Only the first caller for a span is granted ownership.
func (r *exceptionRecorder) claim(span trace.Span) bool {
	key := keyOf(span)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.spans == nil {
		r.spans = make(map[spanKey]struct{})
	}
	if _, claimed := r.spans[key]; claimed {
		return false
	}
	r.spans[key] = struct{}{}
	return true
}

// release forgets the span once it ended so the recorder does not grow.
func (r *exceptionRecorder) release(span trace.Span) {
	key := keyOf(span)
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.spans, key)
}

// recordGenAIException records the GenAI semantic convention exception event
// on a failing model call or tool call span.
//
// The severity the conventions ask for (WARN, severity number 13) cannot be
// expressed on a span event: the OpenTelemetry Go SDK models an event without
// a severity field (sdk/trace.Event only carries Name, Attributes and Time)
// and trace.EventOption only accepts attributes and a timestamp. The span
// status is therefore left at Error, which is the fallback the GenAI
// "Recording Errors" document prescribes.
func recordGenAIException(span trace.Span, state State, exceptions *exceptionRecorder) {
	if !genAIOperation(state.Event) || !span.IsRecording() {
		return
	}
	if !exceptions.claim(span) {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("exception.type", exceptionType(state)),
		attribute.String("exception.message", state.Error),
	}
	// The hook only observes the error message, so the stack is captured at
	// the hook boundary, the same convention the OpenTelemetry Go SDK uses
	// when it records an exception.
	if stack := captureStackTrace(); stack != "" {
		attrs = append(attrs, attribute.String("exception.stacktrace", stack))
	}
	span.AddEvent(genAIExceptionEvent, trace.WithAttributes(attrs...))
}

// genAIOperation reports whether the event belongs to a generative AI
// operation that reports client-operation exceptions: a model call or a tool
// call.
func genAIOperation(event Event) bool {
	switch event {
	case EventModelCallError, EventModelCallEnd, EventToolCallError, EventToolCallEnd:
		return true
	default:
		return false
	}
}

// exceptionType returns the low-cardinality class of the failure. The error
// type reported by the agent wins; the well-known fallback keeps the
// attribute non-empty otherwise.
func exceptionType(state State) string {
	if state.ErrorType != "" {
		return state.ErrorType
	}
	return errorTypeOther
}

func captureStackTrace() string {
	buffer := make([]byte, 2048)
	return string(buffer[:runtime.Stack(buffer, false)])
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
		// The GenAI semantic conventions name the internal invoke_agent span
		// "invoke_agent {gen_ai.agent.name}".
		return "invoke_agent " + AgentName
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
	attrs := make([]attribute.KeyValue, 0, 16)
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
		attrs = append(attrs,
			attribute.String("gen_ai.operation.name", "invoke_agent"),
			attribute.String("gen_ai.agent.name", AgentName),
		)
	case EventModelCallStart, EventModelCallEnd:
		attrs = append(attrs, attribute.String("gen_ai.operation.name", "chat"))
		if model := modelName(state.Model); model != "" {
			attrs = append(attrs, attribute.String("gen_ai.request.model", model))
		}
		if provider := providerName(state.Model); provider != "" {
			attrs = append(attrs, attribute.String("gen_ai.provider.name", provider))
		}
	case EventToolCallStart, EventToolCallEnd:
		attrs = append(attrs, attribute.String("gen_ai.operation.name", "execute_tool"))
	}
	if state.Event.toolCall() {
		// The execute_tool span runs inside the agent, so it carries the
		// agent identity as well.
		attrs = append(attrs, attribute.String("gen_ai.agent.name", AgentName))
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
	switch {
	case state.ErrorType != "":
		attrs = append(attrs, attribute.String("error.type", state.ErrorType))
	case state.Error != "":
		// The GenAI semantic conventions require a low-cardinality error.type
		// on every span that ended in an error.
		attrs = append(attrs, attribute.String("error.type", errorTypeOther))
	}
	if state.Event == EventModelCallEnd && state.InputTokens > 0 {
		attrs = append(attrs, attribute.Int("gen_ai.usage.input_tokens", state.InputTokens))
	}
	if state.Event == EventModelCallEnd && state.OutputTokens > 0 {
		attrs = append(attrs, attribute.Int("gen_ai.usage.output_tokens", state.OutputTokens))
	}
	// The generative AI semantic conventions define no total token attribute:
	// the total is the sum of the input and the output tokens. Consumers such
	// as cost calculators derive it from those two attributes, so Dubbo Admin
	// does not emit a private total.
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
	value = truncateJSONStrings(value, truncatedJSONStringLimit)
	// The strings are shortened first, because a long prompt is usually one
	// long string. When the document still does not fit, the trailing elements
	// of the outermost array are dropped until it does. A long conversation
	// keeps its first turns this way, instead of turning into an empty list
	// that a backend reads as "there was no conversation".
	for {
		encoded, err := json.Marshal(value)
		if err != nil || len(encoded) <= truncatedContentLimit {
			break
		}
		items, ok := value.([]any)
		if !ok || len(items) <= 1 {
			break
		}
		value = items[:len(items)-1]
	}
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
	case EventInteractionEnd:
		// The GenAI conventions let the agent span carry the messages that
		// went into the agent and the messages it produced. The agent closes
		// that span with the whole conversation and the final answer, so the
		// trace root is self-describing.
		if input {
			return state.snapshotInputContent(), "gen_ai.input.messages"
		}
		return state.snapshotOutputContent(), "gen_ai.output.messages"
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

// genAIProviderAliases maps a provider configuration name to the well-known
// gen_ai.provider.name value of the OpenTelemetry generative AI semantic
// conventions.
//
// The conventions state that "gen_ai.provider.name has the following list of
// well-known values. If one of them applies, then the respective value MUST
// be used; otherwise, a custom value MAY be used." A provider is therefore
// never dropped: a configured name that matches no entry of the conventions
// stays a legitimate custom value and is reported verbatim.
//
// Every alias is justified by the endpoint description the conventions give
// for the well-known value, so an alias only exists where the configured
// base_url identifies the well-known provider unambiguously:
//
//	gemini -> gcp.gemini
//	    The conventions describe gcp.gemini as "Used when accessing the
//	    'generativelanguage.googleapis.com' endpoint. Also known as the AI
//	    Studio API", and the gemini provider of
//	    ai/component/models/models.yaml is configured with the matching base
//	    url https://generativelanguage.googleapis.com/v1beta.
//
// The dashscope and siliconflow providers have no well-known counterpart and
// are reported as custom values.
//
// When a new provider configuration is added and it corresponds to one of the
// well-known values, add the mapping here.
var genAIProviderAliases = map[string]string{
	"gemini": "gcp.gemini",
}

// wellKnownGenAIProviders is the reference set of the well-known
// gen_ai.provider.name values defined by the OpenTelemetry generative AI
// semantic conventions. It documents the values the conventions recognize and
// lets a test assert whether a reported provider is a convention value.
//
// The set is a reference only: it never filters a provider out. The
// conventions explicitly allow a custom value for every provider that is not
// listed, so a provider name outside this set is still reported verbatim.
var wellKnownGenAIProviders = map[string]struct{}{
	"anthropic":          {},
	"aws.bedrock":        {},
	"azure.ai.inference": {},
	"azure.ai.openai":    {},
	"cohere":             {},
	"deepseek":           {},
	"gcp.gemini":         {},
	"gcp.gen_ai":         {},
	"gcp.vertex_ai":      {},
	"groq":               {},
	"ibm.watsonx.ai":     {},
	"mistral_ai":         {},
	"moonshot_ai":        {},
	"openai":             {},
	"perplexity":         {},
	"x_ai":               {},
}

// providerName returns the provider prefix of a "provider/model" identifier.
// An identifier without a separator has no known provider, so no provider is
// reported rather than a bogus one.
//
// A configured provider name that maps to a well-known value of the semantic
// conventions is reported as that value; every other name is reported
// verbatim, because the conventions allow a custom value. The model part of
// the identifier is not affected by the mapping.
func providerName(model string) string {
	provider, _, ok := strings.Cut(model, "/")
	if !ok {
		return ""
	}
	if canonical, aliased := genAIProviderAliases[provider]; aliased {
		return canonical
	}
	return provider
}

// modelName returns the model part of a "provider/model" identifier. An
// identifier without a separator names the model itself; an identifier without
// a model part has no model to report.
func modelName(model string) string {
	_, name, ok := strings.Cut(model, "/")
	if !ok {
		return model
	}
	return name
}
