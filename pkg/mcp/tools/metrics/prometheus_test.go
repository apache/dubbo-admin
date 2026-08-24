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

package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPrometheusInstantQueryNormalizesVector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/query" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("query") != "up" {
			t.Fatalf("unexpected query %q", r.Form.Get("query"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"job":"dubbo"},"value":[1710000000.5,"NaN"]}]},"warnings":["partial data"]}`))
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	result, err := newPrometheusClient(baseURL).instant(context.Background(), "up", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.ResultType != "vector" || result.ReturnedSeries != 1 || result.ReturnedSamples != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Series[0].Value.Value != "NaN" || result.Series[0].Labels["job"] != "dubbo" {
		t.Fatalf("unexpected normalized series: %#v", result.Series[0])
	}
}

func TestPrometheusRangeValidation(t *testing.T) {
	baseURL, _ := url.Parse("http://prometheus:9090")
	client := newPrometheusClient(baseURL)
	_, err := client.rangeQuery(context.Background(), "rate(x[5m])", "2026-01-01T00:00:00Z", "2026-01-02T00:00:01Z", "15s")
	if err == nil || !strings.Contains(err.Error(), "must not exceed") {
		t.Fatalf("expected range limit error, got %v", err)
	}
	_, err = client.rangeQuery(context.Background(), "rate(x[5m])", "2026-01-01T00:00:00Z", "2026-01-01T01:00:00Z", "5s")
	if err == nil || !strings.Contains(err.Error(), "at least") {
		t.Fatalf("expected step limit error, got %v", err)
	}
}

func TestNormalizePrometheusMatrixTruncatesSamples(t *testing.T) {
	values := make([]string, 0, maxSamples+1)
	for i := 0; i < maxSamples+1; i++ {
		values = append(values, `[1710000000,"1"]`)
	}
	upstream := &prometheusResponse{Status: "success"}
	upstream.Data.ResultType = "matrix"
	upstream.Data.Result = []byte(`[{"metric":{"job":"dubbo"},"values":[` + strings.Join(values, ",") + `]}]`)
	result, err := normalizePrometheusResult("range", upstream)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated || result.ReturnedSamples != maxSamples {
		t.Fatalf("expected sample truncation, got %#v", result)
	}
}

func TestPrometheusReturnsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("bad query"))
	}))
	defer server.Close()
	baseURL, _ := url.Parse(server.URL)
	_, err := newPrometheusClient(baseURL).instant(context.Background(), "bad(", "")
	if err == nil || !strings.Contains(err.Error(), "status 422") {
		t.Fatalf("expected upstream status error, got %v", err)
	}
}

func TestReadBoundedBodyRejectsOversizedResponse(t *testing.T) {
	_, err := readBoundedBody(strings.NewReader("12345"), 4)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected response limit error, got %v", err)
	}
}
