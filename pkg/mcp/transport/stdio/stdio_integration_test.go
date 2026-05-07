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
	"bufio"
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/core"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
)

// TestTransport_Integration_EndToEnd 端到端集成测试
func TestTransport_Integration_EndToEnd(t *testing.T) {
	// 创建服务器并注册工具
	server := core.NewServer("test-server", "1.0.0")
	reg := server.GetRegistry()

	// 注册一个测试工具
	reg.Register(types.ToolDef{
		Name:        "echo",
		Description: "Echo back the input message",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{"message"},
			Properties: map[string]types.PropertyDef{
				"message": {
					Type:        "string",
					Description: "Message to echo",
				},
			},
		},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			msg, _ := args["message"].(string)
			return types.NewTextResult("echo: "+msg, false), nil
		},
	})

	// 创建管道对
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()
	defer func() {
		clientW.Close()
		serverW.Close()
	}()

	// 创建使用管道的 transport
	transport := NewTransportWithIO(server, serverR, serverW)

	// 启动服务器（在后台）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- transport.Serve(ctx)
	}()

	// 给服务器一点时间启动
	time.Sleep(100 * time.Millisecond)

	// 客户端发送请求
	client := newMCPClient(clientR, clientW)

	// 1. 测试 initialize
	t.Run("Initialize", func(t *testing.T) {
		resp := client.call(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]any{},
		})

		if resp.Error != nil {
			t.Fatalf("Initialize failed: %s", resp.Error.Message)
		}

		result := resp.Result.(map[string]any)
		serverInfo := result["serverInfo"].(map[string]any)
		if serverInfo["name"] != "test-server" {
			t.Errorf("Expected name 'test-server', got '%v'", serverInfo["name"])
		}
	})

	// 2. 测试 tools/list
	t.Run("ToolsList", func(t *testing.T) {
		resp := client.call(map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		})

		if resp.Error != nil {
			t.Fatalf("Tools list failed: %s", resp.Error.Message)
		}

		result := resp.Result.(map[string]any)
		tools := result["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("Expected 1 tool, got %d", len(tools))
		}

		tool := tools[0].(map[string]any)
		if tool["name"] != "echo" {
			t.Errorf("Expected tool name 'echo', got '%v'", tool["name"])
		}
	})

	// 3. 测试 tools/call - 成功调用
	t.Run("ToolCall_Success", func(t *testing.T) {
		resp := client.call(map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "echo",
				"arguments": map[string]any{
					"message": "hello world",
				},
			},
		})

		if resp.Error != nil {
			t.Fatalf("Tool call failed: %s", resp.Error.Message)
		}

		result := resp.Result.(map[string]any)
		content := result["content"].([]any)
		if len(content) != 1 {
			t.Fatalf("Expected 1 content item, got %d", len(content))
		}

		firstContent := content[0].(map[string]any)
		if firstContent["text"] != "echo: hello world" {
			t.Errorf("Expected 'echo: hello world', got '%v'", firstContent["text"])
		}
	})

	// 4. 测试 tools/call - 缺少必需参数
	t.Run("ToolCall_MissingRequired", func(t *testing.T) {
		resp := client.call(map[string]any{
			"jsonrpc": "2.0",
			"id":      4,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "echo",
				"arguments": map[string]any{},
			},
		})

		if resp.Error == nil {
			t.Fatal("Expected error for missing required parameter")
		}

		if resp.Error.Code != core.ErrCodeInvalidParams {
			t.Errorf("Expected invalid params code, got %d", resp.Error.Code)
		}
	})

	// 5. 测试 tools/call - 工具不存在
	t.Run("ToolCall_NotFound", func(t *testing.T) {
		resp := client.call(map[string]any{
			"jsonrpc": "2.0",
			"id":      5,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "nonexistent",
				"arguments": map[string]any{},
			},
		})

		if resp.Error == nil {
			t.Fatal("Expected error for nonexistent tool")
		}

		if resp.Error.Code != core.ErrCodeMethodNotFound {
			t.Errorf("Expected method not found code, got %d", resp.Error.Code)
		}
	})

	// 取消服务器上下文并关闭管道
	cancel()
	clientW.Close() // 关闭客户端写入端，让服务器 ReadString 返回 EOF

	// 等待服务器退出
	select {
	case err := <-serverErr:
		if err != nil && err != context.Canceled {
			t.Logf("Server exited with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Server did not exit in time")
	}
}

// mcpClient 简单的 MCP 客户端
type mcpClient struct {
	r      *bufio.Reader
	w      io.Writer
	wMutex sync.Mutex
	reqID  int
}

func newMCPClient(r io.Reader, w io.WriteCloser) *mcpClient {
	return &mcpClient{
		r:     bufio.NewReader(r),
		w:     w,
		reqID: 0,
	}
}

func (c *mcpClient) call(req map[string]any) *jsonRPCResponse {
	c.wMutex.Lock()
	defer c.wMutex.Unlock()

	c.reqID++
	req["id"] = c.reqID

	// 发送请求
	reqData, _ := json.Marshal(req)
	c.w.Write(reqData)
	c.w.Write([]byte("\n"))

	// 读取响应
	line, err := c.r.ReadString('\n')
	if err != nil {
		return &jsonRPCResponse{
			Error: &core.JSONRPCError{
				Code:    -1,
				Message: err.Error(),
			},
		}
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return &jsonRPCResponse{
			Error: &core.JSONRPCError{
				Code:    -1,
				Message: "Failed to parse response: " + err.Error(),
			},
		}
	}

	return &resp
}

type jsonRPCResponse struct {
	JSONRPC string             `json:"jsonrpc"`
	ID      any                `json:"id"`
	Result  any                `json:"result,omitempty"`
	Error   *core.JSONRPCError `json:"error,omitempty"`
}

// TestTransport_RealIO 使用真实 io 操作的测试
func TestTransport_RealIO(t *testing.T) {
	server := core.NewServer("test-server", "1.0.0")
	reg := server.GetRegistry()

	reg.Register(types.ToolDef{
		Name:        "ping",
		Description: "Ping tool",
		InputSchema: types.InputSchema{Type: "object"},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("pong", false), nil
		},
	})

	// 创建管道对
	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	transport := NewTransportWithIO(server, serverR, serverW)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 启动服务器
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- transport.Serve(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// 客户端发送多个请求
	client := newMCPClient(clientR, clientW)

	responses := make([]*jsonRPCResponse, 0, 3)
	for i := 0; i < 3; i++ {
		resp := client.call(map[string]any{
			"jsonrpc": "2.0",
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "ping",
				"arguments": map[string]any{},
			},
		})
		responses = append(responses, resp)
	}

	// 验证所有响应
	for i, resp := range responses {
		if resp.Error != nil {
			t.Errorf("Request %d failed: %s", i, resp.Error.Message)
		}
	}

	cancel()
	<-serverErr
}

// TestTransport_ConcurrentRequests 并发请求测试
func TestTransport_ConcurrentRequests(t *testing.T) {
	server := core.NewServer("test-server", "1.0.0")
	reg := server.GetRegistry()

	reg.Register(types.ToolDef{
		Name:        "counter",
		Description: "Counting tool",
		InputSchema: types.InputSchema{Type: "object"},
		Handler: func(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
			return types.NewTextResult("count", false), nil
		},
	})

	serverR, clientW := io.Pipe()
	clientR, serverW := io.Pipe()

	transport := NewTransportWithIO(server, serverR, serverW)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- transport.Serve(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// 并发发送多个请求
	// 使用共享客户端（内部有锁保护写操作）
	client := newMCPClient(clientR, clientW)
	const numRequests = 10
	results := make(chan *jsonRPCResponse, numRequests)
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := client.call(map[string]any{
				"jsonrpc": "2.0",
				"method":  "tools/call",
				"params": map[string]any{
					"name":      "counter",
					"arguments": map[string]any{},
				},
			})
			results <- resp
		}()
	}

	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	successCount := 0
	for resp := range results {
		if resp.Error == nil {
			successCount++
		}
	}

	if successCount != numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, successCount)
	}

	cancel()
	clientW.Close()
	<-serverErr
}
