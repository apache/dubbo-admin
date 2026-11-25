/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0.
 */

package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"
)

var errUsageCleanupUnavailable = errors.New("usage cleanup unavailable")

// failingUsageCleanupStore keeps the conversation Store healthy while making
// usage maintenance fail, so retention cleanup must still complete.
type failingUsageCleanupStore struct {
	*memorystore.MemoryStore
}

func (failingUsageCleanupStore) CleanupUsage(context.Context, security.AccessContext, time.Time, int) (int, error) {
	return 0, errUsageCleanupUnavailable
}

func (failingUsageCleanupStore) CleanupAudit(context.Context, security.AccessContext, time.Time, int) (int, error) {
	return 0, nil
}

// conversationStoreOnly hides the optional usage/audit maintenance methods.
type conversationStoreOnly struct {
	conversationstore.BackendStore
}

func prepareDeletedSession(t *testing.T, backend *memorystore.MemoryStore) time.Time {
	t.Helper()
	ctx := context.Background()
	access := security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "owner", AuthMethod: security.AuthMethodStatic})
	now := time.Now()
	session := &conversationstore.Session{ID: "cleanup-retention", TenantID: "tenant", OwnerID: "owner", Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
	if err := backend.Create(ctx, access, session); err != nil {
		t.Fatal(err)
	}
	if err := backend.Delete(ctx, access, session.ID); err != nil {
		t.Fatal(err)
	}
	return now
}

func TestCleanupRemovesConversationsWhenUsageAuditCleanupFails(t *testing.T) {
	backend := memorystore.NewMemoryStore()
	now := prepareDeletedSession(t, backend)
	component := &MemoryComponent{
		store:           failingUsageCleanupStore{MemoryStore: backend},
		rt:              runtime.NewRuntime(),
		cleanupHandlers: make(map[string]conversationstore.DerivedArtifactCleanupHandler),
	}

	deleted, err := component.runCleanupOnce(context.Background(), now.Add(2*time.Minute), 10)
	if !errors.Is(err, errUsageCleanupUnavailable) {
		t.Fatalf("cleanup error = %v, want %v", err, errUsageCleanupUnavailable)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1 conversation removed despite usage cleanup failure", deleted)
	}
}

func TestCleanupRemovesConversationsWhenUsageAuditUnsupported(t *testing.T) {
	backend := memorystore.NewMemoryStore()
	now := prepareDeletedSession(t, backend)
	component := &MemoryComponent{
		store:           conversationStoreOnly{BackendStore: backend},
		rt:              runtime.NewRuntime(),
		cleanupHandlers: make(map[string]conversationstore.DerivedArtifactCleanupHandler),
	}

	deleted, err := component.runCleanupOnce(context.Background(), now.Add(2*time.Minute), 10)
	if err != nil {
		t.Fatalf("cleanup error = %v, want conversation cleanup to succeed", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1 conversation removed without usage maintenance support", deleted)
	}
}

func TestCleanupWorkerDispatchesAndReconcilesDerivedArtifact(t *testing.T) {
	backend := memorystore.NewMemoryStore()
	component := &MemoryComponent{
		store:           backend,
		rt:              runtime.NewRuntime(),
		cleanupHandlers: make(map[string]conversationstore.DerivedArtifactCleanupHandler),
	}
	called := 0
	if err := component.RegisterDerivedArtifactCleanupHandler("test", conversationstore.DerivedArtifactCleanupFunc(func(_ context.Context, artifact conversationstore.DerivedArtifact) error {
		called++
		if artifact.Locator != "test://artifact" {
			t.Fatalf("artifact locator = %q", artifact.Locator)
		}
		if called == 1 {
			return errors.New("temporary cleanup failure")
		}
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	access := security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "owner", AuthMethod: security.AuthMethodStatic})
	now := time.Now()
	session := &conversationstore.Session{ID: "cleanup-dispatch", TenantID: "tenant", OwnerID: "owner", Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
	if err := backend.Create(ctx, access, session); err != nil {
		t.Fatal(err)
	}
	artifact := conversationstore.DerivedArtifact{ID: "artifact", TenantID: "tenant", CanonicalSourceID: "session:" + session.ID, Kind: "test", Locator: "test://artifact"}
	if err := backend.RegisterDerivedArtifact(ctx, security.SystemAccessContext(), artifact); err != nil {
		t.Fatal(err)
	}
	if err := backend.Delete(ctx, access, session.ID); err != nil {
		t.Fatal(err)
	}
	deleted, err := component.runCleanupOnce(ctx, now.Add(time.Second), 10)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 0 || called != 1 {
		t.Fatalf("first cleanup result = deleted %d, handler calls %d; want 0, 1", deleted, called)
	}
	if _, err := backend.Get(ctx, access, session.ID); err != conversationstore.ErrSessionNotFound {
		t.Fatalf("Get() while cleanup is pending error = %v", err)
	}
	deleted, err = component.runCleanupOnce(ctx, now.Add(2*time.Minute), 10)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 || called != 2 {
		t.Fatalf("reconciled cleanup result = deleted %d, handler calls %d; want 1, 2", deleted, called)
	}
}
