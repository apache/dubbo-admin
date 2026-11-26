/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
 */
package store_test

import (
	"context"
	"dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"
	"dubbo-admin-ai/testutils"
	"errors"
	"testing"
	"time"
)

func TestMemoryCanonicalIdentityAndQuota(t *testing.T) {
	a := testutils.TestAccess()
	ctx := context.Background()
	quota := store.DefaultQuotaPolicy()
	quota.ExplicitMemory = store.MemoryPolicy{MaxRecords: 2}
	s := memorystore.NewMemoryStoreWithPolicy(quota)
	w := store.MemoryWrite{Generation: 1, StateVersion: 1, ScopeType: "user", MemoryType: "task_context", FactKey: "project.background", Value: "project context", Applicability: store.MatchContext{ProjectRef: " project-a "}}
	r, err := s.SaveMemory(ctx, a, w)
	if err != nil {
		t.Fatal(err)
	}
	w.Applicability.ProjectRef = "project-a"
	duplicate, err := s.SaveMemory(ctx, a, w)
	if err != nil || duplicate.ID != r.ID || duplicate.CanonicalKey != r.CanonicalKey {
		t.Fatal("normalization changed identity", duplicate, err)
	}
	w.Applicability.ProjectRef = "project-b"
	second, err := s.SaveMemory(ctx, a, w)
	if err != nil {
		t.Fatal(err)
	}
	w.ID = r.ID
	w.ExpectedVersion = r.Version
	w.Value = "try to overwrite B"
	if _, err := s.SaveMemory(ctx, a, w); !errors.Is(err, store.ErrMemoryConflict) {
		t.Fatal("identity conflict overwrote another record", err)
	}
	w.ID = ""
	w.ExpectedVersion = 0
	w.Applicability.ProjectRef = "project-c"
	if _, err := s.SaveMemory(ctx, a, w); !errors.Is(err, store.ErrResourceExhausted) {
		t.Fatal("record quota bypass", err)
	}
	page, err := s.ListMemories(ctx, a, "", 32)
	if err != nil || len(page.Records) != 2 {
		t.Fatal("failed write changed state", page, err)
	}
	for _, record := range page.Records {
		if record.ID == second.ID && record.Value != "project context" {
			t.Fatal("conflicting target overwritten")
		}
	}
	// Returned snapshots, including expiry pointers, must not alias the store.
	if err := s.DeleteMemory(ctx, a, second.ID, store.MemoryMutation{Generation: 1, StateVersion: 1, Version: 1}); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(time.Hour)
	w.ExpiresAt = &expires
	r, err = s.SaveMemory(ctx, a, w)
	if err != nil {
		t.Fatal(err)
	}
	*r.ExpiresAt = time.Now().Add(-time.Hour)
	items, _, err := s.RecallMemories(ctx, a, store.MatchContext{ProjectRef: "project-c"}, 16, 16384)
	if err != nil || len(items) != 1 {
		t.Fatal("returned pointer mutated storage", items, err)
	}
}
