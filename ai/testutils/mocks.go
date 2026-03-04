/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License); you may not use this file except in compliance with
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

package testutils

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"dubbo-admin-ai/runtime"

	"github.com/firebase/genkit/go/genkit"
)

// MockRuntime 提供测试用的Mock Runtime
// 它实现了runtime.Runtime接口，但简化了某些方法
type MockRuntime struct {
	components   map[string]runtime.Component
	mu           sync.RWMutex
	logger       *slog.Logger
	genkit       *genkit.Genkit
	factories    map[string]runtime.ComponentFactory
	factoryOrder []string
}

// NewMockRuntime 创建一个新的MockRuntime实例
func NewMockRuntime() *MockRuntime {
	ctx := context.Background()
	g := genkit.Init(ctx)

	return &MockRuntime{
		components:   make(map[string]runtime.Component),
		logger:       slog.Default(),
		genkit:       g,
		factories:    make(map[string]runtime.ComponentFactory),
		factoryOrder: make([]string, 0),
	}
}

// RegisterComponent 注册一个组件
func (m *MockRuntime) RegisterComponent(comp runtime.Component) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.components[comp.Name()] = comp
}

// GetComponent 根据名称获取组件
func (m *MockRuntime) GetComponent(name string) (runtime.Component, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	comp, ok := m.components[name]
	if !ok {
		return nil, fmt.Errorf("component not found: %s", name)
	}
	return comp, nil
}

// GetLogger 返回logger
func (m *MockRuntime) GetLogger() *slog.Logger {
	return m.logger
}

// GetContext 返回context
func (m *MockRuntime) GetContext() context.Context {
	return context.Background()
}

// GetGenkitRegistry 返回genkit registry
func (m *MockRuntime) GetGenkitRegistry() *genkit.Genkit {
	return m.genkit
}

// SetGenkitRegistry 设置genkit registry
func (m *MockRuntime) SetGenkitRegistry(registry *genkit.Genkit) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.genkit = registry
}

// RegisterFactory 注册组件工厂
func (m *MockRuntime) RegisterFactory(componentType string, factory runtime.ComponentFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.factories[componentType]; !exists {
		m.factoryOrder = append(m.factoryOrder, componentType)
	}

	m.factories[componentType] = factory
}

// GetFactoryFn 获取工厂函数
func (m *MockRuntime) GetFactoryFn(componentType string) (runtime.ComponentFactory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	factory, exists := m.factories[componentType]
	if !exists {
		return nil, fmt.Errorf("component type '%s' not registered", componentType)
	}

	return factory, nil
}
