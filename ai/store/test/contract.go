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
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"

	"github.com/firebase/genkit/go/ai"
)

var contractAccess = security.MustAccessContext(security.Subject{
	TenantID: "tenant-a", UserID: "user-a", AuthMethod: security.AuthMethodStatic,
})

type authorizedStore struct {
	conversationstore.BackendStore
	access security.AccessContext
}

func withAccess(store conversationstore.BackendStore) *authorizedStore {
	return &authorizedStore{BackendStore: store, access: contractAccess}
}

func (s *authorizedStore) Create(ctx context.Context, session *conversationstore.Session) error {
	if session.TenantID == "" {
		session.TenantID = s.access.Subject.TenantID
	}
	if session.OwnerID == "" {
		session.OwnerID = s.access.Subject.UserID
	}
	if session.Version == 0 {
		session.Version = 1
	}
	return s.BackendStore.Create(ctx, s.access, session)
}

func (s *authorizedStore) Get(ctx context.Context, id string) (*conversationstore.Session, error) {
	return s.BackendStore.Get(ctx, s.access, id)
}
func (s *authorizedStore) List(ctx context.Context) ([]*conversationstore.Session, error) {
	return s.BackendStore.List(ctx, s.access)
}
func (s *authorizedStore) Touch(ctx context.Context, id string, at time.Time) error {
	return s.BackendStore.Touch(ctx, s.access, id, at)
}
func (s *authorizedStore) Delete(ctx context.Context, id string) error {
	return s.BackendStore.Delete(ctx, s.access, id)
}
func (s *authorizedStore) DeleteExpired(ctx context.Context, at time.Time) (int, error) {
	return s.BackendStore.DeleteExpired(ctx, security.SystemAccessContext(), at)
}
func (s *authorizedStore) BeginTurn(ctx context.Context, id string) (uint64, error) {
	return s.BackendStore.BeginTurn(ctx, s.access, id)
}
func (s *authorizedStore) AddHistoryToTurn(ctx context.Context, id string, turn uint64, messages ...*ai.Message) error {
	return s.BackendStore.AddHistoryToTurn(ctx, s.access, id, turn, messages...)
}
func (s *authorizedStore) IsTurnEmpty(ctx context.Context, id string, turn uint64) (bool, error) {
	return s.BackendStore.IsTurnEmpty(ctx, s.access, id, turn)
}
func (s *authorizedStore) WindowMemoryForTurn(ctx context.Context, id string, turn uint64) ([]*ai.Message, error) {
	return s.BackendStore.WindowMemoryForTurn(ctx, s.access, id, turn)
}
func (s *authorizedStore) ContextWindowForTurn(ctx context.Context, id string, turn uint64, completedLimit int) ([]*ai.Message, error) {
	return s.BackendStore.ContextWindowForTurn(ctx, s.access, id, turn, completedLimit)
}
func (s *authorizedStore) AllMemory(ctx context.Context, id string) ([]*ai.Message, error) {
	return s.BackendStore.AllMemory(ctx, s.access, id)
}
func (s *authorizedStore) NextTurnForTurn(ctx context.Context, id string, turn uint64) error {
	return s.BackendStore.NextTurnForTurn(ctx, s.access, id, turn)
}
func (s *authorizedStore) AbortTurnForTurn(ctx context.Context, id string, turn uint64) error {
	return s.BackendStore.AbortTurnForTurn(ctx, s.access, id, turn)
}

// RunStoreContractTests verifies behavior shared by every Store backend.
// Backend-specific tests should call this function with their own constructor.
func RunStoreContractTests(t *testing.T, newStore func() conversationstore.BackendStore, limit int) {
	t.Helper()
	if limit <= 0 {
		t.Fatalf("contract test limit must be positive, got %d", limit)
	}
	begin := func(t *testing.T, s *authorizedStore, ctx context.Context, sessionID string) uint64 {
		t.Helper()
		turnID, err := s.BeginTurn(ctx, sessionID)
		if err != nil {
			t.Fatalf("BeginTurn() error = %v", err)
		}
		return turnID
	}

	t.Run("ownership isolation", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		owner := security.MustAccessContext(security.Subject{TenantID: "tenant-a", UserID: "owner", AuthMethod: security.AuthMethodStatic})
		otherUser := security.MustAccessContext(security.Subject{TenantID: "tenant-a", UserID: "other", AuthMethod: security.AuthMethodStatic})
		otherTenant := security.MustAccessContext(security.Subject{TenantID: "tenant-b", UserID: "owner", AuthMethod: security.AuthMethodStatic})
		admin := security.MustAccessContext(security.Subject{
			TenantID: "tenant-a", UserID: "admin", Scopes: []string{security.ScopeSessionAdmin}, AuthMethod: security.AuthMethodStatic,
		})
		now := time.Now()
		expiresAt := now.Add(time.Hour)
		session := &conversationstore.Session{
			ID: "owned-session", TenantID: "tenant-a", OwnerID: "owner", Version: 1,
			CreatedAt: now, UpdatedAt: now, ExpiresAt: &expiresAt, Status: "active",
		}
		if err := s.Create(ctx, owner, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		for name, access := range map[string]security.AccessContext{"other user": otherUser, "other tenant": otherTenant} {
			t.Run(name, func(t *testing.T) {
				if _, err := s.Get(ctx, access, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
					t.Fatalf("Get() error = %v, want ErrSessionNotFound", err)
				}
				listed, err := s.List(ctx, access)
				if err != nil || len(listed) != 0 {
					t.Fatalf("List() = %#v, %v, want empty", listed, err)
				}
				if _, err := s.BeginTurn(ctx, access, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
					t.Fatalf("BeginTurn() error = %v, want ErrSessionNotFound", err)
				}
				if err := s.Delete(ctx, access, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
					t.Fatalf("Delete() error = %v, want ErrSessionNotFound", err)
				}
			})
		}
		if _, err := s.Get(ctx, admin, session.ID); err != nil {
			t.Fatalf("same-tenant admin Get() error = %v", err)
		}
		system := security.SystemAccessContext()
		if _, err := s.Get(ctx, system, session.ID); !errors.Is(err, security.ErrResourceNotAccessible) {
			t.Fatalf("system Get() error = %v, want ErrResourceNotAccessible", err)
		}
		if _, err := s.List(ctx, system); !errors.Is(err, security.ErrResourceNotAccessible) {
			t.Fatalf("system List() error = %v, want ErrResourceNotAccessible", err)
		}
		if _, err := s.List(ctx, security.AccessContext{}); !errors.Is(err, security.ErrAuthenticationRequired) {
			t.Fatalf("unauthenticated List() error = %v, want ErrAuthenticationRequired", err)
		}
	})

	t.Run("session validation", func(t *testing.T) {
		now := time.Now()
		valid := func() *conversationstore.Session {
			return &conversationstore.Session{
				ID: "valid", TenantID: "tenant-a", OwnerID: "user-a", Version: 1,
				CreatedAt: now, UpdatedAt: now, Status: "active",
			}
		}
		tests := map[string]func() *conversationstore.Session{
			"nil":                func() *conversationstore.Session { return nil },
			"missing id":         func() *conversationstore.Session { s := valid(); s.ID = ""; return s },
			"missing tenant":     func() *conversationstore.Session { s := valid(); s.TenantID = ""; return s },
			"missing owner":      func() *conversationstore.Session { s := valid(); s.OwnerID = ""; return s },
			"missing version":    func() *conversationstore.Session { s := valid(); s.Version = 0; return s },
			"missing created_at": func() *conversationstore.Session { s := valid(); s.CreatedAt = time.Time{}; return s },
			"missing updated_at": func() *conversationstore.Session { s := valid(); s.UpdatedAt = time.Time{}; return s },
			"missing status":     func() *conversationstore.Session { s := valid(); s.Status = ""; return s },
		}
		for name, session := range tests {
			t.Run(name, func(t *testing.T) {
				if err := newStore().Create(context.Background(), security.SystemAccessContext(), session()); err == nil {
					t.Fatal("Create() succeeded, want shared model validation error")
				}
			})
		}
	})

	t.Run("maintenance operations require capability", func(t *testing.T) {
		s := newStore()
		forged := security.MustAccessContext(security.Subject{
			TenantID: "system", UserID: "system", AuthMethod: security.AuthMethodSystem,
		})
		if _, err := s.AdoptLegacySessions(context.Background(), forged, "tenant-a", "owner-a"); !errors.Is(err, security.ErrResourceNotAccessible) {
			t.Fatalf("AdoptLegacySessions() error = %v, want ErrResourceNotAccessible", err)
		}
		if _, err := s.DeleteExpired(context.Background(), forged, time.Now()); !errors.Is(err, security.ErrResourceNotAccessible) {
			t.Fatalf("DeleteExpired() error = %v, want ErrResourceNotAccessible", err)
		}
	})

	t.Run("session lifecycle and history deletion", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{
			ID:        "contract-session",
			CreatedAt: now,
			UpdatedAt: now,
			Status:    "active",
		}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserMessage(ai.NewTextPart("hello"))); err != nil {
			t.Fatalf("AddHistory() error = %v", err)
		}
		if err := s.Delete(ctx, session.ID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if _, err := s.Get(ctx, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
			t.Fatalf("Get() after Delete() error = %v, want ErrSessionNotFound", err)
		}
		if _, err := s.AllMemory(ctx, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
			t.Fatalf("AllMemory() after Delete() error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("message roles and turn order", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "order-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID,
			ai.NewUserMessage(ai.NewTextPart("first")),
			ai.NewSystemMessage(ai.NewTextPart("system")),
			ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart("model")),
			nil,
			ai.NewMessage(ai.RoleTool, nil, ai.NewTextPart("ignored")),
		); err != nil {
			t.Fatalf("AddHistory() error = %v", err)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
			t.Fatalf("NextTurn() error = %v", err)
		}
		turnID = begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserMessage(ai.NewTextPart("second"))); err != nil {
			t.Fatalf("second AddHistory() error = %v", err)
		}
		messages, err := s.AllMemory(ctx, session.ID)
		if err != nil {
			t.Fatalf("AllMemory() error = %v", err)
		}
		if len(messages) != 4 || messages[0].Content[0].Text != "second" || messages[1].Role != ai.RoleUser || messages[2].Role != ai.RoleSystem || messages[3].Role != ai.RoleModel {
			t.Fatalf("AllMemory() = %#v, want active turn followed by persisted message order", messages)
		}
	})

	t.Run("concurrent turns are rejected", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "concurrent-turn-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		first, err := s.BeginTurn(ctx, session.ID)
		if err != nil {
			t.Fatalf("BeginTurn(first) error = %v", err)
		}
		if _, err := s.BeginTurn(ctx, session.ID); !errors.Is(err, conversationstore.ErrSessionBusy) {
			t.Fatalf("BeginTurn(second) error = %v, want ErrSessionBusy", err)
		}
		if err := s.AddHistoryToTurn(ctx, session.ID, first, ai.NewUserTextMessage("first")); err != nil {
			t.Fatalf("AddHistoryToTurn(first) error = %v", err)
		}
		firstMessages, err := s.WindowMemoryForTurn(ctx, session.ID, first)
		if err != nil || len(firstMessages) != 1 || firstMessages[0].Content[0].Text != "first" {
			t.Fatalf("first turn messages = %#v, error = %v", firstMessages, err)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, first); err != nil {
			t.Fatalf("NextTurnForTurn(first) error = %v", err)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, first); !errors.Is(err, conversationstore.ErrTurnNotFound) {
			t.Fatalf("repeated NextTurnForTurn(first) error = %v, want ErrTurnNotFound", err)
		}
	})

	t.Run("idempotent interaction lifecycle", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "idempotent-session", CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		first, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "request-1", "hash-1", time.Minute)
		if err != nil {
			t.Fatalf("PrepareInteraction(first) error = %v", err)
		}
		if _, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "request-1", "hash-2", time.Minute); !errors.Is(err, conversationstore.ErrIdempotencyConflict) {
			t.Fatalf("PrepareInteraction(conflict) error = %v, want ErrIdempotencyConflict", err)
		}
		if _, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "request-1", "hash-1", time.Minute); !errors.Is(err, conversationstore.ErrInteractionInProgress) {
			t.Fatalf("PrepareInteraction(retry running) error = %v, want ErrInteractionInProgress", err)
		}
		if _, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "request-2", "hash-2", time.Minute); !errors.Is(err, conversationstore.ErrSessionBusy) {
			t.Fatalf("PrepareInteraction(concurrent) error = %v, want ErrSessionBusy", err)
		}
		if err := s.BackendStore.AddHistoryToTurn(ctx, s.access, session.ID, first.TurnID, ai.NewUserTextMessage("hello"), ai.NewModelTextMessage("answer")); err != nil {
			t.Fatalf("AddHistoryToTurn() error = %v", err)
		}
		if err := s.BackendStore.CompleteInteraction(ctx, s.access, session.ID, first.InteractionID, first.TurnID, "answer"); err != nil {
			t.Fatalf("CompleteInteraction() error = %v", err)
		}
		replayed, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "request-1", "hash-1", time.Minute)
		if err != nil || !replayed.Replayed || replayed.InteractionID != first.InteractionID || replayed.Result != "answer" {
			t.Fatalf("PrepareInteraction(replay) = %#v, %v", replayed, err)
		}

		failed, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "request-3", "hash-3", time.Minute)
		if err != nil {
			t.Fatalf("PrepareInteraction(failed attempt) error = %v", err)
		}
		if err := s.BackendStore.FailInteraction(ctx, s.access, session.ID, failed.InteractionID, failed.TurnID); err != nil {
			t.Fatalf("FailInteraction() error = %v", err)
		}
		retried, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "request-3", "hash-3", time.Minute)
		if err != nil || retried.InteractionID != failed.InteractionID || retried.TurnID == failed.TurnID {
			t.Fatalf("PrepareInteraction(retry failed) = %#v, %v", retried, err)
		}
		if err := s.BackendStore.FailInteraction(ctx, s.access, session.ID, retried.InteractionID, retried.TurnID); err != nil {
			t.Fatal(err)
		}
		expiring, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "request-4", "hash-4", time.Nanosecond)
		if err != nil {
			t.Fatal(err)
		}
		recovered, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "request-4", "hash-4", time.Minute)
		if err != nil || recovered.InteractionID != expiring.InteractionID || recovered.TurnID == expiring.TurnID {
			t.Fatalf("PrepareInteraction(expired lease) = %#v, %v", recovered, err)
		}
	})

	t.Run("event quota is atomic", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "event-quota-session", CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
		if err := s.Create(ctx, session); err != nil {
			t.Fatal(err)
		}
		turnID := begin(t, s, ctx, session.ID)
		messages := make([]*ai.Message, 1001)
		for i := range messages {
			messages[i] = ai.NewUserTextMessage("event")
		}
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, messages...); !errors.Is(err, conversationstore.ErrResourceExhausted) {
			t.Fatalf("AddHistoryToTurn(over quota) error = %v, want ErrResourceExhausted", err)
		}
		stored, err := s.WindowMemoryForTurn(ctx, session.ID, turnID)
		if err != nil && !errors.Is(err, conversationstore.ErrTurnNotFound) {
			t.Fatalf("WindowMemoryForTurn() error = %v", err)
		}
		if len(stored) != 0 {
			t.Fatalf("quota failure left %d partial messages", len(stored))
		}
	})

	t.Run("derived cleanup is reconcilable", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "derived-cleanup-session", CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
		if err := s.Create(ctx, session); err != nil {
			t.Fatal(err)
		}
		artifact := conversationstore.DerivedArtifact{ID: "artifact-1", TenantID: s.access.Subject.TenantID, CanonicalSourceID: "session:" + session.ID, Kind: "test", Locator: "test://artifact"}
		if err := s.BackendStore.RegisterDerivedArtifact(ctx, security.SystemAccessContext(), artifact); err != nil {
			t.Fatal(err)
		}
		if err := s.Delete(ctx, session.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Get(ctx, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
			t.Fatalf("Get() after tombstone = %v", err)
		}
		if cleaned, err := s.DeleteExpired(ctx, now.Add(time.Second)); err != nil || cleaned != 0 {
			t.Fatalf("blocked cleanup = %d, %v", cleaned, err)
		}
		pending, err := s.BackendStore.ListDerivedArtifactsForCleanup(ctx, security.SystemAccessContext(), now.Add(time.Second), 10)
		if err != nil || len(pending) != 1 || pending[0].ID != artifact.ID {
			t.Fatalf("ListDerivedArtifactsForCleanup() = %#v, %v", pending, err)
		}
		if err := s.BackendStore.MarkDerivedArtifactDeleted(ctx, security.SystemAccessContext(), artifact.ID); err != nil {
			t.Fatal(err)
		}
		if cleaned, err := s.DeleteExpired(ctx, now.Add(2*time.Second)); err != nil || cleaned != 1 {
			t.Fatalf("reconciled cleanup = %d, %v", cleaned, err)
		}
	})

	t.Run("failed derived artifact yields cleanup batch until retry", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		for _, id := range []string{"artifact-retry-session-a", "artifact-retry-session-b"} {
			session := &conversationstore.Session{ID: id, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
			if err := s.Create(ctx, session); err != nil {
				t.Fatal(err)
			}
			artifactID := strings.TrimPrefix(id, "artifact-retry-session-")
			artifact := conversationstore.DerivedArtifact{ID: "artifact-" + artifactID, TenantID: s.access.Subject.TenantID, CanonicalSourceID: "session:" + id, Kind: "test", Locator: "test://" + artifactID}
			if err := s.BackendStore.RegisterDerivedArtifact(ctx, security.SystemAccessContext(), artifact); err != nil {
				t.Fatal(err)
			}
			if err := s.Delete(ctx, id); err != nil {
				t.Fatal(err)
			}
		}
		if cleaned, err := s.BackendStore.DeleteExpired(ctx, security.SystemAccessContext(), now.Add(time.Second), 2); err != nil || cleaned != 0 {
			t.Fatalf("DeleteExpired(blocked artifacts) = %d, %v", cleaned, err)
		}

		failedAt := time.Now().Add(time.Second)
		first, err := s.BackendStore.ListDerivedArtifactsForCleanup(ctx, security.SystemAccessContext(), failedAt, 1)
		if err != nil || len(first) != 1 || first[0].ID != "artifact-a" {
			t.Fatalf("first artifact batch = %#v, %v", first, err)
		}
		if err := s.BackendStore.MarkDerivedArtifactCleanupFailed(ctx, security.SystemAccessContext(), first[0].ID, "CLEANUP_FAILED", failedAt); err != nil {
			t.Fatal(err)
		}

		second, err := s.BackendStore.ListDerivedArtifactsForCleanup(ctx, security.SystemAccessContext(), failedAt.Add(time.Second), 1)
		if err != nil || len(second) != 1 || second[0].ID != "artifact-b" {
			t.Fatalf("second artifact batch = %#v, %v; failed artifact blocked progress", second, err)
		}
		if err := s.BackendStore.MarkDerivedArtifactDeleted(ctx, security.SystemAccessContext(), second[0].ID); err != nil {
			t.Fatal(err)
		}
		if pending, err := s.BackendStore.ListDerivedArtifactsForCleanup(ctx, security.SystemAccessContext(), failedAt.Add(2*time.Second), 1); err != nil || len(pending) != 0 {
			t.Fatalf("artifact became eligible before retry = %#v, %v", pending, err)
		}

		retryAt := conversationstore.DerivedArtifactRetryAt(failedAt, 1)
		retried, err := s.BackendStore.ListDerivedArtifactsForCleanup(ctx, security.SystemAccessContext(), retryAt, 1)
		if err != nil || len(retried) != 1 || retried[0].ID != first[0].ID {
			t.Fatalf("retry artifact batch = %#v, %v", retried, err)
		}
		if retried[0].CleanupAttempts != 1 || retried[0].LastErrorCode != "CLEANUP_FAILED" || retried[0].NextRetryAt == nil || !retried[0].NextRetryAt.Equal(retryAt) {
			t.Fatalf("retry metadata = %#v", retried[0])
		}
	})

	t.Run("derived artifact registration follows canonical session lifecycle", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "artifact-registration-session", CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
		if err := s.Create(ctx, session); err != nil {
			t.Fatal(err)
		}
		wrongTenant := conversationstore.DerivedArtifact{ID: "wrong-tenant-artifact", TenantID: "tenant-b", CanonicalSourceID: "session:" + session.ID, Kind: "test", Locator: "test://wrong-tenant"}
		if err := s.BackendStore.RegisterDerivedArtifact(ctx, security.SystemAccessContext(), wrongTenant); !errors.Is(err, conversationstore.ErrCanonicalSourceNotWritable) {
			t.Fatalf("RegisterDerivedArtifact(wrong tenant) error = %v, want ErrCanonicalSourceNotWritable", err)
		}
		if err := s.Delete(ctx, session.ID); err != nil {
			t.Fatal(err)
		}
		late := conversationstore.DerivedArtifact{ID: "late-artifact", TenantID: s.access.Subject.TenantID, CanonicalSourceID: "session:" + session.ID, Kind: "test", Locator: "test://late"}
		if err := s.BackendStore.RegisterDerivedArtifact(ctx, security.SystemAccessContext(), late); !errors.Is(err, conversationstore.ErrCanonicalSourceNotWritable) {
			t.Fatalf("RegisterDerivedArtifact(after delete) error = %v, want ErrCanonicalSourceNotWritable", err)
		}
	})

	t.Run("derived artifact registration cannot race past deletion", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "artifact-delete-race-session", CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
		if err := s.Create(ctx, session); err != nil {
			t.Fatal(err)
		}
		artifact := conversationstore.DerivedArtifact{ID: "artifact-delete-race", TenantID: s.access.Subject.TenantID, CanonicalSourceID: "session:" + session.ID, Kind: "test", Locator: "test://race"}
		start := make(chan struct{})
		deleteResult := make(chan error, 1)
		registerResult := make(chan error, 1)
		go func() {
			<-start
			deleteResult <- s.Delete(ctx, session.ID)
		}()
		go func() {
			<-start
			registerResult <- s.BackendStore.RegisterDerivedArtifact(ctx, security.SystemAccessContext(), artifact)
		}()
		close(start)
		if err := <-deleteResult; err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		registerErr := <-registerResult
		if registerErr != nil && !errors.Is(registerErr, conversationstore.ErrCanonicalSourceNotWritable) {
			t.Fatalf("RegisterDerivedArtifact() error = %v", registerErr)
		}
		pending, err := s.BackendStore.ListDerivedArtifactsForCleanup(ctx, security.SystemAccessContext(), time.Now(), 10)
		if err != nil {
			t.Fatal(err)
		}
		if registerErr == nil && (len(pending) != 1 || pending[0].ID != artifact.ID) {
			t.Fatalf("successful concurrent registration was not attached to cleanup: %#v", pending)
		}
		if errors.Is(registerErr, conversationstore.ErrCanonicalSourceNotWritable) && len(pending) != 0 {
			t.Fatalf("rejected concurrent registration persisted artifacts: %#v", pending)
		}
	})

	t.Run("cleanup honors batch size", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		for _, id := range []string{"batch-cleanup-1", "batch-cleanup-2"} {
			if err := s.Create(ctx, &conversationstore.Session{ID: id, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}); err != nil {
				t.Fatal(err)
			}
			if err := s.Delete(ctx, id); err != nil {
				t.Fatal(err)
			}
		}
		if cleaned, err := s.BackendStore.DeleteExpired(ctx, security.SystemAccessContext(), now.Add(time.Second), 1); err != nil || cleaned != 1 {
			t.Fatalf("first cleanup batch = %d, %v", cleaned, err)
		}
		if cleaned, err := s.BackendStore.DeleteExpired(ctx, security.SystemAccessContext(), now.Add(2*time.Second), 1); err != nil || cleaned != 1 {
			t.Fatalf("second cleanup batch = %d, %v", cleaned, err)
		}
	})

	t.Run("cleanup pages advance by tenant cursor", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		for _, id := range []string{"cursor-cleanup-a", "cursor-cleanup-b"} {
			if err := s.Create(ctx, &conversationstore.Session{ID: id, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}); err != nil {
				t.Fatal(err)
			}
			if err := s.Delete(ctx, id); err != nil {
				t.Fatal(err)
			}
		}
		first, cursor, err := s.BackendStore.DeleteExpiredPage(ctx, security.SystemAccessContext(), now.Add(time.Second), conversationstore.CleanupCursor{}, 1)
		if err != nil || first != 1 || !cursor.Valid {
			t.Fatalf("DeleteExpiredPage(first) = %d, %#v, %v", first, cursor, err)
		}
		second, next, err := s.BackendStore.DeleteExpiredPage(ctx, security.SystemAccessContext(), now.Add(2*time.Second), cursor, 1)
		if err != nil || second != 1 || !next.Valid || next.ID == cursor.ID {
			t.Fatalf("DeleteExpiredPage(second) = %d, %#v, %v", second, next, err)
		}
	})

	t.Run("cleanup cursor resets after an empty sweep", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		expiresAt := now.Add(time.Hour)
		session := &conversationstore.Session{ID: "cursor-future-expiry", CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour), ExpiresAt: &expiresAt, Status: conversationstore.SessionStatusActive}
		if err := s.Create(ctx, session); err != nil {
			t.Fatal(err)
		}
		_, cursor, err := s.BackendStore.DeleteExpiredPage(ctx, security.SystemAccessContext(), now, conversationstore.CleanupCursor{}, 10)
		if err != nil {
			t.Fatal(err)
		}
		if cursor.Valid {
			t.Fatalf("empty sweep cursor = %#v, want invalid cursor", cursor)
		}
		deleted, _, err := s.BackendStore.DeleteExpiredPage(ctx, security.SystemAccessContext(), expiresAt.Add(time.Second), cursor, 10)
		if err != nil || deleted != 1 {
			t.Fatalf("future expiry cleanup = %d, %v; want 1", deleted, err)
		}
	})

	t.Run("context window contains recent completed turns and selected active turn", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "window-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		for _, text := range []string{"oldest", "latest"} {
			turnID := begin(t, s, ctx, session.ID)
			if err := s.AddHistoryToTurn(ctx, session.ID, turnID,
				ai.NewUserTextMessage(text+"-user"), ai.NewModelTextMessage(text+"-model")); err != nil {
				t.Fatalf("AddHistoryToTurn(%s) error = %v", text, err)
			}
			if err := s.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
				t.Fatalf("NextTurnForTurn(%s) error = %v", text, err)
			}
		}

		selected := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, selected, ai.NewUserTextMessage("selected-active")); err != nil {
			t.Fatalf("AddHistoryToTurn(selected active) error = %v", err)
		}

		messages, err := s.ContextWindowForTurn(ctx, session.ID, selected, 1)
		if err != nil {
			t.Fatalf("ContextWindowForTurn() error = %v", err)
		}
		got := messageTexts(messages)
		want := []string{"latest-user", "latest-model", "selected-active"}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("ContextWindowForTurn() texts = %v, want %v", got, want)
		}
		if strings.Contains(strings.Join(got, " "), "oldest") {
			t.Fatalf("ContextWindowForTurn() included an excluded turn: %v", got)
		}

		messages[0].Content[0].Text = "mutated"
		again, err := s.ContextWindowForTurn(ctx, session.ID, selected, 1)
		if err != nil {
			t.Fatalf("second ContextWindowForTurn() error = %v", err)
		}
		if again[0].Text() != "latest-user" {
			t.Fatalf("caller mutation changed stored context: %q", again[0].Text())
		}

		currentOnly, err := s.ContextWindowForTurn(ctx, session.ID, selected, 0)
		if err != nil || len(currentOnly) != 1 || currentOnly[0].Text() != "selected-active" {
			t.Fatalf("zero-limit context = %#v, error = %v, want selected active only", currentOnly, err)
		}
		if _, err := s.ContextWindowForTurn(ctx, session.ID, selected, -1); err == nil {
			t.Fatal("negative completed limit succeeded, want validation error")
		}

		isolated := withAccess(newStore())
		isolatedSession := &conversationstore.Session{ID: "isolated-window-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := isolated.Create(ctx, isolatedSession); err != nil {
			t.Fatalf("Create(isolated) error = %v", err)
		}
		completed := begin(t, isolated, ctx, isolatedSession.ID)
		if err := isolated.AddHistoryToTurn(ctx, isolatedSession.ID, completed, ai.NewUserTextMessage("completed")); err != nil {
			t.Fatalf("AddHistoryToTurn(completed) error = %v", err)
		}
		if err := isolated.NextTurnForTurn(ctx, isolatedSession.ID, completed); err != nil {
			t.Fatalf("NextTurnForTurn(completed) error = %v", err)
		}
		otherActive := begin(t, isolated, ctx, isolatedSession.ID)
		if err := isolated.AddHistoryToTurn(ctx, isolatedSession.ID, otherActive, ai.NewUserTextMessage("other-active")); err != nil {
			t.Fatalf("AddHistoryToTurn(other active) error = %v", err)
		}
		if _, err := isolated.BeginTurn(ctx, isolatedSession.ID); !errors.Is(err, conversationstore.ErrSessionBusy) {
			t.Fatalf("BeginTurn(concurrent) error = %v, want ErrSessionBusy", err)
		}
		if err := isolated.AbortTurnForTurn(ctx, isolatedSession.ID, otherActive); err != nil {
			t.Fatalf("AbortTurnForTurn(other active) error = %v", err)
		}
		selectedActive := begin(t, isolated, ctx, isolatedSession.ID)
		if err := isolated.AddHistoryToTurn(ctx, isolatedSession.ID, selectedActive, ai.NewUserTextMessage("selected")); err != nil {
			t.Fatalf("AddHistoryToTurn(selected) error = %v", err)
		}
		isolatedWindow, err := isolated.ContextWindowForTurn(ctx, isolatedSession.ID, selectedActive, 1)
		if err != nil {
			t.Fatalf("ContextWindowForTurn(isolated) error = %v", err)
		}
		if got := messageTexts(isolatedWindow); fmt.Sprint(got) != fmt.Sprint([]string{"completed", "selected"}) {
			t.Fatalf("isolated context texts = %v, want completed and selected only", got)
		}
	})

	t.Run("context window preserves message write order within a turn", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "message-order-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID,
			ai.NewUserTextMessage("user-1"), ai.NewModelTextMessage("model-1"),
			ai.NewUserTextMessage("user-2"), ai.NewModelTextMessage("model-2")); err != nil {
			t.Fatalf("AddHistoryToTurn() error = %v", err)
		}
		messages, err := s.ContextWindowForTurn(ctx, session.ID, turnID, 0)
		if err != nil {
			t.Fatalf("ContextWindowForTurn() error = %v", err)
		}
		want := []string{"user-1", "model-1", "user-2", "model-2"}
		if got := messageTexts(messages); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("message order = %v, want %v", got, want)
		}
	})

	t.Run("aborting a turn removes its partial history", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "abort-turn-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserTextMessage("partial")); err != nil {
			t.Fatalf("AddHistoryToTurn() error = %v", err)
		}
		if err := s.AbortTurnForTurn(ctx, session.ID, turnID); err != nil {
			t.Fatalf("AbortTurnForTurn() error = %v", err)
		}
		if _, err := s.WindowMemoryForTurn(ctx, session.ID, turnID); !errors.Is(err, conversationstore.ErrTurnNotFound) {
			t.Fatalf("WindowMemoryForTurn() after abort error = %v, want ErrTurnNotFound", err)
		}
		if messages, err := s.AllMemory(ctx, session.ID); err != nil || len(messages) != 0 {
			t.Fatalf("AllMemory() after abort = %#v, error = %v, want empty history", messages, err)
		}
		if nextID, err := s.BeginTurn(ctx, session.ID); err != nil || nextID == 0 {
			t.Fatalf("BeginTurn() after abort = %d, error = %v", nextID, err)
		}
	})

	t.Run("bounded context window snapshots", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "context-window", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatal(err)
		}
		var completed uint64
		for i := 0; i < 2; i++ {
			completed = begin(t, s, ctx, session.ID)
			if err := s.AddHistoryToTurn(ctx, session.ID, completed,
				ai.NewUserTextMessage(fmt.Sprintf("question-%d", i)),
				ai.NewModelTextMessage(fmt.Sprintf("answer-%d", i))); err != nil {
				t.Fatal(err)
			}
			if err := s.NextTurnForTurn(ctx, session.ID, completed); err != nil {
				t.Fatal(err)
			}
		}
		active := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, active, ai.NewUserTextMessage("current")); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			limit int
			want  string
		}{
			{0, "current"},
			{1, "question-1,answer-1,current"},
			{2, "question-0,answer-0,question-1,answer-1,current"},
			{100, "question-0,answer-0,question-1,answer-1,current"},
		} {
			t.Run(fmt.Sprintf("limit=%d", tc.limit), func(t *testing.T) {
				messages, err := s.BackendStore.ContextWindowForTurn(ctx, s.access, session.ID, active, tc.limit)
				if err != nil {
					t.Fatal(err)
				}
				texts := make([]string, len(messages))
				for i, message := range messages {
					texts[i] = message.Text()
				}
				if got := strings.Join(texts, ","); got != tc.want {
					t.Fatalf("ContextWindowForTurn() = %q, want %q", got, tc.want)
				}
				messages[len(messages)-1].Content[0].Text = "caller mutation"
				again, err := s.BackendStore.ContextWindowForTurn(ctx, s.access, session.ID, active, tc.limit)
				if err != nil || again[len(again)-1].Text() != "current" {
					t.Fatalf("snapshot mutation affected storage: %v, %v", again, err)
				}
			})
		}
		for _, turnID := range []uint64{completed, active + 100} {
			if _, err := s.BackendStore.ContextWindowForTurn(ctx, s.access, session.ID, turnID, 2); !errors.Is(err, conversationstore.ErrTurnNotFound) {
				t.Fatalf("ContextWindowForTurn(%d) error = %v, want ErrTurnNotFound", turnID, err)
			}
		}
		before, err := s.BackendStore.ContextWindowForTurn(ctx, s.access, session.ID, active, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.AddHistoryToTurn(ctx, session.ID, active, ai.NewModelTextMessage("later")); err != nil {
			t.Fatal(err)
		}
		if len(before) != 3 || before[2].Text() != "current" {
			t.Fatalf("snapshot changed after append: %v", before)
		}
	})

	t.Run("returned messages are snapshots", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "snapshot-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		message := ai.NewUserMessage(ai.NewTextPart("original"))
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, message); err != nil {
			t.Fatalf("AddHistory() error = %v", err)
		}
		message.Content[0].Text = "caller mutation"
		stored, err := s.WindowMemoryForTurn(ctx, session.ID, turnID)
		if err != nil {
			t.Fatalf("WindowMemory() error = %v", err)
		}
		if len(stored) != 1 || stored[0].Content[0].Text != "original" {
			t.Fatalf("stored message = %#v, want original snapshot", stored)
		}
	})

	t.Run("message batch failure does not expose partial writes", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "batch-failure-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		invalid := &ai.Message{
			Role:     ai.RoleUser,
			Metadata: map[string]any{"unsupported": func() {}},
		}
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserTextMessage("valid"), invalid); err == nil {
			t.Fatal("AddHistoryToTurn() succeeded with an unencodable message")
		}
		messages, err := s.WindowMemoryForTurn(ctx, session.ID, turnID)
		if errors.Is(err, conversationstore.ErrTurnNotFound) {
			return
		}
		if err != nil {
			t.Fatalf("WindowMemoryForTurn() error = %v", err)
		}
		if len(messages) != 0 {
			t.Fatalf("WindowMemoryForTurn() = %#v, want no partial messages", messages)
		}
	})

	t.Run("empty and unsupported history still creates an active turn", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "empty-turn-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, nil, ai.NewMessage(ai.RoleTool, nil, ai.NewTextPart("ignored"))); err != nil {
			t.Fatalf("AddHistory() error = %v", err)
		}
		empty, err := s.IsTurnEmpty(ctx, session.ID, turnID)
		if err != nil {
			t.Fatalf("IsEmpty() error = %v", err)
		}
		if empty {
			t.Fatal("IsEmpty() = true after an empty AddHistory(), want false")
		}
		messages, err := s.WindowMemoryForTurn(ctx, session.ID, turnID)
		if err != nil {
			t.Fatalf("WindowMemory() error = %v", err)
		}
		if len(messages) != 0 {
			t.Fatalf("WindowMemory() = %#v, want empty", messages)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
			t.Fatalf("NextTurn() on empty active turn error = %v", err)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, turnID); !errors.Is(err, conversationstore.ErrTurnNotFound) {
			t.Fatalf("NextTurnForTurn() without active turn = %v, want ErrTurnNotFound", err)
		}
	})

	t.Run("closed session rejects history writes", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "closed-session", CreatedAt: now, UpdatedAt: now, Status: "closed"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if _, err := s.BeginTurn(ctx, session.ID); err == nil || !strings.Contains(err.Error(), "not active") {
			t.Fatalf("BeginTurn() error = %v, want inactive-session error", err)
		}
		if _, err := s.IsTurnEmpty(ctx, session.ID, 1); err == nil || !strings.Contains(err.Error(), "not active") {
			t.Fatalf("IsTurnEmpty() error = %v, want inactive-session error", err)
		}
		if _, err := s.WindowMemoryForTurn(ctx, session.ID, 1); err == nil || !strings.Contains(err.Error(), "not active") {
			t.Fatalf("WindowMemoryForTurn() error = %v, want inactive-session error", err)
		}
		if _, err := s.AllMemory(ctx, session.ID); err == nil || !strings.Contains(err.Error(), "not active") {
			t.Fatalf("AllMemory() error = %v, want inactive-session error", err)
		}
	})

	t.Run("expiration is consistent across session operations", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		expired := &conversationstore.Session{
			ID:        "expired-contract-session",
			CreatedAt: now.Add(-25 * time.Hour),
			UpdatedAt: now.Add(-25 * time.Hour),
			Status:    "active",
		}
		active := &conversationstore.Session{ID: "active-contract-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		historyExpired := &conversationstore.Session{ID: "expired-history-contract-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		for _, session := range []*conversationstore.Session{expired, active, historyExpired} {
			if err := s.Create(ctx, session); err != nil {
				t.Fatalf("Create(%s) error = %v", session.ID, err)
			}
		}
		turnID := begin(t, s, ctx, historyExpired.ID)
		if err := s.AddHistoryToTurn(ctx, historyExpired.ID, turnID, ai.NewUserTextMessage("must not leak")); err != nil {
			t.Fatalf("AddHistoryToTurn() error = %v", err)
		}
		if err := s.Touch(ctx, historyExpired.ID, now.Add(-25*time.Hour)); err != nil {
			t.Fatalf("Touch(history session into expiry) error = %v", err)
		}
		if _, err := s.IsTurnEmpty(ctx, historyExpired.ID, turnID); !errors.Is(err, conversationstore.ErrSessionExpired) {
			t.Fatalf("IsTurnEmpty(expired) error = %v, want ErrSessionExpired", err)
		}
		if _, err := s.WindowMemoryForTurn(ctx, historyExpired.ID, turnID); !errors.Is(err, conversationstore.ErrSessionExpired) {
			t.Fatalf("WindowMemoryForTurn(expired) error = %v, want ErrSessionExpired", err)
		}
		if _, err := s.AllMemory(ctx, historyExpired.ID); !errors.Is(err, conversationstore.ErrSessionExpired) {
			t.Fatalf("AllMemory(expired) error = %v, want ErrSessionExpired", err)
		}
		if _, err := s.Get(ctx, expired.ID); !errors.Is(err, conversationstore.ErrSessionExpired) {
			t.Fatalf("Get(expired) error = %v, want ErrSessionExpired", err)
		}
		if err := s.Touch(ctx, expired.ID, now); !errors.Is(err, conversationstore.ErrSessionExpired) {
			t.Fatalf("Touch(expired) error = %v, want ErrSessionExpired", err)
		}
		listed, err := s.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(listed) != 1 || listed[0].ID != active.ID {
			t.Fatalf("List() = %#v, want only active session", listed)
		}
		deleted, err := s.DeleteExpired(ctx, now)
		if err != nil {
			t.Fatalf("DeleteExpired() error = %v", err)
		}
		if deleted != 2 {
			t.Fatalf("DeleteExpired() = %d, want 2", deleted)
		}
		if _, err := s.Get(ctx, expired.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
			t.Fatalf("Get(expired) after DeleteExpired error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("turn limit and failed transition preserve active turn", func(t *testing.T) {
		ctx := context.Background()
		s := withAccess(newStore())
		now := time.Now()
		session := &conversationstore.Session{ID: "limit-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserTextMessage("turn-0")); err != nil {
			t.Fatalf("initial AddHistory() error = %v", err)
		}
		for i := 1; i < limit; i++ {
			if err := s.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
				t.Fatalf("NextTurnForTurn() at turn %d error = %v", i-1, err)
			}
			turnID = begin(t, s, ctx, session.ID)
			if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserTextMessage(fmt.Sprintf("turn-%d", i))); err != nil {
				t.Fatalf("AddHistory() at turn %d error = %v", i, err)
			}
		}
		if err := s.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
			t.Fatalf("final NextTurn() error = %v, want success", err)
		}
		if _, err := s.BeginTurn(ctx, session.ID); !errors.Is(err, conversationstore.ErrTurnLimitReached) {
			t.Fatalf("overflow BeginTurn() error = %v, want ErrTurnLimitReached", err)
		}
		messages, err := s.AllMemory(ctx, session.ID)
		if err != nil {
			t.Fatalf("AllMemory() after rejected turn error = %v", err)
		}
		if len(messages) != limit {
			t.Fatalf("AllMemory() after rejected turn = %d messages, want %d", len(messages), limit)
		}
	})
}

// RunQuotaContractTests verifies configurable persistent quotas shared by all
// Store backends. Memory-only entity capacity is covered by MemoryStore tests.
func RunQuotaContractTests(t *testing.T, newStore func(conversationstore.QuotaPolicy) conversationstore.BackendStore) {
	t.Helper()
	t.Run("failed interactions do not consume turn quota", func(t *testing.T) {
		quota := conversationstore.DefaultQuotaPolicy()
		quota.MaxTurnsPerSession = 1
		s := withAccess(newStore(quota))
		ctx := context.Background()
		now := time.Now()
		session := &conversationstore.Session{ID: "failed-interaction-turn-quota", CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
		if err := s.Create(ctx, session); err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 3; i++ {
			key := fmt.Sprintf("failed-request-%d", i)
			prepared, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, key, "hash-"+key, time.Minute)
			if err != nil {
				t.Fatalf("PrepareInteraction(%s) error = %v", key, err)
			}
			if err := s.AddHistoryToTurn(ctx, session.ID, prepared.TurnID, ai.NewUserTextMessage("partial")); err != nil {
				t.Fatalf("AddHistoryToTurn(%s) error = %v", key, err)
			}
			if err := s.BackendStore.FailInteraction(ctx, s.access, session.ID, prepared.InteractionID, prepared.TurnID); err != nil {
				t.Fatalf("FailInteraction(%s) error = %v", key, err)
			}
		}
		if messages, err := s.AllMemory(ctx, session.ID); err != nil || len(messages) != 0 {
			t.Fatalf("AllMemory() after failed interactions = %#v, %v; want empty", messages, err)
		}

		completed, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "completed-request", "completed-hash", time.Minute)
		if err != nil {
			t.Fatalf("PrepareInteraction(completed) error = %v", err)
		}
		if err := s.AddHistoryToTurn(ctx, session.ID, completed.TurnID, ai.NewUserTextMessage("complete")); err != nil {
			t.Fatal(err)
		}
		if err := s.BackendStore.CompleteInteraction(ctx, s.access, session.ID, completed.InteractionID, completed.TurnID, "done"); err != nil {
			t.Fatal(err)
		}

		if _, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "overflow-request", "overflow-hash", time.Minute); !errors.Is(err, conversationstore.ErrResourceExhausted) {
			t.Fatalf("PrepareInteraction(over turn quota) error = %v, want ErrResourceExhausted", err)
		}
		if _, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "failed-request-0", "hash-failed-request-0", time.Minute); !errors.Is(err, conversationstore.ErrResourceExhausted) {
			t.Fatalf("PrepareInteraction(failed retry over turn quota) error = %v, want ErrResourceExhausted", err)
		}
		replayed, err := s.BackendStore.PrepareInteraction(ctx, s.access, session.ID, "completed-request", "completed-hash", time.Minute)
		if err != nil {
			t.Fatalf("PrepareInteraction(replay at turn quota) error = %v", err)
		}
		if !replayed.Replayed || replayed.InteractionID != completed.InteractionID || replayed.Result != "done" {
			t.Fatalf("PrepareInteraction(replay) = %#v", replayed)
		}
	})

	t.Run("session quota", func(t *testing.T) {
		quota := conversationstore.DefaultQuotaPolicy()
		quota.MaxSessionsPerTenant = 1
		s := withAccess(newStore(quota))
		now := time.Now()
		if err := s.Create(context.Background(), &conversationstore.Session{ID: "quota-session-1", CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}); err != nil {
			t.Fatal(err)
		}
		if err := s.Create(context.Background(), &conversationstore.Session{ID: "quota-session-2", CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}); !errors.Is(err, conversationstore.ErrResourceExhausted) {
			t.Fatalf("Create(over quota) error = %v, want ErrResourceExhausted", err)
		}
	})

	for _, tc := range []struct {
		name   string
		adjust func(*conversationstore.QuotaPolicy)
	}{
		{name: "session byte quota", adjust: func(quota *conversationstore.QuotaPolicy) { quota.MaxSessionBytes = 1 }},
		{name: "tenant byte quota", adjust: func(quota *conversationstore.QuotaPolicy) { quota.MaxTenantBytes = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			quota := conversationstore.DefaultQuotaPolicy()
			tc.adjust(&quota)
			s := withAccess(newStore(quota))
			now := time.Now()
			session := &conversationstore.Session{ID: strings.ReplaceAll(tc.name, " ", "-"), CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
			if err := s.Create(context.Background(), session); err != nil {
				t.Fatal(err)
			}
			turnID, err := s.BeginTurn(context.Background(), session.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.AddHistoryToTurn(context.Background(), session.ID, turnID, ai.NewUserTextMessage("exceeds one byte")); !errors.Is(err, conversationstore.ErrResourceExhausted) {
				t.Fatalf("AddHistoryToTurn(over quota) error = %v, want ErrResourceExhausted", err)
			}
			messages, err := s.WindowMemoryForTurn(context.Background(), session.ID, turnID)
			if err != nil && !errors.Is(err, conversationstore.ErrTurnNotFound) {
				t.Fatal(err)
			}
			if len(messages) != 0 {
				t.Fatalf("quota failure persisted %d messages", len(messages))
			}
		})
	}
}

func messageTexts(messages []*ai.Message) []string {
	texts := make([]string, 0, len(messages))
	for _, message := range messages {
		texts = append(texts, message.Text())
	}
	return texts
}
