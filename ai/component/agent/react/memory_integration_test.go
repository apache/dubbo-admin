/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
 */
package react

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/store"
	gormstore "dubbo-admin-ai/store/gorm"
	"dubbo-admin-ai/testutils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMemoryProviderRequestsAndTransitiveHistoryInvalidation(t *testing.T) {
	for _, kind := range []string{"memory", "gorm"} {
		t.Run(kind, func(t *testing.T) {
			provider := &providerFixture{first: `{"role":"assistant","content":"answer contains obsolete-memory-codeword"}`}
			unused, mem, spec, g, _ := compilerAgentFixture(t, provider)
			unused.Stop()
			var backend store.BackendStore = mem
			if kind == "gorm" {
				db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "memory.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
				if err != nil {
					t.Fatal(err)
				}
				sqlDB, _ := db.DB()
				sqlDB.SetMaxOpenConns(1)
				persistent, err := gormstore.NewGormStore(db, 20)
				if err != nil {
					t.Fatal(err)
				}
				if err = persistent.Migrate(context.Background()); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = persistent.Close() })
				backend = persistent
				createAgentSession(t, backend, "compiled")
			}
			memory := backend.(store.ExplicitMemoryStore)
			a := testutils.TestAccess()
			ctx := testutils.ContextWithTestAccess(context.Background())
			r, err := memory.SaveMemory(ctx, a, store.MemoryWrite{Generation: 1, StateVersion: 1, ScopeType: "user", MemoryType: "preference", FactKey: "response.style", Value: "obsolete-memory-codeword"})
			if err != nil {
				t.Fatal(err)
			}
			spec.ExplicitMemory.Enabled = true
			ra, err := NewReActAgentWithStore(g, backend, spec, newToolTimeoutResolver(5, nil), nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(ra.Stop)
			runCompiledInteraction(t, ra, backend, "first")
			provider.mu.Lock()
			first, _ := json.Marshal(provider.requests[0])
			provider.mu.Unlock()
			if !strings.Contains(string(first), "obsolete-memory-codeword") {
				t.Fatal("Memory absent at Provider", string(first))
			}
			// The second answer inherits Memory only through the first answer.
			ra.compilerSpec.ExplicitMemory.Enabled = false
			second := runCompiledInteraction(t, ra, backend, "second")
			active, err := backend.BeginTurn(ctx, a, "compiled")
			if err != nil {
				t.Fatal(err)
			}
			turns, err := backend.(store.CompilationStore).ContextTurns(ctx, a, "compiled", active, 10, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, turn := range turns {
				if turn.ID == second {
					found = len(turn.Dependencies) == 1
				}
			}
			if !found {
				t.Fatal("transitive history dependency lost", turns)
			}
			if err := backend.AbortTurnForTurn(ctx, a, "compiled", active); err != nil {
				t.Fatal(err)
			}
			if err := memory.DeleteMemory(ctx, a, r.ID, store.MemoryMutation{Generation: 1, StateVersion: 1, Version: 1}); err != nil {
				t.Fatal(err)
			}
			runCompiledInteraction(t, ra, backend, "after-delete")
			provider.mu.Lock()
			last, _ := json.Marshal(provider.requests[len(provider.requests)-1])
			provider.mu.Unlock()
			if strings.Contains(string(last), "obsolete-memory-codeword") || strings.Contains(string(last), "final answer") {
				t.Fatal("stale direct/transitive answer leaked", string(last))
			}
			original, err := backend.AllMemory(ctx, a, "compiled")
			if err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(original)
			if !strings.Contains(string(data), "obsolete-memory-codeword") {
				t.Fatal("canonical history changed")
			}
			// Independent new Session injection uses the same current-user store.
			_, err = memory.SaveMemory(ctx, a, store.MemoryWrite{Generation: 1, StateVersion: 1, ScopeType: "user", MemoryType: "preference", FactKey: "response.language", Value: "fresh-cross-session-value"})
			if err != nil {
				t.Fatal(err)
			}
			createAgentSession(t, backend, "fresh-session")
			ra.compilerSpec.ExplicitMemory.Enabled = true
			channels := ra.Interact(ctx, &schema.UserInput{Content: "hello"}, "fresh-session")
			deadline := time.After(5 * time.Second)
			done := false
			for !done {
				select {
				case err := <-channels.ErrorChan:
					if err != nil {
						t.Fatal(err)
					}
				case <-channels.UserRespChan:
				case <-channels.Done():
					done = true
				case <-deadline:
					t.Fatal("new Session stalled")
				}
			}
			provider.mu.Lock()
			last, _ = json.Marshal(provider.requests[len(provider.requests)-1])
			provider.mu.Unlock()
			if !strings.Contains(string(last), "fresh-cross-session-value") {
				t.Fatal("cross-session injection missing", string(last))
			}
		})
	}
}

func TestMemoryRecoveryFilteringAfterConcurrentInvalidation(t *testing.T) {
	provider := &providerFixture{}
	unused, mem, _, _, _ := compilerAgentFixture(t, provider)
	unused.Stop()
	a := testutils.TestAccess()
	ctx := testutils.ContextWithTestAccess(context.Background())
	r, err := mem.SaveMemory(ctx, a, store.MemoryWrite{Generation: 1, StateVersion: 1, ScopeType: "user", MemoryType: "preference", FactKey: "response.style", Value: "recovered-value"})
	if err != nil {
		t.Fatal(err)
	}
	ra := unused
	ra.memoryStore = mem
	state := &interactionState{access: a}
	recovery := store.ToolRecovery{Batches: []store.ToolBatch{{ID: "recovered", Dependencies: []store.MemoryDependency{{ID: r.ID, Version: r.Version, Generation: 1, StateVersion: 1}}}}}
	kept, err := ra.filterMemoryRecovery(ctx, state, "compiled", recovery)
	if err != nil || len(kept.Batches) != 1 {
		t.Fatalf("valid recovery batch was filtered: %+v %v", kept, err)
	}
	if err := mem.DeleteMemory(ctx, a, r.ID, store.MemoryMutation{Generation: 1, StateVersion: 1, Version: r.Version}); err != nil {
		t.Fatal(err)
	}
	filtered, err := ra.filterMemoryRecovery(ctx, state, "compiled", kept)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Batches) != 0 || !filtered.Omitted {
		t.Fatalf("invalid recovery batch was retained after Memory deletion: %+v", filtered)
	}
}
