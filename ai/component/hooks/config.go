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
	"fmt"
	"log/slog"
	"strings"

	"gopkg.in/yaml.v3"
)

// HookSpec configures a built-in observer at startup. Name identifies the
// instance; Type selects logging or tracing. Nil selectors mean all events/tools.
type HookSpec struct {
	Name    string    `yaml:"name"`
	Type    string    `yaml:"type"`
	Enabled *bool     `yaml:"enabled,omitempty"`
	Events  []Event   `yaml:"events,omitempty"`
	Tools   []string  `yaml:"tools,omitempty"`
	Config  yaml.Node `yaml:"config,omitempty"`
}

// Spec accepts either the legacy fixed observers or an explicit hook list.
// A non-nil empty Hooks list disables all configured observers.
type Spec struct {
	Logging          LoggingSpec `yaml:"logging,omitempty"`
	Tracing          TracingSpec `yaml:"tracing,omitempty"`
	Hooks            []HookSpec  `yaml:"hooks,omitempty"`
	legacyConfigured bool
}

func (s *Spec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("hooks spec must be a mapping")
	}
	type plainSpec Spec
	var decoded plainSpec
	if err := decodeHookConfig(node, &decoded); err != nil {
		return err
	}
	for i := 0; i < len(node.Content); i += 2 {
		switch node.Content[i].Value {
		case "logging", "tracing":
			decoded.legacyConfigured = true
		case "hooks":
			if node.Content[i+1].Kind != yaml.SequenceNode {
				return fmt.Errorf("hooks must be a sequence")
			}
		}
	}
	if decoded.Hooks != nil && decoded.legacyConfigured {
		return fmt.Errorf("cannot combine hooks with legacy logging/tracing configuration")
	}
	*s = Spec(decoded)
	return nil
}

func (h *HookSpec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("hook entry must be a mapping")
	}
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		switch key {
		case "events", "tools":
			if value.Kind != yaml.SequenceNode {
				return fmt.Errorf("%s must be a sequence", key)
			}
		case "enabled":
			if value.Tag != "!!bool" {
				return fmt.Errorf("enabled must be a boolean")
			}
		case "config":
			if value.Kind != yaml.MappingNode {
				return fmt.Errorf("hook config must be a mapping")
			}
		}
	}
	type plainHook HookSpec
	var decoded plainHook
	if err := decodeHookConfig(node, &decoded); err != nil {
		return err
	}
	*h = HookSpec(decoded)
	return nil
}

// Decode type-specific options strictly even when HookFactory is called
// directly rather than through the schema-validating configuration loader.
func decodeHookConfig(node *yaml.Node, target any) error {
	if node.Kind == 0 {
		return nil
	}
	data, err := yaml.Marshal(node)
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	return decoder.Decode(target)
}

type configuredHook struct {
	name    string
	kind    string
	enabled bool
	events  []Event
	tools   []string
	level   slog.Level
	tracing TracingSpec
}

// tracingOptions excludes enabled: enabling a list entry belongs on the entry,
// while the legacy TracingSpec keeps its original Go/YAML contract.
type tracingOptions struct {
	Protocol       string  `yaml:"protocol"`
	ServiceName    string  `yaml:"service_name"`
	SampleRatio    float64 `yaml:"sample_ratio"`
	CaptureContent string  `yaml:"capture_content"`
}

func (s Spec) normalizedHooks() ([]configuredHook, error) {
	if s.Hooks == nil {
		if s.Tracing.Enabled {
			if err := s.Tracing.validate(); err != nil {
				return nil, err
			}
		}
		return []configuredHook{
			{name: "logging", kind: "logging", enabled: s.Logging.Enabled, events: AllEvents(), tools: []string{AllTools}, level: slog.LevelInfo},
			{name: "tracing", kind: "tracing", enabled: s.Tracing.Enabled, events: AllEvents(), tools: []string{AllTools}, tracing: s.Tracing},
		}, nil
	}
	if s.legacyConfigured || s.Logging != (LoggingSpec{}) || s.Tracing != (TracingSpec{}) {
		return nil, fmt.Errorf("cannot combine hooks with legacy logging/tracing configuration")
	}
	entries := make([]configuredHook, 0, len(s.Hooks))
	names := make(map[string]bool)
	tracingCount := 0
	for _, spec := range s.Hooks {
		if strings.TrimSpace(spec.Name) == "" {
			return nil, fmt.Errorf("hook name is required")
		}
		if names[spec.Name] {
			return nil, fmt.Errorf("duplicate hook name %q", spec.Name)
		}
		names[spec.Name] = true
		entry, err := spec.normalized()
		if err != nil {
			return nil, fmt.Errorf("hook %q: %w", spec.Name, err)
		}
		if entry.kind == "tracing" {
			tracingCount++
			if tracingCount > 1 {
				return nil, fmt.Errorf("only one tracing hook may be configured")
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s HookSpec) normalized() (configuredHook, error) {
	h := configuredHook{name: s.Name, kind: s.Type, enabled: s.Enabled == nil || *s.Enabled, events: s.Events, tools: s.Tools}
	if h.events == nil {
		h.events = AllEvents()
	}
	if h.tools == nil {
		h.tools = []string{AllTools}
	}
	if len(h.events) == 0 {
		return h, fmt.Errorf("events must not be empty; use enabled: false to disable a hook")
	}
	events := make(map[Event]bool)
	for _, event := range h.events {
		if !event.valid() {
			return h, fmt.Errorf("unsupported hook event %q", event)
		}
		if events[event] {
			return h, fmt.Errorf("duplicate event %q", event)
		}
		events[event] = true
	}
	if len(h.tools) == 0 {
		return h, fmt.Errorf("tools must not be empty")
	}
	tools := make(map[string]bool)
	for _, tool := range h.tools {
		if strings.TrimSpace(tool) == "" {
			return h, fmt.Errorf("tool name must not be blank")
		}
		if tools[tool] {
			return h, fmt.Errorf("duplicate tool %q", tool)
		}
		tools[tool] = true
	}
	switch h.kind {
	case "logging":
		options := struct {
			Level string `yaml:"level"`
		}{Level: "info"}
		if err := decodeHookConfig(&s.Config, &options); err != nil {
			return h, err
		}
		switch options.Level {
		case "debug":
			h.level = slog.LevelDebug
		case "info":
			h.level = slog.LevelInfo
		case "warn":
			h.level = slog.LevelWarn
		case "error":
			h.level = slog.LevelError
		default:
			return h, fmt.Errorf("logging level must be debug, info, warn, or error")
		}
	case "tracing":
		if len(events) != len(AllEvents()) {
			return h, fmt.Errorf("tracing requires all lifecycle events")
		}
		if len(h.tools) != 1 || h.tools[0] != AllTools {
			return h, fmt.Errorf("tracing requires all tools (tools: [\"*\"])")
		}
		options := tracingOptions{Protocol: ProtocolGRPC, ServiceName: "dubbo-admin-ai", SampleRatio: 1, CaptureContent: CaptureNone}
		if err := decodeHookConfig(&s.Config, &options); err != nil {
			return h, err
		}
		h.tracing = TracingSpec{Enabled: h.enabled, Protocol: options.Protocol, ServiceName: options.ServiceName, SampleRatio: options.SampleRatio, CaptureContent: options.CaptureContent}
		if err := h.tracing.validate(); err != nil {
			return h, err
		}
	default:
		return h, fmt.Errorf("unsupported hook type %q", h.kind)
	}
	return h, nil
}
