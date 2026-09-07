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
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"dubbo-admin-ai/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

func componentFromHookYAML(t *testing.T, text string) (*Component, error) {
	t.Helper()
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(text), &node); err != nil {
		t.Fatal(err)
	}
	raw, err := HookFactory(node.Content[0])
	if err != nil {
		return nil, err
	}
	c := raw.(*Component)
	return c, c.Validate()
}

func TestConfiguredTracingLifecycle(t *testing.T) {
	oldProvider, oldPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(oldProvider)
		otel.SetTextMapPropagator(oldPropagator)
	})
	for _, protocol := range []string{ProtocolGRPC, ProtocolHTTP} {
		t.Run(protocol, func(t *testing.T) {
			c, err := componentFromHookYAML(t, "hooks: [{name: trace, type: tracing, config: {protocol: '"+protocol+"', sample_ratio: 0}}]")
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Init(runtime.NewRuntime()); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := c.Stop(); err != nil {
					t.Errorf("Stop() error: %v", err)
				}
			})
			manager := c.GetActiveManager()
			if c.provider == nil || manager == nil || len(manager.registrations) != 1 {
				t.Fatal("tracing resources were not initialized")
			}
			ctx := manager.Emit(context.Background(), State{Event: EventInteractionStart})
			span := trace.SpanFromContext(ctx)
			if !span.SpanContext().IsValid() || span.IsRecording() {
				t.Fatal("expected an unsampled trace context with sample_ratio: 0")
			}
			manager.Emit(ctx, State{Event: EventInteractionEnd})
		})
	}
}

func TestConfiguredHookSelectionAndLoggingLevel(t *testing.T) {
	var output bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(oldLogger) })
	c, err := componentFromHookYAML(t, `hooks:
  - name: tool-failures
    type: logging
    events: [tool_call.error]
    tools: [lookup_service]
    config: {level: warn}
  - name: completed
    type: logging
    events: [interaction.end]
  - name: disabled
    type: logging
    enabled: false
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Init(runtime.NewRuntime()); err != nil {
		t.Fatal(err)
	}
	manager := c.GetManager()
	for _, state := range []State{
		{Event: EventToolCallError, ToolName: "other_tool"},
		{Event: EventModelCallEnd},
		{Event: EventToolCallError, ToolName: "lookup_service", Input: "secret-input", Output: "secret-output"},
		{Event: EventInteractionEnd},
	} {
		manager.Emit(context.Background(), state)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two selected events, got %s", output.String())
	}
	if !strings.Contains(lines[0], `"level":"WARN"`) || !strings.Contains(lines[0], `"hook":"tool-failures"`) {
		t.Fatalf("wrong logging config: %s", lines[0])
	}
	if !strings.Contains(lines[1], `"level":"INFO"`) || !strings.Contains(lines[1], `"hook":"completed"`) {
		t.Fatalf("wrong default logging config: %s", lines[1])
	}
	if manager.NeedsContent(EventToolCallError, "lookup_service") || strings.Contains(output.String(), "secret-") {
		t.Fatal("logging unexpectedly captured payloads")
	}
	calls := 0
	if err := manager.Register(Registration{Events: []Event{EventStageStart}, Hook: func(ctx context.Context, _ State) context.Context { calls++; return ctx }}); err != nil {
		t.Fatal(err)
	}
	manager.Emit(context.Background(), State{Event: EventStageStart})
	if calls != 1 {
		t.Fatal("programmatic registration stopped working")
	}
}

func TestConfiguredHooksValidateBeforeInitialization(t *testing.T) {
	tests := []struct{ name, yaml, want string }{
		{"mixed", "hooks: []\nlogging: {enabled: false}", "cannot combine"},
		{"mixed empty legacy", "hooks: []\ntracing: {}", "cannot combine"},
		{"null list", "hooks: null", "sequence"},
		{"unknown hook", "hooks: [{name: custom, type: unknown}]", "unsupported hook type"},
		{"empty name", "hooks: [{name: ' ', type: logging}]", "name"},
		{"duplicate name", "hooks: [{name: log, type: logging}, {name: log, type: logging}]", "duplicate hook name"},
		{"unknown event", "hooks: [{name: log, type: logging, events: [model_call.chunk]}]", "unsupported hook event"},
		{"empty events", "hooks: [{name: log, type: logging, events: []}]", "events"},
		{"duplicate events", "hooks: [{name: log, type: logging, events: [interaction.end, interaction.end]}]", "duplicate event"},
		{"empty tools", "hooks: [{name: log, type: logging, tools: []}]", "tools"},
		{"blank tool", "hooks: [{name: log, type: logging, tools: [' ']}]", "tool"},
		{"duplicate tools", "hooks: [{name: log, type: logging, tools: [x, x]}]", "duplicate tool"},
		{"logging options", "hooks: [{name: log, type: logging, config: {capture_content: full}}]", "capture_content"},
		{"logging level", "hooks: [{name: log, type: logging, config: {level: invalid}}]", "level"},
		{"tracing options", "hooks: [{name: trace, type: tracing, config: {level: info}}]", "level"},
		{"partial tracing", "hooks: [{name: trace, type: tracing, events: [interaction.start, interaction.end]}]", "all lifecycle events"},
		{"tracing tool filter", "hooks: [{name: trace, type: tracing, tools: [lookup]}]", "all tools"},
		{"multiple tracing", "hooks: [{name: trace1, type: tracing}, {name: trace2, type: tracing}]", "one tracing"},
		{"invalid disabled entry", "hooks: [{name: log, type: logging, enabled: false, events: [bad]}]", "unsupported hook event"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := componentFromHookYAML(t, tt.yaml)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
	// Direct Go construction is validated too, before allocating exporters.
	c := NewComponent(Spec{Hooks: []HookSpec{{Name: "bad", Type: "unknown"}}})
	originalManager := c.GetManager()
	if err := c.Init(runtime.NewRuntime()); err == nil {
		t.Fatal("Init accepted an unknown hook type")
	}
	if c.GetManager() != originalManager || c.provider != nil {
		t.Fatal("invalid configuration changed component resources")
	}
}

func TestConfiguredHookDefaultsAndLegacyCompatibility(t *testing.T) {
	for _, input := range []string{"hooks: []", "hooks: [{name: log, type: logging, enabled: false}]", "logging: {enabled: false}\ntracing: {enabled: false}"} {
		c, err := componentFromHookYAML(t, input)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Init(runtime.NewRuntime()); err != nil {
			t.Fatal(err)
		}
		if c.GetActiveManager() != nil {
			t.Fatalf("unexpected active hooks for %s", input)
		}
	}
	for _, input := range []string{"logging: {enabled: true}", "hooks: [{name: logging, type: logging}]"} {
		c, err := componentFromHookYAML(t, input)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Init(runtime.NewRuntime()); err != nil {
			t.Fatal(err)
		}
		regs := c.GetManager().registrations
		if len(regs) != 1 || len(regs[0].events) != len(AllEvents()) || !regs[0].allTools {
			t.Fatalf("wrong defaults for %s: %+v", input, regs)
		}
	}
	c, err := componentFromHookYAML(t, "hooks: [{name: trace, type: tracing, config: {sample_ratio: 0}}]")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := c.spec.normalizedHooks()
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].tracing.SampleRatio != 0 || entries[0].tracing.ServiceName != "dubbo-admin-ai" || entries[0].tracing.CaptureContent != CaptureNone {
		t.Fatalf("wrong tracing defaults: %+v", entries[0].tracing)
	}
}
