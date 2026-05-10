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
	"encoding/json"
	"io"
	"net/http"

	"github.com/apache/dubbo-admin/pkg/mcp/core"
)

// Handler HTTP请求处理器
type Handler struct {
	server *core.Server
}

// NewHandler 创建HTTP处理器
func NewHandler(server *core.Server) *Handler {
	return &Handler{
		server: server,
	}
}

// ServeHTTP 实现http.Handler接口
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 设置CORS headers
	h.setCORSHeaders(w)

	// 处理OPTIONS请求（CORS preflight）
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只接受POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendError(w, nil, core.ErrCodeParseError, "Failed to read request body")
		return
	}

	// 解析JSON-RPC请求
	var req core.JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.sendError(w, nil, core.ErrCodeParseError, "Invalid JSON")
		return
	}

	// 处理请求并获取响应
	resp := h.server.HandleRequest(&req)

	// 发送响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// setCORSHeaders 设置CORS headers
func (h *Handler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// sendError 发送错误响应
func (h *Handler) sendError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	resp := core.JSONRPCResponse{
		JSONRPC: core.JSONRPCVersion,
		ID:      id,
		Error: &core.JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

// HandleMCPRequest 处理MCP请求（公开方法）
func (h *Handler) HandleMCPRequest(w http.ResponseWriter, req *core.JSONRPCRequest) {
	resp := h.server.HandleRequest(req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
