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
	"fmt"
	"strings"
	"sync"

	"dubbo-admin-ai/runtime"
	conversationstore "dubbo-admin-ai/store"
	gormstore "dubbo-admin-ai/store/gorm"
	memorystore "dubbo-admin-ai/store/memory"
	"gorm.io/gorm"
)

// MemoryComponent owns the single conversation Store shared by runtime
// consumers. The legacy HistoryMemory accessors remain for compatibility, but
// are initialized only when explicitly requested and are not a production data
// source.
type MemoryComponent struct {
	instanceName string
	spec         MemorySpec
	historyKey   HistoryKey
	maxTurns     int
	legacyOnce   sync.Once
	legacyErr    error
	memoryCtx    context.Context
	memory       *HistoryMemory
	store        conversationstore.Store
	gormStore    *gormstore.GormStore
}

func NewMemoryComponent(historyKey HistoryKey, maxTurns ...int) (runtime.Component, error) {
	limit := 100
	if len(maxTurns) > 0 {
		limit = maxTurns[0]
	}
	return &MemoryComponent{
		spec: MemorySpec{
			Backend:    DefaultBackend,
			HistoryKey: historyKey,
			MaxTurns:   limit,
		},
		historyKey: historyKey,
		maxTurns:   limit,
	}, nil
}

// NewMemoryComponentFromSpec creates a memory component from the complete
// configuration decoded by the runtime factory.
func NewMemoryComponentFromSpec(spec MemorySpec) (runtime.Component, error) {
	spec.Backend = strings.ToLower(strings.TrimSpace(spec.Backend))
	if spec.Backend == "" {
		spec.Backend = DefaultBackend
	}
	if spec.HistoryKey == "" {
		spec.HistoryKey = ChatHistoryKey
	}
	if spec.MaxTurns == 0 {
		spec.MaxTurns = DefaultMemorySpec().MaxTurns
	}
	if spec.Database != nil {
		spec.Database.applyDefaults()
	}
	return &MemoryComponent{
		spec:       spec,
		historyKey: spec.HistoryKey,
		maxTurns:   spec.MaxTurns,
	}, nil
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
	return m.spec.Validate()
}

func (m *MemoryComponent) Init(rt *runtime.Runtime) error {
	backend := m.spec.Backend
	if backend == "" {
		backend = DefaultBackend
	}
	switch backend {
	case "memory":
		m.store = memorystore.NewMemoryStore(m.maxTurns)
	case "gorm":
		if m.spec.Database == nil {
			return fmt.Errorf("database is required for gorm backend")
		}
		db, err := gormstore.Open(m.spec.Database.Driver, m.spec.Database.DSN, &gorm.Config{})
		if err != nil {
			return fmt.Errorf("failed to open conversation database: %w", err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			if closeDB, closeErr := db.DB(); closeErr == nil {
				_ = closeDB.Close()
			}
			return fmt.Errorf("failed to access conversation database: %w", err)
		}
		sqlDB.SetMaxOpenConns(m.spec.Database.MaxOpenConns)
		sqlDB.SetMaxIdleConns(m.spec.Database.MaxIdleConns)
		gormStore, err := gormstore.NewGormStore(db, m.maxTurns)
		if err != nil {
			_ = sqlDB.Close()
			return fmt.Errorf("failed to create gorm conversation store: %w", err)
		}
		if err := gormStore.Migrate(context.Background()); err != nil {
			_ = gormStore.Close()
			return fmt.Errorf("failed to migrate conversation database: %w", err)
		}
		m.gormStore = gormStore
		m.store = gormStore
	default:
		return fmt.Errorf("unsupported memory backend %q", backend)
	}
	rt.GetLogger().Info("Memory component initialized",
		"history_key", m.historyKey)

	return nil
}

func (m *MemoryComponent) Start() error {
	return nil
}

func (m *MemoryComponent) Stop() error {
	if m.gormStore != nil {
		err := m.gormStore.Close()
		m.gormStore = nil
		return err
	}
	return nil
}

// initLegacyMemory initializes the deprecated context-backed memory only when
// a legacy caller explicitly requests it.
func (m *MemoryComponent) initLegacyMemory() {
	m.legacyOnce.Do(func() {
		m.memoryCtx = NewMemoryContext(m.historyKey)
		m.memory, m.legacyErr = GetHistoryMemory(m.memoryCtx, m.historyKey)
	})
}

// GetContext returns the deprecated context-backed memory context.
func (m *MemoryComponent) GetContext() context.Context {
	m.initLegacyMemory()
	return m.memoryCtx
}

// GetStore returns the single conversation Store shared by runtime consumers.
func (m *MemoryComponent) GetStore() (conversationstore.Store, error) {
	if m.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return m.store, nil
}

// GetMemory returns the deprecated HistoryMemory instance for compatibility.
func (m *MemoryComponent) GetMemory() (*HistoryMemory, error) {
	m.initLegacyMemory()
	return m.memory, m.legacyErr
}
