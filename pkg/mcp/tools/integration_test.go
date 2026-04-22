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
	"encoding/json"
	"testing"

	"github.com/apache/dubbo-admin/pkg/mcp/core"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
	"github.com/apache/dubbo-admin/pkg/mcp/transport/stdio"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	ctx "context"
)

// TestTools_E2E 通过 MCP Server 端到端测试工具
// 这展示了如何在实际场景中测试 MCP 工具
func TestTools_E2E(t *testing.T) {
	// 创建 MCP 服务器
	server := core.NewServer("dubbo-admin-test", "1.0.0")

	// 注册所有默认工具
	reg := server.GetRegistry()
	reg.RegisterRegistrar(&MetricsRegistrar{})
	reg.RegisterRegistrar(&ResourceSearchRegistrar{})
	reg.RegisterRegistrar(&ServiceRegistrar{})
	reg.RegisterAll()

	// 验证工具已注册
	tools := reg.List()
	t.Logf("Registered %d tools:", len(tools))
	for _, tool := range tools {
		t.Logf("  - %s: %s", tool.Name, tool.Description)
	}

	// 测试获取工具列表
	t.Run("ToolsList", func(t *testing.T) {
		req := &core.JSONRPCRequest{
			JSONRPC: core.JSONRPCVersion,
			ID:      1,
			Method:  core.MethodToolsList,
		}

		resp := server.HandleRequest(req)
		if resp.Error != nil {
			t.Fatalf("Tools list failed: %s", resp.Error.Message)
		}

		// 将响应序列化为 JSON 以验证格式
		respData, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal response: %v", err)
		}
		t.Logf("Tools list response:\n%s", string(respData))

		// Result 是 ToolListResult，通过 JSON 反序列化来验证
		var result map[string]any
		err = json.Unmarshal(respData, &result)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		tools := result["result"].(map[string]any)["tools"].([]any)

		// 验证核心工具存在
		toolNames := make(map[string]bool)
		for _, t := range tools {
			tool := t.(map[string]any)
			toolNames[tool["name"].(string)] = true
		}

		expectedTools := []string{
			"get_cluster_info",
			"global_search",
			"search_services",
			"get_service_detail",
		}

		for _, name := range expectedTools {
			if !toolNames[name] {
				t.Errorf("Expected tool '%s' not found", name)
			}
		}

		t.Logf("Found %d tools in tools/list response", len(tools))
	})

	// 测试工具定义可以正确转换为 JSON
	t.Run("ToolSchemaSerialization", func(t *testing.T) {
		tools := reg.List()

		for _, tool := range tools {
			// 创建不包含 Handler 的工具定义用于序列化
			toolForMarshal := struct {
				Name        string             `json:"name"`
				Description string             `json:"description"`
				InputSchema types.InputSchema `json:"inputSchema"`
			}{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			}

			// 尝试序列化为 JSON
			data, err := json.Marshal(toolForMarshal)
			if err != nil {
				t.Errorf("Failed to marshal tool %s: %v", tool.Name, err)
				continue
			}

			// 验证可以反序列化
			var unmarshaled map[string]any
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal tool %s: %v", tool.Name, err)
			}
		}
	})

	// 测试工具 schema 验证
	t.Run("ToolSchemaValidation", func(t *testing.T) {
		tools := reg.List()

		for _, tool := range tools {
			t.Run(tool.Name, func(t *testing.T) {
				// 验证必需参数可以正确检测
				if len(tool.InputSchema.Required) > 0 {
					// 缺少必需参数
					invalidArgs := map[string]any{}

					err := core.ValidateRequired(tool.InputSchema, invalidArgs)
					if err == nil {
						t.Errorf("Tool %s: expected validation error for missing required params", tool.Name)
					}
				}

				// 空参数应该通过（如果无必需参数）
				if len(tool.InputSchema.Required) == 0 {
					validArgs := map[string]any{}
					err := core.ValidateRequired(tool.InputSchema, validArgs)
					if err != nil {
						t.Errorf("Tool %s: unexpected validation error: %v", tool.Name, err)
					}
				}
			})
		}
	})
}

// TestTools_Manual 手动测试示例
// 这展示了如何在需要时手动调用工具 handler
func TestTools_Manual(t *testing.T) {
	t.Skip("跳过手动测试示例 - 需要真实的服务依赖")

	// 示例：如何直接调用 GetClusterInfo
	// 注意：这需要真实的 CounterManager
	/*
		ctx := setupRealContext()
		args := map[string]any{
			"mesh": "test-mesh",
		}

		result, err := GetClusterInfo(ctx, args)
		if err != nil {
			t.Fatalf("GetClusterInfo failed: %v", err)
		}

		t.Logf("Result: %s", result.Content[0].Text)
	*/
}

// MockContext 用于测试的 mock context
type MockContext struct {
	meshName string
	config   *appcfg.AdminConfig
}

func NewMockContext(meshName string) *MockContext {
	return &MockContext{
		meshName: meshName,
		config: &appcfg.AdminConfig{
			// 设置必要的配置
		},
	}
}

func (m *MockContext) Config() appcfg.AdminConfig {
	if m.config == nil {
		m.config = &appcfg.AdminConfig{}
	}
	return *m.config
}

func (m *MockContext) CounterManager() interface{} {
	// 返回 mock CounterManager
	return nil
}

func (m *MockContext) AppContext() ctx.Context {
	return ctx.Background()
}

// TestGetClusterInfoSchema 测试 get_cluster_info 的 schema
func TestGetClusterInfoSchema(t *testing.T) {
	reg := registry.NewRegistry()
	registrar := &MetricsRegistrar{}
	registrar.RegisterTools(reg)

	tool, ok := reg.Get("get_cluster_info")
	if !ok {
		t.Fatal("Tool get_cluster_info not registered")
	}

	// 验证 schema
	t.Run("Schema", func(t *testing.T) {
		if tool.Name != "get_cluster_info" {
			t.Errorf("Expected name 'get_cluster_info', got '%s'", tool.Name)
		}

		if tool.InputSchema.Type != "object" {
			t.Errorf("Expected type 'object', got '%s'", tool.InputSchema.Type)
		}

		// 验证可选参数
		if _, ok := tool.InputSchema.Properties["mesh"]; !ok {
			t.Error("Missing 'mesh' property")
		}
	})

	t.Run("Arguments", func(t *testing.T) {
		// 无参数调用（使用默认 mesh）
		args := map[string]any{}
		err := core.ValidateRequired(tool.InputSchema, args)
		if err != nil {
			t.Errorf("Validation failed for empty args: %v", err)
		}

		// 指定 mesh 参数
		args = map[string]any{"mesh": "custom-mesh"}
		err = core.ValidateRequired(tool.InputSchema, args)
		if err != nil {
			t.Errorf("Validation failed for mesh arg: %v", err)
		}
	})
}

// TestGlobalSearchSchema 测试 global_search 的 schema
func TestGlobalSearchSchema(t *testing.T) {
	reg := registry.NewRegistry()
	registrar := &ResourceSearchRegistrar{}
	registrar.RegisterTools(reg)

	tool, ok := reg.Get("global_search")
	if !ok {
		t.Fatal("Tool global_search not registered")
	}

	t.Run("RequiredParameters", func(t *testing.T) {
		// keyword 现在是可选的，空关键字返回所有数据
		if len(tool.InputSchema.Required) != 0 {
			t.Errorf("Expected 0 required parameters (keyword is optional), got %d", len(tool.InputSchema.Required))
		}
	})

	t.Run("Validation", func(t *testing.T) {
		// 所有参数都是可选的，不应有验证错误
		args := map[string]any{}
		err := core.ValidateRequired(tool.InputSchema, args)
		if err != nil {
			t.Errorf("Expected no validation error for empty args: %v", err)
		}

		// 空字符串也是有效的（返回所有数据）
		args = map[string]any{"keyword": ""}
		err = core.ValidateRequired(tool.InputSchema, args)
		if err != nil {
			t.Errorf("Expected no validation error for empty keyword: %v", err)
		}

		// 有效参数
		args = map[string]any{"keyword": "test-service"}
		err = core.ValidateRequired(tool.InputSchema, args)
		if err != nil {
			t.Errorf("Validation failed for valid args: %v", err)
		}
	})

	t.Run("OptionalParameters", func(t *testing.T) {
		// 验证所有可选参数存在
		expectedProps := []string{"keyword", "searchType", "mesh", "pageSize", "pageNumber"}
		for _, prop := range expectedProps {
			if _, ok := tool.InputSchema.Properties[prop]; !ok {
				t.Errorf("Missing property '%s'", prop)
			}
		}
	})
}

// TestServiceToolsSchema 测试服务工具的 schema
func TestServiceToolsSchema(t *testing.T) {
	reg := registry.NewRegistry()
	registrar := &ServiceRegistrar{}
	registrar.RegisterTools(reg)

	t.Run("SearchServices", func(t *testing.T) {
		tool, ok := reg.Get("search_services")
		if !ok {
			t.Fatal("Tool search_services not registered")
		}

		// 无必需参数
		if len(tool.InputSchema.Required) != 0 {
			t.Errorf("Expected no required parameters, got %v", tool.InputSchema.Required)
		}

		// 测试验证
		args := map[string]any{}
		err := core.ValidateRequired(tool.InputSchema, args)
		if err != nil {
			t.Errorf("Validation failed unexpectedly: %v", err)
		}
	})

	t.Run("GetServiceDetail", func(t *testing.T) {
		tool, ok := reg.Get("get_service_detail")
		if !ok {
			t.Fatal("Tool get_service_detail not registered")
		}

		// serviceName 是必需的
		if len(tool.InputSchema.Required) != 1 {
			t.Errorf("Expected 1 required parameter, got %d", len(tool.InputSchema.Required))
		}

		// 测试验证
		args := map[string]any{}
		err := core.ValidateRequired(tool.InputSchema, args)
		if err == nil {
			t.Error("Expected validation error for missing 'serviceName'")
		}

		args = map[string]any{"serviceName": "com.example.Service"}
		err = core.ValidateRequired(tool.InputSchema, args)
		if err != nil {
			t.Errorf("Validation failed for valid args: %v", err)
		}

		// 测试 side 参数枚举
		sideProp := tool.InputSchema.Properties["side"]
		actualEnum := sideProp.Enum
		if len(actualEnum) != 2 {
			t.Errorf("Expected enum with 2 values, got %d", len(actualEnum))
		}
	})
}

// Example_testUsage 演示如何通过 stdio transport 使用工具
func Example_testUsage() {
	// 创建服务器
	server := core.NewServer("dubbo-admin", "1.0.0")
	reg := server.GetRegistry()

	// 注册工具
	reg.RegisterRegistrar(&MetricsRegistrar{})
	reg.RegisterRegistrar(&ResourceSearchRegistrar{})
	reg.RegisterRegistrar(&ServiceRegistrar{})
	reg.RegisterAll()

	// 创建 stdio transport
	transport := stdio.NewTransport(server)

	// 在实际使用中，你会这样启动：
	// transport.Serve(context.Background())

	// 或者使用自定义 io 进行测试
	// transport := stdio.NewTransportWithIO(server, stdin, stdout)

	_ = transport
}

// TestDefaultRegistry 测试默认注册表
func TestDefaultRegistry(t *testing.T) {
	// 使用 DefaultRegistry 创建包含所有工具的注册表
	reg := registry.NewRegistry()

	// 注册所有默认工具
	reg.RegisterRegistrar(&MetricsRegistrar{})
	reg.RegisterRegistrar(&ResourceSearchRegistrar{})
	reg.RegisterRegistrar(&ServiceRegistrar{})
	reg.RegisterAll()

	tools := reg.List()
	t.Logf("DefaultRegistry has %d tools", len(tools))

	// 验证核心工具存在
	expectedTools := []string{
		"get_cluster_info",
		"global_search",
		"search_services",
		"get_service_detail",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool '%s' not found in default registry", name)
		}
	}
}
