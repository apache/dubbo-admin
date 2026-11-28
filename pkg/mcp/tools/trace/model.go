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

import "time"

// Trace is a backend-independent, bounded representation of a distributed trace.
type Trace struct {
	TraceID           string   `json:"traceId"`
	StartTime         string   `json:"startTime"`
	DurationMicros    int64    `json:"durationMicros"`
	SpanCount         int      `json:"spanCount"`
	ReturnedSpanCount int      `json:"returnedSpanCount"`
	ServiceCount      int      `json:"serviceCount"`
	ErrorSpanCount    int      `json:"errorSpanCount"`
	Services          []string `json:"services"`
	RootSpanIDs       []string `json:"rootSpanIds"`
	Spans             []Span   `json:"spans"`
	SourceEngine      string   `json:"sourceEngine"`
	Truncated         bool     `json:"truncated"`
}

// Span contains the diagnostic fields needed to reconstruct latency and error propagation.
type Span struct {
	SpanID         string            `json:"spanId"`
	ParentSpanID   string            `json:"parentSpanId,omitempty"`
	ServiceName    string            `json:"serviceName"`
	OperationName  string            `json:"operationName"`
	StartTime      string            `json:"startTime"`
	DurationMicros int64             `json:"durationMicros"`
	Status         string            `json:"status"`
	Error          bool              `json:"error"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	Events         []Event           `json:"events,omitempty"`
	Truncated      bool              `json:"truncated"`
}

// Event represents a bounded Jaeger span log entry.
type Event struct {
	Timestamp  string            `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// TraceSearchQuery bounds a provider-side trace lookup.
type TraceSearchQuery struct {
	Service     string
	Operation   string
	TraceID     string
	Tags        map[string]string
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	MinDuration time.Duration
}

// TraceSummary is a bounded per-trace overview produced by trace search.
type TraceSummary struct {
	TraceID        string `json:"traceId"`
	RootService    string `json:"rootService,omitempty"`
	RootOperation  string `json:"rootOperation,omitempty"`
	StartTime      string `json:"startTime"`
	DurationMicros int64  `json:"durationMicros"`
	SpanCount      int    `json:"spanCount"`
	ErrorSpanCount int    `json:"errorSpanCount"`
	ServiceCount   int    `json:"serviceCount"`
	HasError       bool   `json:"hasError"`
	TraceKind      string `json:"traceKind,omitempty"` // "genai" when the trace carries gen_ai.* attributes
}

// TraceSearchResult aggregates the outcome of a trace search.
type TraceSearchResult struct {
	Traces       []TraceSummary `json:"traces"`
	Total        int            `json:"total"`
	Truncated    bool           `json:"truncated"`
	SourceEngine string         `json:"sourceEngine"`
}

// CriticalPathEntry is one hop of the latency-determining root-to-leaf chain.
type CriticalPathEntry struct {
	SpanID         string `json:"spanId"`
	ServiceName    string `json:"serviceName"`
	OperationName  string `json:"operationName"`
	StartTime      string `json:"startTime"`
	DurationMicros int64  `json:"durationMicros"`
	SelfMicros     int64  `json:"selfMicros"`
	WaitMicros     int64  `json:"waitMicros"`
}

// ServiceLatency aggregates the spans contributed by one service.
type ServiceLatency struct {
	ServiceName    string  `json:"serviceName"`
	SpanCount      int     `json:"spanCount"`
	ErrorSpanCount int     `json:"errorSpanCount"`
	TotalMicros    int64   `json:"totalMicros"`
	MaxMicros      int64   `json:"maxMicros"`
	AvgMicros      int64   `json:"avgMicros"`
	SharePercent   float64 `json:"sharePercent"`
}

// SlowSpan is a span ranked by the time it consumed on its own.
type SlowSpan struct {
	SpanID         string  `json:"spanId"`
	ServiceName    string  `json:"serviceName"`
	OperationName  string  `json:"operationName"`
	DurationMicros int64   `json:"durationMicros"`
	SharePercent   float64 `json:"sharePercent"`
	OnCriticalPath bool    `json:"onCriticalPath"`
}

// ErrorSpan describes a failed span together with its root-cause verdict.
type ErrorSpan struct {
	SpanID        string `json:"spanId"`
	ServiceName   string `json:"serviceName"`
	OperationName string `json:"operationName"`
	ErrorType     string `json:"errorType,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
	Status        string `json:"status"`
	RootCause     bool   `json:"rootCause"`
}

// DependencyEdge is a cross-service call observed inside a single trace.
type DependencyEdge struct {
	CallerService string `json:"callerService"`
	CalleeService string `json:"calleeService"`
	SpanCount     int    `json:"spanCount"`
	TotalMicros   int64  `json:"totalMicros"`
	ErrorCount    int    `json:"errorCount"`
}

// GenAIModelCall describes one generative model invocation observed in a trace.
type GenAIModelCall struct {
	SpanID         string `json:"spanId"`
	ServiceName    string `json:"serviceName"`
	OperationName  string `json:"operationName"`
	Operation      string `json:"operation,omitempty"`
	ProviderName   string `json:"providerName,omitempty"`
	RequestModel   string `json:"requestModel,omitempty"`
	ResponseModel  string `json:"responseModel,omitempty"`
	InputTokens    int    `json:"inputTokens,omitempty"`
	OutputTokens   int    `json:"outputTokens,omitempty"`
	TotalTokens    int    `json:"totalTokens,omitempty"`
	DurationMicros int64  `json:"durationMicros"`
	Error          bool   `json:"error"`
}

// GenAIToolCall describes one tool invocation observed in a trace.
type GenAIToolCall struct {
	SpanID         string `json:"spanId"`
	ServiceName    string `json:"serviceName"`
	ToolName       string `json:"toolName,omitempty"`
	ToolCallID     string `json:"toolCallId,omitempty"`
	DurationMicros int64  `json:"durationMicros"`
	Error          bool   `json:"error"`
}

// GenAIAgent describes one agent-scoped span observed in a trace.
type GenAIAgent struct {
	SpanID         string `json:"spanId"`
	ServiceName    string `json:"serviceName"`
	AgentName      string `json:"agentName,omitempty"`
	AgentID        string `json:"agentId,omitempty"`
	DurationMicros int64  `json:"durationMicros"`
	Error          bool   `json:"error"`
}

// GenAISummary aggregates the generative AI footprint of a trace.
type GenAISummary struct {
	Present         bool             `json:"present"`
	ConversationIDs []string         `json:"conversationIds,omitempty"`
	Providers       []string         `json:"providers,omitempty"`
	Models          []string         `json:"models,omitempty"`
	InputTokens     int              `json:"inputTokens"`
	OutputTokens    int              `json:"outputTokens"`
	TotalTokens     int              `json:"totalTokens"`
	ModelCallCount  int              `json:"modelCallCount"`
	ToolCallCount   int              `json:"toolCallCount"`
	AgentCount      int              `json:"agentCount"`
	ModelCalls      []GenAIModelCall `json:"modelCalls,omitempty"`
	ToolCalls       []GenAIToolCall  `json:"toolCalls,omitempty"`
	Agents          []GenAIAgent     `json:"agents,omitempty"`
	Truncated       bool             `json:"truncated"`
}

// TraceAnalysis is the structured diagnosis of a single trace.
type TraceAnalysis struct {
	TraceID          string              `json:"traceId"`
	Summary          string              `json:"summary"`
	StartTime        string              `json:"startTime"`
	DurationMicros   int64               `json:"durationMicros"`
	SpanCount        int                 `json:"spanCount"`
	ServiceCount     int                 `json:"serviceCount"`
	ErrorSpanCount   int                 `json:"errorSpanCount"`
	CriticalPath     []CriticalPathEntry `json:"criticalPath,omitempty"`
	ServiceBreakdown []ServiceLatency    `json:"serviceBreakdown,omitempty"`
	SlowSpans        []SlowSpan          `json:"slowSpans,omitempty"`
	ErrorSpans       []ErrorSpan         `json:"errorSpans,omitempty"`
	Dependencies     []DependencyEdge    `json:"dependencies,omitempty"`
	GenAI            *GenAISummary       `json:"genai,omitempty"`
	Truncated        bool                `json:"truncated"`
}
