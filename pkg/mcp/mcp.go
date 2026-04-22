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

// Package mcp 提供 MCP (Model Context Protocol) 服务器实现。
//
// # 包结构
//
//   - core: MCP 服务器核心功能（Server、类型定义、常量）
//   - registry: 工具注册表和注册器接口
//   - tools: 内置工具实现（cluster、search、service）
//
// # 快速开始
//
//	server := mcp.NewServer("dubbo-admin", "1.0.0")
//	mcp.RegisterDefaultTools(server)
//
// # 使用构建器
//
//	server := mcp.NewServerBuilder().
//	    WithName("custom-server").
//	    WithVersion("2.0.0").
//	    Build()
//	mcp.RegisterDefaultTools(server)
package mcp

import (
	"github.com/apache/dubbo-admin/pkg/mcp/core"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
	"github.com/apache/dubbo-admin/pkg/mcp/tools"
)

// 类型别名，方便外部使用
type (
	Server        = core.Server
	ServerBuilder = core.ServerBuilder
	ToolDef       = core.ToolDef
	ToolResult    = core.ToolResult
	ToolHandler   = core.ToolHandler
	InputSchema   = core.InputSchema
	PropertyDef   = core.PropertyDef
	JSONRPCRequest = core.JSONRPCRequest
	JSONRPCResponse = core.JSONRPCResponse
	ToolRegistrar = registry.ToolRegistrar
	Registry      = registry.Registry
)

// NewServer 创建 MCP 服务器
func NewServer(name, version string) *core.Server {
	return core.NewServer(name, version)
}

// NewServerBuilder 创建服务器构建器
func NewServerBuilder() *core.ServerBuilder {
	return core.NewServerBuilder()
}

// NewRegistry 创建空的工具注册表
func NewRegistry() *registry.Registry {
	return registry.NewRegistry()
}

// RegisterDefaultTools 注册所有默认工具到注册表
func RegisterDefaultTools(reg *registry.Registry) {
	reg.RegisterRegistrar(&tools.MetricsRegistrar{})
	reg.RegisterRegistrar(&tools.ResourceSearchRegistrar{})
	reg.RegisterRegistrar(&tools.ServiceRegistrar{})
	reg.RegisterAll()
}

// DefaultRegistry 创建包含所有内置工具的注册表
func DefaultRegistry() *registry.Registry {
	reg := registry.NewRegistry()
	RegisterDefaultTools(reg)
	return reg
}

// 常量导出
const (
	JSONRPCVersion      = core.JSONRPCVersion
	ProtocolVersion     = core.ProtocolVersion
	MethodInitialize    = core.MethodInitialize
	MethodToolsList     = core.MethodToolsList
	MethodToolsCall     = core.MethodToolsCall
	ContentTypeText     = core.ContentTypeText
	ErrCodeParseError    = core.ErrCodeParseError
	ErrCodeInvalidRequest = core.ErrCodeInvalidRequest
	ErrCodeMethodNotFound = core.ErrCodeMethodNotFound
	ErrCodeInvalidParams  = core.ErrCodeInvalidParams
)

// 工具结果构造函数
var (
	NewToolResult = core.NewToolResult
	NewTextResult = core.NewTextResult
	NewErrorResult = core.NewErrorResult
)
