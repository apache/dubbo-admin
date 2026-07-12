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

package mcp

import "testing"

func TestRegisterServiceDetailTools(t *testing.T) {
	server := NewServer("test", "dev")
	RegisterTools(server)

	detail, ok := server.tools["get_service_detail"]
	if !ok {
		t.Fatal("Tool 'get_service_detail' not registered")
	}
	if detail.Handler == nil {
		t.Fatal("Tool 'get_service_detail' handler is nil")
	}
	if len(detail.InputSchema.Required) != 1 || detail.InputSchema.Required[0] != "serviceName" {
		t.Fatalf("Expected serviceName to be required, got %v", detail.InputSchema.Required)
	}
	for _, prop := range []string{"serviceName", "version", "group", "mesh"} {
		if _, ok := detail.InputSchema.Properties[prop]; !ok {
			t.Fatalf("get_service_detail missing property %q", prop)
		}
	}
	if _, ok := detail.InputSchema.Properties["side"]; ok {
		t.Fatal("get_service_detail should not expose side")
	}

	distribution, ok := server.tools["get_service_distribution"]
	if !ok {
		t.Fatal("Tool 'get_service_distribution' not registered")
	}
	if distribution.Handler == nil {
		t.Fatal("Tool 'get_service_distribution' handler is nil")
	}
	for _, prop := range []string{"serviceName", "version", "group", "side", "mesh"} {
		if _, ok := distribution.InputSchema.Properties[prop]; !ok {
			t.Fatalf("get_service_distribution missing property %q", prop)
		}
	}
}

func TestRegisterObservabilityTools(t *testing.T) {
	server := NewServer("test", "dev")
	RegisterTools(server)

	tests := []struct {
		name     string
		required []string
	}{
		{name: "query_prometheus", required: []string{"query"}},
		{name: "query_prometheus_range", required: []string{"query", "startTime", "endTime", "step"}},
		{name: "get_trace_by_id", required: []string{"traceId"}},
		{name: "get_observability_capabilities"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool, ok := server.tools[test.name]
			if !ok {
				t.Fatalf("tool %q is not registered", test.name)
			}
			if tool.Handler == nil {
				t.Fatalf("tool %q has no handler", test.name)
			}
			if len(tool.InputSchema.Required) != len(test.required) {
				t.Fatalf("tool %q required fields = %v, want %v", test.name, tool.InputSchema.Required, test.required)
			}
			for i, field := range test.required {
				if tool.InputSchema.Required[i] != field {
					t.Fatalf("tool %q required fields = %v, want %v", test.name, tool.InputSchema.Required, test.required)
				}
				if _, ok := tool.InputSchema.Properties[field]; !ok {
					t.Fatalf("tool %q is missing schema property %q", test.name, field)
				}
			}
		})
	}
}
