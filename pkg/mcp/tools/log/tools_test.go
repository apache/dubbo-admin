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
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
)

func TestLogRegistrarRegistersExpectedTools(t *testing.T) {
	reg := registry.NewRegistry()
	(&LogRegistrar{}).RegisterTools(reg)

	if got := reg.Count(); got != 2 {
		t.Fatalf("expected 2 log tools, got %d", got)
	}
	for _, name := range []string{"search_logs", "analyze_error_logs"} {
		tool, ok := reg.Get(name)
		if !ok {
			t.Fatalf("tool %s was not registered", name)
		}
		if tool.Handler == nil {
			t.Fatalf("tool %s handler is nil", name)
		}
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

func TestSearchLogsQueriesLoki(t *testing.T) {
	var seenQueries []string
	loki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/query_range" {
			t.Fatalf("unexpected loki path: %s", r.URL.Path)
		}
		seenQueries = append(seenQueries, r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
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
						"trace_id": "trace-1",
						"span_id": "span-1",
						"namespace": "dubbo-system"
					},
					"values": [["1777110661783444000", "ERROR test log"]]
				}]
			}
		}`))
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
