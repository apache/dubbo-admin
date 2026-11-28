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

package trace

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The generative AI semantic conventions are still in development, so the analyzer pins
// the recognized attribute set instead of trusting an open-ended namespace.
// Reference: OpenTelemetry semantic conventions for generative AI systems, commit 6fd0d76.
const genAIPrefix = "gen_ai."

// Recognized generative AI attributes.
const (
	genAIOperationName         = "gen_ai.operation.name"
	genAIProviderName          = "gen_ai.provider.name"
	genAIRequestModel          = "gen_ai.request.model"
	genAIResponseModel         = "gen_ai.response.model"
	genAIConversationID        = "gen_ai.conversation.id"
	genAIAgentName             = "gen_ai.agent.name"
	genAIAgentID               = "gen_ai.agent.id"
	genAIAgentDescription      = "gen_ai.agent.description"
	genAIAgentVersion          = "gen_ai.agent.version"
	genAIToolName              = "gen_ai.tool.name"
	genAIToolCallID            = "gen_ai.tool.call.id"
	genAIToolCallArguments     = "gen_ai.tool.call.arguments"
	genAIToolCallResult        = "gen_ai.tool.call.result"
	genAIInputMessages         = "gen_ai.input.messages"
	genAIOutputMessages        = "gen_ai.output.messages"
	genAIUsageInputTokens      = "gen_ai.usage.input_tokens"
	genAIUsageOutputTokens     = "gen_ai.usage.output_tokens"
	genAIUsageReasoningTokens  = "gen_ai.usage.reasoning.output_tokens"
	genAIUsageCacheReadTokens  = "gen_ai.usage.cache_read.input_tokens"
	genAIUsageCacheWriteTokens = "gen_ai.usage.cache_write.input_tokens"
	genAIResponseFinishReasons = "gen_ai.response.finish_reasons"
	genAIResponseID            = "gen_ai.response.id"
	genAIDataSourceID          = "gen_ai.data_source.id"
	genAIOutputType            = "gen_ai.output.type"
)

// genAIUsageTotalTokens is not part of the semantic conventions, the conventions
// only define the input and the output token counters. It is honored when a
// provider emits it anyway, because some vendors already report the aggregate.
// The value stays a reference value in the detail of a single call, the aggregate
// of the trace is always the sum of the input and the output tokens.
const genAIUsageTotalTokens = "gen_ai.usage.total_tokens"

// genAIContentAttributes hold prompts, completions and tool payloads. They are Opt-In
// in the semantic conventions and may embed PII, so their values are never read.
var genAIContentAttributes = map[string]struct{}{
	genAIInputMessages:     {},
	genAIOutputMessages:    {},
	genAIToolCallArguments: {},
	genAIToolCallResult:    {},
}

// genAIRecognizedAttributes is the pinned set of attributes the analyzer is allowed to
// read. An unknown gen_ai.* attribute still marks a span as generative AI, but its
// value is never interpreted.
var genAIRecognizedAttributes = map[string]struct{}{
	genAIOperationName:         {},
	genAIProviderName:          {},
	genAIRequestModel:          {},
	genAIResponseModel:         {},
	genAIConversationID:        {},
	genAIAgentName:             {},
	genAIAgentID:               {},
	genAIAgentDescription:      {},
	genAIAgentVersion:          {},
	genAIToolName:              {},
	genAIToolCallID:            {},
	genAIToolCallArguments:     {},
	genAIToolCallResult:        {},
	genAIInputMessages:         {},
	genAIOutputMessages:        {},
	genAIUsageInputTokens:      {},
	genAIUsageOutputTokens:     {},
	genAIUsageReasoningTokens:  {},
	genAIUsageCacheReadTokens:  {},
	genAIUsageCacheWriteTokens: {},
	genAIResponseFinishReasons: {},
	genAIResponseID:            {},
	genAIDataSourceID:          {},
	genAIOutputType:            {},
	genAIUsageTotalTokens:      {},
}

// genAITokenAttributes lists every token attribute the analyzer recognizes. The cache
// and reasoning counters are breakdowns of the input/output counters, so they are never
// added on top of them.
var genAITokenAttributes = []string{
	genAIUsageInputTokens,
	genAIUsageOutputTokens,
	genAIUsageReasoningTokens,
	genAIUsageCacheReadTokens,
	genAIUsageCacheWriteTokens,
	genAIUsageTotalTokens,
}

// genAIModelSignalAttributes mark a span as a model invocation even when the operation
// name is missing or carries a custom value.
var genAIModelSignalAttributes = []string{
	genAIRequestModel,
	genAIResponseModel,
	genAIResponseID,
	genAIResponseFinishReasons,
	genAIOutputType,
	genAIDataSourceID,
}

var (
	genAIModelOperations = map[string]struct{}{
		"chat":             {},
		"generate_content": {},
		"text_completion":  {},
		"embeddings":       {},
		"fetch_response":   {},
	}
	genAIToolOperations = map[string]struct{}{
		"execute_tool": {},
	}
	genAIAgentOperations = map[string]struct{}{
		"invoke_agent":    {},
		"create_agent":    {},
		"invoke_workflow": {},
		"plan":            {},
	}
)

// genAIClass is the mutually exclusive category a generative AI span falls into.
type genAIClass int

const (
	genAIClassNone genAIClass = iota
	genAIClassModel
	genAIClassTool
	genAIClassAgent
)

// genAIOrderedSpan pairs a span with its resolved start time so the per-category
// detail lists can be emitted in chronological order.
type genAIOrderedSpan struct {
	span        *Span
	startMicros int64
}

// HasGenAISpans reports whether any span carries generative AI attributes.
func HasGenAISpans(spans []Span) bool {
	for i := range spans {
		if HasGenAIAttributes(spans[i].Attributes) {
			return true
		}
	}
	return false
}

// HasGenAIAttributes reports whether an attribute map contains generative AI attributes.
// The search path needs this check on raw provider tags, so it must not require a
// normalized span.
func HasGenAIAttributes(tags map[string]string) bool {
	for key := range tags {
		if isGenAIAttributeKey(key) {
			return true
		}
	}
	return false
}

// BuildGenAISummary aggregates the generative AI footprint of the given spans.
// It never fails: spans without generative AI attributes yield Present=false.
func BuildGenAISummary(spans []Span) *GenAISummary {
	summary := &GenAISummary{Present: false}
	ordered := make([]genAIOrderedSpan, 0, len(spans))
	conversationIDs := map[string]struct{}{}
	providers := map[string]struct{}{}
	models := map[string]struct{}{}
	for i := range spans {
		span := &spans[i]
		if !HasGenAIAttributes(span.Attributes) {
			continue
		}
		summary.Present = true
		ordered = append(ordered, genAIOrderedSpan{span: span, startMicros: spanStartMicros(span)})
		addString(conversationIDs, genAIAttribute(span.Attributes, genAIConversationID))
		addString(providers, genAIAttribute(span.Attributes, genAIProviderName))
		addString(models, genAIAttribute(span.Attributes, genAIRequestModel))
		addString(models, genAIAttribute(span.Attributes, genAIResponseModel))
		summary.InputTokens += genAIIntAttribute(span.Attributes, genAIUsageInputTokens)
		summary.OutputTokens += genAIIntAttribute(span.Attributes, genAIUsageOutputTokens)
	}
	// The semantic conventions define the input and the output token counters
	// only, there is no standard total token attribute. The total is therefore a
	// derived value with a single definition: it always equals input plus output.
	// A provider that reports the non standard gen_ai.usage.total_tokens only
	// shows up as a reference value in the detail of a single call, it never
	// takes part in the aggregate, so a consumer never has to choose between
	// two definitions of the total.
	summary.TotalTokens = summary.InputTokens + summary.OutputTokens
	if !summary.Present {
		return summary
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].startMicros != ordered[j].startMicros {
			return ordered[i].startMicros < ordered[j].startMicros
		}
		return ordered[i].span.SpanID < ordered[j].span.SpanID
	})
	summary.ConversationIDs = sortedKeys(conversationIDs)
	summary.Providers = sortedKeys(providers)
	summary.Models = sortedKeys(models)

	for _, item := range ordered {
		span := item.span
		switch classifyGenAISpan(span) {
		case genAIClassModel:
			inputTokens := genAIIntAttribute(span.Attributes, genAIUsageInputTokens)
			outputTokens := genAIIntAttribute(span.Attributes, genAIUsageOutputTokens)
			totalTokens := genAISpanTotalTokens(span.Attributes)
			summary.ModelCallCount++
			if len(summary.ModelCalls) < maxGenAICalls {
				summary.ModelCalls = append(summary.ModelCalls, GenAIModelCall{
					SpanID:         span.SpanID,
					ServiceName:    span.ServiceName,
					OperationName:  span.OperationName,
					Operation:      genAIAttribute(span.Attributes, genAIOperationName),
					ProviderName:   genAIAttribute(span.Attributes, genAIProviderName),
					RequestModel:   genAIAttribute(span.Attributes, genAIRequestModel),
					ResponseModel:  genAIAttribute(span.Attributes, genAIResponseModel),
					InputTokens:    inputTokens,
					OutputTokens:   outputTokens,
					TotalTokens:    totalTokens,
					DurationMicros: span.DurationMicros,
					Error:          spanFailed(span),
				})
			} else {
				summary.Truncated = true
			}
		case genAIClassTool:
			summary.ToolCallCount++
			if len(summary.ToolCalls) < maxGenAICalls {
				summary.ToolCalls = append(summary.ToolCalls, GenAIToolCall{
					SpanID:         span.SpanID,
					ServiceName:    span.ServiceName,
					ToolName:       genAIAttribute(span.Attributes, genAIToolName),
					ToolCallID:     genAIAttribute(span.Attributes, genAIToolCallID),
					DurationMicros: span.DurationMicros,
					Error:          spanFailed(span),
				})
			} else {
				summary.Truncated = true
			}
		case genAIClassAgent:
			summary.AgentCount++
			if len(summary.Agents) < maxGenAICalls {
				summary.Agents = append(summary.Agents, GenAIAgent{
					SpanID:         span.SpanID,
					ServiceName:    span.ServiceName,
					AgentName:      genAIAttribute(span.Attributes, genAIAgentName),
					AgentID:        genAIAttribute(span.Attributes, genAIAgentID),
					DurationMicros: span.DurationMicros,
					Error:          spanFailed(span),
				})
			} else {
				summary.Truncated = true
			}
		}
	}
	return summary
}

// classifyGenAISpan assigns exactly one category to a generative AI span.
// The precedence is model operation, then tool, then agent, and finally a model call
// inferred from model or token attributes.
func classifyGenAISpan(span *Span) genAIClass {
	operation := strings.ToLower(strings.TrimSpace(genAIAttribute(span.Attributes, genAIOperationName)))
	if _, ok := genAIModelOperations[operation]; ok {
		return genAIClassModel
	}
	if _, ok := genAIToolOperations[operation]; ok {
		return genAIClassTool
	}
	if _, ok := genAIAgentOperations[operation]; ok {
		return genAIClassAgent
	}
	if _, ok := genAIAttributeOK(span.Attributes, genAIToolName); ok {
		return genAIClassTool
	}
	if _, ok := genAIAttributeOK(span.Attributes, genAIAgentName); ok {
		return genAIClassAgent
	}
	for _, name := range genAIModelSignalAttributes {
		if _, ok := genAIAttributeOK(span.Attributes, name); ok {
			return genAIClassModel
		}
	}
	for _, name := range genAITokenAttributes {
		if _, ok := genAIAttributeOK(span.Attributes, name); ok {
			return genAIClassModel
		}
	}
	return genAIClassNone
}

// genAISpanTotalTokens returns the token total of a single span. A provider that
// reports the aggregate of the span wins, otherwise the total is derived from the
// input and the output counters of that span. The value is only meaningful per
// span: it is a reference value in the detail of a single call and never takes
// part in the aggregate of the trace, whose total is always input plus output.
func genAISpanTotalTokens(attributes map[string]string) int {
	if override, ok := genAIAttributeOK(attributes, genAIUsageTotalTokens); ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(override)); err == nil {
			return parsed
		}
	}
	return genAIIntAttribute(attributes, genAIUsageInputTokens) + genAIIntAttribute(attributes, genAIUsageOutputTokens)
}

func isGenAIAttributeKey(key string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(key)), genAIPrefix)
}

// genAIAttribute reads a recognized, non-content generative AI attribute.
func genAIAttribute(attributes map[string]string, name string) string {
	value, _ := genAIAttributeOK(attributes, name)
	return value
}

// genAIAttributeOK reads a recognized, non-content generative AI attribute.
func genAIAttributeOK(attributes map[string]string, name string) (string, bool) {
	if _, content := genAIContentAttributes[name]; content {
		return "", false
	}
	if _, recognized := genAIRecognizedAttributes[name]; !recognized {
		return "", false
	}
	return lookupAttributeOK(attributes, name)
}

func genAIIntAttribute(attributes map[string]string, name string) int {
	raw, ok := genAIAttributeOK(attributes, name)
	if !ok {
		return 0
	}
	return parseIntegralAttribute(strings.TrimSpace(raw))
}

// parseIntegralAttribute reads an integer attribute value. The attribute map is
// filled from the provider payload, where a number arrives as a JSON number.
// The decoder turns it into a float64 and the formatter renders it with the
// 'g' format, so a token count of 1000000 becomes "1e+06". A plain integer
// parse drops such a value silently, so the integral scientific notation is
// accepted as well.
func parseIntegralAttribute(raw string) int {
	if raw == "" {
		return 0
	}
	if parsed, err := strconv.Atoi(raw); err == nil {
		return parsed
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	if value != math.Trunc(value) || value > math.MaxInt64 || value < math.MinInt64 {
		return 0
	}
	return int(value)
}

func addString(target map[string]struct{}, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	target[trimmed] = struct{}{}
}

func sortedKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

// spanStartMicros converts a span start timestamp into unix microseconds.
// Unparsable timestamps degrade to zero so the analyzer never panics.
func spanStartMicros(span *Span) int64 {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(span.StartTime))
	if err != nil {
		return 0
	}
	return parsed.UnixMicro()
}

// spanFailed reports whether a span is in an error state.
func spanFailed(span *Span) bool {
	return span.Error || strings.EqualFold(span.Status, "ERROR")
}
