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
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// analysisBase is the reference point of the test timeline.
const analysisBase = "2026-10-24T10:00:00.000000000Z"

func spanAt(offsetMicros int64, spanID, parentID, service, operation string, durationMicros int64) Span {
	return Span{
		SpanID:         spanID,
		ParentSpanID:   parentID,
		ServiceName:    service,
		OperationName:  operation,
		StartTime:      shiftTime(analysisBase, offsetMicros),
		DurationMicros: durationMicros,
	}
}

func shiftTime(base string, offsetMicros int64) string {
	parsed, err := time.Parse(time.RFC3339Nano, base)
	if err != nil {
		panic(err)
	}
	return parsed.Add(time.Duration(offsetMicros) * time.Microsecond).UTC().Format(time.RFC3339Nano)
}

func TestBuildTraceAnalysisLinearChain(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "GET /checkout", 100000),
			spanAt(10000, "02", "01", "order-service", "checkout", 80000),
			spanAt(20000, "03", "02", "payment-service", "charge", 50000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.TraceID != testTraceID {
		t.Fatalf("unexpected trace id: %s", analysis.TraceID)
	}
	if analysis.SpanCount != 3 || analysis.ServiceCount != 3 {
		t.Fatalf("unexpected counters: %#v", analysis)
	}
	if analysis.DurationMicros != 100000 {
		t.Fatalf("unexpected duration: %d", analysis.DurationMicros)
	}
	if len(analysis.CriticalPath) != 3 {
		t.Fatalf("unexpected critical path: %#v", analysis.CriticalPath)
	}
	if analysis.CriticalPath[0].SpanID != "01" || analysis.CriticalPath[2].SpanID != "03" {
		t.Fatalf("unexpected critical path order: %#v", analysis.CriticalPath)
	}
	// The root waits for the child for 80ms, the child waits for its own child for 50ms.
	if analysis.CriticalPath[0].SelfMicros != 20000 || analysis.CriticalPath[0].WaitMicros != 80000 {
		t.Fatalf("unexpected root self time: %#v", analysis.CriticalPath[0])
	}
	if analysis.CriticalPath[1].SelfMicros != 30000 || analysis.CriticalPath[1].WaitMicros != 50000 {
		t.Fatalf("unexpected child self time: %#v", analysis.CriticalPath[1])
	}
	if analysis.CriticalPath[2].SelfMicros != 50000 || analysis.CriticalPath[2].WaitMicros != 0 {
		t.Fatalf("unexpected leaf self time: %#v", analysis.CriticalPath[2])
	}
}

func TestBuildTraceAnalysisCriticalPathFollowsSlowBranch(t *testing.T) {
	// The root fans out into a fast branch and a slow branch. The critical path has
	// to follow the branch that finishes last.
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "GET /checkout", 200000),
			spanAt(1000, "02", "01", "order-service", "fast", 10000),
			spanAt(2000, "03", "01", "payment-service", "slow", 150000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if len(analysis.CriticalPath) != 2 {
		t.Fatalf("unexpected critical path: %#v", analysis.CriticalPath)
	}
	if analysis.CriticalPath[1].SpanID != "03" || analysis.CriticalPath[1].ServiceName != "payment-service" {
		t.Fatalf("expected the critical path to follow the slow branch: %#v", analysis.CriticalPath)
	}
}

func TestBuildTraceAnalysisCriticalPathPrefersLatestRoot(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "early", 10000),
			spanAt(5000, "02", "", "frontend", "late", 60000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if len(analysis.CriticalPath) != 1 || analysis.CriticalPath[0].SpanID != "02" {
		t.Fatalf("expected the latest root to start the critical path: %#v", analysis.CriticalPath)
	}
}

func TestBuildTraceAnalysisSelfTimeUsesUnionOfChildren(t *testing.T) {
	// Two concurrent children overlap, so the covered time is the union and not the
	// sum of both durations.
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			spanAt(10000, "02", "01", "order-service", "a", 50000),
			spanAt(20000, "03", "01", "order-service", "b", 50000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.CriticalPath[0].WaitMicros != 60000 {
		t.Fatalf("expected the union of the concurrent children, got %d", analysis.CriticalPath[0].WaitMicros)
	}
	if analysis.CriticalPath[0].SelfMicros != 40000 {
		t.Fatalf("unexpected self time: %d", analysis.CriticalPath[0].SelfMicros)
	}
}

func TestBuildTraceAnalysisSelfTimeClipsChildrenToParent(t *testing.T) {
	// The child reaches beyond the parent, so only the overlap counts.
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			spanAt(50000, "02", "01", "order-service", "child", 900000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.CriticalPath[0].WaitMicros != 50000 {
		t.Fatalf("expected the child to be clipped, got %d", analysis.CriticalPath[0].WaitMicros)
	}
	if analysis.CriticalPath[0].SelfMicros != 50000 {
		t.Fatalf("unexpected self time: %d", analysis.CriticalPath[0].SelfMicros)
	}
}

func TestBuildTraceAnalysisServiceBreakdown(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			spanAt(10000, "02", "01", "order-service", "a", 60000),
			spanAt(20000, "03", "01", "order-service", "b", 30000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if len(analysis.ServiceBreakdown) != 2 {
		t.Fatalf("unexpected breakdown: %#v", analysis.ServiceBreakdown)
	}
	order := analysis.ServiceBreakdown[0]
	if order.ServiceName != "frontend" || order.TotalMicros != 100000 || order.SpanCount != 1 {
		t.Fatalf("unexpected breakdown entry: %#v", order)
	}
	if order.MaxMicros != 100000 || order.AvgMicros != 100000 {
		t.Fatalf("unexpected breakdown statistics: %#v", order)
	}
	if order.SharePercent != 100 {
		t.Fatalf("unexpected share: %v", order.SharePercent)
	}
	second := analysis.ServiceBreakdown[1]
	if second.ServiceName != "order-service" || second.TotalMicros != 90000 || second.SpanCount != 2 {
		t.Fatalf("unexpected breakdown entry: %#v", second)
	}
	if second.MaxMicros != 60000 || second.AvgMicros != 45000 {
		t.Fatalf("unexpected breakdown statistics: %#v", second)
	}
	// The service workload is 90ms against a 100ms wall clock, concurrent spans may
	// push the share above 100 percent but never below the real ratio.
	if second.SharePercent != 90 {
		t.Fatalf("unexpected share: %v", second.SharePercent)
	}
}

func TestBuildTraceAnalysisServiceBreakdownSortsByTotalTime(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			spanAt(10000, "02", "01", "alpha-service", "a", 50000),
			spanAt(20000, "03", "01", "beta-service", "b", 50000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.ServiceBreakdown[1].ServiceName != "alpha-service" {
		t.Fatalf("expected equal totals to be sorted by service name: %#v", analysis.ServiceBreakdown)
	}
}

func TestBuildTraceAnalysisSlowSpansUseSelfTime(t *testing.T) {
	// The early branch consumes the most time on its own, but it is not the branch
	// that finishes last, so it is not part of the critical path.
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 500000),
			spanAt(0, "02", "01", "order-service", "early-branch", 400000),
			spanAt(100000, "03", "01", "payment-service", "late-branch", 380000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if len(analysis.SlowSpans) != 3 {
		t.Fatalf("unexpected slow spans: %#v", analysis.SlowSpans)
	}
	if analysis.SlowSpans[0].SpanID != "02" {
		t.Fatalf("expected the branch with the largest self time to rank first: %#v", analysis.SlowSpans)
	}
	if analysis.SlowSpans[0].DurationMicros != 400000 {
		t.Fatalf("expected the total duration in the slow span: %#v", analysis.SlowSpans[0])
	}
	if analysis.SlowSpans[0].SharePercent != 80 {
		t.Fatalf("unexpected share: %v", analysis.SlowSpans[0].SharePercent)
	}
	if analysis.SlowSpans[0].OnCriticalPath {
		t.Fatal("did not expect the early branch to be part of the critical path")
	}
	if !analysis.SlowSpans[1].OnCriticalPath {
		t.Fatal("expected the late branch to be part of the critical path")
	}
	if len(analysis.CriticalPath) != 2 || analysis.CriticalPath[1].SpanID != "03" {
		t.Fatalf("unexpected critical path: %#v", analysis.CriticalPath)
	}
}

func TestBuildTraceAnalysisDependencyEdges(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			spanAt(10000, "02", "01", "order-service", "a", 40000),
			spanAt(20000, "03", "01", "order-service", "b", 30000),
			spanAt(30000, "04", "02", "payment-service", "charge", 20000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if len(analysis.Dependencies) != 2 {
		t.Fatalf("unexpected dependencies: %#v", analysis.Dependencies)
	}
	edges := map[string]DependencyEdge{}
	for _, edge := range analysis.Dependencies {
		edges[edge.CallerService+"->"+edge.CalleeService] = edge
	}
	frontend := edges["frontend->order-service"]
	if frontend.SpanCount != 2 || frontend.TotalMicros != 70000 || frontend.ErrorCount != 0 {
		t.Fatalf("unexpected edge: %#v", frontend)
	}
	payment := edges["order-service->payment-service"]
	if payment.SpanCount != 1 || payment.TotalMicros != 20000 {
		t.Fatalf("unexpected edge: %#v", payment)
	}
	// The edge with the largest total time comes first.
	if analysis.Dependencies[0].CallerService != "frontend" {
		t.Fatalf("unexpected dependency order: %#v", analysis.Dependencies)
	}
}

func TestBuildTraceAnalysisErrorAttributionAndRootCause(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			spanAt(10000, "02", "01", "order-service", "checkout", 80000),
			spanAt(20000, "03", "02", "payment-service", "charge", 50000),
		},
	}
	trace.Spans[0].Error = true
	trace.Spans[0].Status = "ERROR"
	trace.Spans[0].Attributes = map[string]string{"error.type": "_OTHER"}
	trace.Spans[1].Error = true
	trace.Spans[1].Status = "ERROR"
	trace.Spans[1].Attributes = map[string]string{"error.type": "timeout", "error.message": "payment gateway timeout"}
	trace.Spans[2].Error = true
	trace.Spans[2].Status = "ERROR"

	analysis := BuildTraceAnalysis(trace)
	if analysis.ErrorSpanCount != 3 {
		t.Fatalf("unexpected error count: %d", analysis.ErrorSpanCount)
	}
	if len(analysis.ErrorSpans) != 3 {
		t.Fatalf("unexpected error spans: %#v", analysis.ErrorSpans)
	}
	// The deepest failure is the root cause, the ancestors only propagate it.
	if !analysis.ErrorSpans[0].RootCause || analysis.ErrorSpans[0].ServiceName != "payment-service" {
		t.Fatalf("expected the deepest failure to be the root cause: %#v", analysis.ErrorSpans)
	}
	for _, span := range analysis.ErrorSpans[1:] {
		if span.RootCause {
			t.Fatalf("expected the propagated failures to not be root causes: %#v", analysis.ErrorSpans)
		}
	}
	byService := map[string]ErrorSpan{}
	for _, span := range analysis.ErrorSpans {
		byService[span.ServiceName] = span
	}
	order := byService["order-service"]
	if order.ErrorType != "timeout" {
		t.Fatalf("unexpected error type: %#v", order)
	}
	if order.ErrorMessage != "payment gateway timeout" {
		t.Fatalf("unexpected error message: %#v", order)
	}
	if byService["frontend"].ErrorType != "_OTHER" {
		t.Fatalf("unexpected error type: %#v", byService["frontend"])
	}
	if byService["payment-service"].ErrorType != "unknown" {
		t.Fatalf("expected a fallback error type: %#v", byService["payment-service"])
	}
}

func TestBuildTraceAnalysisReadsExceptionEvent(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "payment-service", "charge", 50000),
		},
	}
	trace.Spans[0].Error = true
	trace.Spans[0].Status = "ERROR"
	trace.Spans[0].Events = []Event{{
		Timestamp: analysisBase,
		Attributes: map[string]string{
			"event":             "exception",
			"exception.type":    "java.net.SocketTimeoutException",
			"exception.message": "Read timed out",
		},
	}}
	analysis := BuildTraceAnalysis(trace)
	if analysis.ErrorSpans[0].ErrorType != "java.net.SocketTimeoutException" {
		t.Fatalf("unexpected error type: %#v", analysis.ErrorSpans[0])
	}
	if analysis.ErrorSpans[0].ErrorMessage != "Read timed out" {
		t.Fatalf("unexpected error message: %#v", analysis.ErrorSpans[0])
	}
}

func TestBuildTraceAnalysisPrefersGenAIExceptionEventOverErrorMessage(t *testing.T) {
	// error.message is not an OpenTelemetry standard attribute, so the typed
	// exception of the generative AI event has to win over it.
	trace := &Trace{
		TraceID: testTraceID,
		Spans:   []Span{spanAt(0, "01", "", "dubbo-admin-sre-agent", "chat gpt-4o", 50000)},
	}
	trace.Spans[0].Error = true
	trace.Spans[0].Status = "ERROR"
	trace.Spans[0].Attributes = map[string]string{"error.message": "upstream timeout"}
	trace.Spans[0].Events = []Event{{
		Timestamp: analysisBase,
		Attributes: map[string]string{
			"event":             "gen_ai.client.operation.exception",
			"exception.type":    "openai.APIConnectionError",
			"exception.message": "connection reset by peer",
		},
	}}
	analysis := BuildTraceAnalysis(trace)
	if analysis.ErrorSpans[0].ErrorMessage != "connection reset by peer" {
		t.Fatalf("expected the generative AI exception message to win: %#v", analysis.ErrorSpans[0])
	}
	if analysis.ErrorSpans[0].ErrorType != "openai.APIConnectionError" {
		t.Fatalf("unexpected error type: %#v", analysis.ErrorSpans[0])
	}
}

func TestBuildTraceAnalysisPrefersGenAIExceptionEventOverStandardExceptionEvent(t *testing.T) {
	// Both events describe the same failure. The generative AI one wins because
	// this project serves the generative AI execution of the agent loop.
	trace := &Trace{
		TraceID: testTraceID,
		Spans:   []Span{spanAt(0, "01", "", "dubbo-admin-sre-agent", "chat gpt-4o", 50000)},
	}
	trace.Spans[0].Error = true
	trace.Spans[0].Status = "ERROR"
	trace.Spans[0].Events = []Event{
		{
			Timestamp: analysisBase,
			Attributes: map[string]string{
				"event":             "exception",
				"exception.type":    "java.lang.IllegalStateException",
				"exception.message": "state mismatch",
			},
		},
		{
			Timestamp: analysisBase,
			Attributes: map[string]string{
				"event":             "gen_ai.client.operation.exception",
				"exception.type":    "openai.RateLimitError",
				"exception.message": "rate limit exceeded",
			},
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.ErrorSpans[0].ErrorType != "openai.RateLimitError" {
		t.Fatalf("expected the generative AI exception type to win: %#v", analysis.ErrorSpans[0])
	}
	if analysis.ErrorSpans[0].ErrorMessage != "rate limit exceeded" {
		t.Fatalf("expected the generative AI exception message to win: %#v", analysis.ErrorSpans[0])
	}
}

func TestBuildTraceAnalysisFallsBackToErrorMessageAttribute(t *testing.T) {
	// error.message is the only signal left, so it provides the message and
	// degrades into the error type as well.
	trace := &Trace{
		TraceID: testTraceID,
		Spans:   []Span{spanAt(0, "01", "", "payment-service", "charge", 50000)},
	}
	trace.Spans[0].Error = true
	trace.Spans[0].Status = "ERROR"
	trace.Spans[0].Attributes = map[string]string{"error.message": "payment gateway timeout"}
	analysis := BuildTraceAnalysis(trace)
	if analysis.ErrorSpans[0].ErrorMessage != "payment gateway timeout" {
		t.Fatalf("unexpected error message: %#v", analysis.ErrorSpans[0])
	}
	if analysis.ErrorSpans[0].ErrorType != "payment gateway timeout" {
		t.Fatalf("expected the message to degrade into the error type: %#v", analysis.ErrorSpans[0])
	}
}

func TestBuildTraceAnalysisCountsErrorsBeyondListLimit(t *testing.T) {
	spans := make([]Span, 0, maxErrorSpans+10)
	for i := 0; i < maxErrorSpans+10; i++ {
		span := spanAt(int64(i)*1000, fmt.Sprintf("%02d", i), "", "payment-service", "charge", 1000)
		span.Error = true
		span.Status = "ERROR"
		spans = append(spans, span)
	}
	analysis := BuildTraceAnalysis(&Trace{TraceID: testTraceID, Spans: spans})
	if len(analysis.ErrorSpans) != maxErrorSpans {
		t.Fatalf("expected the list to be capped, got %d", len(analysis.ErrorSpans))
	}
	if analysis.ErrorSpanCount != maxErrorSpans+10 {
		t.Fatalf("expected the real error count, got %d", analysis.ErrorSpanCount)
	}
	if !analysis.Truncated {
		t.Fatal("expected the analysis to be marked as truncated")
	}
}

func TestBuildTraceAnalysisHandlesOrphanSpans(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			spanAt(10000, "02", "missing-parent", "order-service", "orphan", 40000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.SpanCount != 2 || analysis.ServiceCount != 2 {
		t.Fatalf("unexpected counters: %#v", analysis)
	}
	// The orphan still contributes to the service breakdown.
	found := false
	for _, entry := range analysis.ServiceBreakdown {
		if entry.ServiceName == "order-service" && entry.SpanCount == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the orphan in the breakdown: %#v", analysis.ServiceBreakdown)
	}
}

func TestBuildTraceAnalysisSurvivesCyclicReferences(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "02", "frontend", "a", 100000),
			spanAt(1000, "02", "01", "order-service", "b", 50000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if len(analysis.CriticalPath) > 2 {
		t.Fatalf("expected the cycle to be broken, got %#v", analysis.CriticalPath)
	}
	if analysis.SpanCount != 2 {
		t.Fatalf("unexpected span count: %d", analysis.SpanCount)
	}
}

func TestBuildTraceAnalysisSurvivesMalformedStartTime(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			{SpanID: "02", ParentSpanID: "01", ServiceName: "order-service", OperationName: "broken", StartTime: "not-a-timestamp", DurationMicros: 5000},
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.SpanCount != 2 {
		t.Fatalf("unexpected span count: %d", analysis.SpanCount)
	}
	if !analysis.Truncated {
		t.Fatal("expected the analysis to be marked as truncated")
	}
}

func TestBuildTraceAnalysisSurvivesSelfReference(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "01", "frontend", "self", 100000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.SpanCount != 1 {
		t.Fatalf("unexpected span count: %d", analysis.SpanCount)
	}
}

func TestBuildTraceAnalysisSummary(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "order-service", "checkout", 250000),
			spanAt(10000, "02", "01", "payment-service", "charge", 200000),
		},
	}
	trace.Spans[1].Error = true
	trace.Spans[1].Status = "ERROR"
	trace.Spans[1].Attributes = map[string]string{"error.type": "timeout"}
	analysis := BuildTraceAnalysis(trace)
	if !strings.HasPrefix(analysis.Summary, "trace 2 services, 2 spans, 250ms, 1 error") {
		t.Fatalf("unexpected summary: %q", analysis.Summary)
	}
	// The charge span consumes 200ms on its own, the checkout span only 50ms.
	if !strings.Contains(analysis.Summary, "slowest: payment-service/charge self 200ms of 200ms (80% of trace)") {
		t.Fatalf("unexpected summary: %q", analysis.Summary)
	}
	if !strings.Contains(analysis.Summary, "root cause: payment-service/charge ERROR timeout") {
		t.Fatalf("unexpected summary: %q", analysis.Summary)
	}
	if utf8.RuneCountInString(analysis.Summary) > maxAnalysisSummaryChars {
		t.Fatalf("expected a bounded summary, got %d characters", utf8.RuneCountInString(analysis.Summary))
	}
	if strings.ContainsAny(analysis.Summary, "\r\n") {
		t.Fatalf("expected a single line summary, got %q", analysis.Summary)
	}
}

func TestBuildTraceAnalysisSummaryStaysBounded(t *testing.T) {
	spans := []Span{
		spanAt(0, "01", "", strings.Repeat("very-long-service-name-", 8), strings.Repeat("very-long-operation-name-", 8), 1000),
	}
	spans[0].Error = true
	spans[0].Status = "ERROR"
	analysis := BuildTraceAnalysis(&Trace{TraceID: testTraceID, Spans: spans})
	if utf8.RuneCountInString(analysis.Summary) > maxAnalysisSummaryChars {
		t.Fatalf("expected a bounded summary, got %d characters", utf8.RuneCountInString(analysis.Summary))
	}
}

func TestBuildTraceAnalysisHandlesEmptyTrace(t *testing.T) {
	analysis := BuildTraceAnalysis(&Trace{TraceID: testTraceID})
	if analysis.SpanCount != 0 || analysis.ServiceCount != 0 {
		t.Fatalf("unexpected counters: %#v", analysis)
	}
	if analysis.Summary == "" {
		t.Fatal("expected a summary even for an empty trace")
	}
}

func TestAnalyzePropertiesExposeRequiredParameters(t *testing.T) {
	properties := AnalyzeProperties()
	for _, name := range []string{"traceId", "provider"} {
		if _, ok := properties[name]; !ok {
			t.Fatalf("analyze properties are missing %q", name)
		}
	}
}

// criticalPathChain builds a linear chain of spans for the critical path budget.
func criticalPathChain(hops int) []*analysisSpan {
	spans := make([]Span, 0, hops)
	for i := 0; i < hops; i++ {
		parentID := ""
		if i > 0 {
			parentID = fmt.Sprintf("%02d", i-1)
		}
		spans = append(spans, spanAt(int64(i)*1000, fmt.Sprintf("%02d", i), parentID, "order-service", "hop", 10000))
	}
	return buildAnalysisTree(spans)
}

func TestBuildCriticalPathStopsAtBudgetWithoutTruncation(t *testing.T) {
	// A chain that ends exactly at the budget does not lose any hop, so the
	// analysis must not be reported as truncated.
	analysis := &TraceAnalysis{}
	entries, _ := buildCriticalPath(criticalPathChain(maxCriticalPath), analysis)
	if len(entries) != maxCriticalPath {
		t.Fatalf("unexpected critical path length: %d", len(entries))
	}
	if analysis.Truncated {
		t.Fatal("did not expect the analysis to be truncated when the chain ends at the budget")
	}

	// One hop more than the budget does lose a hop.
	analysis = &TraceAnalysis{}
	entries, _ = buildCriticalPath(criticalPathChain(maxCriticalPath+1), analysis)
	if len(entries) != maxCriticalPath {
		t.Fatalf("unexpected critical path length: %d", len(entries))
	}
	if !analysis.Truncated {
		t.Fatal("expected the analysis to be truncated when the chain exceeds the budget")
	}
}

func TestBuildTraceAnalysisUsesProviderEnvelopeForClippedSpans(t *testing.T) {
	// The provider computes the duration over every raw span and only then clips
	// the span list. The analysis has to use the provider envelope as the
	// latency denominator, otherwise the reported duration is shorter than the
	// one the search reported and every latency percentage is inflated.
	trace := &Trace{
		TraceID:        testTraceID,
		DurationMicros: 500000,
		Truncated:      true,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "GET /checkout", 100000),
			spanAt(10000, "02", "01", "order-service", "checkout", 80000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.DurationMicros != 500000 {
		t.Fatalf("duration = %d, want %d", analysis.DurationMicros, 500000)
	}
	// The service breakdown divides by the envelope, so the frontend service
	// covers a fifth of the whole trace and not of the retained spans only.
	var frontend *ServiceLatency
	for _, entry := range analysis.ServiceBreakdown {
		if entry.ServiceName == "frontend" {
			frontend = &entry
			break
		}
	}
	if frontend == nil {
		t.Fatalf("expected a frontend entry in the breakdown: %#v", analysis.ServiceBreakdown)
	}
	if frontend.SharePercent != 20 {
		t.Fatalf("frontend share = %v, want 20", frontend.SharePercent)
	}
}

func TestBuildTraceAnalysisDoesNotReportTruncationForTheSlowSpanRanking(t *testing.T) {
	// The slow span list is a ranking: cutting it to the best entries does not
	// lose any analysis data. The truncated flag would tell the agent that the
	// diagnosis is based on incomplete data, so it must stay unset.
	spans := make([]Span, 0, maxSlowSpans+5)
	for i := 0; i < maxSlowSpans+5; i++ {
		spans = append(spans, spanAt(int64(i)*1000, fmt.Sprintf("%02d", i), "", "frontend", "GET /checkout", 10000))
	}
	trace := &Trace{TraceID: testTraceID, Spans: spans}
	analysis := BuildTraceAnalysis(trace)
	if len(analysis.SlowSpans) != maxSlowSpans {
		t.Fatalf("slow spans = %d, want %d", len(analysis.SlowSpans), maxSlowSpans)
	}
	if analysis.Truncated {
		t.Fatal("did not expect the analysis to be truncated by the slow span ranking")
	}
}

func TestBuildTraceAnalysisHandlesSpansWithoutSpanID(t *testing.T) {
	// Both spans share an empty span id. An index keyed by the business id would
	// let one span overwrite the other and mislabel the critical path.
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			{SpanID: "", ParentSpanID: "", ServiceName: "frontend", OperationName: "root", StartTime: analysisBase, DurationMicros: 100000},
			{SpanID: "", ParentSpanID: "", ServiceName: "order-service", OperationName: "orphan", StartTime: shiftTime(analysisBase, 10000), DurationMicros: 50000},
		},
	}
	analysis := BuildTraceAnalysis(trace)
	// The critical path starts at the root that finishes last.
	if len(analysis.CriticalPath) != 1 || analysis.CriticalPath[0].ServiceName != "frontend" {
		t.Fatalf("unexpected critical path: %#v", analysis.CriticalPath)
	}
	// Only the span on the critical path may be marked as such.
	onPath := 0
	for _, slow := range analysis.SlowSpans {
		if !slow.OnCriticalPath {
			continue
		}
		onPath++
		if slow.ServiceName != "frontend" {
			t.Fatalf("expected only the root to be on the critical path: %#v", analysis.SlowSpans)
		}
	}
	if onPath != 1 {
		t.Fatalf("expected exactly one span on the critical path: %#v", analysis.SlowSpans)
	}
	// The summary must quote the self time of the slowest span and not the one of
	// another span that happens to share the same identifier.
	if !strings.Contains(analysis.Summary, "slowest: frontend/root self 100ms of 100ms") {
		t.Fatalf("unexpected summary: %q", analysis.Summary)
	}
}

func TestBuildTraceAnalysisSkipsSelfTimeForUnparsableStartTime(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			{SpanID: "02", ParentSpanID: "01", ServiceName: "order-service", OperationName: "broken", StartTime: "not-a-timestamp", DurationMicros: 50000},
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if !analysis.Truncated {
		t.Fatal("expected the analysis to be marked as truncated")
	}
	// The span without a timestamp must not claim a time of its own, otherwise
	// the parent would account for the same 50ms a second time.
	for _, slow := range analysis.SlowSpans {
		if slow.SpanID == "02" && slow.SharePercent != 0 {
			t.Fatalf("expected no self time for the span without a timestamp: %#v", slow)
		}
	}
	if analysis.CriticalPath[0].SelfMicros != 100000 {
		t.Fatalf("unexpected self time of the root: %#v", analysis.CriticalPath[0])
	}
}

func TestBuildTraceAnalysisReadsExceptionSpanAttributes(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans:   []Span{spanAt(0, "01", "", "payment-service", "charge", 50000)},
	}
	trace.Spans[0].Error = true
	trace.Spans[0].Status = "ERROR"
	trace.Spans[0].Attributes = map[string]string{
		"exception.type":    "java.lang.IllegalStateException",
		"exception.message": "state mismatch",
	}
	analysis := BuildTraceAnalysis(trace)
	if analysis.ErrorSpans[0].ErrorType != "java.lang.IllegalStateException" {
		t.Fatalf("unexpected error type: %#v", analysis.ErrorSpans[0])
	}
	if analysis.ErrorSpans[0].ErrorMessage != "state mismatch" {
		t.Fatalf("unexpected error message: %#v", analysis.ErrorSpans[0])
	}
}

func TestBuildTraceAnalysisPrefersExceptionEventOverSpanAttributes(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans:   []Span{spanAt(0, "01", "", "payment-service", "charge", 50000)},
	}
	trace.Spans[0].Error = true
	trace.Spans[0].Status = "ERROR"
	trace.Spans[0].Attributes = map[string]string{
		"exception.type":    "java.lang.IllegalStateException",
		"exception.message": "state mismatch",
	}
	trace.Spans[0].Events = []Event{{
		Timestamp: analysisBase,
		Attributes: map[string]string{
			"event":             "exception",
			"exception.type":    "java.net.SocketTimeoutException",
			"exception.message": "Read timed out",
		},
	}}
	analysis := BuildTraceAnalysis(trace)
	if analysis.ErrorSpans[0].ErrorType != "java.net.SocketTimeoutException" {
		t.Fatalf("expected the event to win over the span attribute: %#v", analysis.ErrorSpans[0])
	}
	if analysis.ErrorSpans[0].ErrorMessage != "Read timed out" {
		t.Fatalf("expected the event to win over the span attribute: %#v", analysis.ErrorSpans[0])
	}
}

func TestBuildTraceAnalysisSummaryCountsRunes(t *testing.T) {
	// A Chinese name uses three bytes per character, so a byte budget would drop
	// the diagnosis even though it still fits into the character budget.
	longName := strings.Repeat("服", 60)
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", longName, longName, 100000),
			spanAt(10000, "02", "01", "order-service", "charge", 50000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	if !strings.Contains(analysis.Summary, "slowest:") {
		t.Fatalf("expected the slowest span in the summary: %q", analysis.Summary)
	}
	if utf8.RuneCountInString(analysis.Summary) > maxAnalysisSummaryChars {
		t.Fatalf("expected a bounded summary, got %d characters", utf8.RuneCountInString(analysis.Summary))
	}
}

func TestBuildTraceAnalysisSummaryUsesSingularForms(t *testing.T) {
	trace := &Trace{
		TraceID: testTraceID,
		Spans:   []Span{spanAt(0, "01", "", "frontend", "root", 100000)},
	}
	analysis := BuildTraceAnalysis(trace)
	if !strings.HasPrefix(analysis.Summary, "trace 1 service, 1 span, 100ms, 0 errors") {
		t.Fatalf("unexpected summary: %q", analysis.Summary)
	}
}

func TestBuildTraceAnalysisServiceShareCanExceedHundred(t *testing.T) {
	// Two concurrent children each cover the whole window of the parent, so the
	// service workload is twice the wall clock of the trace. The share is a
	// workload ratio and not a time slice, so it may exceed 100 percent.
	trace := &Trace{
		TraceID: testTraceID,
		Spans: []Span{
			spanAt(0, "01", "", "frontend", "root", 100000),
			spanAt(0, "02", "01", "order-service", "a", 100000),
			spanAt(0, "03", "01", "order-service", "b", 100000),
		},
	}
	analysis := BuildTraceAnalysis(trace)
	share := 0.0
	for _, entry := range analysis.ServiceBreakdown {
		if entry.ServiceName == "order-service" {
			share = entry.SharePercent
		}
	}
	if share <= 100 {
		t.Fatalf("expected the concurrent workload to exceed 100 percent, got %v", share)
	}
}
