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
		return nil, fmt.Errorf("jaeger trace query failed: %w", err)
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

func (p *jaegerProvider) logFailure(traceID string, startedAt time.Time, status int, errorType string, err error) {
	fields := []any{"provider", p.config.Name, "engine", p.config.Type, "trace_id", maskedTraceID(traceID), "elapsed_ms", time.Since(startedAt).Milliseconds(), "error_type", errorType}
	if status != 0 {
		fields = append(fields, "upstream_status", status)
	}
	if err != nil {
		fields = append(fields, "error", err)
	}
	logger.Sugar().Warnw("trace query failed", fields...)
}
