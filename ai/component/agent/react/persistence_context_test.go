/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. */
package react

import (
	"context"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"
	"dubbo-admin-ai/testutils"
)

func TestFinishPersistsAfterInteractionDeadline(t *testing.T) {
	access := testutils.TestAccess()
	store := memorystore.NewMemoryStore()
	now := time.Now()
	session := &conversationstore.Session{
		ID: "persistence-deadline", TenantID: access.Subject.TenantID, OwnerID: access.Subject.UserID,
		Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive,
	}
	if err := store.Create(context.Background(), access, session); err != nil {
		t.Fatal(err)
	}
	turn, err := store.BeginTurn(context.Background(), access, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	parent := security.WithAccessContext(context.Background(), access)
	deadlineCtx, cancel := context.WithDeadline(parent, time.Now().Add(-time.Second))
	defer cancel()

	ra := &ReActAgent{messageStore: store}
	if err := ra.finish(deadlineCtx, nil, nil, session.ID, turn, "slow model answer"); err != nil {
		t.Fatalf("finish() persisted through an expired interaction context: %v", err)
	}
}
