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
	"strings"
	"testing"

	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/config/observability"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

type traceToolTestContext struct {
	config app.AdminConfig
}

func (c *traceToolTestContext) RuleVersioning() *versioning.Service { return nil }

func (c *traceToolTestContext) ResourceManager() manager.ResourceManager { return nil }

func (c *traceToolTestContext) CounterManager() counter.CounterManager { return nil }

func (c *traceToolTestContext) Config() app.AdminConfig { return c.config }

func (c *traceToolTestContext) AppContext() context.Context { return context.Background() }

func (c *traceToolTestContext) LockManager() lock.Lock { return nil }

func newTraceToolTestContext(endpoint string) consolectx.Context {
	return &traceToolTestContext{
		config: app.AdminConfig{
			Observability: &observability.Config{
				Tracing: &observability.TracingConfig{
					DefaultProvider: "jaeger-main",
					Providers: []observability.TraceProviderConfig{{
						Name:     "jaeger-main",
						Type:     observability.TraceProviderJaeger,
						Endpoint: endpoint,
					}},
				},
			},
		},
	}
}

const traceToolTestPayload = `{"data":[{"traceID":"` + testSearchTraceID + `","spans":[
	{"traceID":"` + testSearchTraceID + `","spanID":"01","operationName":"checkout","references":[],"startTime":2000000,"duration":100000,"tags":[],"logs":[],"processID":"p1"},
	{"traceID":"` + testSearchTraceID + `","spanID":"02","operationName":"charge","references":[{"refType":"CHILD_OF","traceID":"` + testSearchTraceID + `","spanID":"01"}],"startTime":2010000,"duration":50000,"tags":[{"key":"error","type":"bool","value":true},{"key":"error.type","type":"string","value":"timeout"}],"logs":[],"processID":"p2"}
],"processes":{"p1":{"serviceName":"order-service","tags":[]},"p2":{"serviceName":"payment-service","tags":[]}}}]}`

func TestSearchTracesToolReturnsSummaries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(traceToolTestPayload))
	}))
	defer server.Close()

	result, err := SearchTraces(newTraceToolTestContext(server.URL), map[string]any{"service": "order-service"})
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
	if payload.Traces[0].TraceID != testSearchTraceID || !payload.Traces[0].HasError {
		t.Fatalf("unexpected summary: %#v", payload.Traces[0])
	}
	if payload.Traces[0].ErrorSpanCount != 1 || payload.Traces[0].SpanCount != 2 {
		t.Fatalf("unexpected summary counters: %#v", payload.Traces[0])
	}
}

func TestSearchTracesToolRejectsMissingService(t *testing.T) {
	result, err := SearchTraces(newTraceToolTestContext("http://127.0.0.1:1"), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
	if !strings.Contains(result.Content[0].Text, "service is required") {
		t.Fatalf("unexpected error message: %s", result.Content[0].Text)
	}
}

func TestSearchTracesToolRequiresConfiguredProvider(t *testing.T) {
	result, err := SearchTraces(&traceToolTestContext{}, map[string]any{"service": "order-service"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
	if !strings.Contains(result.Content[0].Text, "trace provider is not configured") {
		t.Fatalf("unexpected error message: %s", result.Content[0].Text)
	}
}

func TestSearchTracesToolReportsUnknownProvider(t *testing.T) {
	result, err := SearchTraces(newTraceToolTestContext("http://127.0.0.1:1"), map[string]any{
		"service":  "order-service",
		"provider": "missing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
	if !strings.Contains(result.Content[0].Text, "missing") {
		t.Fatalf("unexpected error message: %s", result.Content[0].Text)
	}
}

func TestSearchTracesToolReturnsEmptyResultWithoutError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	result, err := SearchTraces(newTraceToolTestContext(server.URL), map[string]any{"service": "unknown-service"})
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
	if payload.Total != 0 || len(payload.Traces) != 0 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestAnalyzeTraceToolReturnsDiagnosis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(traceToolTestPayload))
	}))
	defer server.Close()

	result, err := AnalyzeTrace(newTraceToolTestContext(server.URL), map[string]any{"traceId": testSearchTraceID})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Content[0].Text)
	}
	var payload TraceAnalysis
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.TraceID != testSearchTraceID || payload.SpanCount != 2 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if len(payload.CriticalPath) != 2 || payload.CriticalPath[1].ServiceName != "payment-service" {
		t.Fatalf("unexpected critical path: %#v", payload.CriticalPath)
	}
	if payload.ErrorSpanCount != 1 || len(payload.ErrorSpans) != 1 {
		t.Fatalf("unexpected error spans: %#v", payload.ErrorSpans)
	}
	if !payload.ErrorSpans[0].RootCause || payload.ErrorSpans[0].ErrorType != "timeout" {
		t.Fatalf("unexpected error span: %#v", payload.ErrorSpans[0])
	}
	if len(payload.Dependencies) != 1 || payload.Dependencies[0].CallerService != "order-service" {
		t.Fatalf("unexpected dependencies: %#v", payload.Dependencies)
	}
	if payload.Summary == "" {
		t.Fatal("expected a summary")
	}
}

func TestAnalyzeTraceToolRejectsInvalidTraceID(t *testing.T) {
	result, err := AnalyzeTrace(newTraceToolTestContext("http://127.0.0.1:1"), map[string]any{"traceId": "not-a-trace-id"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
	if !strings.Contains(result.Content[0].Text, "traceId must be a 16 or 32 character hexadecimal value") {
		t.Fatalf("unexpected error message: %s", result.Content[0].Text)
	}
}

func TestAnalyzeTraceToolReportsUpstreamFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream exploded"))
	}))
	defer server.Close()

	result, err := AnalyzeTrace(newTraceToolTestContext(server.URL), map[string]any{"traceId": testSearchTraceID})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
}
