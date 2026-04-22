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

package registry

import (
	"testing"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
)

// mockRegistrar 模拟注册器
type mockRegistrar struct {
	registered bool
}

func (m *mockRegistrar) RegisterTools(reg *Registry) {
	m.registered = true
	reg.Register(types.ToolDef{
		Name: "mock_tool",
		InputSchema: types.InputSchema{
			Type: "object",
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("mock", false), nil
		},
	})
}

func TestRegistry_RegisterAll(t *testing.T) {
	reg := NewRegistry()

	mock := &mockRegistrar{}
	reg.RegisterRegistrar(mock)

	reg.RegisterAll()

	if !mock.registered {
		t.Error("expected registrar to be called")
	}

	if !reg.Has("mock_tool") {
		t.Error("expected tool to be registered")
	}
}

func TestRegistry_RegistrarsCount(t *testing.T) {
	reg := NewRegistry()

	if reg.RegistrarsCount() != 0 {
		t.Errorf("expected 0 registrars, got %d", reg.RegistrarsCount())
	}

	reg.RegisterRegistrar(&mockRegistrar{})
	reg.RegisterRegistrar(&mockRegistrar{})

	if reg.RegistrarsCount() != 2 {
		t.Errorf("expected 2 registrars, got %d", reg.RegistrarsCount())
	}
}

func TestRegistry_ClearRegistrars(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterRegistrar(&mockRegistrar{})
	reg.RegisterRegistrar(&mockRegistrar{})

	reg.ClearRegistrars()

	if reg.RegistrarsCount() != 0 {
		t.Errorf("expected 0 registrars after clear, got %d", reg.RegistrarsCount())
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()

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

	// 测试 Get
	retrieved, ok := reg.Get("test_tool")
	if !ok {
		t.Error("expected tool to be found")
	}

	if retrieved.Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%s'", retrieved.Name)
	}

	// 测试获取不存在的工具
	_, ok = reg.Get("non_existent")
	if ok {
		t.Error("expected non-existent tool to not be found")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()

	reg.Register(types.ToolDef{
		Name: "tool1",
		InputSchema: types.InputSchema{
			Type: "object",
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("test", false), nil
		},
	})

	reg.Register(types.ToolDef{
		Name: "tool2",
		InputSchema: types.InputSchema{
			Type: "object",
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("test", false), nil
		},
	})

	tools := reg.List()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestRegistry_RegisterEmptyName(t *testing.T) {
	reg := NewRegistry()

	err := reg.Register(types.ToolDef{
		Name: "",
		InputSchema: types.InputSchema{
			Type: "object",
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("test", false), nil
		},
	})

	if err == nil {
		t.Error("expected error when registering tool with empty name")
	}
}

