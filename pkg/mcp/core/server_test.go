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

package core

import (
	"testing"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
)

func TestServer_NewServer(t *testing.T) {
	server := NewServer("test", "1.0.0")

	if server.name != "test" {
		t.Errorf("expected name 'test', got '%s'", server.name)
	}

	if server.version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", server.version)
	}

	if server.registry == nil {
		t.Error("expected registry to be initialized")
	}
}

func TestServer_NewServerWithRegistry(t *testing.T) {
	reg := registry.NewRegistry()
	server := NewServerWithRegistry("test", "1.0.0", reg)

	if server.registry != reg {
		t.Error("expected server to use provided registry")
	}
}

func TestServer_GetRegistry(t *testing.T) {
	server := NewServer("test", "1.0.0")

	reg := server.GetRegistry()
	if reg == nil {
		t.Error("expected registry to be returned")
	}

	if reg != server.registry {
		t.Error("expected returned registry to be the same as server's registry")
	}
}

func TestRegistry_Register(t *testing.T) {
	reg := registry.NewRegistry()

	tool := types.ToolDef{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: types.InputSchema{
			Type: "object",
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("test", false), nil
		},
	}

	err := reg.Register(tool)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !reg.Has("test_tool") {
		t.Error("expected tool to be registered")
	}

	if reg.Count() != 1 {
		t.Errorf("expected 1 tool, got %d", reg.Count())
	}
}

func TestRegistry_Unregister(t *testing.T) {
	reg := registry.NewRegistry()

	tool := types.ToolDef{
		Name: "test_tool",
		InputSchema: types.InputSchema{
			Type: "object",
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("test", false), nil
		},
	}

	reg.Register(tool)
	removed := reg.Unregister("test_tool")

	if !removed {
		t.Error("expected tool to be removed")
	}

	if reg.Has("test_tool") {
		t.Error("expected tool to be unregistered")
	}

	// 再次移除应该返回 false
	removed = reg.Unregister("test_tool")
	if removed {
		t.Error("expected second removal to return false")
	}
}

func TestRegistry_Clear(t *testing.T) {
	reg := registry.NewRegistry()

	tool := types.ToolDef{
		Name: "test_tool",
		InputSchema: types.InputSchema{
			Type: "object",
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("test", false), nil
		},
	}

	reg.Register(tool)
	reg.Clear()

	if reg.Count() != 0 {
		t.Errorf("expected 0 tools after clear, got %d", reg.Count())
	}
}

func TestServerBuilder(t *testing.T) {
	server := NewServerBuilder().
		WithName("custom-server").
		WithVersion("2.0.0").
		Build()

	if server.name != "custom-server" {
		t.Errorf("expected name 'custom-server', got '%s'", server.name)
	}

	if server.version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got '%s'", server.version)
	}
}
