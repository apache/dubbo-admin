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

package types

import (
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
)

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
		Content: []Content{{Type: "text", Text: text}},
		IsError: isError,
	}
}

// NewErrorResult 创建错误结果
func NewErrorResult(err error) *ToolResult {
	return &ToolResult{
		Content: []Content{{Type: "text", Text: err.Error()}},
		IsError: true,
	}
}
