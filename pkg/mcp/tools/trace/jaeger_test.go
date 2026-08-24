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
	"net/http"
	"net/http/httptest"
	"testing"

	observabilitycfg "github.com/apache/dubbo-admin/pkg/config/observability"
)

const testTraceID = "faba6a688ea3070b1613f50fb081c578"

func TestJaegerProviderQueriesAndNormalizesTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/traces/"+testTraceID {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" || r.Header.Get(tenantHeader) != "tenant-a" {
			t.Fatalf("missing auth headers: %#v", r.Header)
		}
		_, _ = w.Write([]byte(`{"data":[{"traceID":"` + testTraceID + `","spans":[{"traceID":"` + testTraceID + `","spanID":"01","operationName":"root","references":[],"startTime":1000000,"duration":5000,"tags":[],"logs":[],"processID":"p1"},{"traceID":"` + testTraceID + `","spanID":"02","operationName":"child","references":[{"refType":"CHILD_OF","traceID":"` + testTraceID + `","spanID":"01"}],"startTime":1001000,"duration":2000,"tags":[{"key":"error","type":"bool","value":true}],"logs":[],"processID":"p2"}],"processes":{"p1":{"serviceName":"frontend","tags":[]},"p2":{"serviceName":"backend","tags":[]}}}]}`))
	}))
	defer server.Close()

	provider, err := newJaegerProvider(observabilitycfg.TraceProviderConfig{
		Name: "jaeger-main", Type: observabilitycfg.TraceProviderJaeger, Endpoint: server.URL, BearerToken: "secret", Tenant: "tenant-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.GetTraceByID(context.Background(), testTraceID)
	if err != nil {
		t.Fatal(err)
	}
	if result.SpanCount != 2 || result.ReturnedSpanCount != 2 || result.ErrorSpanCount != 1 || result.ServiceCount != 2 {
		t.Fatalf("unexpected trace summary: %#v", result)
	}
	if result.Spans[1].ParentSpanID != "01" || result.Spans[1].Status != "ERROR" {
		t.Fatalf("unexpected child span: %#v", result.Spans[1])
	}
}

func TestTraceIDValidation(t *testing.T) {
	for _, value := range []string{"abc", "../trace", "1234567890abcdefg"} {
		if traceIDPattern.MatchString(value) {
			t.Fatalf("expected invalid trace ID %q", value)
		}
	}
	for _, value := range []string{"1234567890abcdef", testTraceID} {
		if !traceIDPattern.MatchString(value) {
			t.Fatalf("expected valid trace ID %q", value)
		}
	}
}

func TestJaegerProviderReturnsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	provider, err := newJaegerProvider(observabilitycfg.TraceProviderConfig{
		Name: "jaeger-main", Type: observabilitycfg.TraceProviderJaeger, Endpoint: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.GetTraceByID(context.Background(), testTraceID); err == nil {
		t.Fatal("expected trace not found error")
	}
}

func TestNormalizeJaegerTraceTruncatesSpans(t *testing.T) {
	input := jaegerTrace{TraceID: testTraceID, Spans: make([]jaegerSpan, maxTraceSpans+1)}
	for i := range input.Spans {
		input.Spans[i] = jaegerSpan{SpanID: string(rune(i + 1)), StartTime: int64(i + 1), Duration: 1}
	}
	result := normalizeJaegerTrace(input)
	if !result.Truncated || result.SpanCount != maxTraceSpans+1 || result.ReturnedSpanCount != maxTraceSpans {
		t.Fatalf("expected trace truncation, got %#v", result)
	}
}
