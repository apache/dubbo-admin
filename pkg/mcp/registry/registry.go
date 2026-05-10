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

package registry

import (
	"fmt"

	"github.com/apache/dubbo-admin/pkg/mcp/types"
)

// ToolRegistrar 工具注册器接口
type ToolRegistrar interface {
	RegisterTools(registry *Registry)
}

// Registry 工具注册表
type Registry struct {
	tools      map[string]types.ToolDef
	registrars []ToolRegistrar
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools:      make(map[string]types.ToolDef),
		registrars: make([]ToolRegistrar, 0),
	}
}

// ==================== Tool CRUD 操作 ====================

// Register 注册单个工具
func (r *Registry) Register(tool types.ToolDef) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	r.tools[tool.Name] = tool
	return nil
}

// Unregister 注销工具
func (r *Registry) Unregister(name string) bool {
	if _, exists := r.tools[name]; !exists {
		return false
	}
	delete(r.tools, name)
	return true
}

// Get 获取指定工具
func (r *Registry) Get(name string) (types.ToolDef, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// Has 检查工具是否存在
func (r *Registry) Has(name string) bool {
	_, exists := r.tools[name]
	return exists
}

// List 列出所有工具
func (r *Registry) List() []types.ToolDef {
	result := make([]types.ToolDef, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// Count 获取工具数量
func (r *Registry) Count() int {
	return len(r.tools)
}

// Clear 清空所有工具
func (r *Registry) Clear() {
	r.tools = make(map[string]types.ToolDef)
}

// ==================== Registrar 管理 ====================

// RegisterRegistrar 注册注册器
func (r *Registry) RegisterRegistrar(registrar ToolRegistrar) {
	r.registrars = append(r.registrars, registrar)
}

// RegisterAll 注册所有注册器的工具
func (r *Registry) RegisterAll() {
	for _, registrar := range r.registrars {
		registrar.RegisterTools(r)
	}
}

// RegistrarsCount 返回注册器数量
func (r *Registry) RegistrarsCount() int {
	return len(r.registrars)
}

// ClearRegistrars 清空注册器
func (r *Registry) ClearRegistrars() {
	r.registrars = make([]ToolRegistrar, 0)
}
