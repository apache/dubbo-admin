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

package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/config"

	"github.com/xeipuuv/gojsonschema"
)

func loadHooksConfig(t *testing.T, spec string) (*config.Config, error) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SCHEMA_DIR", repoSchemaDir(t))
	writeFile(t, filepath.Join(dir, "hooks.yaml"), "type: hooks\nspec:\n"+spec)
	return config.NewLoader(filepath.Join(dir, "config.yaml")).LoadComponent("hooks.yaml")
}

func hookSpecMap(t *testing.T, cfg *config.Config) map[string]any {
	t.Helper()
	var spec map[string]any
	if err := cfg.Spec.Decode(&spec); err != nil {
		t.Fatal(err)
	}
	return spec
}

func validateLoadedHooks(t *testing.T, cfg *config.Config) {
	t.Helper()
	component, err := hooks.HookFactory(&cfg.Spec)
	if err != nil {
		t.Fatal(err)
	}
	if err = component.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestHooksLoaderLegacyDefaults(t *testing.T) {
	cfg, err := loadHooksConfig(t, "  {}\n")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"logging": map[string]any{"enabled": true},
		"tracing": map[string]any{"enabled": false, "protocol": "grpc", "service_name": "dubbo-admin-ai", "sample_ratio": 1, "capture_content": "none"},
	}
	if got := hookSpecMap(t, cfg); !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy defaults = %#v, want %#v", got, want)
	}
	validateLoadedHooks(t, cfg)
}

func TestHooksLoaderRegistrationDefaults(t *testing.T) {
	for _, kind := range []string{"logging", "tracing"} {
		t.Run(kind, func(t *testing.T) {
			cfg, err := loadHooksConfig(t, "  hooks:\n    - name: audit\n      type: "+kind+"\n")
			if err != nil {
				t.Fatal(err)
			}
			spec := hookSpecMap(t, cfg)
			if len(spec) != 1 {
				t.Fatalf("legacy defaults leaked into list configuration: %#v", spec)
			}
			entry := spec["hooks"].([]any)[0].(map[string]any)
			if entry["enabled"] != true {
				t.Fatalf("enabled = %#v", entry["enabled"])
			}
			var gotEvents []hooks.Event
			for _, event := range entry["events"].([]any) {
				gotEvents = append(gotEvents, hooks.Event(event.(string)))
			}
			if !reflect.DeepEqual(gotEvents, hooks.AllEvents()) {
				t.Fatalf("events = %v, want %v", gotEvents, hooks.AllEvents())
			}
			if !reflect.DeepEqual(entry["tools"], []any{"*"}) {
				t.Fatalf("tools = %#v", entry["tools"])
			}
			want := map[string]any{"level": "info"}
			if kind == "tracing" {
				want = map[string]any{"protocol": "grpc", "service_name": "dubbo-admin-ai", "sample_ratio": 1, "capture_content": "none"}
			}
			if !reflect.DeepEqual(entry["config"], want) {
				t.Fatalf("config = %#v, want %#v", entry["config"], want)
			}
			validateLoadedHooks(t, cfg)
		})
	}
}

func TestHooksLoaderEmptyListDisablesLegacyDefaults(t *testing.T) {
	cfg, err := loadHooksConfig(t, "  hooks: []\n")
	if err != nil {
		t.Fatal(err)
	}
	spec := hookSpecMap(t, cfg)
	if len(spec) != 1 || len(spec["hooks"].([]any)) != 0 {
		t.Fatalf("empty list changed: %#v", spec)
	}
	var typed hooks.Spec
	if err = cfg.Spec.Decode(&typed); err != nil {
		t.Fatal(err)
	}
	if typed.Hooks == nil || typed.Logging.Enabled || typed.Tracing.Enabled {
		t.Fatalf("explicit empty list lost or legacy hook enabled: %#v", typed)
	}
	validateLoadedHooks(t, cfg)
}

func TestHooksLoaderExplicitRegistrationValues(t *testing.T) {
	cfg, err := loadHooksConfig(t, `  hooks:
    - name: tool-audit
      type: logging
      enabled: false
      events: [tool_call.start, tool_call.error]
      tools: [search, lookup]
      config:
        level: warn
    - name: traces
      type: tracing
      config:
        protocol: http/protobuf
        service_name: custom-service
        sample_ratio: 0
        capture_content: truncated
`)
	if err != nil {
		t.Fatal(err)
	}
	entries := hookSpecMap(t, cfg)["hooks"].([]any)
	logging := entries[0].(map[string]any)
	if logging["enabled"] != false || !reflect.DeepEqual(logging["events"], []any{"tool_call.start", "tool_call.error"}) || !reflect.DeepEqual(logging["tools"], []any{"search", "lookup"}) || logging["config"].(map[string]any)["level"] != "warn" {
		t.Fatalf("explicit logging values lost: %#v", logging)
	}
	wantTracing := map[string]any{"protocol": "http/protobuf", "service_name": "custom-service", "sample_ratio": 0, "capture_content": "truncated"}
	if got := entries[1].(map[string]any)["config"]; !reflect.DeepEqual(got, wantTracing) {
		t.Fatalf("tracing config = %#v, want %#v", got, wantTracing)
	}
	validateLoadedHooks(t, cfg)
}

func TestHooksLoaderRejectsInvalidStructure(t *testing.T) {
	tests := map[string]string{
		"mixed logging":          "  logging: {}\n  hooks: []\n",
		"mixed tracing":          "  tracing: {}\n  hooks: []\n",
		"null list":              "  hooks: null\n",
		"null spec":              "  null\n",
		"list is object":         "  hooks: {}\n",
		"missing name":           "  hooks: [{type: logging}]\n",
		"blank name":             "  hooks: [{name: '  ', type: logging}]\n",
		"missing type":           "  hooks: [{name: audit}]\n",
		"unknown type":           "  hooks: [{name: audit, type: custom}]\n",
		"null entry":             "  hooks: [null]\n",
		"unknown entry field":    "  hooks: [{name: audit, type: logging, priority: 1}]\n",
		"enabled string":         "  hooks: [{name: audit, type: logging, enabled: 'false'}]\n",
		"null enabled":           "  hooks: [{name: audit, type: logging, enabled: null}]\n",
		"empty events":           "  hooks: [{name: audit, type: logging, events: []}]\n",
		"null events":            "  hooks: [{name: audit, type: logging, events: null}]\n",
		"unknown event":          "  hooks: [{name: audit, type: logging, events: [unknown]}]\n",
		"reserved chunk event":   "  hooks: [{name: audit, type: logging, events: [model_call.chunk]}]\n",
		"duplicate event":        "  hooks: [{name: audit, type: logging, events: [tool_call.start, tool_call.start]}]\n",
		"empty tools":            "  hooks: [{name: audit, type: logging, tools: []}]\n",
		"blank tool":             "  hooks: [{name: audit, type: logging, tools: ['  ']}]\n",
		"null tools":             "  hooks: [{name: audit, type: logging, tools: null}]\n",
		"duplicate tool":         "  hooks: [{name: audit, type: logging, tools: [search, search]}]\n",
		"null config":            "  hooks: [{name: audit, type: logging, config: null}]\n",
		"logging tracing config": "  hooks: [{name: audit, type: logging, config: {protocol: grpc}}]\n",
		"tracing logging config": "  hooks: [{name: audit, type: tracing, config: {level: info}}]\n",
		"logging enabled config": "  hooks: [{name: audit, type: logging, config: {enabled: false}}]\n",
		"tracing enabled config": "  hooks: [{name: audit, type: tracing, config: {enabled: false}}]\n",
		"invalid level":          "  hooks: [{name: audit, type: logging, config: {level: verbose}}]\n",
		"invalid ratio":          "  hooks: [{name: audit, type: tracing, config: {sample_ratio: 2}}]\n",
		"invalid protocol":       "  hooks: [{name: audit, type: tracing, config: {protocol: udp}}]\n",
		"invalid capture":        "  hooks: [{name: audit, type: tracing, config: {capture_content: raw}}]\n",
	}
	for name, spec := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := loadHooksConfig(t, spec)
			mustContain(t, err, "structural error")
		})
	}
}

func TestHooksSchemaEventEnumMatchesRuntime(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoSchemaDir(t), "hooks.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs struct {
			Event struct {
				Enum []hooks.Event `json:"enum"`
			} `json:"event"`
		} `json:"$defs"`
	}
	if err = json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(schema.Defs.Event.Enum, hooks.AllEvents()) {
		t.Fatalf("schema events = %v, runtime events = %v", schema.Defs.Event.Enum, hooks.AllEvents())
	}
	for _, event := range hooks.AllEvents() {
		t.Run(string(event), func(t *testing.T) {
			cfg, err := loadHooksConfig(t, "  hooks: [{name: audit, type: logging, events: ["+string(event)+"]}]\n")
			if err != nil {
				t.Fatal(err)
			}
			validateLoadedHooks(t, cfg)
		})
	}
}

func TestHooksLoaderSemanticValidation(t *testing.T) {
	tests := map[string]string{
		"duplicate names":        "  hooks: [{name: audit, type: logging}, {name: audit, type: tracing}]\n",
		"multiple tracers":       "  hooks: [{name: first, type: tracing}, {name: second, type: tracing, enabled: false}]\n",
		"partial tracing events": "  hooks: [{name: traces, type: tracing, events: [interaction.start]}]\n",
		"filtered tracing tools": "  hooks: [{name: traces, type: tracing, tools: [search]}]\n",
	}
	for name, spec := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := loadHooksConfig(t, spec)
			if err != nil {
				t.Fatalf("expected structurally valid configuration: %v", err)
			}
			component, err := hooks.HookFactory(&cfg.Spec)
			if err != nil {
				t.Fatal(err)
			}
			if err = component.Validate(); err == nil {
				t.Fatal("expected semantic validation error")
			}
		})
	}
}

func TestHooksSchemaValidatesConfigBeforeDefaults(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoSchemaDir(t), "hooks.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	schema, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(data))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		kind, config string
		valid        bool
	}{
		{"logging", `{}`, true},
		{"tracing", `{}`, true},
		{"logging", `{"level":"warn"}`, true},
		{"tracing", `{"sample_ratio":0}`, true},
		{"logging", `{"protocol":"grpc"}`, false},
		{"tracing", `{"level":"info"}`, false},
	} {
		t.Run(tt.kind+tt.config, func(t *testing.T) {
			doc := `{"type":"hooks","spec":{"hooks":[{"name":"observer","type":"` + tt.kind + `","config":` + tt.config + `}]}}`
			result, err := schema.Validate(gojsonschema.NewStringLoader(doc))
			if err != nil {
				t.Fatal(err)
			}
			if result.Valid() != tt.valid {
				t.Fatalf("valid = %v, want %v: %v", result.Valid(), tt.valid, result.Errors())
			}
		})
	}
}
