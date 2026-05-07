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

package stdio

import (
	"encoding/json"
	"testing"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/core"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
)

func TestTransport_ServeOnce_Initialize(t *testing.T) {
	server := core.NewServer("test-server", "1.0.0")
	transport := NewTransport(server)

	// 测试 initialize 请求
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	}
	reqJSON, _ := json.Marshal(req)

	resp, err := transport.ServeOnce(string(reqJSON) + "\n")
	if err != nil {
		t.Fatalf("ServeOnce failed: %v", err)
	}

	var jsonResp core.JSONRPCResponse
	if err := json.Unmarshal([]byte(resp), &jsonResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if jsonResp.Error != nil {
		t.Fatalf("Response error: %s", jsonResp.Error.Message)
	}

	result, ok := jsonResp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("serverInfo not found")
	}

	if serverInfo["name"] != "test-server" {
		t.Errorf("Expected name 'test-server', got '%v'", serverInfo["name"])
	}
}

func TestTransport_ServeOnce_ToolsList(t *testing.T) {
	server := core.NewServer("test-server", "1.0.0")
	transport := NewTransport(server)

	// 注册一个测试工具
	reg := server.GetRegistry()
	reg.Register(types.ToolDef{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: types.InputSchema{
			Type: "object",
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("ok", false), nil
		},
	})

	// 测试 tools/list 请求
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}
	reqJSON, _ := json.Marshal(req)

	resp, err := transport.ServeOnce(string(reqJSON) + "\n")
	if err != nil {
		t.Fatalf("ServeOnce failed: %v", err)
	}

	var jsonResp core.JSONRPCResponse
	if err := json.Unmarshal([]byte(resp), &jsonResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if jsonResp.Error != nil {
		t.Fatalf("Response error: %s", jsonResp.Error.Message)
	}

	result, ok := jsonResp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatal("tools not found")
	}

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
}

func TestTransport_ServeOnce_ParseError(t *testing.T) {
	server := core.NewServer("test-server", "1.0.0")
	transport := NewTransport(server)

	// 测试无效 JSON - ServeOnce 在解析失败时返回错误
	_, err := transport.ServeOnce("invalid json\n")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}

	// 测试无效方法名 - 这会返回 JSON 错误响应
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "invalid_method",
	}
	reqJSON, _ := json.Marshal(req)

	resp, err := transport.ServeOnce(string(reqJSON) + "\n")
	if err != nil {
		t.Fatalf("ServeOnce failed: %v", err)
	}

	var jsonResp core.JSONRPCResponse
	if err := json.Unmarshal([]byte(resp), &jsonResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if jsonResp.Error == nil {
		t.Fatal("Expected error response")
	}

	if jsonResp.Error.Code != core.ErrCodeMethodNotFound {
		t.Errorf("Expected method not found code, got %d", jsonResp.Error.Code)
	}
}

func TestTransport_Close(t *testing.T) {
	server := core.NewServer("test-server", "1.0.0")
	transport := NewTransport(server)

	if err := transport.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 再次关闭应该成功
	if err := transport.Close(); err != nil {
		t.Fatalf("Second Close failed: %v", err)
	}
}
