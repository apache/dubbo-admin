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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
	"github.com/gin-gonic/gin"
)

// Server MCP 服务器
type Server struct {
	name           string
	version        string
	registry       *registry.Registry
	consoleContext consolectx.Context
}

// NewServer 创建 MCP 服务器
func NewServer(name, version string) *Server {
	return &Server{
		name:     name,
		version:  version,
		registry: registry.NewRegistry(),
	}
}

// NewServerWithRegistry 使用指定 Registry 创建 MCP 服务器
func NewServerWithRegistry(name, version string, reg *registry.Registry) *Server {
	return &Server{
		name:     name,
		version:  version,
		registry: reg,
	}
}

// GetRegistry 获取工具注册表
func (s *Server) GetRegistry() *registry.Registry {
	return s.registry
}

// SetConsoleContext 设置 console context
func (s *Server) SetConsoleContext(ctx consolectx.Context) {
	s.consoleContext = ctx
}

// GetConsoleContext 获取 console context
func (s *Server) GetConsoleContext() consolectx.Context {
	return s.consoleContext
}

// ==================== 请求处理 ====================

// HandleRequest 处理 JSON-RPC 请求（公开方法，供 transport 层使用）
func (s *Server) HandleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	return s.handleRequest(req)
}

// HandleHTTP 处理 HTTP 请求
func (s *Server) HandleHTTP(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.respondWithError(c, nil, ErrCodeParseError, "Parse error")
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		s.respondWithError(c, nil, ErrCodeParseError, "Parse error")
		return
	}

	c.JSON(http.StatusOK, s.handleRequest(&req))
}

// respondWithError 返回错误响应
func (s *Server) respondWithError(c *gin.Context, id interface{}, code int, message string) {
	c.JSON(http.StatusBadRequest, JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	})
}

// handleRequest 处理 JSON-RPC 请求
func (s *Server) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(req)
	case MethodToolsList:
		return s.handleToolsList(req)
	case MethodToolsCall:
		return s.handleToolsCall(req)
	default:
		return s.methodNotFoundResponse(req)
	}
}

// methodNotFoundResponse 方法未找到响应
func (s *Server) methodNotFoundResponse(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Error: &JSONRPCError{
			Code:    ErrCodeMethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		},
	}
}

// newErrorResponse 创建错误响应
func (s *Server) newErrorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}

// handleInitialize 处理 initialize 请求
func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo: ServerInfo{
				Name:    s.name,
				Version: s.version,
			},
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{},
			},
		},
	}
}

// handleToolsList 处理 tools/list 请求
func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	toolDefs := s.registry.List()
	tools := make([]Tool, 0, len(toolDefs))
	for _, def := range toolDefs {
		tools = append(tools, Tool{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		})
	}

	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  ToolListResult{Tools: tools},
	}
}

// handleToolsCall 处理 tools/call 请求
func (s *Server) handleToolsCall(req *JSONRPCRequest) *JSONRPCResponse {
	params, ok := req.Params.(map[string]any)
	if !ok {
		return s.newErrorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	name, ok := params["name"].(string)
	if !ok {
		return s.newErrorResponse(req.ID, ErrCodeInvalidParams, "Tool name is required")
	}

	tool, ok := s.registry.Get(name)
	if !ok {
		return s.newErrorResponse(req.ID, ErrCodeMethodNotFound, "Tool not found: "+name)
	}

	arguments, _ := params["arguments"].(map[string]any)

	// 验证必需参数
	if err := ValidateRequired(tool.InputSchema, arguments); err != nil {
		return s.newErrorResponse(req.ID, ErrCodeInvalidParams, err.Error())
	}

	result, err := tool.Handler(s.consoleContext, arguments)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result: CallToolResult{
				Content: []types.Content{{Type: ContentTypeText, Text: err.Error()}},
				IsError: true,
			},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  s.convertToCallToolResult(result),
	}
}

// convertToCallToolResult 转换 ToolResult 到 CallToolResult
func (s *Server) convertToCallToolResult(result *ToolResult) CallToolResult {
	content := make([]types.Content, len(result.Content))
	for i, c := range result.Content {
		content[i] = types.Content{Type: c.Type, Text: c.Text}
	}
	return CallToolResult{
		Content: content,
		IsError: result.IsError,
	}
}
