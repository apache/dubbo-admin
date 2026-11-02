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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	observabilitycfg "github.com/apache/dubbo-admin/pkg/config/observability"
	"github.com/apache/dubbo-admin/pkg/core/logger"
)

const (
	traceQueryTimeout      = 15 * time.Second
	maxTraceResponseBytes  = int64(8 * 1024 * 1024)
	maxTraceErrorBodyBytes = int64(4096)
	maxTraceSpans          = 500
	maxTraceRoots          = 50
	maxTraceServices       = 100
	maxSpanAttributes      = 32
	maxSpanEvents          = 10
	maxEventAttributes     = 16
	maxAttributeValueBytes = 512
	tenantHeader           = "X-Scope-OrgID"
)

type jaegerProvider struct {
	config  observabilitycfg.TraceProviderConfig
	baseURL *url.URL
	client  *http.Client
}

var jaegerTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	MaxIdleConns:          20,
	MaxIdleConnsPerHost:   10,
	IdleConnTimeout:       90 * time.Second,
	ResponseHeaderTimeout: traceQueryTimeout,
}

type jaegerResponse struct {
	Data   []jaegerTrace `json:"data"`
	Errors []struct {
		Code    int    `json:"code"`
		Message string `json:"msg"`
	} `json:"errors,omitempty"`
}

type jaegerTrace struct {
	TraceID   string                   `json:"traceID"`
	Spans     []jaegerSpan             `json:"spans"`
	Processes map[string]jaegerProcess `json:"processes"`
}

type jaegerSpan struct {
	TraceID       string            `json:"traceID"`
	SpanID        string            `json:"spanID"`
	OperationName string            `json:"operationName"`
	References    []jaegerReference `json:"references"`
	StartTime     int64             `json:"startTime"`
	Duration      int64             `json:"duration"`
	Tags          []jaegerKeyValue  `json:"tags"`
	Logs          []jaegerLog       `json:"logs"`
	ProcessID     string            `json:"processID"`
	Process       *jaegerProcess    `json:"process,omitempty"`
}

type jaegerReference struct {
	RefType string `json:"refType"`
	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`
}

type jaegerProcess struct {
	ServiceName string           `json:"serviceName"`
	Tags        []jaegerKeyValue `json:"tags"`
}

type jaegerLog struct {
	Timestamp int64            `json:"timestamp"`
	Fields    []jaegerKeyValue `json:"fields"`
}

type jaegerKeyValue struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

func newJaegerProvider(cfg observabilitycfg.TraceProviderConfig) (*jaegerProvider, error) {
	baseURL, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid jaeger endpoint: %w", err)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid jaeger endpoint: scheme must be http or https")
	}
	if baseURL.Host == "" {
		return nil, fmt.Errorf("invalid jaeger endpoint: host is missing")
	}
	// A query string on the endpoint would be carried into every request URL and
	// into the transport error of a failed lookup, which reaches the tool result
	// and the model context. Credentials belong in bearerToken, so the query is
	// rejected instead of being forwarded. The error message does not repeat the
	// endpoint, because the endpoint is exactly what may hold the secret.
	if baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, fmt.Errorf("invalid jaeger endpoint: query parameters are not supported, use bearerToken for credentials")
	}
	return &jaegerProvider{
		config:  cfg,
		baseURL: baseURL,
		client: &http.Client{
			Timeout:   traceQueryTimeout,
			Transport: jaegerTransport,
		},
	}, nil
}

func (p *jaegerProvider) GetTraceByID(ctx context.Context, traceID string) (*Trace, error) {
	endpoint := *p.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/traces/" + traceID
	startedAt := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create jaeger request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if p.config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.BearerToken)
	}
	if p.config.Tenant != "" {
		req.Header.Set(tenantHeader, p.config.Tenant)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.logFailure(traceID, startedAt, 0, "transport", err)
		// The transport error embeds the request URL, which repeats the configured
		// endpoint and carries the retrieval conditions. It is sanitized before it
		// reaches the tool result and the model context. The trace identifier is
		// kept, because the model chose it and has to be able to report which
		// lookup failed.
		return nil, fmt.Errorf("jaeger trace query failed for trace %s: %w", traceID, sanitizedTransportError(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		p.logFailure(traceID, startedAt, resp.StatusCode, "not_found", nil)
		return nil, fmt.Errorf("trace %s was not found", traceID)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxTraceErrorBodyBytes))
		p.logFailure(traceID, startedAt, resp.StatusCode, "upstream_status", nil)
		return nil, fmt.Errorf("jaeger trace query failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := readTraceBody(resp.Body)
	if err != nil {
		p.logFailure(traceID, startedAt, resp.StatusCode, "response_limit", err)
		return nil, err
	}
	var upstream jaegerResponse
	if err := json.Unmarshal(body, &upstream); err != nil {
		p.logFailure(traceID, startedAt, resp.StatusCode, "decode", err)
		return nil, fmt.Errorf("failed to decode jaeger response: %w", err)
	}
	if len(upstream.Errors) > 0 {
		p.logFailure(traceID, startedAt, resp.StatusCode, "upstream_error", nil)
		return nil, fmt.Errorf("jaeger trace query failed: %s", upstream.Errors[0].Message)
	}
	if len(upstream.Data) == 0 {
		p.logFailure(traceID, startedAt, resp.StatusCode, "not_found", nil)
		return nil, fmt.Errorf("trace %s was not found", traceID)
	}
	if len(upstream.Data) != 1 || !strings.EqualFold(upstream.Data[0].TraceID, traceID) {
		p.logFailure(traceID, startedAt, resp.StatusCode, "inconsistent_response", nil)
		return nil, fmt.Errorf("jaeger returned an inconsistent response for trace %s", traceID)
	}
	result := normalizeJaegerTrace(upstream.Data[0])
	logger.Sugar().Infow("trace query completed", "provider", p.config.Name, "engine", p.config.Type, "trace_id", maskedTraceID(traceID), "elapsed_ms", time.Since(startedAt).Milliseconds(), "spans", result.SpanCount, "services", result.ServiceCount, "error_spans", result.ErrorSpanCount, "truncated", result.Truncated)
	return result, nil
}

// SearchTraces discovers candidate traces through the Jaeger v1 query API.
// The summaries are computed from the raw spans so that no information is lost to
// the normalization applied by the single trace lookup.
func (p *jaegerProvider) SearchTraces(ctx context.Context, query TraceSearchQuery) (*TraceSearchResult, error) {
	if strings.TrimSpace(query.Service) == "" {
		return nil, fmt.Errorf("service is required for trace search")
	}
	values := url.Values{}
	values.Set("service", query.Service)
	if query.Operation != "" {
		values.Set("operation", query.Operation)
	}
	if query.TraceID != "" {
		values.Set("traceID", query.TraceID)
	}
	if len(query.Tags) > 0 {
		encoded, err := json.Marshal(query.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to encode trace search tags: %w", err)
		}
		values.Set("tags", string(encoded))
	}
	if !query.StartTime.IsZero() {
		values.Set("start", strconv.FormatInt(query.StartTime.UnixMicro(), 10))
	}
	if !query.EndTime.IsZero() {
		values.Set("end", strconv.FormatInt(query.EndTime.UnixMicro(), 10))
	}
	if query.MinDuration > 0 {
		// Jaeger parses minDuration with time.ParseDuration, so the value has to
		// carry a unit. A bare microsecond count is rejected with a 400
		// ("missing unit in duration"), so the duration is sent in its string
		// form ("250ms") instead.
		values.Set("minDuration", query.MinDuration.String())
	}
	if query.Limit > 0 {
		// One trace more than the caller asked for is requested. A full result
		// then proves that more traces match than were returned, which is what
		// the truncated flag has to tell the agent.
		values.Set("limit", strconv.Itoa(query.Limit+1))
	}

	endpoint := *p.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/traces"
	endpoint.RawQuery = values.Encode()
	startedAt := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create jaeger search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if p.config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.BearerToken)
	}
	if p.config.Tenant != "" {
		req.Header.Set(tenantHeader, p.config.Tenant)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.logSearchFailure(query, startedAt, 0, "transport", err)
		// The transport error embeds the request URL, which repeats the configured
		// endpoint and carries the retrieval conditions. It is sanitized before it
		// reaches the tool result and the model context. The service anchor is
		// kept, because the model chose it and has to be able to report which
		// lookup failed.
		return nil, fmt.Errorf("jaeger trace search failed for service %s: %w", query.Service, sanitizedTransportError(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxTraceErrorBodyBytes))
		p.logSearchFailure(query, startedAt, resp.StatusCode, "upstream_status", nil)
		return nil, fmt.Errorf("jaeger trace search failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := readTraceBody(resp.Body)
	if err != nil {
		p.logSearchFailure(query, startedAt, resp.StatusCode, "response_limit", err)
		return nil, err
	}
	var upstream jaegerResponse
	if err := json.Unmarshal(body, &upstream); err != nil {
		p.logSearchFailure(query, startedAt, resp.StatusCode, "decode", err)
		return nil, fmt.Errorf("failed to decode jaeger search response: %w", err)
	}
	if len(upstream.Errors) > 0 {
		p.logSearchFailure(query, startedAt, resp.StatusCode, "upstream_error", nil)
		return nil, fmt.Errorf("jaeger trace search failed: %s", upstream.Errors[0].Message)
	}
	result := summarizeJaegerTraces(upstream.Data, query.Limit)
	logger.Sugar().Infow("trace search completed", "provider", p.config.Name, "engine", p.config.Type, "service", query.Service, "elapsed_ms", time.Since(startedAt).Milliseconds(), "traces", len(result.Traces), "total", result.Total, "truncated", result.Truncated)
	return result, nil
}

// jaegerTraceSummary keeps the numeric start time next to the summary so the result
// can be ordered on the absolute timeline instead of on the formatted timestamp.
type jaegerTraceSummary struct {
	summary     TraceSummary
	startMicros int64
}

// summarizeJaegerTraces converts the raw traces into bounded per-trace summaries.
func summarizeJaegerTraces(traces []jaegerTrace, limit int) *TraceSearchResult {
	result := &TraceSearchResult{Traces: []TraceSummary{}, SourceEngine: "jaeger"}
	summaries := make([]jaegerTraceSummary, 0, len(traces))
	for _, trace := range traces {
		summary, startMicros := summarizeJaegerTrace(trace)
		summaries = append(summaries, jaegerTraceSummary{summary: summary, startMicros: startMicros})
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		if summaries[i].startMicros != summaries[j].startMicros {
			return summaries[i].startMicros < summaries[j].startMicros
		}
		return summaries[i].summary.TraceID < summaries[j].summary.TraceID
	})
	// The provider was asked for one trace more than the caller wants, so a
	// result that reached that limit proves that more traces match than were
	// returned. Without the extra trace a full result would look complete.
	total := len(traces)
	if limit > 0 && len(summaries) > limit {
		result.Truncated = true
		summaries = summaries[:limit]
		total = limit
	}
	if len(summaries) > maxSearchTraceSummaries {
		summaries = summaries[:maxSearchTraceSummaries]
		result.Truncated = true
	}
	for _, item := range summaries {
		result.Traces = append(result.Traces, item.summary)
	}
	result.Total = total
	return result
}

// summarizeJaegerTrace derives the summary of one trace from all of its raw spans.
func summarizeJaegerTrace(input jaegerTrace) (TraceSummary, int64) {
	summary := TraceSummary{TraceID: input.TraceID}
	services := map[string]struct{}{}
	var minStart, maxEnd int64
	hasStart := false
	var rootSpan jaegerSpan
	hasRoot := false
	genai := false
	// The span ids of the trace decide which spans are roots: a span whose parent
	// is not part of the trace is an orphan and roots the trace on its own.
	spanIDs := make(map[string]struct{}, len(input.Spans))
	for _, rawSpan := range input.Spans {
		if rawSpan.SpanID != "" {
			spanIDs[rawSpan.SpanID] = struct{}{}
		}
	}
	for _, rawSpan := range input.Spans {
		process := input.Processes[rawSpan.ProcessID]
		if rawSpan.Process != nil {
			process = *rawSpan.Process
		}
		if process.ServiceName != "" {
			services[process.ServiceName] = struct{}{}
		}
		if errorSpan, _ := spanStatus(rawSpan.Tags); errorSpan {
			summary.ErrorSpanCount++
		}
		if hasGenAIJaegerTags(rawSpan.Tags) {
			genai = true
		}
		if !hasStart || rawSpan.StartTime < minStart {
			minStart = rawSpan.StartTime
			hasStart = true
		}
		if end := rawSpan.StartTime + rawSpan.Duration; end > maxEnd {
			maxEnd = end
		}
		if isJaegerRootSpan(rawSpan, spanIDs) && (!hasRoot || betterJaegerRoot(rawSpan, rootSpan)) {
			rootSpan = rawSpan
			hasRoot = true
			summary.RootService = process.ServiceName
			summary.RootOperation = rawSpan.OperationName
		}
	}
	summary.SpanCount = len(input.Spans)
	summary.ServiceCount = len(services)
	summary.HasError = summary.ErrorSpanCount > 0
	if hasStart {
		summary.StartTime = jaegerMicrosTime(minStart)
		summary.DurationMicros = maxEnd - minStart
	}
	if genai {
		summary.TraceKind = "genai"
	}
	return summary, minStart
}

// isJaegerRootSpan reports whether the span roots the trace. A span is a root when
// it has no parent reference at all, or when the referenced parent is not part of
// the trace. The latter happens for orphans whose parent was dropped by the
// sampling or by the span limit. The analyzer uses the same rule, so the search
// summary and the diagnosis agree on the root of a trace.
func isJaegerRootSpan(rawSpan jaegerSpan, spanIDs map[string]struct{}) bool {
	parentID := parentSpanID(rawSpan.References)
	if parentID == "" {
		return true
	}
	_, present := spanIDs[parentID]
	return !present
}

// betterJaegerRoot prefers the root span that finishes last, because that span
// determines the end of the request.
func betterJaegerRoot(candidate, current jaegerSpan) bool {
	candidateEnd := candidate.StartTime + candidate.Duration
	currentEnd := current.StartTime + current.Duration
	if candidateEnd != currentEnd {
		return candidateEnd > currentEnd
	}
	if candidate.Duration != current.Duration {
		return candidate.Duration > current.Duration
	}
	return candidate.SpanID < current.SpanID
}

// hasGenAIJaegerTags detects generative AI attributes on raw provider tags without
// allocating a normalized attribute map.
func hasGenAIJaegerTags(tags []jaegerKeyValue) bool {
	for _, tag := range tags {
		if isGenAIAttributeKey(tag.Key) {
			return true
		}
	}
	return false
}

func normalizeJaegerTrace(input jaegerTrace) *Trace {
	result := &Trace{TraceID: input.TraceID, SourceEngine: "jaeger", SpanCount: len(input.Spans), Spans: []Span{}}
	serviceSet := map[string]struct{}{}
	var minStart, maxEnd int64
	spans := input.Spans
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].StartTime < spans[j].StartTime })
	for _, rawSpan := range spans {
		process := input.Processes[rawSpan.ProcessID]
		if rawSpan.Process != nil {
			process = *rawSpan.Process
		}
		if process.ServiceName != "" {
			serviceSet[process.ServiceName] = struct{}{}
		}
		if errorSpan, _ := spanStatus(rawSpan.Tags); errorSpan {
			result.ErrorSpanCount++
		}
		if parentSpanID(rawSpan.References) == "" {
			if len(result.RootSpanIDs) < maxTraceRoots {
				result.RootSpanIDs = append(result.RootSpanIDs, rawSpan.SpanID)
			} else {
				result.Truncated = true
			}
		}
		end := rawSpan.StartTime + rawSpan.Duration
		if minStart == 0 || rawSpan.StartTime < minStart {
			minStart = rawSpan.StartTime
		}
		if end > maxEnd {
			maxEnd = end
		}
	}
	if len(spans) > maxTraceSpans {
		spans = spans[:maxTraceSpans]
		result.Truncated = true
	}
	for _, rawSpan := range spans {
		process := input.Processes[rawSpan.ProcessID]
		if rawSpan.Process != nil {
			process = *rawSpan.Process
		}
		span := normalizeJaegerSpan(rawSpan, process)
		result.Spans = append(result.Spans, span)
		if span.Truncated {
			result.Truncated = true
		}
	}
	result.ReturnedSpanCount = len(result.Spans)
	for service := range serviceSet {
		result.Services = append(result.Services, service)
	}
	sort.Strings(result.Services)
	result.ServiceCount = len(result.Services)
	if len(result.Services) > maxTraceServices {
		result.Services = result.Services[:maxTraceServices]
		result.Truncated = true
	}
	if minStart > 0 {
		result.StartTime = jaegerMicrosTime(minStart)
		result.DurationMicros = maxEnd - minStart
	}
	return result
}

func normalizeJaegerSpan(input jaegerSpan, process jaegerProcess) Span {
	attributes, attributesTruncated := normalizeKeyValues(input.Tags, maxSpanAttributes)
	for _, processTag := range process.Tags {
		key := "process." + processTag.Key
		if _, exists := attributes[key]; exists {
			continue
		}
		if len(attributes) >= maxSpanAttributes {
			attributesTruncated = true
			break
		}
		value := formatAttributeValue(processTag.Value)
		if truncatedValue, truncated := truncateUTF8(value, maxAttributeValueBytes); truncated {
			value = truncatedValue
			attributesTruncated = true
		}
		attributes[key] = value
	}
	span := Span{
		SpanID:         input.SpanID,
		ParentSpanID:   parentSpanID(input.References),
		ServiceName:    process.ServiceName,
		OperationName:  input.OperationName,
		StartTime:      jaegerMicrosTime(input.StartTime),
		DurationMicros: input.Duration,
		Status:         "UNSET",
		Attributes:     attributes,
		Truncated:      attributesTruncated,
	}
	span.Error, span.Status = spanStatus(input.Tags)
	for i, logEntry := range input.Logs {
		if i >= maxSpanEvents {
			span.Truncated = true
			break
		}
		eventAttributes, truncated := normalizeKeyValues(logEntry.Fields, maxEventAttributes)
		span.Events = append(span.Events, Event{Timestamp: jaegerMicrosTime(logEntry.Timestamp), Attributes: eventAttributes})
		span.Truncated = span.Truncated || truncated
	}
	return span
}

func normalizeKeyValues(values []jaegerKeyValue, limit int) (map[string]string, bool) {
	result := make(map[string]string, min(limit, len(values)))
	truncated := false
	for _, item := range values {
		if len(result) >= limit {
			truncated = true
			break
		}
		value := formatAttributeValue(item.Value)
		if truncatedValue, valueTruncated := truncateUTF8(value, maxAttributeValueBytes); valueTruncated {
			value = truncatedValue
			truncated = true
		}
		result[item.Key] = value
	}
	return result, truncated
}

func spanStatus(tags []jaegerKeyValue) (bool, string) {
	status := "UNSET"
	errorSpan := false
	for _, tag := range tags {
		key := strings.ToLower(tag.Key)
		value := strings.ToLower(formatAttributeValue(tag.Value))
		switch key {
		case "error":
			if value == "true" || value == "1" {
				errorSpan = true
				status = "ERROR"
			}
		case "otel.status_code", "status.code":
			if value == "error" {
				errorSpan = true
				status = "ERROR"
			} else if value == "ok" && !errorSpan {
				status = "OK"
			}
		}
	}
	return errorSpan, status
}

func parentSpanID(references []jaegerReference) string {
	for _, reference := range references {
		if strings.EqualFold(reference.RefType, "CHILD_OF") {
			return reference.SpanID
		}
	}
	if len(references) > 0 {
		return references[0].SpanID
	}
	return ""
}

func formatAttributeValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64)
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(encoded)
	}
}

func jaegerMicrosTime(value int64) string {
	return time.Unix(0, value*int64(time.Microsecond)).UTC().Format(time.RFC3339Nano)
}

func readTraceBody(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxTraceResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read jaeger response: %w", err)
	}
	if int64(len(body)) > maxTraceResponseBytes {
		return nil, fmt.Errorf("jaeger response exceeds %d bytes", maxTraceResponseBytes)
	}
	return body, nil
}

func maskedTraceID(traceID string) string {
	if len(traceID) <= 8 {
		return traceID
	}
	return "..." + traceID[len(traceID)-8:]
}

func truncateUTF8(value string, limit int) (string, bool) {
	if len(value) <= limit {
		return value, false
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value, true
}

// logFailure records a failed trace query. The error is sanitized, because the
// design document section 10 only allows the provider name, the service name, the
// elapsed time, the error type and the upstream status code in a failure log: the
// query payload must never be logged.
func (p *jaegerProvider) logFailure(traceID string, startedAt time.Time, status int, errorType string, err error) {
	fields := []any{"provider", p.config.Name, "engine", p.config.Type, "trace_id", maskedTraceID(traceID), "elapsed_ms", time.Since(startedAt).Milliseconds(), "error_type", errorType}
	if status != 0 {
		fields = append(fields, "upstream_status", status)
	}
	if sanitized := sanitizeTransportError(err); sanitized != nil {
		fields = append(fields, "error", sanitized)
	}
	logger.Sugar().Warnw("trace query failed", fields...)
}

// logSearchFailure records a failed search. The query payload and the credentials
// are deliberately left out so the log never leaks the lookup filters. The service
// name is kept because it is the only anchor of the failed lookup, but the error
// is sanitized because it would otherwise repeat the whole request URL.
func (p *jaegerProvider) logSearchFailure(query TraceSearchQuery, startedAt time.Time, status int, errorType string, err error) {
	fields := []any{"provider", p.config.Name, "engine", p.config.Type, "service", query.Service, "elapsed_ms", time.Since(startedAt).Milliseconds(), "error_type", errorType}
	if query.TraceID != "" {
		fields = append(fields, "trace_id", maskedTraceID(query.TraceID))
	}
	if status != 0 {
		fields = append(fields, "upstream_status", status)
	}
	if sanitized := sanitizeTransportError(err); sanitized != nil {
		fields = append(fields, "error", sanitized)
	}
	logger.Sugar().Warnw("trace search failed", fields...)
}

// TODO: the tool result still returns the upstream error text of a failed
// response unchanged, and that text can carry sensitive data of the returned
// spans, such as user identifiers or business data. The request URL is no
// longer part of it, because the transport error is sanitized. A production
// deployment still has to decide per deployment scenario whether the span
// content of the tool result is filtered or redacted. That mechanism is not
// implemented yet, this round only records the decision.

// sanitizeTransportError removes the request URL from a transport failure.
// http.Client.Do reports transport problems as *url.Error, whose message embeds
// the full request URL including the query string. The query carries the caller's
// filters and may contain business identifiers or secrets, so only the inner
// error is kept: it still explains the failure ("dial tcp ...: connection refused")
// without repeating the payload. The function guards the failure logs and the
// tool result.
func sanitizeTransportError(err error) error {
	if err == nil {
		return nil
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Err == nil {
			return nil
		}
		return urlErr.Err
	}
	return err
}

// sanitizedTransportError returns the sanitized form of a transport failure and
// never returns nil. The tool result needs a message in every case, while the
// failure logs only add the field when there is something to report.
func sanitizedTransportError(err error) error {
	if sanitized := sanitizeTransportError(err); sanitized != nil {
		return sanitized
	}
	return errors.New("unknown transport error")
}
