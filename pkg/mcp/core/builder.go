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
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
)

// ServerBuilder 服务器构建器
type ServerBuilder struct {
	name           string
	version        string
	consoleContext consolectx.Context
}

// NewServerBuilder 创建服务器构建器
func NewServerBuilder() *ServerBuilder {
	return &ServerBuilder{
		name:    "mcp-server",
		version: "1.0.0",
	}
}

// WithName 设置服务器名称
func (b *ServerBuilder) WithName(name string) *ServerBuilder {
	b.name = name
	return b
}

// WithVersion 设置服务器版本
func (b *ServerBuilder) WithVersion(version string) *ServerBuilder {
	b.version = version
	return b
}

// WithConsoleContext 设置 console context
func (b *ServerBuilder) WithConsoleContext(ctx consolectx.Context) *ServerBuilder {
	b.consoleContext = ctx
	return b
}

// Build 构建服务器
func (b *ServerBuilder) Build() *Server {
	server := NewServer(b.name, b.version)
	if b.consoleContext != nil {
		server.SetConsoleContext(b.consoleContext)
	}
	return server
}
