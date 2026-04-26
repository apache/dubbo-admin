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

package memory

import (
	"context"
	"dubbo-admin-ai/runtime"
	"fmt"
)

// MemoryComponent implements the memory component
type MemoryComponent struct {
	instanceName string
	historyKey   HistoryKey
	config       MemoryConfig
	memoryCtx    context.Context
	memory       *ChatMemory
}

// NewMemoryComponent creates a new MemoryComponent with the given config
func NewMemoryComponent(config MemoryConfig) (runtime.Component, error) {
	if config.MaxMessages <= 0 {
		config.MaxMessages = 20 // default
	}
	return &MemoryComponent{
		historyKey: ChatHistoryKey,
		config:     config,
	}, nil
}

// Deprecated: Use NewMemoryComponent with MemoryConfig instead
func NewMemoryComponentWithKey(historyKey HistoryKey, maxTurns ...int) (runtime.Component, error) {
	maxMessages := 20
	if len(maxTurns) > 0 && maxTurns[0] > 0 {
		maxMessages = maxTurns[0]
	}
	return NewMemoryComponent(MemoryConfig{
		MaxMessages: maxMessages,
	})
}

func (m *MemoryComponent) Name() string {
	if m.instanceName != "" {
		return m.instanceName
	}
	return "memory"
}

func (m *MemoryComponent) SetName(name string) {
	m.instanceName = name
}

func (m *MemoryComponent) Validate() error {
	if m.config.MaxMessages <= 0 {
		return fmt.Errorf("max_messages must be greater than 0")
	}
	return nil
}

func (m *MemoryComponent) Init(rt *runtime.Runtime) error {
	m.memoryCtx = NewMemoryContext(m.historyKey)
	history, err := GetHistoryMemory(m.memoryCtx, m.historyKey)
	if err != nil {
		return fmt.Errorf("failed to initialize history: %w", err)
	}

	// Update max messages if configured
	if m.config.MaxMessages > 0 {
		history.max = m.config.MaxMessages
	}

	m.memory = history

	rt.GetLogger().Info("Memory component initialized",
		"history_key", m.historyKey,
		"max_messages", m.config.MaxMessages)

	return nil
}

func (m *MemoryComponent) Start() error {
	return nil
}

func (m *MemoryComponent) Stop() error {
	return nil
}

// GetContext returns the memory context
func (m *MemoryComponent) GetContext() context.Context {
	return m.memoryCtx
}

// GetMemory returns the underlying ChatMemory instance
func (m *MemoryComponent) GetMemory() (*ChatMemory, error) {
	if m.memory == nil {
		return nil, fmt.Errorf("history not initialized")
	}
	return m.memory, nil
}

// GetHistory returns the underlying HistoryMemory instance (alias for backward compatibility)
func (m *MemoryComponent) GetHistory() (*HistoryMemory, error) {
	return m.GetMemory()
}
