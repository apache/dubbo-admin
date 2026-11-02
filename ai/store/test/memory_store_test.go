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
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"

	"github.com/firebase/genkit/go/ai"
)

func TestMemoryStore_Contract(t *testing.T) {
	RunStoreContractTests(t, func() conversationstore.BackendStore {
		return memorystore.NewMemoryStore(3)
	}, 3)
}

func TestMemoryStore_QuotaContract(t *testing.T) {
	RunQuotaContractTests(t, func(quota conversationstore.QuotaPolicy) conversationstore.BackendStore {
		return memorystore.NewMemoryStoreWithPolicy(quota)
	})
}

func TestMemoryStore_UsageAuditContract(t *testing.T) {
	RunUsageAuditContractTests(t, func() conversationstore.UsageAuditMaintenanceStore { return memorystore.NewMemoryStore(3) })
}

func TestMemoryUsageAuditCleanupBoundsScanningAndAdvances(t *testing.T) {
	for _, kind := range []string{"usage", "audit"} {
		t.Run(kind, func(t *testing.T) {
			s := memorystore.NewMemoryStore()
			s.SetUsageAuditPolicy(conversationstore.UsageAuditPolicy{MaxUsageEvents: 4, MaxAuditEvents: 4, CleanupBatchSize: 1})
			ctx := security.WithAccessContext(context.Background(), contractAccess)
			cleanup := s.CleanupUsage
			if kind == "audit" {
				cleanup = s.CleanupAudit
			}
			record := func(id string, at time.Time) error {
				if kind == "audit" {
					return s.RecordAudit(ctx, conversationstore.AuditEvent{ID: id, TenantID: contractAccess.Subject.TenantID, CreatedAt: at})
				}
				return s.RecordUsage(ctx, conversationstore.UsageEvent{ID: id, TenantID: contractAccess.Subject.TenantID, CreatedAt: at})
			}
			for i := 0; i < 4; i++ {
				at := time.Now()
				if i == 3 {
					at = at.Add(-40 * 24 * time.Hour)
				}
				if err := record(fmt.Sprint(i), at); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 4; i++ {
				n, err := cleanup(ctx, security.SystemAccessContext(), time.Now(), 100)
				want := 0
				if i == 3 {
					want = 1
				}
				if err != nil || n != want {
					t.Fatalf("batch %d removed %d, %v; want %d", i, n, err, want)
				}
			}
			if err := record("replacement", time.Now()); err != nil {
				t.Fatalf("deleted record retained capacity: %v", err)
			}
		})
	}
}

func newTestSession(id string, updatedAt time.Time) *conversationstore.Session {
	return &conversationstore.Session{
		ID:        id,
		TenantID:  contractAccess.Subject.TenantID,
		OwnerID:   contractAccess.Subject.UserID,
		Version:   1,
		CreatedAt: updatedAt,
		UpdatedAt: updatedAt,
		Status:    "active",
	}
}

func createSession(t *testing.T, s *memorystore.MemoryStore, id string) {
	t.Helper()
	now := time.Now()
	if err := s.Create(context.Background(), contractAccess, newTestSession(id, now)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestMemoryStore_SessionLifecycle(t *testing.T) {
	ctx := context.Background()
	s := memorystore.NewMemoryStore()
	now := time.Now()
	session := newTestSession("session-1", now)

	if err := s.Create(ctx, contractAccess, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	got, err := s.Get(ctx, contractAccess, session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != session.ID || got.Status != "active" {
		t.Fatalf("Get() = %#v, want %#v", got, session)
	}

	touched := now.Add(time.Minute)
	if err := s.Touch(ctx, contractAccess, session.ID, touched); err != nil {
		t.Fatalf("Touch() error = %v", err)
	}
	got, err = s.Get(ctx, contractAccess, session.ID)
	if err != nil {
		t.Fatalf("Get() after Touch error = %v", err)
	}
	if !got.UpdatedAt.Equal(touched) {
		t.Fatalf("UpdatedAt = %v, want %v", got.UpdatedAt, touched)
	}
	if got.Version != 2 || got.ExpiresAt == nil || !got.ExpiresAt.Equal(touched.Add(24*time.Hour)) {
		t.Fatalf("Touch() version/expires = %d/%v", got.Version, got.ExpiresAt)
	}
	*got.ExpiresAt = time.Time{}
	storedAgain, err := s.Get(ctx, contractAccess, session.ID)
	if err != nil || storedAgain.ExpiresAt == nil || storedAgain.ExpiresAt.IsZero() {
		t.Fatalf("Get() returned a mutable Store snapshot: %#v, %v", storedAgain, err)
	}

	listed, err := s.List(ctx, contractAccess)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != session.ID {
		t.Fatalf("List() = %#v, want one session", listed)
	}

	if err := s.Delete(ctx, contractAccess, session.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := s.Get(ctx, contractAccess, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
		t.Fatalf("Get() after Delete error = %v, want ErrSessionNotFound", err)
	}
	if err := s.Delete(ctx, contractAccess, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
		t.Fatalf("second Delete() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_Expiration(t *testing.T) {
	ctx := context.Background()
	s := memorystore.NewMemoryStore()
	now := time.Now()
	old := newTestSession("expired", now.Add(-25*time.Hour))
	active := newTestSession("active", now)
	if err := s.Create(ctx, contractAccess, old); err != nil {
		t.Fatalf("Create(expired) error = %v", err)
	}
	if err := s.Create(ctx, contractAccess, active); err != nil {
		t.Fatalf("Create(active) error = %v", err)
	}

	if _, err := s.Get(ctx, contractAccess, old.ID); !errors.Is(err, conversationstore.ErrSessionExpired) {
		t.Fatalf("Get(expired) error = %v, want ErrSessionExpired", err)
	}
	listed, err := s.List(ctx, contractAccess)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != active.ID {
		t.Fatalf("List() = %#v, want active session only", listed)
	}

	deleted, err := s.DeleteExpired(ctx, security.SystemAccessContext(), now)
	if err != nil {
		t.Fatalf("DeleteExpired() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("DeleteExpired() = %d, want 1", deleted)
	}
}

func TestMemoryStore_CreateAndTouchValidation(t *testing.T) {
	ctx := context.Background()
	s := memorystore.NewMemoryStore()
	now := time.Now()

	for name, session := range map[string]*conversationstore.Session{
		"missing created_at": newTestSession("missing-created", time.Time{}),
		"missing updated_at": {
			ID:        "missing-updated",
			CreatedAt: now,
			Status:    "active",
		},
		"missing status": {
			ID:        "missing-status",
			CreatedAt: now,
			UpdatedAt: now,
		},
	} {
		if name == "missing created_at" {
			session.UpdatedAt = now
		}
		if err := s.Create(ctx, contractAccess, session); err == nil {
			t.Errorf("Create(%s) succeeded, want validation error", name)
		}
	}

	closed := newTestSession("closed", now)
	closed.Status = "closed"
	if err := s.Create(ctx, contractAccess, closed); err != nil {
		t.Fatalf("Create(closed) error = %v", err)
	}
	if err := s.Touch(ctx, contractAccess, closed.ID, now.Add(time.Minute)); err == nil {
		t.Fatal("Touch(closed) succeeded, want an error")
	}
}

func TestMemoryStore_HistoryOrderingAndFiltering(t *testing.T) {
	ctx := context.Background()
	s := memorystore.NewMemoryStore()
	createSession(t, s, "session-1")

	first := ai.NewUserMessage(ai.NewTextPart("first"))
	if err := s.AddHistory(ctx, contractAccess, "session-1",
		ai.NewSystemMessage(ai.NewTextPart("system")),
		first,
		ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart("model")),
		nil,
		ai.NewMessage(ai.RoleTool, nil, ai.NewTextPart("ignored")),
	); err != nil {
		t.Fatalf("AddHistory() error = %v", err)
	}

	window, err := s.WindowMemory(ctx, contractAccess, "session-1")
	if err != nil {
		t.Fatalf("WindowMemory() error = %v", err)
	}
	if len(window) != 3 || window[0].Role != ai.RoleSystem || window[1].Role != ai.RoleUser || window[2].Role != ai.RoleModel {
		t.Fatalf("WindowMemory() roles = %#v, want system/user/model", roles(window))
	}

	if err := s.NextTurn(ctx, contractAccess, "session-1"); err != nil {
		t.Fatalf("NextTurn() error = %v", err)
	}
	if err := s.AddHistory(ctx, contractAccess, "session-1", ai.NewUserMessage(ai.NewTextPart("second"))); err != nil {
		t.Fatalf("second AddHistory() error = %v", err)
	}
	all, err := s.AllMemory(ctx, contractAccess, "session-1")
	if err != nil {
		t.Fatalf("AllMemory() error = %v", err)
	}
	if len(all) != 4 || all[0].Content[0].Text != "second" || all[3].Content[0].Text != "model" {
		t.Fatalf("AllMemory() = %#v, want active turn before committed turn", all)
	}

	first.Content[0].Text = "mutated"
	window, err = s.WindowMemory(ctx, contractAccess, "session-1")
	if err != nil {
		t.Fatalf("WindowMemory() after caller mutation error = %v", err)
	}
	if window[0].Content[0].Text != "second" {
		t.Fatalf("stored history was unexpectedly changed by caller mutation")
	}
}

func TestMemoryStore_TurnLimitAndNoActiveTurn(t *testing.T) {
	ctx := context.Background()
	s := memorystore.NewMemoryStore(3)
	createSession(t, s, "session-1")

	if err := s.NextTurn(ctx, contractAccess, "session-1"); !errors.Is(err, conversationstore.ErrNoActiveTurn) {
		t.Fatalf("NextTurn() without active turn = %v, want ErrNoActiveTurn", err)
	}
	if err := s.AddHistory(ctx, contractAccess, "session-1", ai.NewUserMessage(ai.NewTextPart("one"))); err != nil {
		t.Fatalf("AddHistory() error = %v", err)
	}
	if err := s.NextTurn(ctx, contractAccess, "session-1"); err != nil {
		t.Fatalf("first NextTurn() error = %v", err)
	}
	if err := s.AddHistory(ctx, contractAccess, "session-1", ai.NewUserMessage(ai.NewTextPart("two"))); err != nil {
		t.Fatalf("second AddHistory() error = %v", err)
	}
	if err := s.NextTurn(ctx, contractAccess, "session-1"); err != nil {
		t.Fatalf("second NextTurn() error = %v", err)
	}
	if err := s.AddHistory(ctx, contractAccess, "session-1", ai.NewUserMessage(ai.NewTextPart("three"))); err != nil {
		t.Fatalf("third AddHistory() error = %v", err)
	}
	if err := s.NextTurn(ctx, contractAccess, "session-1"); err != nil {
		t.Fatalf("final NextTurn() = %v, want success", err)
	}

	if err := s.AddHistory(ctx, contractAccess, "session-1", ai.NewUserMessage(ai.NewTextPart("overflow"))); !errors.Is(err, conversationstore.ErrTurnLimitReached) {
		t.Fatalf("overflow AddHistory() = %v, want ErrTurnLimitReached", err)
	}

	window, err := s.WindowMemory(ctx, contractAccess, "session-1")
	if err != nil {
		t.Fatalf("WindowMemory() after rejected turn error = %v", err)
	}
	if len(window) != 0 {
		t.Fatalf("active turn after rejected AddHistory = %#v, want empty", window)
	}
}

func TestMemoryStore_ConcurrentHistoryWrites(t *testing.T) {
	ctx := context.Background()
	s := memorystore.NewMemoryStore()
	createSession(t, s, "session-1")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := s.AddHistory(ctx, contractAccess, "session-1", ai.NewUserMessage(ai.NewTextPart(fmt.Sprintf("message-%d", i)))); err != nil {
				t.Errorf("AddHistory(%d) error = %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	window, err := s.WindowMemory(ctx, contractAccess, "session-1")
	if err != nil {
		t.Fatalf("WindowMemory() error = %v", err)
	}
	if len(window) != 100 {
		t.Fatalf("WindowMemory() length = %d, want 100", len(window))
	}
}

func roles(messages []*ai.Message) []ai.Role {
	result := make([]ai.Role, len(messages))
	for i, message := range messages {
		result[i] = message.Role
	}
	return result
}
