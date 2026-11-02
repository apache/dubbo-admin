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
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	observabilitycfg "github.com/apache/dubbo-admin/pkg/config/observability"
)

const testSearchTraceID = "aabbccddeeff00112233445566778899"

func newSearchProvider(t *testing.T, endpoint string) *jaegerProvider {
	t.Helper()
	provider, err := newJaegerProvider(observabilitycfg.TraceProviderConfig{
		Name: "jaeger-main", Type: observabilitycfg.TraceProviderJaeger, Endpoint: endpoint, BearerToken: "secret", Tenant: "tenant-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func TestJaegerProviderSearchTracesBuildsRequest(t *testing.T) {
	var captured *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		_, _ = w.Write([]byte(`{"data":[{"traceID":"` + testSearchTraceID + `","spans":[{"traceID":"` + testSearchTraceID + `","spanID":"01","operationName":"root","references":[],"startTime":2000000,"duration":3000,"tags":[],"logs":[],"processID":"p1"}],"processes":{"p1":{"serviceName":"shop-order","tags":[]}}}]}`))
	}))
	defer server.Close()

	provider := newSearchProvider(t, server.URL)
	start := time.Unix(1700000000, 0).UTC()
	end := time.Unix(1700003600, 0).UTC()
	result, err := provider.SearchTraces(context.Background(), TraceSearchQuery{
		Service:     "shop-order",
		Operation:   "checkout",
		TraceID:     testSearchTraceID,
		Tags:        map[string]string{"rpc.system": "apache_dubbo"},
		StartTime:   start,
		EndTime:     end,
		Limit:       17,
		MinDuration: 250 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured == nil {
		t.Fatal("expected the provider to issue a request")
	}
	if captured.URL.Path != "/api/traces" {
		t.Fatalf("unexpected path %s", captured.URL.Path)
	}
	if captured.Header.Get("Authorization") != "Bearer secret" || captured.Header.Get(tenantHeader) != "tenant-a" {
		t.Fatalf("missing auth headers: %#v", captured.Header)
	}
	query := captured.URL.Query()
	if query.Get("service") != "shop-order" || query.Get("operation") != "checkout" {
		t.Fatalf("unexpected service filter: %v", query)
	}
	if query.Get("traceID") != testSearchTraceID {
		t.Fatalf("unexpected trace id filter: %v", query)
	}
	if query.Get("tags") != `{"rpc.system":"apache_dubbo"}` {
		t.Fatalf("unexpected tags filter: %v", query)
	}
	if query.Get("start") != "1700000000000000" || query.Get("end") != "1700003600000000" {
		t.Fatalf("unexpected time window: %v", query)
	}
	// Jaeger parses minDuration with time.ParseDuration, so the wire format is a
	// duration string and not a bare microsecond count.
	minDuration := query.Get("minDuration")
	if minDuration != "250ms" {
		t.Fatalf("unexpected min duration: %v", query)
	}
	// The stub does not validate the parameter like Jaeger does, so the value is
	// parsed here to keep a unit-less microsecond count from slipping through.
	parsed, err := time.ParseDuration(minDuration)
	if err != nil {
		t.Fatalf("min duration %q is not a duration Jaeger can parse: %v", minDuration, err)
	}
	if parsed != 250*time.Millisecond {
		t.Fatalf("unexpected parsed min duration: %v", parsed)
	}
	// One trace more than the caller asked for is requested, so a full result
	// proves that more traces match than were returned.
	if query.Get("limit") != "18" {
		t.Fatalf("unexpected limit: %v", query)
	}
	if result.Total != 1 || len(result.Traces) != 1 || result.SourceEngine != "jaeger" {
		t.Fatalf("unexpected search result: %#v", result)
	}
	summary := result.Traces[0]
	if summary.TraceID != testSearchTraceID || summary.RootService != "shop-order" || summary.RootOperation != "root" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.SpanCount != 1 || summary.ServiceCount != 1 || summary.ErrorSpanCount != 0 || summary.HasError {
		t.Fatalf("unexpected summary counters: %#v", summary)
	}
	if summary.DurationMicros != 3000 {
		t.Fatalf("unexpected summary duration: %#v", summary)
	}
}

func TestJaegerProviderSearchTracesOmitsEmptyParameters(t *testing.T) {
	var captured *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	provider := newSearchProvider(t, server.URL)
	result, err := provider.SearchTraces(context.Background(), TraceSearchQuery{Service: "shop-order"})
	if err != nil {
		t.Fatal(err)
	}
	query := captured.URL.Query()
	for _, key := range []string{"operation", "traceID", "tags", "start", "end", "minDuration", "limit"} {
		if query.Has(key) {
			t.Fatalf("expected parameter %q to be omitted, got %v", key, query)
		}
	}
	if query.Get("service") != "shop-order" {
		t.Fatalf("unexpected service filter: %v", query)
	}
	if result.Total != 0 || len(result.Traces) != 0 || result.Truncated {
		t.Fatalf("expected an empty but successful result, got %#v", result)
	}
}

func TestJaegerProviderSearchTracesRequiresService(t *testing.T) {
	provider := newSearchProvider(t, "http://127.0.0.1:1")
	_, err := provider.SearchTraces(context.Background(), TraceSearchQuery{})
	if err == nil {
		t.Fatal("expected a missing service error")
	}
	if !strings.Contains(err.Error(), "service") {
		t.Fatalf("expected the error to name the missing service, got %v", err)
	}
}

func TestJaegerProviderSearchTracesSummarizesOrphanRoot(t *testing.T) {
	// The parent of the span is not part of the trace, so the span is an orphan
	// and roots the trace on its own. The summary and the analyzer have to agree
	// on that root.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"traceID":"` + testSearchTraceID + `","spans":[{"traceID":"` + testSearchTraceID + `","spanID":"02","operationName":"orphan","references":[{"refType":"CHILD_OF","traceID":"` + testSearchTraceID + `","spanID":"missing"}],"startTime":2000000,"duration":3000,"tags":[],"logs":[],"processID":"p1"}],"processes":{"p1":{"serviceName":"shop-order","tags":[]}}}]}`))
	}))
	defer server.Close()

	provider := newSearchProvider(t, server.URL)
	result, err := provider.SearchTraces(context.Background(), TraceSearchQuery{Service: "shop-order"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Traces) != 1 {
		t.Fatalf("unexpected number of traces: %#v", result.Traces)
	}
	summary := result.Traces[0]
	if summary.RootService != "shop-order" || summary.RootOperation != "orphan" {
		t.Fatalf("expected the orphan to be reported as the root: %#v", summary)
	}
}

func TestJaegerProviderSearchTracesDetectsGenAIKind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"traceID":"` + testSearchTraceID + `","spans":[{"traceID":"` + testSearchTraceID + `","spanID":"01","operationName":"chat","references":[],"startTime":2000000,"duration":3000,"tags":[{"key":"gen_ai.operation.name","type":"string","value":"chat"}],"logs":[],"processID":"p1"}],"processes":{"p1":{"serviceName":"dubbo-admin-sre-agent","tags":[]}}}]}`))
	}))
	defer server.Close()

	provider := newSearchProvider(t, server.URL)
	result, err := provider.SearchTraces(context.Background(), TraceSearchQuery{Service: "dubbo-admin-sre-agent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Traces) != 1 || result.Traces[0].TraceKind != "genai" {
		t.Fatalf("expected a genai trace kind, got %#v", result.Traces)
	}
}

func TestJaegerProviderSearchTracesSortsByStartTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// The traces are returned out of order and use timestamps whose RFC3339
		// rendering would sort differently than the numeric microsecond value.
		_, _ = w.Write([]byte(`{"data":[
			{"traceID":"cccccccccccccccccccccccccccccccc","spans":[{"spanID":"01","operationName":"late","references":[],"startTime":9000000,"duration":10,"tags":[],"logs":[],"processID":"p1"}],"processes":{"p1":{"serviceName":"shop-order","tags":[]}}},
			{"traceID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","spans":[{"spanID":"01","operationName":"early","references":[],"startTime":1000000,"duration":10,"tags":[],"logs":[],"processID":"p1"}],"processes":{"p1":{"serviceName":"shop-order","tags":[]}}},
			{"traceID":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","spans":[{"spanID":"01","operationName":"middle","references":[],"startTime":5000000,"duration":10,"tags":[],"logs":[],"processID":"p1"}],"processes":{"p1":{"serviceName":"shop-order","tags":[]}}}
		]}`))
	}))
	defer server.Close()

	provider := newSearchProvider(t, server.URL)
	result, err := provider.SearchTraces(context.Background(), TraceSearchQuery{Service: "shop-order"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "cccccccccccccccccccccccccccccccc"}
	if len(result.Traces) != len(expected) {
		t.Fatalf("unexpected number of traces: %#v", result.Traces)
	}
	for i, traceID := range expected {
		if result.Traces[i].TraceID != traceID {
			t.Fatalf("unexpected trace order: %v", result.Traces)
		}
	}
}

func TestJaegerProviderSearchTracesTruncatesSummaries(t *testing.T) {
	spans := make([]string, 0, maxSearchTraceSummaries+5)
	for i := 0; i < maxSearchTraceSummaries+5; i++ {
		spans = append(spans, `{"traceID":"trace`+strconv.Itoa(i)+`","spans":[{"spanID":"01","operationName":"op","references":[],"startTime":`+strconv.Itoa(1000000+i)+`,"duration":10,"tags":[],"logs":[],"processID":"p1"}],"processes":{"p1":{"serviceName":"shop-order","tags":[]}}}`)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[` + strings.Join(spans, ",") + `]}`))
	}))
	defer server.Close()

	provider := newSearchProvider(t, server.URL)
	result, err := provider.SearchTraces(context.Background(), TraceSearchQuery{Service: "shop-order"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated {
		t.Fatal("expected the result to be marked as truncated")
	}
	if len(result.Traces) != maxSearchTraceSummaries {
		t.Fatalf("unexpected number of summaries: %d", len(result.Traces))
	}
	if result.Total != maxSearchTraceSummaries+5 {
		t.Fatalf("expected the backend total to be preserved, got %d", result.Total)
	}
}

func TestJaegerProviderSearchTracesReportsUpstreamFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream exploded"))
	}))
	defer server.Close()

	provider := newSearchProvider(t, server.URL)
	result, err := provider.SearchTraces(context.Background(), TraceSearchQuery{Service: "shop-order"})
	if err == nil {
		t.Fatal("expected an upstream error")
	}
	if result != nil {
		t.Fatalf("expected no result on failure, got %#v", result)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected the status code in the error, got %v", err)
	}
}

func TestJaegerProviderSearchTracesReportsEnvelopeErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"errors":[{"code":500,"msg":"query rejected"}]}`))
	}))
	defer server.Close()

	provider := newSearchProvider(t, server.URL)
	if _, err := provider.SearchTraces(context.Background(), TraceSearchQuery{Service: "shop-order"}); err == nil {
		t.Fatal("expected an envelope error")
	}
}

func TestBuildTraceSearchQuery(t *testing.T) {
	query, err := buildTraceSearchQuery(map[string]any{
		"service":       "shop-order",
		"operation":     "checkout",
		"traceId":       testSearchTraceID,
		"tags":          "rpc.system=apache_dubbo, http.status_code=500",
		"startTime":     "2026-10-24T10:00:00Z",
		"endTime":       "2026-10-24T11:00:00Z",
		"minDurationMs": 250,
		"limit":         33,
	})
	if err != nil {
		t.Fatal(err)
	}
	if query.Service != "shop-order" || query.Operation != "checkout" || query.TraceID != testSearchTraceID {
		t.Fatalf("unexpected query: %#v", query)
	}
	if len(query.Tags) != 2 || query.Tags["rpc.system"] != "apache_dubbo" || query.Tags["http.status_code"] != "500" {
		t.Fatalf("unexpected tags: %#v", query.Tags)
	}
	if query.StartTime.UTC().Format(time.RFC3339) != "2026-10-24T10:00:00Z" {
		t.Fatalf("unexpected start time: %v", query.StartTime)
	}
	if query.EndTime.UTC().Format(time.RFC3339) != "2026-10-24T11:00:00Z" {
		t.Fatalf("unexpected end time: %v", query.EndTime)
	}
	if query.MinDuration != 250*time.Millisecond {
		t.Fatalf("unexpected min duration: %v", query.MinDuration)
	}
	if query.Limit != 33 {
		t.Fatalf("unexpected limit: %d", query.Limit)
	}
}

func TestBuildTraceSearchQueryDefaultsAndLimits(t *testing.T) {
	query, err := buildTraceSearchQuery(map[string]any{"service": "shop-order"})
	if err != nil {
		t.Fatal(err)
	}
	if query.Limit != defaultSearchLimit {
		t.Fatalf("unexpected default limit: %d", query.Limit)
	}
	if !query.StartTime.IsZero() || !query.EndTime.IsZero() || query.MinDuration != 0 || query.Tags != nil {
		t.Fatalf("expected empty optional filters, got %#v", query)
	}

	query, err = buildTraceSearchQuery(map[string]any{"service": "shop-order", "limit": float64(1000)})
	if err != nil {
		t.Fatal(err)
	}
	if query.Limit != maxSearchLimit {
		t.Fatalf("expected the limit to be capped, got %d", query.Limit)
	}

	query, err = buildTraceSearchQuery(map[string]any{"service": "shop-order", "limit": float64(-5)})
	if err != nil {
		t.Fatal(err)
	}
	if query.Limit != defaultSearchLimit {
		t.Fatalf("expected a negative limit to fall back, got %d", query.Limit)
	}
}

func TestBuildTraceSearchQueryRejectsInvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "missing service", args: map[string]any{}},
		{name: "blank service", args: map[string]any{"service": "  "}},
		{name: "invalid trace id", args: map[string]any{"service": "shop-order", "traceId": "not-hex"}},
		{name: "malformed tags", args: map[string]any{"service": "shop-order", "tags": "rpc.system"}},
		{name: "tags without key", args: map[string]any{"service": "shop-order", "tags": "=value"}},
		{name: "invalid start time", args: map[string]any{"service": "shop-order", "startTime": "2026-10-24"}},
		{name: "invalid end time", args: map[string]any{"service": "shop-order", "endTime": "yesterday"}},
		{name: "reversed window", args: map[string]any{"service": "shop-order", "startTime": "2026-10-24T11:00:00Z", "endTime": "2026-10-24T10:00:00Z"}},
		{name: "negative duration", args: map[string]any{"service": "shop-order", "minDurationMs": float64(-1)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := buildTraceSearchQuery(test.args); err == nil {
				t.Fatal("expected a validation error")
			}
		})
	}
}

func TestParseSearchTags(t *testing.T) {
	tags, err := parseSearchTags("")
	if err != nil || tags != nil {
		t.Fatalf("expected no tags, got %#v (%v)", tags, err)
	}
	tags, err = parseSearchTags(" a = 1 , b=2 ")
	if err != nil {
		t.Fatal(err)
	}
	if tags["a"] != "1" || tags["b"] != "2" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
	tags, err = parseSearchTags("a=1,a=2")
	if err != nil {
		t.Fatal(err)
	}
	if tags["a"] != "2" {
		t.Fatalf("expected the last value to win, got %#v", tags)
	}
	tags, err = parseSearchTags("a=b=c")
	if err != nil {
		t.Fatal(err)
	}
	if tags["a"] != "b=c" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
	if _, err := parseSearchTags("broken"); err == nil {
		t.Fatal("expected a malformed tag error")
	}
}

func TestSearchPropertiesExposeRequiredParameters(t *testing.T) {
	properties := SearchProperties()
	for _, name := range []string{"service", "operation", "traceId", "tags", "startTime", "endTime", "minDurationMs", "limit", "provider"} {
		if _, ok := properties[name]; !ok {
			t.Fatalf("search properties are missing %q", name)
		}
	}
	if properties["minDurationMs"].Type != "integer" {
		t.Fatalf("unexpected minDurationMs type: %s", properties["minDurationMs"].Type)
	}
}

func TestSearchTracesToolResultIsSerializable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"traceID":"` + testSearchTraceID + `","spans":[{"traceID":"` + testSearchTraceID + `","spanID":"01","operationName":"chat","references":[],"startTime":2000000,"duration":3000,"tags":[{"key":"gen_ai.operation.name","type":"string","value":"chat"}],"logs":[],"processID":"p1"}],"processes":{"p1":{"serviceName":"shop-order","tags":[]}}}]}`))
	}))
	defer server.Close()

	// The handler must produce a payload that survives a round trip.
	result, err := SearchTraces(newTraceToolTestContext(server.URL), map[string]any{"service": "shop-order"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Content[0].Text)
	}
	var payload TraceSearchResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Total != 1 || len(payload.Traces) != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if payload.Traces[0].TraceID != testSearchTraceID {
		t.Fatalf("unexpected summary: %#v", payload.Traces[0])
	}
	if !strings.Contains(result.Content[0].Text, `"traceId":"`+testSearchTraceID+`"`) {
		t.Fatalf("unexpected payload: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, `"traceKind":"genai"`) {
		t.Fatalf("expected the genai kind in the payload: %s", result.Content[0].Text)
	}
	// The tool result envelope must be serializable as well.
	if _, err := json.Marshal(result); err != nil {
		t.Fatal(err)
	}
}

func TestJaegerProviderSearchTracesReportsTruncationAtTheBackendLimit(t *testing.T) {
	// The provider is asked for one trace more than the caller wants, so a result
	// that reached the limit proves that more traces match than were returned.
	// Without the extra trace a full result would look complete.
	available := 30
	traces := make([]string, 0, available)
	for i := 0; i < available; i++ {
		traces = append(traces, `{"traceID":"trace`+strconv.Itoa(i)+`","spans":[{"spanID":"01","operationName":"op","references":[],"startTime":`+strconv.Itoa(1000000+i)+`,"duration":10,"tags":[],"logs":[],"processID":"p1"}],"processes":{"p1":{"serviceName":"shop-order","tags":[]}}}`)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit > len(traces) {
			limit = len(traces)
		}
		_, _ = w.Write([]byte(`{"data":[` + strings.Join(traces[:limit], ",") + `]}`))
	}))
	defer server.Close()
	provider := newSearchProvider(t, server.URL)

	// The backend has fewer traces than the caller asked for, so the result is
	// complete even though the request asked for one trace more.
	result, err := provider.SearchTraces(context.Background(), TraceSearchQuery{Service: "shop-order", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if result.Truncated {
		t.Fatal("did not expect the result to be truncated")
	}
	if len(result.Traces) != available || result.Total != available {
		t.Fatalf("unexpected result: %d traces, total %d", len(result.Traces), result.Total)
	}

	// The backend reached the requested limit, so more traces match than were
	// returned and the agent has to be told about it.
	result, err = provider.SearchTraces(context.Background(), TraceSearchQuery{Service: "shop-order", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated {
		t.Fatal("expected the result to be marked as truncated")
	}
	if len(result.Traces) != 20 || result.Total != 20 {
		t.Fatalf("unexpected result: %d traces, total %d", len(result.Traces), result.Total)
	}
}
