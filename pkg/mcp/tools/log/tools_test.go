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

package log

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/config/observability"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
)

func TestLogToolPropertiesAreAvailable(t *testing.T) {
	searchProperties := LogSearchProperties()
	for _, name := range []string{"mesh", "appName", "serviceName", "instanceName", "traceId", "keywords", "startTime", "endTime", "limit"} {
		if _, ok := searchProperties[name]; !ok {
			t.Fatalf("search property %s was not configured", name)
		}
	}

	capabilityProperties := LogCapabilitiesProperties()
	for _, name := range []string{"startTime", "endTime"} {
		if _, ok := capabilityProperties[name]; !ok {
			t.Fatalf("capability property %s was not configured", name)
		}
	}
}

func TestGetLogCapabilitiesReturnsAvailableLabelsAndResolvedFilters(t *testing.T) {
	loki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/labels" {
			t.Fatalf("unexpected loki path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":["pod","namespace","service_name","mesh"]}`))
	}))
	defer loki.Close()

	result, err := GetLogCapabilities(newLogToolTestContext(loki.URL), map[string]any{
		"startTime": "2026-04-01T00:00:00Z",
		"endTime":   "2026-04-01T01:00:00Z",
	})
	if err != nil {
		t.Fatalf("GetLogCapabilities returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("GetLogCapabilities returned error result: %s", result.Content[0].Text)
	}

	var payload LogCapabilitiesResp
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("failed to decode tool result: %v", err)
	}
	expectedLabels := []string{"mesh", "namespace", "pod", "service_name"}
	if len(payload.AvailableLabels) != len(expectedLabels) {
		t.Fatalf("expected labels %v, got %v", expectedLabels, payload.AvailableLabels)
	}
	for i := range expectedLabels {
		if payload.AvailableLabels[i] != expectedLabels[i] {
			t.Fatalf("label[%d] expected %q, got %q", i, expectedLabels[i], payload.AvailableLabels[i])
		}
	}
	if payload.FallbackLabel != "namespace" {
		t.Fatalf("expected fallbackLabel namespace, got %q", payload.FallbackLabel)
	}
	if got := payload.LabelFilters["serviceName"]; len(got) != 1 || got[0] != "service_name" {
		t.Fatalf("expected serviceName to resolve to service_name, got %v", got)
	}
	if got := payload.LabelFilters["instanceName"]; len(got) != 1 || got[0] != "pod" {
		t.Fatalf("expected instanceName to resolve to pod, got %v", got)
	}
	if got := payload.LabelFilters["appName"]; len(got) != 0 {
		t.Fatalf("expected appName to have no available labels, got %v", got)
	}
	if len(payload.ContentFilters) != 2 || payload.ContentFilters[0] != "traceId" || payload.ContentFilters[1] != "keywords" {
		t.Fatalf("unexpected content filters: %v", payload.ContentFilters)
	}
	if payload.SourceEngine != "loki" {
		t.Fatalf("expected sourceEngine loki, got %q", payload.SourceEngine)
	}
}

func TestBuildLogQLQueriesUsesCommonLabelAliases(t *testing.T) {
	queries := buildLogQLQueries(&SearchLogsReq{
		ServiceName: "org.apache.DemoService",
		Keywords:    "Error",
	})

	expected := []string{
		`{service="org.apache.DemoService"} |= "Error"`,
		`{serviceName="org.apache.DemoService"} |= "Error"`,
		`{service_name="org.apache.DemoService"} |= "Error"`,
	}
	if len(queries) != len(expected) {
		t.Fatalf("expected %d queries, got %d: %v", len(expected), len(queries), queries)
	}
	for i := range expected {
		if queries[i] != expected[i] {
			t.Fatalf("query[%d] expected %q, got %q", i, expected[i], queries[i])
		}
	}
}

func TestBuildLogQLQueriesFiltersTraceIDFromLogContent(t *testing.T) {
	queries := buildLogQLQueries(&SearchLogsReq{
		ServiceName: "org.apache.DemoService",
		TraceID:     "trace-1",
		Keywords:    "ERROR",
	})

	expected := []string{
		`{service="org.apache.DemoService"} |= "ERROR" |= "trace-1"`,
		`{serviceName="org.apache.DemoService"} |= "ERROR" |= "trace-1"`,
		`{service_name="org.apache.DemoService"} |= "ERROR" |= "trace-1"`,
	}
	if len(queries) != len(expected) {
		t.Fatalf("expected %d queries, got %d: %v", len(expected), len(queries), queries)
	}
	for i := range expected {
		if queries[i] != expected[i] {
			t.Fatalf("query[%d] expected %q, got %q", i, expected[i], queries[i])
		}
	}
}

func TestBuildLogQLQueriesUsesNamespaceSelectorWhenOnlyTraceIDIsProvided(t *testing.T) {
	queries := buildLogQLQueries(&SearchLogsReq{
		TraceID: "trace-1",
	})

	expected := []string{
		`{namespace=~".+"} |= "trace-1"`,
	}
	if len(queries) != len(expected) {
		t.Fatalf("expected %d queries, got %d: %v", len(expected), len(queries), queries)
	}
	for i := range expected {
		if queries[i] != expected[i] {
			t.Fatalf("query[%d] expected %q, got %q", i, expected[i], queries[i])
		}
	}
}

func TestNormalizeLokiLogsExtractsTraceContextFromMessage(t *testing.T) {
	logs := normalizeLokiLogs(lokiQueryRangeResp{
		Status: "success",
		Data: struct {
			Result []lokiStream `json:"result"`
		}{
			Result: []lokiStream{{
				Stream: map[string]string{
					"app":      "demo-provider",
					"trace_id": "label-trace",
					"span_id":  "label-span",
				},
				Values: [][]string{{
					"1777110661783444000",
					`ERROR trace_id=message-trace span_id=message-span metadata report failed`,
				}},
			}},
		},
	})

	if len(logs) != 1 {
		t.Fatalf("expected one log, got %d", len(logs))
	}
	if logs[0].TraceID != "message-trace" || logs[0].SpanID != "message-span" {
		t.Fatalf("expected trace context from message, got traceID=%q spanID=%q", logs[0].TraceID, logs[0].SpanID)
	}
}

func TestNormalizeLokiLogsParsesDubboGoJSONLog(t *testing.T) {
	raw := `{"level":"error","msg":"error message","span_id":"36d69a3dd9bea02c","time":"2026-06-02T14:56:37+08:00","trace_flags":"01","trace_id":"faba6a688ea3070b1613f50fb081c578"}`
	logs := normalizeLokiLogs(lokiQueryRangeResp{
		Status: "success",
		Data: struct {
			Result []lokiStream `json:"result"`
		}{
			Result: []lokiStream{{
				Stream: map[string]string{
					"app":          "dubbo-go-provider",
					"service_name": "org.apache.DemoService",
				},
				Values: [][]string{{
					"1777110661783444000",
					raw,
				}},
			}},
		},
	})

	if len(logs) != 1 {
		t.Fatalf("expected one log, got %d", len(logs))
	}
	logItem := logs[0]
	if logItem.Timestamp != "2026-06-02T06:56:37Z" {
		t.Fatalf("expected timestamp from JSON log time, got %q", logItem.Timestamp)
	}
	if logItem.Severity != "error" || logItem.Message != "error message" {
		t.Fatalf("expected level and msg from JSON log, got severity=%q message=%q", logItem.Severity, logItem.Message)
	}
	if logItem.TraceID != "faba6a688ea3070b1613f50fb081c578" || logItem.SpanID != "36d69a3dd9bea02c" || logItem.TraceFlags != "01" {
		t.Fatalf("unexpected trace context: traceID=%q spanID=%q traceFlags=%q", logItem.TraceID, logItem.SpanID, logItem.TraceFlags)
	}
	if logItem.Raw != raw {
		t.Fatalf("expected raw JSON log to be preserved, got %q", logItem.Raw)
	}
}

func TestSearchLogsQueriesLoki(t *testing.T) {
	var seenQueries []string
	loki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/loki/api/v1/labels":
			http.Error(w, "labels unavailable", http.StatusInternalServerError)
		case "/loki/api/v1/query_range":
			seenQueries = append(seenQueries, r.URL.Query().Get("query"))
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": {
					"resultType": "streams",
					"result": [{
						"stream": {
							"app": "demo-provider",
							"service_name": "org.apache.DemoService",
							"instance": "127.0.0.1:20880",
							"level": "ERROR",
							"namespace": "dubbo-system"
						},
						"values": [["1777110661783444000", "ERROR trace_id=trace-1 span_id=span-1 test log"]]
					}]
				}
			}`))
		default:
			t.Fatalf("unexpected loki path: %s", r.URL.Path)
		}
	}))
	defer loki.Close()

	result, err := SearchLogs(newLogToolTestContext(loki.URL), map[string]any{
		"serviceName": "org.apache.DemoService",
		"keywords":    "ERROR",
		"startTime":   "2026-04-01T00:00:00Z",
		"endTime":     "2026-04-01T01:00:00Z",
		"limit":       float64(10),
	})
	if err != nil {
		t.Fatalf("SearchLogs returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("SearchLogs returned error result: %s", result.Content[0].Text)
	}

	var payload SearchLogsResp
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("failed to decode tool result: %v", err)
	}
	if len(payload.Logs) != 1 {
		t.Fatalf("expected one deduplicated log, got %d", len(payload.Logs))
	}
	logItem := payload.Logs[0]
	if logItem.ServiceName != "org.apache.DemoService" || logItem.TraceID != "trace-1" || logItem.Severity != "ERROR" {
		t.Fatalf("unexpected normalized log: %+v", logItem)
	}
	if len(seenQueries) != 3 {
		t.Fatalf("expected three service label alias queries, got %d: %v", len(seenQueries), seenQueries)
	}
}

func TestSearchLogsUsesLokiLabelsToReduceAliasQueries(t *testing.T) {
	var seenQueries []string
	labelsRequested := false
	loki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/loki/api/v1/labels":
			labelsRequested = true
			_, _ = w.Write([]byte(`{"status":"success","data":["namespace","service_name","pod"]}`))
		case "/loki/api/v1/query_range":
			seenQueries = append(seenQueries, r.URL.Query().Get("query"))
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": {
					"result": [{
						"stream": {"service_name": "org.apache.DemoService", "namespace": "dubbo-system"},
						"values": [["1777110661783444000", "ERROR trace_id=trace-1 test log"]]
					}]
				}
			}`))
		default:
			t.Fatalf("unexpected loki path: %s", r.URL.Path)
		}
	}))
	defer loki.Close()

	result, err := SearchLogs(newLogToolTestContext(loki.URL), map[string]any{
		"serviceName": "org.apache.DemoService",
		"keywords":    "ERROR",
		"startTime":   "2026-04-01T00:00:00Z",
		"endTime":     "2026-04-01T01:00:00Z",
	})
	if err != nil {
		t.Fatalf("SearchLogs returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("SearchLogs returned error result: %s", result.Content[0].Text)
	}
	if !labelsRequested {
		t.Fatal("expected labels endpoint to be requested")
	}
	expected := []string{`{service_name="org.apache.DemoService"} |= "ERROR"`}
	if len(seenQueries) != len(expected) {
		t.Fatalf("expected %d queries, got %d: %v", len(expected), len(seenQueries), seenQueries)
	}
	for i := range expected {
		if seenQueries[i] != expected[i] {
			t.Fatalf("query[%d] expected %q, got %q", i, expected[i], seenQueries[i])
		}
	}
}

func TestSearchLogsUsesAvailableFallbackLabelWhenOnlyTraceIDIsProvided(t *testing.T) {
	var seenQueries []string
	loki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/loki/api/v1/labels":
			_, _ = w.Write([]byte(`{"status":"success","data":["pod","job"]}`))
		case "/loki/api/v1/query_range":
			seenQueries = append(seenQueries, r.URL.Query().Get("query"))
			_, _ = w.Write([]byte(`{"status":"success","data":{"result":[]}}`))
		default:
			t.Fatalf("unexpected loki path: %s", r.URL.Path)
		}
	}))
	defer loki.Close()

	result, err := SearchLogs(newLogToolTestContext(loki.URL), map[string]any{
		"traceId":   "trace-1",
		"startTime": "2026-04-01T00:00:00Z",
		"endTime":   "2026-04-01T01:00:00Z",
	})
	if err != nil {
		t.Fatalf("SearchLogs returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("SearchLogs returned error result: %s", result.Content[0].Text)
	}
	expected := []string{`{job=~".+"} |= "trace-1"`}
	if len(seenQueries) != len(expected) {
		t.Fatalf("expected %d queries, got %d: %v", len(expected), len(seenQueries), seenQueries)
	}
	for i := range expected {
		if seenQueries[i] != expected[i] {
			t.Fatalf("query[%d] expected %q, got %q", i, expected[i], seenQueries[i])
		}
	}
}

func TestAnalyzeErrorsGroupsPatterns(t *testing.T) {
	resp := analyzeErrors([]LogItem{
		{Timestamp: "2026-04-01T00:00:00Z", Severity: "ERROR", Message: "Error 500 for request 123"},
		{Timestamp: "2026-04-01T00:01:00Z", Severity: "ERROR", Message: "Error 503 for request 456"},
		{Timestamp: "2026-04-01T00:02:00Z", Severity: "INFO", Message: "normal log"},
	}, "loki")

	if resp.TotalErrors != 2 {
		t.Fatalf("expected 2 errors, got %d", resp.TotalErrors)
	}
	if len(resp.Patterns) != 1 {
		t.Fatalf("expected one pattern, got %d", len(resp.Patterns))
	}
	if resp.Patterns[0].Pattern != "Error ? for request ?" || resp.Patterns[0].Count != 2 {
		t.Fatalf("unexpected pattern: %+v", resp.Patterns[0])
	}
}

func newLogToolTestContext(endpoint string) consolectx.Context {
	return &logToolTestContext{
		config: app.AdminConfig{
			Observability: &observability.Config{
				Logs: &observability.LogsConfig{
					DefaultProvider: "loki-main",
					Providers: []observability.LogProviderConfig{{
						Name:     "loki-main",
						Type:     observability.LogProviderLoki,
						Endpoint: endpoint,
					}},
				},
			},
		},
	}
}

type logToolTestContext struct {
	config app.AdminConfig
}

func (c *logToolTestContext) ResourceManager() manager.ResourceManager {
	return nil
}

func (c *logToolTestContext) CounterManager() counter.CounterManager {
	return nil
}

func (c *logToolTestContext) Config() app.AdminConfig {
	return c.config
}

func (c *logToolTestContext) AppContext() context.Context {
	return context.Background()
}

func (c *logToolTestContext) LockManager() lock.Lock {
	return nil
}
