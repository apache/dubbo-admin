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
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/runtime"
	conversationstore "dubbo-admin-ai/store"
	gormstore "dubbo-admin-ai/store/gorm"
	memorystore "dubbo-admin-ai/store/memory"

	"github.com/firebase/genkit/go/ai"
)

func TestMemorySpecValidate(t *testing.T) {
	tests := []struct {
		name string
		spec MemorySpec
		want string
	}{
		{
			name: "gorm requires database",
			spec: MemorySpec{Backend: "gorm", MaxTurns: 1},
			want: "database is required",
		},
		{
			name: "unknown backend",
			spec: MemorySpec{Backend: "unknown", MaxTurns: 1},
			want: "unsupported memory backend",
		},
		{
			name: "idle cannot exceed open",
			spec: MemorySpec{
				Backend:  "gorm",
				MaxTurns: 1,
				Database: &DatabaseSpec{Driver: "mysql", DSN: "dsn", MaxOpenConns: 2, MaxIdleConns: 3},
			},
			want: "must not exceed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.spec.Validate(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestMemoryComponent_GormBackendLifecycle(t *testing.T) {
	component, err := NewMemoryComponentFromSpec(MemorySpec{
		Backend:    "gorm",
		HistoryKey: ChatHistoryKey,
		MaxTurns:   1,
		Database: &DatabaseSpec{
			Driver:       "sqlite",
			DSN:          filepath.Join(t.TempDir(), "memory.db"),
			MaxOpenConns: 2,
			MaxIdleConns: 1,
		},
	})
	if err != nil {
		t.Fatalf("NewMemoryComponentFromSpec() error = %v", err)
	}
	if err := component.Init(runtime.NewRuntime()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	store, err := component.(*MemoryComponent).GetStore()
	if err != nil {
		t.Fatalf("GetStore() error = %v", err)
	}
	gormBackend, ok := store.(*gormstore.GormStore)
	if !ok {
		t.Fatalf("store type = %T, want *gormstore.GormStore", store)
	}
	assertConfiguredTurnLimit(t, store, 1)
	sqlDB, err := gormBackend.DB().DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	if got := sqlDB.Stats().MaxOpenConnections; got != 2 {
		t.Fatalf("max open connections = %d, want 2", got)
	}
	if err := component.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := sqlDB.PingContext(context.Background()); err == nil {
		t.Fatal("PingContext() after Stop() succeeded, want closed database")
	}
}

func TestMemoryComponent_PassesMaxTurnsToMemoryStore(t *testing.T) {
	component, err := NewMemoryComponentFromSpec(MemorySpec{
		Backend:  "memory",
		MaxTurns: 2,
	})
	if err != nil {
		t.Fatalf("NewMemoryComponentFromSpec() error = %v", err)
	}
	if err := component.Init(runtime.NewRuntime()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	t.Cleanup(func() { _ = component.Stop() })

	store, err := component.(*MemoryComponent).GetStore()
	if err != nil {
		t.Fatalf("GetStore() error = %v", err)
	}
	if _, ok := store.(*memorystore.MemoryStore); !ok {
		t.Fatalf("store type = %T, want *memorystore.MemoryStore", store)
	}
	assertConfiguredTurnLimit(t, store, 2)
}

func TestMemoryComponentLazilyInitializesLegacyHistory(t *testing.T) {
	component, err := NewMemoryComponentFromSpec(MemorySpec{
		Backend:  "memory",
		MaxTurns: 2,
	})
	if err != nil {
		t.Fatalf("NewMemoryComponentFromSpec() error = %v", err)
	}
	memoryComponent := component.(*MemoryComponent)
	if err := memoryComponent.Init(runtime.NewRuntime()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	t.Cleanup(func() { _ = memoryComponent.Stop() })

	if memoryComponent.memory != nil || memoryComponent.memoryCtx != nil {
		t.Fatal("Init() initialized legacy HistoryMemory; want Store-only initialization")
	}

	history, err := memoryComponent.GetMemory()
	if err != nil {
		t.Fatalf("GetMemory() error = %v", err)
	}
	if history == nil || memoryComponent.GetContext() == nil {
		t.Fatal("legacy HistoryMemory was not initialized on explicit request")
	}
}

func assertConfiguredTurnLimit(t *testing.T, store conversationstore.Store, limit int) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	session := &conversationstore.Session{ID: "configured-limit", CreatedAt: now, UpdatedAt: now, Status: "active"}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	for i := 0; i < limit; i++ {
		turnID, err := store.BeginTurn(ctx, session.ID)
		if err != nil {
			t.Fatalf("BeginTurn(%d) error = %v", i, err)
		}
		if err := store.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserTextMessage("turn")); err != nil {
			t.Fatalf("AddHistory(%d) error = %v", i, err)
		}
		if err := store.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
			t.Fatalf("NextTurn(%d) error = %v", i, err)
		}
	}
	if _, err := store.BeginTurn(ctx, session.ID); !errors.Is(err, conversationstore.ErrTurnLimitReached) {
		t.Fatalf("overflow BeginTurn() error = %v, want ErrTurnLimitReached", err)
	}
}
