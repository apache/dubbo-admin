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
	"encoding/json"
	"strings"
	"testing"
)

func genAISpan(spanID string, attributes map[string]string) Span {
	return Span{
		SpanID:        spanID,
		ServiceName:   "dubbo-admin-sre-agent",
		OperationName: spanID,
		StartTime:     analysisBase,
		Attributes:    attributes,
	}
}

func TestHasGenAIAttributes(t *testing.T) {
	if HasGenAIAttributes(map[string]string{"rpc.system": "apache_dubbo"}) {
		t.Fatal("did not expect a non generative AI attribute to match")
	}
	if !HasGenAIAttributes(map[string]string{"gen_ai.operation.name": "chat"}) {
		t.Fatal("expected a generative AI attribute to match")
	}
	if !HasGenAIAttributes(map[string]string{" GEN_AI.Operation.Name ": "chat"}) {
		t.Fatal("expected the detection to tolerate casing and padding")
	}
}

func TestHasGenAISpans(t *testing.T) {
	if HasGenAISpans([]Span{{SpanID: "01"}}) {
		t.Fatal("did not expect a plain span to match")
	}
	if !HasGenAISpans([]Span{{SpanID: "01"}, {SpanID: "02", Attributes: map[string]string{"gen_ai.agent.name": "sre"}}}) {
		t.Fatal("expected a generative AI span to match")
	}
}

func TestBuildGenAISummaryClassifiesModelCall(t *testing.T) {
	summary := BuildGenAISummary([]Span{
		genAISpan("01", map[string]string{
			"gen_ai.operation.name":      "chat",
			"gen_ai.provider.name":       "openai",
			"gen_ai.request.model":       "gpt-4o",
			"gen_ai.response.model":      "gpt-4o-2024-08-06",
			"gen_ai.conversation.id":     "conv-1",
			"gen_ai.usage.input_tokens":  "120",
			"gen_ai.usage.output_tokens": "30",
		}),
	})
	if !summary.Present {
		t.Fatal("expected the summary to be present")
	}
	if summary.ModelCallCount != 1 || summary.ToolCallCount != 0 || summary.AgentCount != 0 {
		t.Fatalf("unexpected call counts: %#v", summary)
	}
	if len(summary.ModelCalls) != 1 {
		t.Fatalf("unexpected model calls: %#v", summary.ModelCalls)
	}
	call := summary.ModelCalls[0]
	if call.Operation != "chat" || call.ProviderName != "openai" || call.RequestModel != "gpt-4o" || call.ResponseModel != "gpt-4o-2024-08-06" {
		t.Fatalf("unexpected model call: %#v", call)
	}
	if call.InputTokens != 120 || call.OutputTokens != 30 || call.TotalTokens != 150 {
		t.Fatalf("unexpected token usage: %#v", call)
	}
	if summary.InputTokens != 120 || summary.OutputTokens != 30 || summary.TotalTokens != 150 {
		t.Fatalf("unexpected token totals: %#v", summary)
	}
	if len(summary.Providers) != 1 || summary.Providers[0] != "openai" {
		t.Fatalf("unexpected providers: %#v", summary.Providers)
	}
	if len(summary.Models) != 2 || summary.Models[0] != "gpt-4o" {
		t.Fatalf("unexpected models: %#v", summary.Models)
	}
	if len(summary.ConversationIDs) != 1 || summary.ConversationIDs[0] != "conv-1" {
		t.Fatalf("unexpected conversations: %#v", summary.ConversationIDs)
	}
}

func TestBuildGenAISummaryClassifiesToolAndAgent(t *testing.T) {
	summary := BuildGenAISummary([]Span{
		genAISpan("01", map[string]string{
			"gen_ai.operation.name": "execute_tool",
			"gen_ai.tool.name":      "search_logs",
			"gen_ai.tool.call.id":   "call-1",
		}),
		genAISpan("02", map[string]string{
			"gen_ai.operation.name": "invoke_agent",
			"gen_ai.agent.name":     "dubbo-admin-sre-agent",
			"gen_ai.agent.id":       "agent-42",
		}),
	})
	if summary.ToolCallCount != 1 || summary.AgentCount != 1 || summary.ModelCallCount != 0 {
		t.Fatalf("unexpected call counts: %#v", summary)
	}
	if len(summary.ToolCalls) != 1 || summary.ToolCalls[0].ToolName != "search_logs" || summary.ToolCalls[0].ToolCallID != "call-1" {
		t.Fatalf("unexpected tool calls: %#v", summary.ToolCalls)
	}
	if len(summary.Agents) != 1 || summary.Agents[0].AgentName != "dubbo-admin-sre-agent" || summary.Agents[0].AgentID != "agent-42" {
		t.Fatalf("unexpected agents: %#v", summary.Agents)
	}
}

func TestBuildGenAISummaryClassifiesByAttributesWithoutOperationName(t *testing.T) {
	summary := BuildGenAISummary([]Span{
		genAISpan("01", map[string]string{"gen_ai.tool.name": "search_logs"}),
		genAISpan("02", map[string]string{"gen_ai.agent.name": "sre"}),
		genAISpan("03", map[string]string{"gen_ai.request.model": "gpt-4o"}),
		genAISpan("04", map[string]string{"gen_ai.usage.input_tokens": "10"}),
	})
	if summary.ToolCallCount != 1 || summary.AgentCount != 1 || summary.ModelCallCount != 2 {
		t.Fatalf("unexpected call counts: %#v", summary)
	}
}

func TestBuildGenAISummaryKeepsProviderTokenTotalAsCallDetailOnly(t *testing.T) {
	// The semantic conventions define the input and the output token counters
	// only. The total of the summary is therefore always their sum, the value
	// reported by the provider stays a reference in the detail of the call.
	summary := BuildGenAISummary([]Span{
		genAISpan("01", map[string]string{
			"gen_ai.operation.name":      "chat",
			"gen_ai.usage.input_tokens":  "10",
			"gen_ai.usage.output_tokens": "5",
			"gen_ai.usage.total_tokens":  "99",
		}),
	})
	if summary.ModelCalls[0].TotalTokens != 99 {
		t.Fatalf("expected the provider total to be kept in the call detail: %#v", summary.ModelCalls[0])
	}
	if summary.TotalTokens != summary.InputTokens+summary.OutputTokens {
		t.Fatalf("expected the total to equal the input plus the output tokens: %#v", summary)
	}
	if summary.TotalTokens != 15 {
		t.Fatalf("expected the total to be derived from the input and the output tokens: %#v", summary)
	}
}

func TestBuildGenAISummaryWithoutGenAIAttributes(t *testing.T) {
	summary := BuildGenAISummary([]Span{{SpanID: "01", ServiceName: "order-service"}})
	if summary.Present {
		t.Fatal("did not expect the summary to be present")
	}
	if summary.TotalTokens != 0 || summary.ModelCallCount != 0 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestBuildGenAISummaryDerivesAggregateTotalFromInputAndOutput(t *testing.T) {
	// Only one span reports its own total. The aggregate must not be influenced
	// by it, the total is always the sum of the input and the output tokens.
	summary := BuildGenAISummary([]Span{
		genAISpan("01", map[string]string{
			"gen_ai.operation.name":      "chat",
			"gen_ai.usage.input_tokens":  "100",
			"gen_ai.usage.output_tokens": "50",
		}),
		genAISpan("02", map[string]string{
			"gen_ai.operation.name":      "chat",
			"gen_ai.usage.input_tokens":  "10",
			"gen_ai.usage.output_tokens": "5",
			"gen_ai.usage.total_tokens":  "999",
		}),
	})
	if summary.InputTokens != 110 || summary.OutputTokens != 55 {
		t.Fatalf("unexpected token counters: %#v", summary)
	}
	// The reported total of the second span does not take part in the aggregate.
	if summary.TotalTokens != 110+55 {
		t.Fatalf("expected the total to equal the input plus the output tokens, got %d", summary.TotalTokens)
	}
	if len(summary.ModelCalls) != 2 {
		t.Fatalf("unexpected model calls: %#v", summary.ModelCalls)
	}
	totals := map[string]int{}
	for _, call := range summary.ModelCalls {
		totals[call.SpanID] = call.TotalTokens
	}
	// The detail of a single call keeps the value reported by the provider.
	if totals["01"] != 150 || totals["02"] != 999 {
		t.Fatalf("unexpected per call totals: %#v", totals)
	}
}

func TestBuildGenAISummaryDoesNotReadContentAttributes(t *testing.T) {
	summary := BuildGenAISummary([]Span{
		genAISpan("01", map[string]string{
			"gen_ai.operation.name":      "chat",
			"gen_ai.request.model":       "gpt-4o",
			"gen_ai.input.messages":      `{"messages":[{"role":"user","content":"secret prompt"}]}`,
			"gen_ai.output.messages":     `{"messages":[{"role":"assistant","content":"secret answer"}]}`,
			"gen_ai.tool.call.arguments": `{"query":"secret argument"}`,
			"gen_ai.tool.call.result":    `{"rows":["secret result"]}`,
		}),
	})
	if summary.ModelCallCount != 1 {
		t.Fatalf("unexpected call counts: %#v", summary)
	}
	for _, call := range summary.ModelCalls {
		if call.RequestModel != "gpt-4o" {
			t.Fatalf("unexpected model call: %#v", call)
		}
	}
	// The content attributes must not reach the tool output at all.
	encoded, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret prompt", "secret answer", "secret argument", "secret result"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("expected the content attribute to be omitted, got %s", encoded)
		}
	}
}

func TestBuildGenAISummaryTruncatesDetailLists(t *testing.T) {
	spans := make([]Span, 0, maxGenAICalls+5)
	for i := 0; i < maxGenAICalls+5; i++ {
		spans = append(spans, genAISpan(string(rune('a'+i%26))+string(rune('a'+i/26)), map[string]string{
			"gen_ai.operation.name": "chat",
			"gen_ai.request.model":  "gpt-4o",
		}))
	}
	summary := BuildGenAISummary(spans)
	if summary.ModelCallCount != maxGenAICalls+5 {
		t.Fatalf("expected the real call count, got %d", summary.ModelCallCount)
	}
	if len(summary.ModelCalls) != maxGenAICalls {
		t.Fatalf("expected the detail list to be capped, got %d", len(summary.ModelCalls))
	}
	if !summary.Truncated {
		t.Fatal("expected the summary to be marked as truncated")
	}
}

func TestBuildGenAISummarySortsDetailsByStartTime(t *testing.T) {
	late := genAISpan("02", map[string]string{"gen_ai.operation.name": "chat"})
	late.StartTime = shiftTime(analysisBase, 5000)
	early := genAISpan("01", map[string]string{"gen_ai.operation.name": "chat"})
	early.StartTime = shiftTime(analysisBase, 1000)
	summary := BuildGenAISummary([]Span{late, early})
	if len(summary.ModelCalls) != 2 {
		t.Fatalf("unexpected model calls: %#v", summary.ModelCalls)
	}
	if summary.ModelCalls[0].SpanID != "01" || summary.ModelCalls[1].SpanID != "02" {
		t.Fatalf("unexpected model call order: %#v", summary.ModelCalls)
	}
}

func TestGenAIOperationClassification(t *testing.T) {
	tests := []struct {
		operation string
		expected  genAIClass
	}{
		{operation: "chat", expected: genAIClassModel},
		{operation: "generate_content", expected: genAIClassModel},
		{operation: "text_completion", expected: genAIClassModel},
		{operation: "embeddings", expected: genAIClassModel},
		{operation: "fetch_response", expected: genAIClassModel},
		{operation: "execute_tool", expected: genAIClassTool},
		{operation: "invoke_agent", expected: genAIClassAgent},
		{operation: "create_agent", expected: genAIClassAgent},
		{operation: "invoke_workflow", expected: genAIClassAgent},
		{operation: "plan", expected: genAIClassAgent},
	}
	for _, test := range tests {
		t.Run(test.operation, func(t *testing.T) {
			span := genAISpan("01", map[string]string{"gen_ai.operation.name": test.operation})
			if actual := classifyGenAISpan(&span); actual != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestGenAIAnalysisEmbedsSummaryInTraceAnalysis(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "dubbo-admin-sre-agent", "chat gpt-4o", 1000),
			spanAt(100, "02", "01", "dubbo-admin-sre-agent", "execute_tool search_logs", 500),
		},
	}
	trace.Spans[0].Attributes = map[string]string{"gen_ai.operation.name": "chat", "gen_ai.usage.input_tokens": "5"}
	trace.Spans[1].Attributes = map[string]string{"gen_ai.operation.name": "execute_tool", "gen_ai.tool.name": "search_logs"}
	analysis := BuildTraceAnalysis(trace)
	if analysis.GenAI == nil {
		t.Fatal("expected a generative AI summary")
	}
	if !analysis.GenAI.Present || analysis.GenAI.ModelCallCount != 1 || analysis.GenAI.ToolCallCount != 1 {
		t.Fatalf("unexpected generative AI summary: %#v", analysis.GenAI)
	}
}

func TestGenAIIntAttributeAcceptsIntegralScientificNotation(t *testing.T) {
	// The attribute map is filled from a JSON payload, so a number arrives as a
	// float64 and the formatter renders it with the 'g' format: 1000000 becomes
	// "1e+06". A plain integer parse drops such a value silently, which would
	// report a million token model call as a call without any token.
	attributes := map[string]string{
		"gen_ai.usage.input_tokens":  "1e+06",
		"gen_ai.usage.output_tokens": "1500000",
		"gen_ai.usage.total_tokens":  "12.5",
	}
	if got := genAIIntAttribute(attributes, "gen_ai.usage.input_tokens"); got != 1000000 {
		t.Fatalf("input tokens = %d, want %d", got, 1000000)
	}
	if got := genAIIntAttribute(attributes, "gen_ai.usage.output_tokens"); got != 1500000 {
		t.Fatalf("output tokens = %d, want %d", got, 1500000)
	}
	// A fractional value is not a token count and must not be rounded.
	if got := genAIIntAttribute(attributes, "gen_ai.usage.total_tokens"); got != 0 {
		t.Fatalf("total tokens = %d, want %d", got, 0)
	}
}
