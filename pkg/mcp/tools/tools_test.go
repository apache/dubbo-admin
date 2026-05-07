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

package tools

import (
	"errors"
	"testing"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
)

// TestArgsHelper 测试参数辅助器
func TestArgsHelper(t *testing.T) {
	t.Run("GetString", func(t *testing.T) {
		args := map[string]any{
			"name":  "test",
			"empty": "",
		}
		helper := NewArgsHelper(args)

		if v := helper.GetString("name", "default"); v != "test" {
			t.Errorf("Expected 'test', got '%s'", v)
		}

		if v := helper.GetString("empty", "default"); v != "" {
			t.Errorf("Expected '', got '%s'", v)
		}

		if v := helper.GetString("notexist", "default"); v != "default" {
			t.Errorf("Expected 'default', got '%s'", v)
		}
	})

	t.Run("GetInt", func(t *testing.T) {
		args := map[string]any{
			"intVal":   42,
			"floatVal": 3.14,
		}
		helper := NewArgsHelper(args)

		if v := helper.GetInt("intVal", 0); v != 42 {
			t.Errorf("Expected 42, got %d", v)
		}

		if v := helper.GetInt("floatVal", 0); v != 3 {
			t.Errorf("Expected 3, got %d", v)
		}

		if v := helper.GetInt("notexist", 10); v != 10 {
			t.Errorf("Expected 10, got %d", v)
		}
	})

	t.Run("GetBool", func(t *testing.T) {
		args := map[string]any{
			"true":  true,
			"false": false,
		}
		helper := NewArgsHelper(args)

		if v := helper.GetBool("true", false); !v {
			t.Error("Expected true")
		}

		if v := helper.GetBool("false", true); v {
			t.Error("Expected false")
		}

		if v := helper.GetBool("notexist", true); !v {
			t.Error("Expected default true")
		}
	})

	t.Run("GetRequiredString", func(t *testing.T) {
		args := map[string]any{
			"valid": "value",
			"empty": "",
		}
		helper := NewArgsHelper(args)

		if v, ok := helper.GetRequiredString("valid"); !ok || v != "value" {
			t.Errorf("Expected 'value', got '%s', ok=%v", v, ok)
		}

		if v, ok := helper.GetRequiredString("empty"); ok || v != "" {
			t.Errorf("Expected empty and false, got '%s', ok=%v", v, ok)
		}

		if v, ok := helper.GetRequiredString("notexist"); ok || v != "" {
			t.Errorf("Expected empty and false, got '%s', ok=%v", v, ok)
		}
	})
}

// TestBuildPageReq 测试分页请求构建
func TestBuildPageReq(t *testing.T) {
	tests := []struct {
		name       string
		pageNumber int
		pageSize   int
		wantOffset int
		wantSize   int
	}{
		{"正常分页", 2, 10, 10, 10},
		{"第一页", 1, 20, 0, 20},
		{"无效页码", 0, 10, 0, 10},
		{"无效大小", 1, 0, 0, 10},
		{"负数页码", -1, 10, 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildPageReq(tt.pageNumber, tt.pageSize)
			if req.PageOffset != tt.wantOffset {
				t.Errorf("Expected offset %d, got %d", tt.wantOffset, req.PageOffset)
			}
			if req.PageSize != tt.wantSize {
				t.Errorf("Expected size %d, got %d", tt.wantSize, req.PageSize)
			}
		})
	}
}

// TestJsonResult 测试 JSON 结果创建
func TestJsonResult(t *testing.T) {
	data := map[string]any{
		"key":   "value",
		"count": 42,
	}

	result, err := JsonResult(data)
	if err != nil {
		t.Fatalf("JsonResult failed: %v", err)
	}

	if result.IsError {
		t.Error("Expected non-error result")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	// 验证包含 expected 内容
	text := result.Content[0].Text
	if text == "" {
		t.Error("Expected non-empty content text")
	}
}

// TestErrorResult 测试错误结果创建
func TestErrorResult(t *testing.T) {
	err := errors.New("test error")
	result := ErrorResult(err)

	if !result.IsError {
		t.Error("Expected error result")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Text != "test error" {
		t.Errorf("Expected 'test error', got '%s'", result.Content[0].Text)
	}
}

// TestToolValidation 测试工具参数验证
func TestToolValidation(t *testing.T) {
	// 创建测试工具
	testTool := types.ToolDef{
		Name:        "test_tool",
		Description: "Test tool for validation",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]types.PropertyDef{
				"name": {
					Type:        "string",
					Description: "Name parameter",
				},
				"age": {
					Type:        "integer",
					Description: "Age parameter",
				},
			},
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("ok", false), nil
		},
	}

	t.Run("ValidArguments", func(t *testing.T) {
		args := map[string]any{
			"name": "test",
			"age":  25,
		}

		err := types.ValidateRequired(testTool.InputSchema, args)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("MissingRequired", func(t *testing.T) {
		args := map[string]any{
			"age": 25,
		}

		err := types.ValidateRequired(testTool.InputSchema, args)
		if err == nil {
			t.Error("Expected validation error")
		}
	})

	t.Run("EmptyString", func(t *testing.T) {
		args := map[string]any{
			"name": "",
		}

		err := types.ValidateRequired(testTool.InputSchema, args)
		if err == nil {
			t.Error("Expected validation error for empty string")
		}
	})
}

// TestMetricsRegistrar 测试集群信息工具注册
func TestMetricsRegistrar(t *testing.T) {
	registrar := &MetricsRegistrar{}
	reg := registry.NewRegistry()

	registrar.RegisterTools(reg)

	// 验证工具已注册
	tool, ok := reg.Get("get_cluster_info")
	if !ok {
		t.Fatal("Tool 'get_cluster_info' not registered")
	}

	if tool.Name != "get_cluster_info" {
		t.Errorf("Expected name 'get_cluster_info', got '%s'", tool.Name)
	}

	if len(tool.InputSchema.Required) != 0 {
		t.Errorf("Expected no required params, got %v", tool.InputSchema.Required)
	}
}

// TestResourceSearchRegistrar 测试搜索工具注册
func TestResourceSearchRegistrar(t *testing.T) {
	registrar := &ResourceSearchRegistrar{}
	reg := registry.NewRegistry()

	registrar.RegisterTools(reg)

	// 验证工具已注册
	tool, ok := reg.Get("global_search")
	if !ok {
		t.Fatal("Tool 'global_search' not registered")
	}

	// 验证必需参数（keyword 现在是可选的）
	if len(tool.InputSchema.Required) != 0 {
		t.Errorf("Expected required=[], got %v", tool.InputSchema.Required)
	}

	// 验证所有属性定义
	expectedProps := []string{"keyword", "searchType", "mesh", "pageSize", "pageNumber"}
	for _, prop := range expectedProps {
		if _, ok := tool.InputSchema.Properties[prop]; !ok {
			t.Errorf("Missing property: %s", prop)
		}
	}
}

// TestServiceRegistrar 测试服务发现工具注册
func TestServiceRegistrar(t *testing.T) {
	registrar := &ServiceRegistrar{}
	reg := registry.NewRegistry()

	registrar.RegisterTools(reg)

	// 验证工具已注册
	tools := reg.List()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	expectedTools := []string{"search_services", "get_service_detail"}
	for _, name := range expectedTools {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("Tool '%s' not registered", name)
		}
	}
}

// TestRegistryList 测试注册表列表功能
func TestRegistryList(t *testing.T) {
	reg := registry.NewRegistry()

	// 注册一个测试工具
	reg.Register(types.ToolDef{
		Name:        "test1",
		Description: "Test tool 1",
		InputSchema: types.InputSchema{Type: "object"},
		Handler:     func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) { return nil, nil },
	})

	reg.Register(types.ToolDef{
		Name:        "test2",
		Description: "Test tool 2",
		InputSchema: types.InputSchema{Type: "object"},
		Handler:     func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) { return nil, nil },
	})

	tools := reg.List()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

// TestRegistryUnregister 测试工具注销
func TestRegistryUnregister(t *testing.T) {
	reg := registry.NewRegistry()

	reg.Register(types.ToolDef{
		Name:        "test",
		Description: "Test tool",
		InputSchema: types.InputSchema{Type: "object"},
		Handler:     func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) { return nil, nil },
	})

	if _, ok := reg.Get("test"); !ok {
		t.Error("Tool not registered")
	}

	reg.Unregister("test")

	if _, ok := reg.Get("test"); ok {
		t.Error("Tool still exists after unregister")
	}
}

// TestDefaultPageValues 测试默认分页值
func TestDefaultPageValues(t *testing.T) {
	if DefaultPageSize != 10 {
		t.Errorf("Expected DefaultPageSize=10, got %d", DefaultPageSize)
	}

	if DefaultPageNumber != 1 {
		t.Errorf("Expected DefaultPageNumber=1, got %d", DefaultPageNumber)
	}
}

// TestToolHandlerExecution 测试工具处理器执行
func TestToolHandlerExecution(t *testing.T) {
	// 创建简单的测试工具
	var called bool
	var receivedArgs map[string]any

	testTool := types.ToolDef{
		Name:        "echo",
		Description: "Echo tool",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{"message"},
			Properties: map[string]types.PropertyDef{
				"message": {Type: "string", Description: "Message"},
			},
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			called = true
			receivedArgs = args
			msg := args["message"].(string)
			return types.NewTextResult("echo: "+msg, false), nil
		},
	}

	t.Run("成功调用", func(t *testing.T) {
		called = false
		receivedArgs = nil

		args := map[string]any{
			"message": "hello",
		}

		result, err := testTool.Handler(nil, args)
		if err != nil {
			t.Fatalf("Handler failed: %v", err)
		}

		if !called {
			t.Error("Handler was not called")
		}

		if msg, ok := receivedArgs["message"].(string); !ok || msg != "hello" {
			t.Errorf("Handler did not receive correct args, got %+v", receivedArgs)
		}

		if result.IsError {
			t.Error("Expected non-error result")
		}

		if result.Content[0].Text != "echo: hello" {
			t.Errorf("Expected 'echo: hello', got '%s'", result.Content[0].Text)
		}
	})
}

// TestSearchType 测试搜索类型
func TestSearchType(t *testing.T) {
	types := map[SearchType]string{
		SearchTypeIP:           "ip",
		SearchTypeInstanceName: "instanceName",
		SearchTypeAppName:      "appName",
		SearchTypeName:         "serviceName",
	}

	for k, v := range types {
		if string(k) != v {
			t.Errorf("Expected '%s', got '%s'", v, string(k))
		}
	}
}

// TestServiceSide 测试服务端类型
func TestServiceSide(t *testing.T) {
	types := map[ServiceSide]string{
		ServiceSideProvider: "provider",
		ServiceSideConsumer: "consumer",
	}

	for k, v := range types {
		if string(k) != v {
			t.Errorf("Expected '%s', got '%s'", v, string(k))
		}
	}
}

// TestIsEmpty 测试空值判断
func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"空字符串", "", true},
		{"非空字符串", "hello", false},
		{"nil", nil, true},
		{"空数组", []any{}, true},
		{"非空数组", []any{1, 2}, false},
		{"空map", map[string]any{}, true},
		{"非空map", map[string]any{"key": "value"}, false},
		{"数字", 42, false},
		{"布尔", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := types.IsEmpty(tt.value)
			if result != tt.expected {
				t.Errorf("IsEmpty(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}
