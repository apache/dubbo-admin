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
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestJaegerOTLPEndToEnd(t *testing.T) {
	if os.Getenv("DUBBO_ADMIN_OTEL_E2E") != "1" {
		t.Skip("set DUBBO_ADMIN_OTEL_E2E=1 with a Jaeger OTLP endpoint to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exporter, err := newTraceExporter(ctx, ProtocolGRPC)
	if err != nil {
		t.Fatalf("create exporter: %v", err)
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", "dubbo-admin-ai-e2e"))),
	)
	defer func() { _ = provider.Shutdown(context.Background()) }()

	manager := testManager()
	if err := manager.Register(NewTracingRegistration(provider.Tracer("dubbo-admin-ai/hooks/e2e"), CaptureNone)); err != nil {
		t.Fatalf("register tracing hook: %v", err)
	}

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
	if err := provider.ForceFlush(ctx); err != nil {
		t.Fatalf("flush traces: %v", err)
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
				t.Logf("verified Jaeger trace %s", traceID)
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
