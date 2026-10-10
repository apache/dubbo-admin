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
	"testing"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
)

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
