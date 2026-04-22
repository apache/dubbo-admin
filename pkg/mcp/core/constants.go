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

const (
	// JSONRPCVersion JSON-RPC 协议版本
	JSONRPCVersion = "2.0"

	// ProtocolVersion MCP 协议版本
	ProtocolVersion = "2024-11-05"

	// MethodInitialize 初始化方法
	MethodInitialize = "initialize"
	// MethodToolsList 工具列表方法
	MethodToolsList = "tools/list"
	// MethodToolsCall 工具调用方法
	MethodToolsCall = "tools/call"

	// ContentTypeText 文本内容类型
	ContentTypeText = "text"
)

// JSONRPC 错误码
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603
)
