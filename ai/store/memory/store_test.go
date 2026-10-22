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
	"sort"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"

	"github.com/firebase/genkit/go/ai"
)

func TestAuditRecordsRetainCallIdentifiers(t *testing.T) {
	s := NewMemoryStore()
	access := security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "owner", AuthMethod: security.AuthMethodJWT})
	ctx := security.WithAccessContext(context.Background(), access)
	events := []conversationstore.AuditEvent{
		{EventType: "interaction.start"},
		{EventType: "model_call.end", CallID: "model-1"},
		{EventType: "model_call.end", CallID: "model-2", Status: "error"},
		{EventType: "tool_call.end", ToolCallID: "tool-1"},
		{EventType: "tool_call.end", ToolCallID: "tool-2", Status: "error"},
	}
	for _, event := range events {
		event.TenantID, event.InteractionID = "tenant", "interaction"
		if err := s.RecordAudit(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	if s.auditEvents.Len() != len(events) {
		t.Fatalf("stored %d events", s.auditEvents.Len())
	}
	element := s.auditEvents.Front()
	for i, expected := range events {
		stored := element.Value.(conversationstore.AuditEvent)
		if stored.InteractionID != "interaction" || stored.CallID != expected.CallID || stored.ToolCallID != expected.ToolCallID {
			t.Fatalf("audit %d lost call identity: %+v", i, stored)
		}
		element = element.Next()
	}
}

func TestCustomRetentionAppliesToDirectCreate(t *testing.T) {
	quota := conversationstore.DefaultQuotaPolicy()
	lifecycle := conversationstore.DefaultLifecyclePolicy()
	lifecycle.Retention = 2 * time.Hour
	store := NewMemoryStoreWithPolicies(quota, lifecycle)
	access := security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "owner", AuthMethod: security.AuthMethodStatic})
	now := time.Now()
	session := &conversationstore.Session{ID: "custom-retention", TenantID: "tenant", OwnerID: "owner", Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
	if err := store.Create(context.Background(), access, session); err != nil {
		t.Fatal(err)
	}
	stored := store.sessions[session.ID]
	if stored.ExpiresAt == nil || !stored.ExpiresAt.Equal(now.Add(2*time.Hour)) {
		t.Fatalf("ExpiresAt = %v, want %v", stored.ExpiresAt, now.Add(2*time.Hour))
	}
}

func TestMemoryEntityQuotaBoundsInProcessConversationState(t *testing.T) {
	quota := conversationstore.DefaultQuotaPolicy()
	quota.MaxInMemoryEntities = 3
	store := NewMemoryStoreWithPolicy(quota)
	access := security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "owner", AuthMethod: security.AuthMethodStatic})
	now := time.Now()
	session := &conversationstore.Session{ID: "entity-quota", TenantID: "tenant", OwnerID: "owner", Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
	if err := store.Create(context.Background(), access, session); err != nil {
		t.Fatal(err)
	}
	prepared, err := store.PrepareInteraction(context.Background(), access, session.ID, "request", "hash", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddHistoryToTurn(context.Background(), access, session.ID, prepared.TurnID, ai.NewUserTextMessage("over limit")); err != conversationstore.ErrResourceExhausted {
		t.Fatalf("AddHistoryToTurn() error = %v, want ErrResourceExhausted", err)
	}
}

func TestAdoptLegacySessionsOnlyAdoptsFullyUnownedRows(t *testing.T) {
	now := time.Now()
	store := NewMemoryStore()
	store.sessions["legacy"] = conversationstore.Session{ID: "legacy", CreatedAt: now, UpdatedAt: now, Status: "active"}
	store.sessions["partial-tenant"] = conversationstore.Session{
		ID: "partial-tenant", TenantID: "tenant-existing", Version: 1, CreatedAt: now, UpdatedAt: now, Status: "active",
	}
	store.sessions["partial-owner"] = conversationstore.Session{
		ID: "partial-owner", OwnerID: "owner-existing", Version: 1, CreatedAt: now, UpdatedAt: now, Status: "active",
	}

	adopted, err := store.AdoptLegacySessions(context.Background(), security.SystemAccessContext(), "tenant-a", "owner-a")
	if err != nil || adopted != 1 {
		t.Fatalf("AdoptLegacySessions() = %d, %v, want 1, nil", adopted, err)
	}
	if got := store.sessions["legacy"]; got.TenantID != "tenant-a" || got.OwnerID != "owner-a" || got.Version != 1 || got.ExpiresAt == nil {
		t.Fatalf("adopted legacy session = %+v", got)
	}
	if got := store.sessions["partial-tenant"]; got.TenantID != "tenant-existing" || got.OwnerID != "" {
		t.Fatalf("partial tenant session changed: %+v", got)
	}
	if got := store.sessions["partial-owner"]; got.TenantID != "" || got.OwnerID != "owner-existing" {
		t.Fatalf("partial owner session changed: %+v", got)
	}
}

func TestListDerivedArtifactsForCleanupFindsInconsistentSources(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()
	store.sessions["healthy"] = conversationstore.Session{ID: "healthy", TenantID: "tenant-a", Status: conversationstore.SessionStatusActive}
	store.sessions["mismatched"] = conversationstore.Session{ID: "mismatched", TenantID: "tenant-a", Status: conversationstore.SessionStatusActive}
	for _, artifact := range []conversationstore.DerivedArtifact{
		{ID: "healthy", TenantID: "tenant-a", CanonicalSourceID: "session:healthy", Status: "active", CreatedAt: now},
		{ID: "invalid", TenantID: "tenant-a", CanonicalSourceID: "invalid", Status: "active", CreatedAt: now},
		{ID: "missing", TenantID: "tenant-a", CanonicalSourceID: "session:missing", Status: "active", CreatedAt: now},
		{ID: "tenant-mismatch", TenantID: "tenant-b", CanonicalSourceID: "session:mismatched", Status: "active", CreatedAt: now},
	} {
		store.derivedArtifacts[artifact.ID] = artifact
	}

	pending, err := store.ListDerivedArtifactsForCleanup(context.Background(), security.SystemAccessContext(), now, 10)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(pending))
	for i := range pending {
		ids = append(ids, pending[i].ID)
	}
	sort.Strings(ids)
	want := []string{"invalid", "missing", "tenant-mismatch"}
	if len(ids) != len(want) {
		t.Fatalf("cleanup candidates = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("cleanup candidates = %v, want %v", ids, want)
		}
	}
}

func TestCleanupJobProjectsEarliestArtifactRetry(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	access := security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "owner", AuthMethod: security.AuthMethodStatic})
	now := time.Now()
	session := &conversationstore.Session{ID: "retry-projection", TenantID: "tenant", OwnerID: "owner", Version: 1, Status: conversationstore.SessionStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(ctx, access, session); err != nil {
		t.Fatal(err)
	}
	canonicalID := "session:" + session.ID
	for _, id := range []string{"artifact-a", "artifact-b"} {
		artifact := conversationstore.DerivedArtifact{ID: id, TenantID: "tenant", CanonicalSourceID: canonicalID, Kind: "test", Locator: "test://" + id}
		if err := store.RegisterDerivedArtifact(ctx, security.SystemAccessContext(), artifact); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Delete(ctx, access, session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteExpired(ctx, security.SystemAccessContext(), now.Add(time.Second), 10); err != nil {
		t.Fatal(err)
	}

	failedA := now.Add(2 * time.Second)
	if err := store.MarkDerivedArtifactCleanupFailed(ctx, security.SystemAccessContext(), "artifact-a", "A_FAILED", failedA); err != nil {
		t.Fatal(err)
	}
	if job := store.cleanupJobs[canonicalID]; job.NextRetryAt != nil {
		t.Fatalf("NextRetryAt = %v, want nil while artifact-b is immediately eligible", job.NextRetryAt)
	}
	failedB := failedA.Add(time.Second)
	if err := store.MarkDerivedArtifactCleanupFailed(ctx, security.SystemAccessContext(), "artifact-b", "B_FAILED", failedB); err != nil {
		t.Fatal(err)
	}
	wantA := conversationstore.DerivedArtifactRetryAt(failedA, 1)
	if job := store.cleanupJobs[canonicalID]; job.NextRetryAt == nil || !job.NextRetryAt.Equal(wantA) {
		t.Fatalf("NextRetryAt = %v, want earliest %v", job.NextRetryAt, wantA)
	}
	if err := store.MarkDerivedArtifactDeleted(ctx, security.SystemAccessContext(), "artifact-a"); err != nil {
		t.Fatal(err)
	}
	wantB := conversationstore.DerivedArtifactRetryAt(failedB, 1)
	if job := store.cleanupJobs[canonicalID]; job.NextRetryAt == nil || !job.NextRetryAt.Equal(wantB) {
		t.Fatalf("NextRetryAt after deleting artifact-a = %v, want %v", job.NextRetryAt, wantB)
	}
}
