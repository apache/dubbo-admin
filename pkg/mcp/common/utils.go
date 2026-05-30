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
	"encoding/json"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// ArgsHelper 参数辅助器
type ArgsHelper struct {
	args map[string]any
}

// NewArgsHelper 创建参数辅助器
func NewArgsHelper(args map[string]any) *ArgsHelper {
	return &ArgsHelper{args: args}
}

// GetString 获取字符串参数
func (h *ArgsHelper) GetString(key, defaultValue string) string {
	if v, ok := h.args[key].(string); ok {
		return v
	}
	return defaultValue
}

// GetInt 获取整数参数
func (h *ArgsHelper) GetInt(key string, defaultValue int) int {
	switch v := h.args[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return defaultValue
}

// GetBool 获取布尔参数
func (h *ArgsHelper) GetBool(key string, defaultValue bool) bool {
	if v, ok := h.args[key].(bool); ok {
		return v
	}
	return defaultValue
}

// GetRequiredString 获取必需的字符串参数
func (h *ArgsHelper) GetRequiredString(key string) (string, bool) {
	v, ok := h.args[key].(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

// BuildPageReq 构建分页请求参数
func BuildPageReq(pageNumber, pageSize int) coremodel.PageReq {
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageNumber <= 0 {
		pageNumber = DefaultPageNumber
	}
	return coremodel.PageReq{
		PageOffset: (pageNumber - 1) * pageSize,
		PageSize:   pageSize,
	}
}

// FormatJSON 格式化 JSON
func FormatJSON(data any) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// JsonResult 创建 JSON 结果
func JsonResult(data any) (*ToolResult, error) {
	jsonData, err := FormatJSON(data)
	if err != nil {
		return nil, err
	}
	return NewTextResult(jsonData, false), nil
}

// ErrorResult 创建错误结果
func ErrorResult(err error) *ToolResult {
	return NewErrorResult(err)
}

// GetMeshArg 获取 mesh 参数，默认使用配置中的 discovery id 作为 mesh
func GetMeshArg(ctx consolectx.Context, args map[string]any) string {
	helper := NewArgsHelper(args)
	if mesh := helper.GetString("mesh", ""); mesh != "" {
		return mesh
	}
	// 默认使用第一个 discovery 配置的 id 作为 mesh 名称
	if len(ctx.Config().Discovery) > 0 {
		return ctx.Config().Discovery[0].ID
	}
	// fallback 到 engine name
	return ctx.Config().Engine.Name
}
