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

package common

import (
	"fmt"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
)

// ==================== Constants ====================

const (
	// JSONRPCVersion JSON-RPC 版本
	JSONRPCVersion = "2.0"
	// ProtocolVersion MCP 协议版本
	ProtocolVersion = "2024-11-05"

	// Methods
	MethodInitialize = "initialize"
	MethodToolsList  = "tools/list"
	MethodToolsCall  = "tools/call"

	// Error codes
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603

	// Content types
	ContentTypeText = "text"
)

// ==================== JSON-RPC 2.0 Types ====================

// JSONRPCRequest JSON-RPC 2.0 请求
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// JSONRPCResponse JSON-RPC 2.0 响应
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError JSON-RPC 2.0 错误
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ==================== MCP Protocol Types ====================

// InitializeResult 初始化结果
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
}

// ServerInfo 服务器信息
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities 服务器能力
type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability 工具能力
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ToolListResult 工具列表结果
type ToolListResult struct {
	Tools []Tool `json:"tools"`
}

// Tool 工具定义（用于响应）
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// CallToolResult 调用工具结果
type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// ==================== Tool Definition Types ====================

// ToolDef 工具定义
type ToolDef struct {
	Name        string
	Description string
	InputSchema InputSchema
	Handler     ToolHandler
}

// InputSchema 输入参数 schema
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]PropertyDef `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// PropertyDef 属性定义
type PropertyDef struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     any      `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolHandler 工具处理器类型
type ToolHandler func(ctx consolectx.Context, args map[string]any) (*ToolResult, error)

// ToolResult 工具执行结果
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content 内容块
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ==================== ToolResult Constructors ====================

// NewToolResult 创建工具结果
func NewToolResult(content []Content, isError bool) *ToolResult {
	return &ToolResult{
		Content: content,
		IsError: isError,
	}
}

// NewTextResult 创建文本结果
func NewTextResult(text string, isError bool) *ToolResult {
	return &ToolResult{
		Content: []Content{{Type: ContentTypeText, Text: text}},
		IsError: isError,
	}
}

// NewErrorResult 创建错误结果
func NewErrorResult(err error) *ToolResult {
	return &ToolResult{
		Content: []Content{{Type: ContentTypeText, Text: err.Error()}},
		IsError: true,
	}
}

// ==================== Validation ====================

// ValidateRequired 验证必需参数是否存在且非空
func ValidateRequired(schema InputSchema, args map[string]any) error {
	if args == nil {
		args = make(map[string]any)
	}

	for _, required := range schema.Required {
		val, exists := args[required]
		if !exists {
			return fmt.Errorf("missing required parameter: %s", required)
		}

		if IsEmpty(val) {
			return fmt.Errorf("required parameter %s cannot be empty", required)
		}
	}
	return nil
}

// IsEmpty 判断值是否为空
func IsEmpty(val any) bool {
	switch v := val.(type) {
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case nil:
		return true
	default:
		return false
	}
}
