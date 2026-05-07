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

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/apache/dubbo-admin/pkg/mcp/core"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
)

// TestHTTPTransport 测试HTTP传输
func TestHTTPTransport(t *testing.T) {
	// 创建测试服务器
	server := core.NewServer("test-server", "1.0.0")

	// 注册测试工具
	reg := server.GetRegistry()
	reg.Register(core.ToolDef{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: core.InputSchema{
			Type: "object",
			Properties: map[string]core.PropertyDef{
				"message": {
					Type:        "string",
					Description: "Test message",
				},
			},
		},
		Handler: func(ctx interface{}, args map[string]any) (*core.ToolResult, error) {
			msg, _ := args["message"].(string)
			return core.NewTextResult("Echo: " + msg), nil
		},
	})
	reg.RegisterAll()

	// 创建HTTP传输层
	transport := NewTransportWithConfig(server, &Config{
		Host:          "127.0.0.1",
		Port:          0, // 随机端口
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  5 * time.Second,
	})

	// 异步启动
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := transport.StartAsync(ctx); err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 测试initialize请求
	t.Run("Initialize", func(t *testing.T) {
		req := core.JSONRPCRequest{
			JSONRPC: core.JSONRPCVersion,
			ID:      "1",
			Method:  core.MethodInitialize,
		}

		resp := makeRequest(t, transport, req)
		if resp.Error != nil {
			t.Fatalf("Initialize failed: %v", resp.Error)
		}

		result, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatal("Invalid result type")
		}

		serverInfo := result["serverInfo"].(map[string]any)
		if serverInfo["name"] != "test-server" {
			t.Errorf("Expected server name 'test-server', got '%v'", serverInfo["name"])
		}
	})

	// 测试tools/list请求
	t.Run("ToolsList", func(t *testing.T) {
		req := core.JSONRPCRequest{
			JSONRPC: core.JSONRPCVersion,
			ID:      "2",
			Method:  core.MethodToolsList,
		}

		resp := makeRequest(t, transport, req)
		if resp.Error != nil {
			t.Fatalf("Tools list failed: %v", resp.Error)
		}

		result, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatal("Invalid result type")
		}

		tools := result["tools"].([]any)
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}
	})

	// 测试tools/call请求
	t.Run("ToolsCall", func(t *testing.T) {
		req := core.JSONRPCRequest{
			JSONRPC: core.JSONRPCVersion,
			ID:      "3",
			Method:  core.MethodToolsCall,
			Params: map[string]any{
				"name": "test_tool",
				"arguments": map[string]any{
					"message": "Hello, MCP!",
				},
			},
		}

		resp := makeRequest(t, transport, req)
		if resp.Error != nil {
			t.Fatalf("Tools call failed: %v", resp.Error)
		}

		result, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatal("Invalid result type")
		}

		content := result["content"].([]any)
		if len(content) == 0 {
			t.Fatal("Empty content")
		}

		firstContent := content[0].(map[string]any)
		text := firstContent["text"].(string)
		if text != "Echo: Hello, MCP!" {
			t.Errorf("Expected 'Echo: Hello, MCP!', got '%s'", text)
		}
	})

	// 关闭传输层
	if err := transport.Shutdown(); err != nil {
		t.Fatalf("Failed to shutdown transport: %v", err)
	}
}

// makeRequest 发送请求到HTTP传输层
func makeRequest(t *testing.T, transport *Transport, req core.JSONRPCRequest) core.JSONRPCResponse {
	t.Helper()

	// 使用httptest直接测试handler
	handler := transport.GetHandler()
	body, _ := json.Marshal(req)

	httpReq := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httpReq)

	var resp core.JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return resp
}

// TestHandlerCORS 测试CORS支持
func TestHandlerCORS(t *testing.T) {
	server := core.NewServer("test", "1.0.0")
	handler := NewHandler(server)

	req := httptest.NewRequest("OPTIONS", "/mcp", nil)
	req.Header.Set("Origin", "https://example.com")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	corsHeader := w.Header().Get("Access-Control-Allow-Origin")
	if corsHeader != "*" {
		t.Errorf("Expected CORS origin '*', got '%s'", corsHeader)
	}
}

// TestConcurrentRequests 测试并发请求
func TestConcurrentRequests(t *testing.T) {
	server := core.NewServer("test", "1.0.0")
	transport := NewTransportWithConfig(server, &Config{
		Host: "127.0.0.1",
		Port: 0,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := transport.StartAsync(ctx); err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}
	defer transport.Shutdown()

	time.Sleep(100 * time.Millisecond)

	// 并发发送10个请求
	const numRequests = 10
	errCh := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := core.JSONRPCRequest{
				JSONRPC: core.JSONRPCVersion,
				ID:      fmt.Sprintf("req-%d", id),
				Method:  core.MethodInitialize,
			}
			_ = makeRequest(t, transport, req)
			errCh <- nil
		}(i)
	}

	// 等待所有请求完成
	for i := 0; i < numRequests; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("Request failed: %v", err)
		}
	}
}

// TestSSETransport 测试SSE传输
func TestSSETransport(t *testing.T) {
	server := core.NewServer("test", "1.0.0")
	sseTransport := NewSSETransport(server)

	// 测试SSE连接
	t.Run("SSEConnection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		w := httptest.NewRecorder()

		// SSE需要flusher支持，httptest.NewRecorder实现了Flusher接口
		sseTransport.HandleSSE(w, req)

		// 验证响应headers
		contentType := w.Header().Get("Content-Type")
		if contentType != "text/event-stream" {
			t.Errorf("Expected content type 'text/event-stream', got '%s'", contentType)
		}
	})
}
