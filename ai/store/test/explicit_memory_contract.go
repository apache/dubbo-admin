/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
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

	"dubbo-admin-ai/component/memory/summary"
	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	"dubbo-admin-ai/testutils"
	"github.com/firebase/genkit/go/ai"
)

func memoryWrite(settings store.MemorySettings) store.MemoryWrite {
	return store.MemoryWrite{Generation: settings.Generation, StateVersion: settings.StateVersion, ScopeType: "user", MemoryType: "preference", FactKey: "response.language", Value: "Chinese"}
}

func RunExplicitMemoryContractTests(t *testing.T, newStore func() store.BackendStore) {
	runMemoryMutationRaces(t, newStore)
	t.Run("expired records remain manageable", func(t *testing.T) {
		ctx := context.Background()
		a := contractAccess
		s := newStore()
		mem := s.(store.ExplicitMemoryStore)
		settings, err := mem.GetMemorySettings(ctx, a)
		if err != nil {
			t.Fatal(err)
		}
		expires := time.Now().Add(3 * time.Second)
		w := memoryWrite(settings)
		w.ExpiresAt = &expires
		if _, err := mem.SaveMemory(ctx, a, w); err != nil {
			t.Fatal(err)
		}
		live, _, err := mem.RecallMemories(ctx, a, store.MatchContext{}, 32, 16384)
		if err != nil || len(live) != 1 {
			t.Fatalf("record must be live after setup: %#v, %v", live, err)
		}
		testutils.WaitFor(t, 10*time.Second, "memory expiry", func() bool {
			items, _, err := mem.RecallMemories(ctx, a, store.MatchContext{}, 32, 16384)
			if err != nil {
				t.Fatal(err)
			}
			return !time.Now().Before(expires) && len(items) == 0
		})
		page, err := mem.ListMemories(ctx, a, "", 32)
		if err != nil || len(page.Records) != 1 || page.Records[0].ExpiresAt == nil {
			t.Fatalf("expired record missing from management list: %#v, %v", page, err)
		}
		items, _, err := mem.RecallMemories(ctx, a, store.MatchContext{}, 32, 16384)
		if err != nil || len(items) != 0 {
			t.Fatalf("expired record was recalled: %#v, %v", items, err)
		}
	})
	t.Run("expired source is hidden from management", func(t *testing.T) {
		ctx := context.Background()
		a := contractAccess
		s := newStore()
		mem := s.(store.ExplicitMemoryStore)
		now := time.Now()
		sourceExpiry := now.Add(3 * time.Second)
		if err := s.Create(ctx, a, &store.Session{ID: "expired-memory-source", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, Version: 1, Status: store.SessionStatusActive, CreatedAt: now, UpdatedAt: now, ExpiresAt: &sourceExpiry}); err != nil {
			t.Fatal(err)
		}
		settings, err := mem.GetMemorySettings(ctx, a)
		if err != nil {
			t.Fatal(err)
		}
		w := memoryWrite(settings)
		w.SourceSessionID = "expired-memory-source"
		if _, err := mem.SaveMemory(ctx, a, w); err != nil {
			t.Fatal(err)
		}
		live, _, err := mem.RecallMemories(ctx, a, store.MatchContext{}, 32, 16384)
		if err != nil || len(live) != 1 {
			t.Fatalf("source memory must be live after setup: %#v, %v", live, err)
		}
		testutils.WaitFor(t, 10*time.Second, "source session expiry", func() bool {
			_, err := s.Get(ctx, a, w.SourceSessionID)
			if err != nil && !errors.Is(err, store.ErrSessionExpired) {
				t.Fatal(err)
			}
			return errors.Is(err, store.ErrSessionExpired)
		})
		page, err := mem.ListMemories(ctx, a, "", 32)
		if err != nil || len(page.Records) != 1 || page.Records[0].Value != "" || page.Records[0].Status != "source_unavailable" {
			t.Fatalf("source-expired memory exposed with content: %#v, %v", page, err)
		}
		items, _, err := mem.RecallMemories(ctx, a, store.MatchContext{}, 32, 16384)
		if err != nil || len(items) != 0 {
			t.Fatalf("source-expired memory recalled: %#v, %v", items, err)
		}
	})
	for _, scenario := range []string{"identity", "lifecycle", "clear race", "source", "summary rebuild", "fragment invalidation", "summary commit race"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			a := contractAccess
			s := newStore()
			mem := s.(store.ExplicitMemoryStore)
			settings, err := mem.GetMemorySettings(ctx, a)
			if err != nil {
				t.Fatal(err)
			}
			w := memoryWrite(settings)
			r, err := mem.SaveMemory(ctx, a, w)
			if err != nil {
				t.Fatal(err)
			}
			duplicate, err := mem.SaveMemory(ctx, a, w)
			if err != nil || duplicate.ID != r.ID || duplicate.Version != r.Version {
				t.Fatal("non-idempotent save", duplicate, err)
			}
			for _, other := range []security.AccessContext{security.MustAccessContext(security.Subject{TenantID: a.Subject.TenantID, UserID: "someone-else", AuthMethod: "test", Scopes: []string{security.ScopeSessionAdmin}}), security.MustAccessContext(security.Subject{TenantID: "other-tenant", UserID: a.Subject.UserID, AuthMethod: "test"})} {
				page, err := mem.ListMemories(ctx, other, "", 32)
				if err != nil || len(page.Records) != 0 {
					t.Fatal("Memory crossed identity", page, err)
				}
				attack := w
				attack.ID = r.ID
				attack.ExpectedVersion = r.Version
				if _, err := mem.SaveMemory(ctx, other, attack); !errors.Is(err, store.ErrMemoryConflict) {
					t.Fatal("foreign update", err)
				}
			}
			if scenario == "identity" {
				for _, project := range []string{"A", "B"} {
					projectWrite := w
					projectWrite.MemoryType = "task_context"
					projectWrite.FactKey = "project.background"
					projectWrite.Applicability.ProjectRef = project
					projectWrite.Value = "background for " + project
					if _, err := mem.SaveMemory(ctx, a, projectWrite); err != nil {
						t.Fatal(err)
					}
				}
				items, _, err := mem.RecallMemories(ctx, a, store.MatchContext{ProjectRef: "A"}, 32, 16384)
				if err != nil || len(items) != 2 {
					t.Fatal("project matching", items, err)
				}
				plain, _, err := mem.RecallMemories(ctx, a, store.MatchContext{}, 32, 16384)
				if err != nil || len(plain) != 1 {
					t.Fatal("implicit project inheritance", plain, err)
				}
				for _, key := range []string{"diagnostic.priority", "diagnostic.read_only"} {
					v := w
					v.MemoryType = "diagnostic_constraint"
					v.FactKey = key
					v.Value = "true"
					v.Applicability = store.MatchContext{ProjectRef: "A", TaskType: "diagnosis"}
					if _, err := mem.SaveMemory(ctx, a, v); err != nil {
						t.Fatal(err)
					}
				}
				page, err := mem.ListMemories(ctx, a, "", 1)
				if err != nil || len(page.Records) != 1 || page.NextCursor == "" {
					t.Fatal("pagination", page, err)
				}
				next, err := mem.ListMemories(ctx, a, page.NextCursor, 32)
				if err != nil || len(next.Records) != 4 {
					t.Fatal("pagination suffix", next, err)
				}
				w.ID = r.ID
				w.ExpectedVersion = r.Version
				w.Value = "English"
				updated, err := mem.SaveMemory(ctx, a, w)
				if err != nil || updated.ID != r.ID || updated.Version != 2 || updated.CanonicalKey != r.CanonicalKey {
					t.Fatal("replacement", updated, err)
				}
				if _, err := mem.SaveMemory(ctx, a, w); !errors.Is(err, store.ErrMemoryConflict) {
					t.Fatal("stale update", err)
				}
				w.ExpectedVersion = updated.Version
				w.FactKey = "response.style"
				w.Value = "brief"
				if _, err := mem.SaveMemory(ctx, a, w); err != nil {
					t.Fatal(err)
				}
				unknown := memoryWrite(settings)
				unknown.ScopeType = "resource"
				if _, err := mem.SaveMemory(ctx, a, unknown); !errors.Is(err, store.ErrMemoryInvalid) {
					t.Fatal("scope", err)
				}
				unknown = memoryWrite(settings)
				unknown.Value = "password=super-secret-value"
				if _, err := mem.SaveMemory(ctx, a, unknown); !errors.Is(err, store.ErrMemoryInvalid) {
					t.Fatal("secret", err)
				}
				return
			}
			if scenario == "lifecycle" {
				mutation := store.MemoryMutation{Generation: settings.Generation, StateVersion: settings.StateVersion, Version: r.Version}
				if err := mem.DeleteMemory(ctx, a, r.ID, mutation); err != nil {
					t.Fatal(err)
				}
				w.ID = r.ID
				w.ExpectedVersion = r.Version
				if _, err := mem.SaveMemory(ctx, a, w); !errors.Is(err, store.ErrMemoryConflict) {
					t.Fatal("deleted record revived", err)
				}
				w = memoryWrite(settings)
				r, err = mem.SaveMemory(ctx, a, w)
				if err != nil {
					t.Fatal(err)
				}
				disabled := false
				mutation.Enabled = &disabled
				off, err := mem.ChangeMemorySettings(ctx, a, mutation, false)
				if err != nil {
					t.Fatal(err)
				}
				items, _, err := mem.RecallMemories(ctx, a, store.MatchContext{}, 32, 16384)
				if err != nil || len(items) != 0 {
					t.Fatal("disabled injected", items, err)
				}
				if _, err := mem.SaveMemory(ctx, a, w); !errors.Is(err, store.ErrMemoryConflict) {
					t.Fatal("old state write", err)
				}
				w.StateVersion = off.StateVersion
				if _, err := mem.SaveMemory(ctx, a, w); !errors.Is(err, store.ErrMemoryDisabled) {
					t.Fatal("disabled write", err)
				}
				return
			}
			if scenario == "clear race" {
				var wg sync.WaitGroup
				start := make(chan struct{})
				errs := make(chan error, 2)
				wg.Add(2)
				go func() {
					defer wg.Done()
					<-start
					_, err := mem.ChangeMemorySettings(ctx, a, store.MemoryMutation{Generation: settings.Generation, StateVersion: settings.StateVersion}, true)
					errs <- err
				}()
				go func() {
					defer wg.Done()
					<-start
					v := w
					v.FactKey = "response.style"
					_, err := mem.SaveMemory(ctx, a, v)
					errs <- err
				}()
				close(start)
				wg.Wait()
				close(errs)
				for err := range errs {
					if err != nil && !errors.Is(err, store.ErrMemoryConflict) {
						t.Fatal(err)
					}
				}
				page, err := mem.ListMemories(ctx, a, "", 32)
				if err != nil || len(page.Records) != 0 || page.Generation != settings.Generation+1 {
					t.Fatal("clear lost", page, err)
				}
				if _, err := mem.SaveMemory(ctx, a, w); !errors.Is(err, store.ErrMemoryConflict) {
					t.Fatal("old generation committed", err)
				}
				w.Generation = page.Generation
				if _, err := mem.SaveMemory(ctx, a, w); err != nil {
					t.Fatal("fresh generation blocked", err)
				}
				return
			}
			if err := s.Create(ctx, a, &store.Session{ID: "memory-source", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, Version: 1, Status: store.SessionStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
				t.Fatal(err)
			}
			dep := store.MemoryDependency{ID: r.ID, Version: r.Version, Generation: settings.Generation, StateVersion: settings.StateVersion}
			if scenario == "source" {
				v := w
				v.FactKey = "response.style"
				v.SourceSessionID = "memory-source"
				r, err := mem.SaveMemory(ctx, a, v)
				if err != nil {
					t.Fatal(err)
				}
				if err := s.Delete(ctx, a, "memory-source"); err != nil {
					t.Fatal(err)
				}
				items, _, err := mem.RecallMemories(ctx, a, store.MatchContext{}, 32, 16384)
				if err != nil || len(items) != 1 {
					t.Fatal("deleted source readable", items, err)
				}
				if _, err := mem.SaveMemory(ctx, a, v); err == nil {
					t.Fatal("deleted source writable")
				}
				artifacts, err := s.ListDerivedArtifactsForCleanup(ctx, security.SystemAccessContext(), time.Now(), 32)
				if err != nil || len(artifacts) != 1 {
					t.Fatal("missing source relationship", artifacts, err)
				}
				if err := s.(store.ExplicitMemoryMaintenanceStore).DeleteSourceMemory(ctx, security.SystemAccessContext(), artifacts[0]); err != nil {
					t.Fatal(err)
				}
				v.ID = r.ID
				v.ExpectedVersion = r.Version
				v.SourceSessionID = ""
				if _, err := mem.SaveMemory(ctx, a, v); !errors.Is(err, store.ErrMemoryConflict) {
					t.Fatal("cleaned record revived", err)
				}
				return
			}
			first, err := s.PrepareInteraction(ctx, a, "memory-source", "first", "first", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			content := "answer repeats obsolete-memory-value"
			if scenario == "fragment invalidation" {
				content = strings.Repeat(content, 100)
			}
			if err := s.AddHistoryToTurn(ctx, a, "memory-source", first.TurnID, ai.NewUserTextMessage("independent early goal"), ai.NewModelTextMessage(content)); err != nil {
				t.Fatal(err)
			}
			if err := mem.RecordTurnMemoryDependencies(ctx, a, "memory-source", first.TurnID, []store.MemoryDependency{dep}); err != nil {
				t.Fatal(err)
			}
			if err := s.CompleteInteraction(ctx, a, "memory-source", first.InteractionID, first.TurnID, "done"); err != nil {
				t.Fatal(err)
			}
			active, err := s.PrepareInteraction(ctx, a, "memory-source", "active", "active", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			backend := s.(store.SummaryStore)
			generated := 0
			generator := summary.GenerateFunc(func(_ context.Context, req summary.GenerateRequest) (string, error) {
				generated++
				if scenario == "summary commit race" && generated == 1 {
					if err := mem.DeleteMemory(ctx, a, r.ID, store.MemoryMutation{Generation: settings.Generation, StateVersion: settings.StateVersion, Version: r.Version}); err != nil {
						t.Fatal(err)
					}
				}
				payload := req.Previous
				if strings.Contains(req.Batch.Text, "obsolete-memory-value") {
					payload.Goals = []store.SummaryStatement{{Text: "obsolete-memory-value", Sources: []string{req.Batch.Source}}}
				}
				b, _ := json.Marshal(payload)
				return string(b), nil
			})
			cfg := summary.Config{Enabled: true, Model: "test", FragmentBytes: 8192, MaxCalls: 1, RecoveryMaxCalls: 1, TriggerHeadroomTurns: 1}
			if scenario == "fragment invalidation" {
				cfg.FragmentBytes = 256
			}
			result, err := (summary.Service{Store: backend, Generator: generator, Config: cfg}).Advance(ctx, a, "memory-source", active.InteractionID, active.TurnID, 1)
			if scenario == "summary commit race" {
				if err != nil || result.Summary != nil || result.Reason != "SUMMARY_COMMIT_CONFLICT" {
					t.Fatal("concurrent deletion allowed stale commit", result, err)
				}
				result, err = (summary.Service{Store: backend, Generator: generator, Config: cfg}).Advance(ctx, a, "memory-source", active.InteractionID, active.TurnID, 1)
				if err != nil || result.Summary == nil || result.Summary.Covered == 0 || len(result.Summary.Dependencies) != 0 {
					t.Fatal("recovery after stale commit", result, err)
				}
				data, _ := json.Marshal(result.Summary)
				if strings.Contains(string(data), "obsolete-memory-value") {
					t.Fatal("race resurrected Memory")
				}
				return
			}
			if err != nil || result.Summary == nil || len(result.Summary.Dependencies) != 1 {
				t.Fatal("dependencies not propagated", result, err)
			}
			if scenario == "fragment invalidation" && result.Summary.Progress == nil {
				t.Fatal("missing partial progress")
			}
			if err := mem.DeleteMemory(ctx, a, r.ID, store.MemoryMutation{Generation: settings.Generation, StateVersion: settings.StateVersion, Version: r.Version}); err != nil {
				t.Fatal(err)
			}
			if err := mem.ValidateMemoryDependencies(ctx, a, "memory-source", []store.MemoryDependency{dep}); !errors.Is(err, store.ErrMemoryDependency) {
				t.Fatal("stale dependency accepted", err)
			}
			invalid, err := backend.GetSummary(ctx, a, "memory-source")
			if err != nil || invalid.Covered != 0 || invalid.Progress != nil || invalid.Version != result.Summary.Version {
				t.Fatal("invalid summary kept coverage or lost CAS", invalid, err)
			}
			cfg.FragmentBytes = 8192
			result, err = (summary.Service{Store: backend, Generator: generator, Config: cfg}).Advance(ctx, a, "memory-source", active.InteractionID, active.TurnID, 1)
			if err != nil || result.Summary == nil || result.Summary.Covered == 0 || len(result.Summary.Dependencies) != 0 {
				t.Fatal("rebuild failed", result, err)
			}
			data, _ := json.Marshal(result.Summary)
			if strings.Contains(string(data), "obsolete-memory-value") {
				t.Fatal("summary resurrected deleted Memory", string(data))
			}
			history, err := s.AllMemory(ctx, a, "memory-source")
			if err != nil {
				t.Fatal(err)
			}
			data, _ = json.Marshal(history)
			if !strings.Contains(string(data), "obsolete-memory-value") {
				t.Fatal("canonical history was rewritten")
			}
		})
	}
}

func runMemoryMutationRaces(t *testing.T, newStore func() store.BackendStore) {
	for _, scenario := range []string{"source deletion race", "record deletion race", "disable race"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			a := contractAccess
			s := newStore()
			mem := s.(store.ExplicitMemoryStore)
			w := memoryWrite(store.MemorySettings{Generation: 1, StateVersion: 1, Enabled: true})
			r, err := mem.SaveMemory(ctx, a, w)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "source deletion race" {
				if err := s.Create(ctx, a, &store.Session{ID: "racing-source", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, Status: store.SessionStatusActive, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
					t.Fatal(err)
				}
				w.FactKey = "response.style"
				w.SourceSessionID = "racing-source"
			} else if scenario == "record deletion race" {
				w.ID = r.ID
				w.ExpectedVersion = r.Version
				w.Value = "English"
			} else {
				w.FactKey = "response.style"
			}
			start := make(chan struct{})
			results := make(chan error, 2)
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); <-start; _, err := mem.SaveMemory(ctx, a, w); results <- err }()
			go func() {
				defer wg.Done()
				<-start
				var err error
				switch scenario {
				case "source deletion race":
					err = s.Delete(ctx, a, "racing-source")
				case "record deletion race":
					err = mem.DeleteMemory(ctx, a, r.ID, store.MemoryMutation{Generation: 1, StateVersion: 1, Version: 1})
				case "disable race":
					enabled := false
					_, err = mem.ChangeMemorySettings(ctx, a, store.MemoryMutation{Generation: 1, StateVersion: 1, Enabled: &enabled}, false)
				}
				results <- err
			}()
			close(start)
			wg.Wait()
			close(results)
			for err := range results {
				if err != nil && !errors.Is(err, store.ErrMemoryConflict) && !errors.Is(err, store.ErrMemoryDisabled) && !errors.Is(err, store.ErrSessionNotFound) && !errors.Is(err, security.ErrResourceNotAccessible) {
					t.Fatal(err)
				}
			}
			items, settings, err := mem.RecallMemories(ctx, a, store.MatchContext{}, 32, 16384)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "source deletion race":
				if len(items) != 1 || items[0].ID != r.ID {
					t.Fatal("source race left readable Memory", items)
				}
			case "record deletion race":
				if len(items) > 1 || len(items) == 1 && (items[0].Value != "English" || items[0].Version != 2) {
					t.Fatal("delete/update race restored old value", items)
				}
			case "disable race":
				if settings.Enabled || len(items) != 0 {
					t.Fatal("disable race injected Memory", items, settings)
				}
				if _, err := mem.SaveMemory(ctx, a, w); !errors.Is(err, store.ErrMemoryConflict) {
					t.Fatal("old state request committed", err)
				}
			}
		})
	}
}
