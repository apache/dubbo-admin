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

package storetest

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	store "dubbo-admin-ai/store"
	"github.com/firebase/genkit/go/ai"
)

func RunCompilationContractTests(t *testing.T, newStore func() store.BackendStore) {
	t.Helper()
	for _, scenario := range []string{"complete", "failed", "lease", "quota", "ownership", "snapshot", "pin", "concurrent", "bounded history", "recovery new request", "recovery retry"} {
		t.Run(scenario, func(t *testing.T) {
			s := newStore()
			c, ok := s.(store.CompilationStore)
			if !ok {
				t.Fatal("CompilationStore missing")
			}
			ctx := context.Background()
			a := contractAccess
			now := time.Now()
			if err := s.Create(ctx, a, &store.Session{ID: "compile", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, Version: 1, Status: store.SessionStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
				t.Fatal(err)
			}
			prepared, err := s.PrepareInteraction(ctx, a, "compile", "request", "hash", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.AddHistoryToTurn(ctx, a, "compile", prepared.TurnID, ai.NewUserTextMessage("question")); err != nil {
				t.Fatal(err)
			}
			batch := store.ToolBatch{ID: "batch", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "compile", TurnID: prepared.TurnID, InteractionID: prepared.InteractionID, Sequence: 1, Version: 1, SchemaVersion: "v1", Calls: []store.ToolCall{
				{ID: "call1", Name: "lookup", Input: json.RawMessage(`{"name":"one"}`), Status: "pending"},
				{ID: "call2", Name: "lookup", Input: json.RawMessage(`{"name":"two"}`), Status: "pending"},
			}}
			if err := c.CreateToolBatch(ctx, a, batch); err != nil {
				t.Fatal(err)
			}
			terminal := batch.Calls[0]
			terminal.Status = "succeeded"
			terminal.Output = json.RawMessage(`{"ok":true}`)
			switch scenario {
			case "recovery new request", "recovery retry":
				if err := c.CompleteToolCall(ctx, a, "compile", batch.ID, 1, 0, terminal); err != nil {
					t.Fatal(err)
				}
				// Simulate a crash: the second call remains pending, not known unexecuted.
				if err := s.FailInteraction(ctx, a, "compile", prepared.InteractionID, prepared.TurnID); err != nil {
					t.Fatal(err)
				}
				key, hash := "next", "next"
				if scenario == "recovery retry" {
					key, hash = "request", "hash"
				}
				next, err := s.PrepareInteraction(ctx, a, "compile", key, hash, time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				recovered, err := c.RecoveryToolBatches(ctx, a, "compile", next.TurnID, 16, 65536)
				if err != nil || recovered.Omitted || len(recovered.Batches) != 1 || recovered.Batches[0].Calls[0].Status != "succeeded" || recovered.Batches[0].Calls[1].Status != "pending" {
					t.Fatalf("released attempt unavailable: %+v %v", recovered, err)
				}
				recovered.Batches[0].Calls[0].Output[0] = 'x'
				again, err := c.RecoveryToolBatches(ctx, a, "compile", next.TurnID, 16, 65536)
				if err != nil || !json.Valid(again.Batches[0].Calls[0].Output) {
					t.Fatal("recovery returned mutable storage")
				}
				limited, err := c.RecoveryToolBatches(ctx, a, "compile", next.TurnID, 16, 1)
				if err != nil || !limited.Omitted || len(limited.Batches) != 0 {
					t.Fatalf("recovery byte bound: %+v %v", limited, err)
				}
				// A second released attempt exercises the count cap and stable ordering.
				second := batch
				second.ID = "second"
				second.TurnID = next.TurnID
				second.InteractionID = next.InteractionID
				if err := c.CreateToolBatch(ctx, a, second); err != nil {
					t.Fatal(err)
				}
				if err := s.FailInteraction(ctx, a, "compile", next.InteractionID, next.TurnID); err != nil {
					t.Fatal(err)
				}
				current, err := s.PrepareInteraction(ctx, a, "compile", "third", "third", time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				limited, err = c.RecoveryToolBatches(ctx, a, "compile", current.TurnID, 1, 65536)
				if err != nil || !limited.Omitted || len(limited.Batches) != 1 || limited.Batches[0].ID != "second" {
					t.Fatalf("recovery count bound: %+v %v", limited, err)
				}
			case "complete":
				if err := c.CompleteToolCall(ctx, a, "compile", batch.ID, 1, 0, terminal); err != nil {
					t.Fatal(err)
				}
				if err := c.CompleteToolCall(ctx, a, "compile", batch.ID, 1, 0, terminal); !errors.Is(err, store.ErrContextConflict) {
					t.Fatalf("stale update: %v", err)
				}
				if err := s.CompleteInteraction(ctx, a, "compile", prepared.InteractionID, prepared.TurnID, "answer"); err != nil {
					t.Fatal(err)
				}
				batches, err := c.ToolBatches(ctx, a, "compile", prepared.TurnID, 5)
				if err != nil || len(batches) != 1 || batches[0].Version != 2 || batches[0].Calls[1].Status != "pending" {
					t.Fatalf("partial completion lost: %+v %v", batches, err)
				}
				batches[0].Calls[0].Input[0] = 'x'
				again, err := c.ToolBatches(ctx, a, "compile", prepared.TurnID, 5)
				if err != nil || !json.Valid(again[0].Calls[0].Input) {
					t.Fatal("returned mutable storage")
				}
				next, err := s.PrepareInteraction(ctx, a, "compile", "next", "next", time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				recovered, err := c.RecoveryToolBatches(ctx, a, "compile", next.TurnID, 16, 65536)
				if err != nil || len(recovered.Batches) != 0 || recovered.Omitted {
					t.Fatalf("completed history duplicated in recovery: %+v %v", recovered, err)
				}
			case "failed", "lease":
				if err := s.FailInteraction(ctx, a, "compile", prepared.InteractionID, prepared.TurnID); err != nil {
					t.Fatal(err)
				}
				if scenario == "lease" {
					if _, err := s.PrepareInteraction(ctx, a, "compile", "request", "hash", time.Hour); err != nil {
						t.Fatal(err)
					}
				}
				if err := c.CompleteToolCall(ctx, a, "compile", batch.ID, 1, 0, terminal); !errors.Is(err, store.ErrContextConflict) {
					t.Fatalf("stale worker wrote: %v", err)
				}
				batches, err := c.ToolBatches(ctx, a, "compile", prepared.TurnID, 5)
				if err != nil || len(batches) != 1 || batches[0].Calls[0].Status != "pending" {
					t.Fatalf("uncertain execution lost: %+v %v", batches, err)
				}
			case "quota":
				terminal.Output = json.RawMessage(`"` + strings.Repeat("x", store.MaxToolBatchBytes) + `"`)
				if err := c.CompleteToolCall(ctx, a, "compile", batch.ID, 1, 0, terminal); !errors.Is(err, store.ErrResourceExhausted) {
					t.Fatalf("oversized result accepted: %v", err)
				}
				batches, err := c.ToolBatches(ctx, a, "compile", prepared.TurnID, 5)
				if err != nil || batches[0].Version != 1 {
					t.Fatal("quota failure partially committed")
				}
			case "ownership":
				for _, other := range []security.AccessContext{security.MustAccessContext(security.Subject{TenantID: a.Subject.TenantID, UserID: "other", AuthMethod: "test"}), security.MustAccessContext(security.Subject{TenantID: "other", UserID: a.Subject.UserID, AuthMethod: "test"})} {
					if _, err := c.ToolBatches(ctx, other, "compile", prepared.TurnID, 5); err == nil {
						t.Fatal("ledger crossed ownership boundary")
					}
					if _, err := c.ContextTurns(ctx, other, "compile", prepared.TurnID, 1, 65536); err == nil {
						t.Fatal("snapshot crossed ownership boundary")
					}
					if _, err := c.RecoveryToolBatches(ctx, other, "compile", prepared.TurnID, 16, 65536); err == nil {
						t.Fatal("recovery crossed ownership boundary")
					}
					if err := c.CompleteToolCall(ctx, other, "compile", batch.ID, 1, 0, terminal); err == nil {
						t.Fatal("cross-owner write")
					}
				}
			case "snapshot":
				turns, err := c.ContextTurns(ctx, a, "compile", prepared.TurnID, 2, 65536)
				if err != nil || len(turns) != 1 || len(turns[0].Batches) != 1 || turns[0].ID != prepared.TurnID {
					t.Fatalf("snapshot: %+v %v", turns, err)
				}
				if _, err := c.ContextTurns(ctx, a, "compile", prepared.TurnID, 2, 1); !errors.Is(err, store.ErrResourceExhausted) {
					t.Fatalf("unbounded snapshot: %v", err)
				}
			case "bounded history":
				if err := s.AddHistoryToTurn(ctx, a, "compile", prepared.TurnID, ai.NewModelTextMessage(strings.Repeat("large evidence ", 1000))); err != nil {
					t.Fatal(err)
				}
				if err := s.CompleteInteraction(ctx, a, "compile", prepared.InteractionID, prepared.TurnID, "done"); err != nil {
					t.Fatal(err)
				}
				next, err := s.PrepareInteraction(ctx, a, "compile", "next", "next", time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				if err := s.AddHistoryToTurn(ctx, a, "compile", next.TurnID, ai.NewUserTextMessage("latest")); err != nil {
					t.Fatal(err)
				}
				turns, err := c.ContextTurns(ctx, a, "compile", next.TurnID, 2, 1024)
				if err != nil || len(turns) != 2 || !turns[0].ByteLimited || len(turns[0].Messages) != 0 || turns[1].ByteLimited || len(turns[1].Messages) != 1 {
					t.Fatalf("byte-limited history did not preserve active input: %+v %v", turns, err)
				}
			case "pin":
				ref, err := c.PinTemplate(ctx, a, "compile", "template@1:hash")
				if err != nil {
					t.Fatal(err)
				}
				again, err := c.PinTemplate(ctx, a, "compile", "template@2:hash")
				if err != nil || ref != again {
					t.Fatal("template pin changed")
				}
				stored, err := s.Get(ctx, a, "compile")
				if err != nil || stored.TemplateRef != ref {
					t.Fatal("pin not persisted")
				}
			case "concurrent":
				var wg sync.WaitGroup
				errs := make(chan error, 2)
				for range 2 {
					wg.Add(1)
					go func() { defer wg.Done(); errs <- c.CompleteToolCall(ctx, a, "compile", batch.ID, 1, 0, terminal) }()
				}
				wg.Wait()
				close(errs)
				success := 0
				for err := range errs {
					if err == nil {
						success++
					} else if !errors.Is(err, store.ErrContextConflict) {
						t.Fatal(err)
					}
				}
				if success != 1 {
					t.Fatalf("CAS winners: %d", success)
				}
			}
			if err := s.Delete(ctx, a, "compile"); err != nil {
				t.Fatal(err)
			}
			if _, err := c.ToolBatches(ctx, a, "compile", prepared.TurnID, 5); err == nil {
				t.Fatal("deleted ledger remains readable")
			}
			if _, err := c.RecoveryToolBatches(ctx, a, "compile", prepared.TurnID, 16, 65536); err == nil {
				t.Fatal("deleted recovery remains readable")
			}
			if err := c.CompleteToolCall(ctx, a, "compile", batch.ID, 1, 0, terminal); err == nil {
				t.Fatal("deleted session revived")
			}
		})
	}
}

func RunCompilationQuotaContractTests(t *testing.T, newStore func(store.QuotaPolicy) store.BackendStore) {
	t.Helper()
	for _, kind := range []string{"session", "tenant"} {
		t.Run("tool ledger "+kind+" quota rollback", func(t *testing.T) {
			quota := store.DefaultQuotaPolicy()
			quota.MaxSessionBytes = 1024
			quota.MaxTenantBytes = 4096
			if kind == "tenant" {
				quota.MaxSessionBytes = 4096
				quota.MaxTenantBytes = 1024
			}
			s := newStore(quota)
			c := s.(store.CompilationStore)
			ctx := context.Background()
			a := contractAccess
			now := time.Now()
			if err := s.Create(ctx, a, &store.Session{ID: "quota-ledger", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, Version: 1, Status: store.SessionStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
				t.Fatal(err)
			}
			p, err := s.PrepareInteraction(ctx, a, "quota-ledger", "request", "hash", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			b := store.ToolBatch{ID: "batch", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "quota-ledger", TurnID: p.TurnID, InteractionID: p.InteractionID, Sequence: 1, Version: 1, SchemaVersion: "v1", Calls: []store.ToolCall{{ID: "call", Name: "lookup", Input: json.RawMessage(`{}`), Status: "pending"}}}
			if err := c.CreateToolBatch(ctx, a, b); err != nil {
				t.Fatal(err)
			}
			call := b.Calls[0]
			call.Status = "succeeded"
			call.Output = json.RawMessage(`"` + strings.Repeat("x", 2000) + `"`)
			if err := c.CompleteToolCall(ctx, a, b.SessionID, b.ID, 1, 0, call); !errors.Is(err, store.ErrResourceExhausted) {
				t.Fatalf("ledger escaped %s quota: %v", kind, err)
			}
			call.Output = json.RawMessage(`{}`)
			if err := c.CompleteToolCall(ctx, a, b.SessionID, b.ID, 1, 0, call); err != nil {
				t.Fatalf("quota reservation leaked after rollback: %v", err)
			}
			rows, err := c.ToolBatches(ctx, a, b.SessionID, b.TurnID, 10)
			if err != nil || len(rows) != 1 || rows[0].Version != 2 {
				t.Fatalf("quota update corrupted ledger: %+v %v", rows, err)
			}
		})
	}
}
