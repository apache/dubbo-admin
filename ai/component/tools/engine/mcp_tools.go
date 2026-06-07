package engine

import (
	"bytes"
	"context"
	"dubbo-admin-ai/runtime"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// HTTPMCPClient HTTP MCP 客户端
type HTTPMCPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
}

// NewHTTPMCPClient 创建 HTTP MCP 客户端
func NewHTTPMCPClient(host string, port int, apiKey string, maxRetries int) *HTTPMCPClient {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &HTTPMCPClient{
		baseURL:    fmt.Sprintf("http://%s:%d/api/mcp", host, port),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxRetries: maxRetries,
	}
}

// JSONRPCRequest JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError MCP 错误
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolListResult 工具列表结果
type ToolListResult struct {
	Tools []MCPTool `json:"tools"`
}

// MCPTool MCP 工具定义
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// CallToolResult 调用工具结果
type CallToolResult struct {
	Content []interface{} `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ListTools 列出所有工具
func (c *HTTPMCPClient) ListTools(ctx context.Context) ([]MCPTool, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result ToolListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools list: %w", err)
	}

	return result.Tools, nil
}

// isRetriableError 判断错误是否可重试
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// 网络错误、超时错误、连接错误可重试
	retriableErrors := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"EOF",
		"broken pipe",
		"no such host",
		"temporary failure",
		"dial tcp",
		"failed to send request",
	}
	for _, retriable := range retriableErrors {
		if contains(errMsg, retriable) {
			return true
		}
	}
	// HTTP 状态码错误
	if contains(errMsg, "status ") && (contains(errMsg, "503") || contains(errMsg, "502") || contains(errMsg, "504")) {
		return true
	}
	return false
}

// contains 检查字符串是否包含子串（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		containsIgnoreCase(s, substr)))
}

func containsIgnoreCase(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}

// CallTool 调用工具（带重试）
func (c *HTTPMCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*CallToolResult, error) {
	var lastErr error
	baseDelay := 100 * time.Millisecond

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：100ms, 200ms, 400ms, 800ms...
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
			runtime.GetLogger().Info("MCP tool retry",
				"tool", toolName,
				"attempt", attempt,
				"max_retries", c.maxRetries,
				"delay", delay)

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("tool %s unavailable: request cancelled during retry", toolName)
			case <-time.After(delay):
			}
		}

		req := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name":      toolName,
				"arguments": args,
			},
		}

		resp, err := c.sendRequest(ctx, req)
		if err != nil {
			lastErr = err
			if attempt < c.maxRetries && isRetriableError(err) {
				continue
			}
			break
		}

		var result CallToolResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool result: %w", err)
		}

		return &result, nil
	}

	// 所有重试都失败，返回简洁的错误信息给 LLM
	if isRetriableError(lastErr) {
		return nil, fmt.Errorf("tool '%s' is currently unavailable after %d retries", toolName, c.maxRetries)
	}
	return nil, fmt.Errorf("tool '%s' failed: %v", toolName, lastErr)
}

// sendRequest 发送 HTTP 请求
func (c *HTTPMCPClient) sendRequest(ctx context.Context, req JSONRPCRequest) (*JSONRPCResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Add API Key if provided
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned status %d", httpResp.StatusCode)
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
	}

	return &resp, nil
}

// MCPToolManager MCP 工具管理器
type MCPToolManager struct {
	registry       *genkit.Genkit
	client         *HTTPMCPClient
	availableTools map[string]ai.Tool
}

// NewMCPToolManager 创建 MCP 工具管理器
func NewMCPToolManager(rt *runtime.Runtime, host string, port int, apiKey string, maxRetries int) (*MCPToolManager, error) {
	if rt == nil {
		return nil, fmt.Errorf("runtime is nil")
	}
	g := rt.GetGenkitRegistry()
	if g == nil {
		return nil, fmt.Errorf("genkit registry is nil")
	}

	client := NewHTTPMCPClient(host, port, apiKey, maxRetries)

	// 获取工具列表
	tools, err := client.ListTools(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// 注册工具
	availableTools := make(map[string]ai.Tool)
	for _, tool := range tools {
		mcpTool := tool
		handler := func(ctx *ai.ToolContext, input any) (any, error) {
			args, ok := input.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid input type")
			}

			result, err := client.CallTool(ctx.Context, mcpTool.Name, args)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"tool_name": mcpTool.Name,
				"result":    result.Content,
				"summary":   "",
			}, nil
		}

		// 创建工具
		var registered ai.Tool
		if len(tool.InputSchema) > 0 {
			schemaBytes, _ := json.Marshal(tool.InputSchema)
			var schema map[string]interface{}
			json.Unmarshal(schemaBytes, &schema)
			registered = genkit.DefineToolWithInputSchema(g, tool.Name, tool.Description, schema, handler)
		} else {
			registered = genkit.DefineTool(g, tool.Name, tool.Description, handler)
		}
		availableTools[tool.Name] = registered
	}

	return &MCPToolManager{
		registry:       g,
		client:         client,
		availableTools: availableTools,
	}, nil
}

func (mtm *MCPToolManager) AllTools() []ai.Tool {
	var tools []ai.Tool
	for _, tool := range mtm.availableTools {
		tools = append(tools, tool)
	}
	return tools
}

func (mtm *MCPToolManager) ToolRefs() (toolRef []ai.ToolRef) {
	for _, tool := range mtm.availableTools {
		toolRef = append(toolRef, tool)
	}
	return toolRef
}

func (mtm *MCPToolManager) GetToolByName(name string) ai.Tool {
	return mtm.availableTools[name]
}
