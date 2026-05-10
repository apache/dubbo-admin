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

import "github.com/apache/dubbo-admin/pkg/mcp/types"

// 类型别名，保持向后兼容
type (
	ToolDef     = types.ToolDef
	InputSchema = types.InputSchema
	PropertyDef = types.PropertyDef
	ToolHandler = types.ToolHandler
	ToolResult  = types.ToolResult
	Content     = types.Content
)

// 工具结果构造函数
var (
	NewToolResult  = types.NewToolResult
	NewTextResult  = types.NewTextResult
	NewErrorResult = types.NewErrorResult
)

// 验证函数
var (
	ValidateRequired = types.ValidateRequired
	IsEmpty          = types.IsEmpty
)
