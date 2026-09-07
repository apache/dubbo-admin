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

package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/config"
	"dubbo-admin-ai/runtime"

	"go.opentelemetry.io/otel"
)

func TestJaegerOTLPEndToEnd(t *testing.T) {
	if os.Getenv("DUBBO_ADMIN_OTEL_E2E") != "1" {
		t.Skip("set DUBBO_ADMIN_OTEL_E2E=1 with a Jaeger OTLP endpoint to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	protocol := os.Getenv("DUBBO_ADMIN_OTEL_E2E_PROTOCOL")
	if protocol == "" {
		protocol = ProtocolGRPC
	}
	var logs strings.Builder
	oldLogger := slog.Default()
	oldProvider, oldPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(oldLogger)
		otel.SetTracerProvider(oldProvider)
		otel.SetTextMapPropagator(oldPropagator)
	})
	dir := t.TempDir()
	schemaDir, err := filepath.Abs("../../schema/json")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCHEMA_DIR", schemaDir)
	configYAML := fmt.Sprintf(`type: hooks
spec:
  hooks:
    - name: selected-tool
      type: logging
      events: [tool_call.end]
      tools: [lookup_service]
      config: {level: warn}
    - name: disabled-log
      type: logging
      enabled: false
    - name: jaeger
      type: tracing
      config:
        protocol: %q
        service_name: dubbo-admin-ai-e2e
`, protocol)
	if err := os.WriteFile(filepath.Join(dir, "hooks.yaml"), []byte(configYAML), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.NewLoader(filepath.Join(dir, "config.yaml")).LoadComponent("hooks.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := HookFactory(&loaded.Spec)
	if err != nil {
		t.Fatal(err)
	}
	component := raw.(*Component)
	if err := component.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := component.Init(runtime.NewRuntime()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = component.Stop() })
	manager := component.GetActiveManager()

	interaction := manager.Emit(context.Background(), State{
		Event: EventInteractionStart, InteractionID: "jaeger-e2e", SessionID: "jaeger-session",
	})
	traceID := TraceIDFromContext(interaction)
	stage := manager.Emit(interaction, State{
		Event: EventStageStart, InteractionID: "jaeger-e2e", SessionID: "jaeger-session", Stage: "reasonAct",
	})
	model := manager.Emit(stage, State{
		Event: EventModelCallStart, InteractionID: "jaeger-e2e", SessionID: "jaeger-session", Model: "dashscope/qwen-max",
	})
	manager.Emit(model, State{
		Event: EventModelCallEnd, InteractionID: "jaeger-e2e", SessionID: "jaeger-session", Model: "dashscope/qwen-max",
		InputTokens: 5, OutputTokens: 3, TotalTokens: 8,
	})
	tool := manager.Emit(stage, State{
		Event: EventToolCallStart, InteractionID: "jaeger-e2e", SessionID: "jaeger-session",
		ToolName: "lookup_service", ToolCallID: "call-jaeger-e2e",
	})
	manager.Emit(tool, State{
		Event: EventToolCallEnd, InteractionID: "jaeger-e2e", SessionID: "jaeger-session",
		ToolName: "lookup_service", ToolCallID: "call-jaeger-e2e",
	})
	manager.Emit(stage, State{
		Event: EventStageEnd, InteractionID: "jaeger-e2e", SessionID: "jaeger-session", Stage: "reasonAct",
	})
	manager.Emit(interaction, State{
		Event: EventInteractionEnd, InteractionID: "jaeger-e2e", SessionID: "jaeger-session",
	})
	// Shutdown must flush the actual batch exporter before querying Jaeger.
	if err := component.Stop(); err != nil {
		t.Fatalf("stop and flush traces: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], `"hook":"selected-tool"`) || !strings.Contains(lines[0], `"level":"WARN"`) {
		t.Fatalf("unexpected configured logging output: %s", logs.String())
	}

	queryURL := strings.TrimRight(os.Getenv("JAEGER_QUERY_URL"), "/")
	if queryURL == "" {
		queryURL = "http://localhost:16686"
	}
	want := []string{"invoke_agent", "agent.stage reasonAct", "chat qwen-max", "execute_tool lookup_service", "gen_ai.provider.name", "gen_ai.tool.call.id"}
	deadline := time.Now().Add(5 * time.Second)
	for {
		body, queryErr := fetchJaegerTrace(ctx, queryURL, traceID)
		if queryErr == nil {
			missing := ""
			for _, value := range want {
				if !strings.Contains(body, value) {
					missing = value
					break
				}
			}
			if missing == "" {
				verifyJaegerSpanTree(t, body)
				t.Logf("verified configured %s export, log filtering and shutdown flush; Jaeger trace %s", protocol, traceID)
				return
			}
			queryErr = fmt.Errorf("trace is missing %q", missing)
		}
		if time.Now().After(deadline) {
			t.Fatalf("query Jaeger trace %s: %v", traceID, queryErr)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func verifyJaegerSpanTree(t *testing.T, body string) {
	t.Helper()
	var result struct {
		Data []struct {
			Spans []struct {
				SpanID        string `json:"spanID"`
				OperationName string `json:"operationName"`
				References    []struct {
					RefType string `json:"refType"`
					SpanID  string `json:"spanID"`
				} `json:"references"`
			} `json:"spans"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || len(result.Data[0].Spans) != 4 {
		t.Fatalf("expected one trace with four spans: %s", body)
	}
	ids := make(map[string]string)
	for _, span := range result.Data[0].Spans {
		ids[span.OperationName] = span.SpanID
	}
	parents := map[string]string{"agent.stage reasonAct": "invoke_agent", "chat qwen-max": "agent.stage reasonAct", "execute_tool lookup_service": "agent.stage reasonAct"}
	for _, span := range result.Data[0].Spans {
		parent, child := parents[span.OperationName]
		if child && (len(span.References) != 1 || span.References[0].RefType != "CHILD_OF" || span.References[0].SpanID != ids[parent]) {
			t.Fatalf("wrong parent for %s: %+v", span.OperationName, span.References)
		}
	}
}

func fetchJaegerTrace(ctx context.Context, queryURL, traceID string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL+"/api/traces/"+traceID, nil)
	if err != nil {
		return "", err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %s: %s", response.Status, body)
	}
	return string(body), nil
}
