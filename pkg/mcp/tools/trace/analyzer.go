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
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/common"
)

const (
	// Bounded output limits. They keep the tool payload small enough to stay inside
	// the model context window while still describing the most relevant spans.
	maxCriticalPath     = 32
	maxServiceBreakdown = 50
	maxSlowSpans        = 10
	maxErrorSpans       = 20
	maxDependencyEdges  = 50
	maxGenAICalls       = 20

	maxAnalysisSummaryChars = 300
	maxErrorTypeChars       = 96
	maxErrorMessageChars    = 256
)

const (
	errorTypeAttribute        = "error.type"
	errorMessageAttribute     = "error.message"
	exceptionTypeAttribute    = "exception.type"
	exceptionMessageAttribute = "exception.message"
	spanEventNameAttribute    = "event"
	standardExceptionEvent    = "exception"
	genAIExceptionEvent       = "gen_ai.client.operation.exception"
)

// analysisSpan is the analyzer's internal view of a span with a resolved timeline.
type analysisSpan struct {
	span        *Span
	position    int
	startMicros int64
	endMicros   int64
	timeValid   bool
	selfMicros  int64
	waitMicros  int64
	parent      *analysisSpan
	children    []*analysisSpan
}

// timelineInterval is a half-open interval on the absolute microsecond timeline.
type timelineInterval struct {
	start int64
	end   int64
}

// AnalyzeTrace diagnoses a single distributed trace and returns a structured report.
func AnalyzeTrace(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	traceID := helper.GetString("traceId", "")
	if !traceIDPattern.MatchString(traceID) {
		return common.ErrorResult(fmt.Errorf("traceId must be a 16 or 32 character hexadecimal value")), nil
	}
	providerConfig, err := providerConfigFromContext(ctx, helper.GetString("provider", ""))
	if err != nil {
		return common.ErrorResult(err), nil
	}
	provider, err := newProvider(providerConfig)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	trace, err := provider.GetTraceByID(traceRequestContext(ctx), traceID)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	return common.JsonResult(BuildTraceAnalysis(trace))
}

// AnalyzeProperties returns the MCP input schema properties for trace diagnosis.
func AnalyzeProperties() map[string]common.PropertyDef {
	return map[string]common.PropertyDef{
		"traceId":  {Type: "string", Description: "Jaeger 64-bit or 128-bit hexadecimal TraceID"},
		"provider": {Type: "string", Description: "Optional trace provider name; defaults to defaultProvider"},
	}
}

// BuildTraceAnalysis turns a normalized trace into a structured diagnosis.
// It is a pure function so the diagnosis rules can be verified without a backend.
func BuildTraceAnalysis(trace *Trace) *TraceAnalysis {
	analysis := &TraceAnalysis{}
	if trace == nil {
		analysis.Summary = "no trace data available"
		return analysis
	}
	analysis.TraceID = trace.TraceID
	analysis.Truncated = trace.Truncated

	nodes := buildAnalysisTree(trace.Spans)
	computeSelfTimes(nodes, analysis)
	criticalPath, criticalPathSet := buildCriticalPath(nodes, analysis)
	analysis.CriticalPath = criticalPath

	wallClock, traceStart := analysisEnvelope(nodes, trace)
	analysis.DurationMicros = wallClock
	analysis.StartTime = traceStart
	analysis.SpanCount = maxInt(trace.SpanCount, len(trace.Spans))

	breakdown := buildServiceBreakdown(nodes, wallClock, analysis)
	analysis.ServiceBreakdown = breakdown
	analysis.ServiceCount = maxInt(trace.ServiceCount, len(breakdown))

	slowSpans := buildSlowSpans(nodes, wallClock, criticalPathSet)
	analysis.SlowSpans = make([]SlowSpan, 0, len(slowSpans))
	for _, entry := range slowSpans {
		analysis.SlowSpans = append(analysis.SlowSpans, entry.slow)
	}

	errorSpans, errorCount := buildErrorSpans(nodes, analysis)
	analysis.ErrorSpans = errorSpans
	// The provider already counted errors over the complete trace; that count must not
	// be hidden by the bounded error span list.
	analysis.ErrorSpanCount = maxInt(trace.ErrorSpanCount, errorCount)

	analysis.Dependencies = buildDependencyEdges(nodes, analysis)

	if genai := BuildGenAISummary(trace.Spans); genai.Present {
		analysis.GenAI = genai
	}

	analysis.Summary = buildAnalysisSummary(analysis, slowSpans)
	return analysis
}

// buildAnalysisTree links the spans into a parent/child forest. A span whose parent
// is missing from the trace degrades into an orphan root: it still contributes to the
// aggregate statistics but it cannot extend a parent chain.
func buildAnalysisTree(spans []Span) []*analysisSpan {
	nodes := make([]*analysisSpan, 0, len(spans))
	index := make(map[string]*analysisSpan, len(spans))
	for i := range spans {
		span := &spans[i]
		node := &analysisSpan{span: span, position: i}
		if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(span.StartTime)); err == nil {
			node.timeValid = true
			node.startMicros = parsed.UnixMicro()
			node.endMicros = node.startMicros + span.DurationMicros
		} else {
			node.endMicros = span.DurationMicros
		}
		nodes = append(nodes, node)
		if span.SpanID != "" {
			index[span.SpanID] = node
		}
	}
	for _, node := range nodes {
		parentID := node.span.ParentSpanID
		if parentID == "" {
			continue
		}
		parent, ok := index[parentID]
		if !ok || parent == node {
			continue
		}
		node.parent = parent
		parent.children = append(parent.children, node)
	}
	return nodes
}

// computeSelfTimes derives the exclusive time of every span. The covered time is the
// union of the clipped child intervals, so concurrent children are not counted twice.
func computeSelfTimes(nodes []*analysisSpan, analysis *TraceAnalysis) {
	for _, node := range nodes {
		if !node.timeValid {
			// Without a trustworthy start time the span degrades to a leaf. It
			// cannot be placed on the timeline, so it must not claim any time
			// of its own: counting its full duration here would let the parent
			// account for the same time a second time.
			node.selfMicros = 0
			node.waitMicros = 0
			analysis.Truncated = true
			continue
		}
		intervals := make([]timelineInterval, 0, len(node.children))
		for _, child := range node.children {
			if !child.timeValid {
				analysis.Truncated = true
				continue
			}
			start := maxInt64(child.startMicros, node.startMicros)
			end := minInt64(child.endMicros, node.endMicros)
			if end <= start {
				continue
			}
			intervals = append(intervals, timelineInterval{start: start, end: end})
		}
		covered := unionLength(intervals)
		node.waitMicros = covered
		node.selfMicros = node.span.DurationMicros - covered
		if node.selfMicros < 0 {
			node.selfMicros = 0
		}
	}
}

// unionLength returns the length of the union of the given intervals.
func unionLength(intervals []timelineInterval) int64 {
	if len(intervals) == 0 {
		return 0
	}
	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i].start != intervals[j].start {
			return intervals[i].start < intervals[j].start
		}
		return intervals[i].end < intervals[j].end
	})
	total := int64(0)
	current := intervals[0]
	for _, next := range intervals[1:] {
		if next.start > current.end {
			total += current.end - current.start
			current = next
			continue
		}
		if next.end > current.end {
			current.end = next.end
		}
	}
	return total + current.end - current.start
}

// buildCriticalPath walks the latency determining root-to-leaf chain.
// The second return value holds the analyzer internal positions of the spans on the
// path. The position is used instead of the business span id because a span id can
// be empty or duplicated, which would make two different spans overwrite each other.
func buildCriticalPath(nodes []*analysisSpan, analysis *TraceAnalysis) ([]CriticalPathEntry, map[int]struct{}) {
	roots := make([]*analysisSpan, 0, len(nodes))
	for _, node := range nodes {
		if node.parent == nil {
			roots = append(roots, node)
		}
	}
	entries := make([]CriticalPathEntry, 0, maxCriticalPath)
	onPath := map[int]struct{}{}
	visited := map[int]struct{}{}
	current := selectCriticalRoot(roots)
	for current != nil {
		if len(entries) >= maxCriticalPath {
			// The budget is exhausted while the chain still continues, so the
			// remaining hops are dropped and the analysis is incomplete.
			analysis.Truncated = true
			break
		}
		if _, seen := visited[current.position]; seen {
			break
		}
		visited[current.position] = struct{}{}
		onPath[current.position] = struct{}{}
		entries = append(entries, CriticalPathEntry{
			SpanID:         current.span.SpanID,
			ServiceName:    current.span.ServiceName,
			OperationName:  current.span.OperationName,
			StartTime:      current.span.StartTime,
			DurationMicros: current.span.DurationMicros,
			SelfMicros:     current.selfMicros,
			WaitMicros:     current.waitMicros,
		})
		current = selectCriticalChild(current)
	}
	return entries, onPath
}

// selectCriticalRoot picks the root that finishes last, because that root determines
// the end of the request.
func selectCriticalRoot(roots []*analysisSpan) *analysisSpan {
	var best *analysisSpan
	for _, root := range roots {
		if !root.timeValid {
			continue
		}
		if best == nil || betterCriticalRoot(root, best) {
			best = root
		}
	}
	if best != nil {
		return best
	}
	for _, root := range roots {
		if best == nil || betterCriticalRoot(root, best) {
			best = root
		}
	}
	return best
}

func betterCriticalRoot(candidate, current *analysisSpan) bool {
	if candidate.endMicros != current.endMicros {
		return candidate.endMicros > current.endMicros
	}
	if candidate.span.DurationMicros != current.span.DurationMicros {
		return candidate.span.DurationMicros > current.span.DurationMicros
	}
	// The start time is derived from the end time and the duration, so it is
	// already equal here and cannot be used as a tie breaker.
	return candidate.span.SpanID < current.span.SpanID
}

// selectCriticalChild picks the child that finishes last, because it is the activity
// that determined when the parent completed.
func selectCriticalChild(node *analysisSpan) *analysisSpan {
	var best *analysisSpan
	for _, child := range node.children {
		if !child.timeValid {
			// A span without a trustworthy timestamp degrades to a leaf and is not
			// allowed to extend the chain.
			continue
		}
		if best == nil || betterCriticalChild(child, best) {
			best = child
		}
	}
	return best
}

func betterCriticalChild(candidate, current *analysisSpan) bool {
	if candidate.endMicros != current.endMicros {
		return candidate.endMicros > current.endMicros
	}
	if candidate.span.DurationMicros != current.span.DurationMicros {
		return candidate.span.DurationMicros > current.span.DurationMicros
	}
	return candidate.span.SpanID < current.span.SpanID
}

// analysisEnvelope returns the wall clock duration and the start timestamp of the trace.
func analysisEnvelope(nodes []*analysisSpan, trace *Trace) (int64, string) {
	var minStart, maxEnd int64
	hasStart := false
	startText := trace.StartTime
	for _, node := range nodes {
		if !node.timeValid {
			continue
		}
		if !hasStart || node.startMicros < minStart {
			minStart = node.startMicros
			startText = node.span.StartTime
			hasStart = true
		}
		if node.endMicros > maxEnd {
			maxEnd = node.endMicros
		}
	}
	if !hasStart {
		return trace.DurationMicros, trace.StartTime
	}
	// The provider envelope covers every raw span, the node envelope only the
	// retained ones. A clipped span list would make the node envelope shorter
	// than the trace and inflate every latency percentage that uses it as the
	// denominator, so the provider value is preferred whenever it is available.
	if trace.DurationMicros > 0 {
		return trace.DurationMicros, startText
	}
	return maxEnd - minStart, startText
}

// buildServiceBreakdown aggregates the spans per service.
func buildServiceBreakdown(nodes []*analysisSpan, wallClock int64, analysis *TraceAnalysis) []ServiceLatency {
	aggregates := map[string]*ServiceLatency{}
	for _, node := range nodes {
		span := node.span
		entry, ok := aggregates[span.ServiceName]
		if !ok {
			entry = &ServiceLatency{ServiceName: span.ServiceName}
			aggregates[span.ServiceName] = entry
		}
		entry.SpanCount++
		if spanFailed(span) {
			entry.ErrorSpanCount++
		}
		entry.TotalMicros += span.DurationMicros
		if span.DurationMicros > entry.MaxMicros {
			entry.MaxMicros = span.DurationMicros
		}
	}
	result := make([]ServiceLatency, 0, len(aggregates))
	for _, entry := range aggregates {
		if entry.SpanCount > 0 {
			entry.AvgMicros = entry.TotalMicros / int64(entry.SpanCount)
		}
		// The share is the service workload relative to the wall clock of the trace.
		// Concurrent spans can push it above 100 percent on purpose.
		entry.SharePercent = percentOf(entry.TotalMicros, wallClock)
		result = append(result, *entry)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].TotalMicros != result[j].TotalMicros {
			return result[i].TotalMicros > result[j].TotalMicros
		}
		return result[i].ServiceName < result[j].ServiceName
	})
	if len(result) > maxServiceBreakdown {
		result = result[:maxServiceBreakdown]
		analysis.Truncated = true
	}
	return result
}

// slowSpanEntry pairs a slow span with the analyzer internal identity of the span
// it was derived from. The position is used instead of the business span id so
// that spans without an id, or with duplicated ids, cannot overwrite each other.
type slowSpanEntry struct {
	slow       SlowSpan
	position   int
	selfMicros int64
}

// buildSlowSpans ranks the spans by the time they consumed on their own.
// The exclusive time is returned alongside the report so the summary can quote it
// even for spans that are not part of the critical path.
func buildSlowSpans(nodes []*analysisSpan, wallClock int64, criticalPath map[int]struct{}) []slowSpanEntry {
	candidates := make([]slowSpanEntry, 0, len(nodes))
	for _, node := range nodes {
		span := node.span
		_, onPath := criticalPath[node.position]
		candidates = append(candidates, slowSpanEntry{
			slow: SlowSpan{
				SpanID:         span.SpanID,
				ServiceName:    span.ServiceName,
				OperationName:  span.OperationName,
				DurationMicros: span.DurationMicros,
				SharePercent:   percentOf(node.selfMicros, wallClock),
				OnCriticalPath: onPath,
			},
			position:   node.position,
			selfMicros: node.selfMicros,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].selfMicros != candidates[j].selfMicros {
			return candidates[i].selfMicros > candidates[j].selfMicros
		}
		return candidates[i].position < candidates[j].position
	})
	// The list is a ranking, so cutting it to the top entries does not lose any
	// analysis data. The truncated flag stays untouched: it would tell the agent
	// that the diagnosis is based on incomplete data, and a ranking is complete
	// even when only its best entries are reported.
	if len(candidates) > maxSlowSpans {
		candidates = candidates[:maxSlowSpans]
	}
	return candidates
}

// errorSpanEntry pairs an error span with the analyzer node it was derived from.
// The node is kept so the ordering can use the start time of the span without
// indexing the table by the business span id.
type errorSpanEntry struct {
	span ErrorSpan
	node *analysisSpan
}

// buildErrorSpans collects the failed spans and marks the root causes.
func buildErrorSpans(nodes []*analysisSpan, analysis *TraceAnalysis) ([]ErrorSpan, int) {
	failed := make([]*analysisSpan, 0, len(nodes))
	for _, node := range nodes {
		if spanFailed(node.span) {
			failed = append(failed, node)
		}
	}
	entries := make([]errorSpanEntry, 0, len(failed))
	for _, node := range failed {
		span := node.span
		errorType, errorMessage := errorDetails(span)
		entries = append(entries, errorSpanEntry{
			span: ErrorSpan{
				SpanID:        span.SpanID,
				ServiceName:   span.ServiceName,
				OperationName: span.OperationName,
				ErrorType:     errorType,
				ErrorMessage:  errorMessage,
				Status:        errorStatus(span),
				RootCause:     !hasFailedDescendant(node),
			},
			node: node,
		})
	}
	// Root causes come first so the agent sees the deepest failure before the
	// ancestors that only propagate it.
	sort.SliceStable(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]
		if left.span.RootCause != right.span.RootCause {
			return left.span.RootCause
		}
		if left.node.startMicros != right.node.startMicros {
			return left.node.startMicros < right.node.startMicros
		}
		return left.node.position < right.node.position
	})
	count := len(entries)
	if len(entries) > maxErrorSpans {
		entries = entries[:maxErrorSpans]
		analysis.Truncated = true
	}
	result := make([]ErrorSpan, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.span)
	}
	return result, count
}

// errorDetails resolves the error classification and the human readable description.
//
// The sources are ordered "generative AI first, standard second, non standard
// last":
//   - The span attribute error.type stays the leading source because the OpenTelemetry
//     error registry defines it as the stable error type of a span.
//   - The generative AI exception event comes before the standard exception event
//     because this project serves the generative AI execution of the agent loop, so
//     the typed error of the model call is the more relevant classification.
//   - error.message is only the last resort because it is not an OpenTelemetry
//     standard attribute: the registry only defines error.type, and the stable
//     attribute for the exception text is exception.message. error.message is a
//     habit of the Java ecosystem and of some collectors, so it is used only when
//     no typed error is available, and it then degrades into the error type.
func errorDetails(span *Span) (string, string) {
	errorType := ""
	if value, ok := lookupAttributeOK(span.Attributes, errorTypeAttribute); ok {
		errorType = strings.TrimSpace(value)
	}
	exceptionType, exceptionMessage := exceptionDetails(span)
	if errorType == "" {
		errorType = exceptionType
	}
	errorMessage := exceptionMessage
	if errorMessage == "" {
		if value, ok := lookupAttributeOK(span.Attributes, errorMessageAttribute); ok {
			errorMessage = strings.TrimSpace(value)
		}
	}
	if errorType == "" && errorMessage != "" {
		// Degraded classification: without a typed error the message is the only
		// signal left, so it degrades into the error type.
		errorType = errorMessage
	}
	if errorType == "" {
		errorType = "unknown"
	}
	return truncateRunes(errorType, maxErrorTypeChars), truncateRunes(errorMessage, maxErrorMessageChars)
}

// exceptionDetails reads the exception type and message from the span events.
// The generative AI exception event wins over the standard exception event,
// because this project serves the generative AI execution of the agent loop and
// the generative AI event carries the typed error of the model call. Collectors
// also report the exception as a plain span attribute, so the attributes are read
// as a fallback once the events have been exhausted.
func exceptionDetails(span *Span) (string, string) {
	exceptionType := ""
	exceptionMessage := ""
	collect := func(attributes map[string]string) {
		if exceptionType == "" {
			if value, ok := lookupAttributeOK(attributes, exceptionTypeAttribute); ok {
				exceptionType = strings.TrimSpace(value)
			}
		}
		if exceptionMessage == "" {
			if value, ok := lookupAttributeOK(attributes, exceptionMessageAttribute); ok {
				exceptionMessage = strings.TrimSpace(value)
			}
		}
	}
	for _, event := range span.Events {
		if strings.EqualFold(eventName(event), genAIExceptionEvent) {
			collect(event.Attributes)
		}
	}
	for _, event := range span.Events {
		if strings.EqualFold(eventName(event), standardExceptionEvent) {
			collect(event.Attributes)
		}
	}
	// The span attributes are the last resort: the two documented exception
	// events stay the primary source, and no other event is consulted.
	collect(span.Attributes)
	return exceptionType, exceptionMessage
}

func eventName(event Event) string {
	if value, ok := lookupAttributeOK(event.Attributes, spanEventNameAttribute); ok {
		return value
	}
	return ""
}

// errorStatus returns the normalized status of a failed span.
func errorStatus(span *Span) string {
	if spanFailed(span) {
		return "ERROR"
	}
	if span.Status == "" {
		return "UNSET"
	}
	return span.Status
}

// hasFailedDescendant reports whether a failed span only propagates an error that
// happened deeper in the trace.
func hasFailedDescendant(node *analysisSpan) bool {
	visited := map[int]struct{}{}
	stack := make([]*analysisSpan, 0, len(node.children))
	stack = append(stack, node.children...)
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, seen := visited[current.position]; seen {
			continue
		}
		visited[current.position] = struct{}{}
		if spanFailed(current.span) {
			return true
		}
		stack = append(stack, current.children...)
	}
	return false
}

// buildDependencyEdges derives the cross service call graph of the trace.
func buildDependencyEdges(nodes []*analysisSpan, analysis *TraceAnalysis) []DependencyEdge {
	aggregates := map[string]*DependencyEdge{}
	for _, node := range nodes {
		if node.parent == nil {
			continue
		}
		caller := node.parent.span.ServiceName
		callee := node.span.ServiceName
		if caller == callee {
			continue
		}
		key := caller + "\x00" + callee
		edge, ok := aggregates[key]
		if !ok {
			edge = &DependencyEdge{CallerService: caller, CalleeService: callee}
			aggregates[key] = edge
		}
		edge.SpanCount++
		edge.TotalMicros += node.span.DurationMicros
		if spanFailed(node.span) {
			edge.ErrorCount++
		}
	}
	result := make([]DependencyEdge, 0, len(aggregates))
	for _, edge := range aggregates {
		result = append(result, *edge)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].TotalMicros != result[j].TotalMicros {
			return result[i].TotalMicros > result[j].TotalMicros
		}
		if result[i].CallerService != result[j].CallerService {
			return result[i].CallerService < result[j].CallerService
		}
		return result[i].CalleeService < result[j].CalleeService
	})
	if len(result) > maxDependencyEdges {
		result = result[:maxDependencyEdges]
		analysis.Truncated = true
	}
	return result
}

// buildAnalysisSummary renders the one line conclusion of the diagnosis.
func buildAnalysisSummary(analysis *TraceAnalysis, slowSpans []slowSpanEntry) string {
	base := fmt.Sprintf("trace %d %s, %d %s, %s, %d %s",
		analysis.ServiceCount, pluralize(analysis.ServiceCount, "service", "services"),
		analysis.SpanCount, pluralize(analysis.SpanCount, "span", "spans"),
		formatMicros(analysis.DurationMicros),
		analysis.ErrorSpanCount, pluralize(analysis.ErrorSpanCount, "error", "errors"))

	var slowPart string
	if len(slowSpans) > 0 {
		slowest := slowSpans[0]
		slowPart = fmt.Sprintf("; slowest: %s/%s self %s of %s (%.0f%% of trace)",
			sanitizeSummaryText(slowest.slow.ServiceName), sanitizeSummaryText(slowest.slow.OperationName),
			formatMicros(slowest.selfMicros), formatMicros(slowest.slow.DurationMicros), slowest.slow.SharePercent)
	}
	var rootPart string
	for _, span := range analysis.ErrorSpans {
		if !span.RootCause {
			continue
		}
		rootPart = fmt.Sprintf("; root cause: %s/%s %s %s",
			sanitizeSummaryText(span.ServiceName), sanitizeSummaryText(span.OperationName),
			span.Status, sanitizeSummaryText(span.ErrorType))
		break
	}
	for _, candidate := range []string{base + slowPart + rootPart, base + rootPart, base} {
		// The budget counts runes, not bytes: a Chinese service name uses three
		// bytes per character and would otherwise push the diagnosis out of the
		// summary even though it still fits.
		if utf8.RuneCountInString(candidate) <= maxAnalysisSummaryChars {
			return candidate
		}
	}
	return truncateRunes(base+slowPart+rootPart, maxAnalysisSummaryChars)
}

// pluralize returns the singular or the plural form depending on the count.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// lookupAttributeOK resolves an attribute value, tolerating casing and padding
// differences introduced by different collectors.
func lookupAttributeOK(attributes map[string]string, name string) (string, bool) {
	if value, ok := attributes[name]; ok {
		return value, true
	}
	bestKey := ""
	bestValue := ""
	found := false
	for key, value := range attributes {
		if !strings.EqualFold(strings.TrimSpace(key), name) {
			continue
		}
		if !found || key < bestKey {
			bestKey, bestValue, found = key, value, true
		}
	}
	return bestValue, found
}

// percentOf computes a share with two decimals. A zero denominator yields zero
// instead of a division by zero.
func percentOf(value, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(value)/float64(total)*100*100) / 100
}

// formatMicros renders a microsecond duration in a compact human readable form.
func formatMicros(micros int64) string {
	switch {
	case micros < 1000:
		return strconv.FormatInt(micros, 10) + "us"
	case micros < 1000*1000:
		return trimDecimal(float64(micros)/1000, 1) + "ms"
	default:
		return trimDecimal(float64(micros)/(1000*1000), 2) + "s"
	}
}

func trimDecimal(value float64, decimals int) string {
	formatted := strconv.FormatFloat(value, 'f', decimals, 64)
	if strings.Contains(formatted, ".") {
		formatted = strings.TrimRight(formatted, "0")
		formatted = strings.TrimSuffix(formatted, ".")
	}
	return formatted
}

// sanitizeSummaryText keeps the one line summary free of line breaks.
func sanitizeSummaryText(value string) string {
	replaced := strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	return strings.TrimSpace(replaced)
}

// truncateRunes limits a string by rune count so multi byte characters stay intact.
func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// traceRequestContext derives the outbound request context from the console context.
func traceRequestContext(ctx consolectx.Context) context.Context {
	if ctx == nil || ctx.AppContext() == nil {
		return context.Background()
	}
	return ctx.AppContext()
}
