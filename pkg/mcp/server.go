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

package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/mcp/common"
)

// Server MCP 服务器
type Server struct {
	name           string
	version        string
	tools          map[string]*common.ToolDef
	consoleContext consolectx.Context
	mu             sync.RWMutex
}

// NewServer 创建 MCP 服务器
func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   make(map[string]*common.ToolDef),
	}
}

// RegisterTool 注册单个工具
func (s *Server) RegisterTool(tool *common.ToolDef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = tool
}

// SetConsoleContext 设置 console context
func (s *Server) SetConsoleContext(ctx consolectx.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consoleContext = ctx
}

// GetConsoleContext 获取 console context
func (s *Server) GetConsoleContext() consolectx.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.consoleContext
}

// ==================== 请求处理 ====================

// HandleRequest 处理 JSON-RPC 请求（公开方法，供 transport 层使用）
func (s *Server) HandleRequest(req *common.JSONRPCRequest) *common.JSONRPCResponse {
	return s.handleRequest(req)
}

// HandleHTTP 处理 HTTP 请求
func (s *Server) HandleHTTP(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.respondWithError(c, nil, common.ErrCodeParseError, "Parse error")
		return
	}

	var req common.JSONRPCRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		s.respondWithError(c, nil, common.ErrCodeParseError, "Parse error")
		return
	}

	c.JSON(http.StatusOK, s.handleRequest(&req))
}

// respondWithError 返回错误响应
func (s *Server) respondWithError(c *gin.Context, id interface{}, code int, message string) {
	c.JSON(http.StatusBadRequest, common.JSONRPCResponse{
		JSONRPC: common.JSONRPCVersion,
		Error: &common.JSONRPCError{
			Code:    code,
			Message: message,
		},
	})
}

// handleRequest 处理 JSON-RPC 请求
func (s *Server) handleRequest(req *common.JSONRPCRequest) *common.JSONRPCResponse {
	switch req.Method {
	case common.MethodInitialize:
		return s.handleInitialize(req)
	case common.MethodToolsList:
		return s.handleToolsList(req)
	case common.MethodToolsCall:
		return s.handleToolsCall(req)
	default:
		return s.methodNotFoundResponse(req)
	}
}

// methodNotFoundResponse 方法未找到响应
func (s *Server) methodNotFoundResponse(req *common.JSONRPCRequest) *common.JSONRPCResponse {
	return &common.JSONRPCResponse{
		JSONRPC: common.JSONRPCVersion,
		ID:      req.ID,
		Error: &common.JSONRPCError{
			Code:    common.ErrCodeMethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		},
	}
}

// newErrorResponse 创建错误响应
func (s *Server) newErrorResponse(id interface{}, code int, message string) *common.JSONRPCResponse {
	return &common.JSONRPCResponse{
		JSONRPC: common.JSONRPCVersion,
		ID:      id,
		Error: &common.JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}

// handleInitialize 处理 initialize 请求
func (s *Server) handleInitialize(req *common.JSONRPCRequest) *common.JSONRPCResponse {
	s.mu.RLock()
	name, version := s.name, s.version
	s.mu.RUnlock()

	return &common.JSONRPCResponse{
		JSONRPC: common.JSONRPCVersion,
		ID:      req.ID,
		Result: common.InitializeResult{
			ProtocolVersion: common.ProtocolVersion,
			ServerInfo: common.ServerInfo{
				Name:    name,
				Version: version,
			},
			Capabilities: common.ServerCapabilities{
				Tools: &common.ToolsCapability{},
			},
		},
	}
}

// handleToolsList 处理 tools/list 请求
func (s *Server) handleToolsList(req *common.JSONRPCRequest) *common.JSONRPCResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]common.Tool, 0, len(s.tools))
	for _, def := range s.tools {
		tools = append(tools, common.Tool{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		})
	}

	return &common.JSONRPCResponse{
		JSONRPC: common.JSONRPCVersion,
		ID:      req.ID,
		Result:  common.ToolListResult{Tools: tools},
	}
}

// handleToolsCall 处理 tools/call 请求
func (s *Server) handleToolsCall(req *common.JSONRPCRequest) *common.JSONRPCResponse {
	s.mu.RLock()
	ctx := s.consoleContext
	s.mu.RUnlock()

	params, ok := req.Params.(map[string]any)
	if !ok {
		return s.newErrorResponse(req.ID, common.ErrCodeInvalidParams, "Invalid params")
	}

	name, ok := params["name"].(string)
	if !ok {
		return s.newErrorResponse(req.ID, common.ErrCodeInvalidParams, "Tool name is required")
	}

	s.mu.RLock()
	tool, ok := s.tools[name]
	s.mu.RUnlock()

	if !ok {
		return s.newErrorResponse(req.ID, common.ErrCodeMethodNotFound, "Tool not found: "+name)
	}

	arguments, _ := params["arguments"].(map[string]any)

	// 验证必需参数
	if err := common.ValidateRequired(tool.InputSchema, arguments); err != nil {
		return s.newErrorResponse(req.ID, common.ErrCodeInvalidParams, err.Error())
	}

	startedAt := time.Now()
	result, err := tool.Handler(ctx, arguments)
	if err != nil {
		logger.Sugar().Warnw("MCP tool call failed", "tool", name, "request_id", req.ID, "elapsed_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return &common.JSONRPCResponse{
			JSONRPC: common.JSONRPCVersion,
			ID:      req.ID,
			Result: common.CallToolResult{
				Content: []common.Content{{Type: common.ContentTypeText, Text: err.Error()}},
				IsError: true,
			},
		}
	}
	if result == nil {
		logger.Sugar().Errorw("MCP tool returned nil result", "tool", name, "request_id", req.ID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return s.newErrorResponse(req.ID, common.ErrCodeInternalError, "Tool returned no result")
	}
	if result.IsError {
		logger.Sugar().Warnw("MCP tool call completed with error", "tool", name, "request_id", req.ID, "elapsed_ms", time.Since(startedAt).Milliseconds(), "content_blocks", len(result.Content))
	} else {
		logger.Sugar().Infow("MCP tool call completed", "tool", name, "request_id", req.ID, "elapsed_ms", time.Since(startedAt).Milliseconds(), "content_blocks", len(result.Content))
	}

	return &common.JSONRPCResponse{
		JSONRPC: common.JSONRPCVersion,
		ID:      req.ID,
		Result:  s.convertToCallToolResult(result),
	}
}

// convertToCallToolResult 转换 ToolResult 到 CallToolResult
func (s *Server) convertToCallToolResult(result *common.ToolResult) common.CallToolResult {
	content := make([]common.Content, len(result.Content))
	for i, c := range result.Content {
		content[i] = common.Content{Type: c.Type, Text: c.Text}
	}
	return common.CallToolResult{
		Content: content,
		IsError: result.IsError,
	}
}
