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

package gormstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	conversationstore "dubbo-admin-ai/store"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestCleanupSessionDataReleasesTenantUsageOnce covers a replica that read a
// candidate before another replica finalized it. The late caller must not
// decrement tenant counters that still describe live Sessions.
func TestCleanupSessionDataReleasesTenantUsageOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cleanup-once.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&SessionModel{}, &TurnModel{}, &MessageModel{}, &InteractionModel{},
		&DerivedArtifactModel{}, &CleanupJobModel{}, &OutboxEventModel{}, &TenantUsageModel{},
	); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, session := range []SessionModel{
		{ID: "cleanup-once", TenantID: "tenant", OwnerID: "owner", Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusDeletionPending, StoredBytes: 10},
		{ID: "still-active", TenantID: "tenant", OwnerID: "owner", Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive, StoredBytes: 10},
	} {
		if err := db.Create(&session).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&TenantUsageModel{TenantID: "tenant", SessionCount: 2, StoredBytes: 20}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&CleanupJobModel{ID: "job", TenantID: "tenant", CanonicalSourceID: "session:cleanup-once", Status: "pending", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	run := func() bool {
		t.Helper()
		finalized := false
		if err := db.Transaction(func(tx *gorm.DB) error {
			var err error
			finalized, err = cleanupSessionData(tx, "cleanup-once", "tenant", now)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return finalized
	}
	if !run() {
		t.Fatal("first cleanup did not finalize the Session")
	}
	if run() {
		t.Fatal("second cleanup finalized an already-deleted Session")
	}
	var usage TenantUsageModel
	if err := db.First(&usage, "tenant_id = ?", "tenant").Error; err != nil {
		t.Fatal(err)
	}
	if usage.SessionCount != 1 || usage.StoredBytes != 10 {
		t.Fatalf("tenant usage = %+v, want the live Session only (1, 10)", usage)
	}
}

func openCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cleanup.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&SessionModel{}, &TurnModel{}, &MessageModel{}, &InteractionModel{},
		&DerivedArtifactModel{}, &CleanupJobModel{}, &OutboxEventModel{}, &TenantUsageModel{},
		&SchemaMigrationModel{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func runCleanupSessionData(t *testing.T, db *gorm.DB, sessionID string, now time.Time) bool {
	t.Helper()
	finalized := false
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		finalized, err = cleanupSessionData(tx, sessionID, "tenant", now)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return finalized
}

// TestCleanupSessionDataRefusesActiveSession guards the helper from deleting
// canonical data unless persisted state proves the Session is being deleted.
func TestCleanupSessionDataRefusesActiveSession(t *testing.T) {
	db := openCleanupTestDB(t)
	now := time.Now()
	if err := db.Create(&SessionModel{ID: "active", TenantID: "tenant", OwnerID: "owner", Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	turn := TurnModel{SessionID: "active", Status: conversationstore.InteractionStatusRunning, CreatedAt: now}
	if err := db.Create(&turn).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&MessageModel{TurnID: turn.ID, Sequence: 0, Payload: []byte(`{}`), CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if runCleanupSessionData(t, db, "active", now) {
		t.Fatal("cleanupSessionData finalized an active Session")
	}
	var session SessionModel
	if err := db.First(&session, "id = ?", "active").Error; err != nil {
		t.Fatal(err)
	}
	if session.Status != conversationstore.SessionStatusActive {
		t.Fatalf("active Session status = %q, want active", session.Status)
	}
	var messages int64
	if err := db.Model(&MessageModel{}).Where("turn_id = ?", turn.ID).Count(&messages).Error; err != nil {
		t.Fatal(err)
	}
	if messages != 1 {
		t.Fatalf("messages = %d, want 1; active Session data must be untouched", messages)
	}
}

// TestCleanupSessionDataDoesNotReopenCompletedJob covers a stale replica whose
// artifact count still blocks cleanup after another replica completed the job.
func TestCleanupSessionDataDoesNotReopenCompletedJob(t *testing.T) {
	db := openCleanupTestDB(t)
	now := time.Now()
	canonicalID := "session:derived"
	if err := db.Create(&SessionModel{ID: "derived", TenantID: "tenant", OwnerID: "owner", Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusDerivedCleanup}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&DerivedArtifactModel{ID: "artifact", TenantID: "tenant", CanonicalSourceID: canonicalID, Kind: "test", Locator: "test://late", Status: "active", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&CleanupJobModel{ID: "job", TenantID: "tenant", CanonicalSourceID: canonicalID, Status: "completed", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if runCleanupSessionData(t, db, "derived", now) {
		t.Fatal("cleanupSessionData finalized a Session with a live artifact")
	}
	var job CleanupJobModel
	if err := db.First(&job, "canonical_source_id = ?", canonicalID).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != "completed" {
		t.Fatalf("job status = %q, want completed; a late replica must not reopen it", job.Status)
	}
}

func assertUsageBackfillMarkerCount(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&SchemaMigrationModel{}).Where("name = ?", usageBackfillMigration).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("usage backfill marker count = %d, want %d", count, want)
	}
}

func TestClaimUsageBackfillSkipsExistingMarker(t *testing.T) {
	db := openCleanupTestDB(t)
	if err := db.Create(&SchemaMigrationModel{Name: usageBackfillMigration, AppliedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	claimed := false
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		claimed, err = claimUsageBackfill(tx)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if claimed {
		t.Fatal("claimUsageBackfill() claimed an existing marker")
	}
}

// TestMigrateReleasesUsageBackfillClaimOnFailure injects a failure into the
// one-time backfill and verifies that the claim is released so a later start
// can retry instead of being skipped forever by the fast path.
func TestMigrateReleasesUsageBackfillClaimOnFailure(t *testing.T) {
	db := openCleanupTestDB(t)
	ctx := context.Background()
	store, err := NewGormStore(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Now()
	expiresAt := now.Add(time.Hour)
	if err := db.Create(&SessionModel{
		ID: "trigger-session", TenantID: "tenant", OwnerID: "owner", Version: 1,
		CreatedAt: now, UpdatedAt: now, ExpiresAt: &expiresAt, Status: conversationstore.SessionStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TRIGGER fail_stored_bytes BEFORE UPDATE OF stored_bytes ON ai_sessions BEGIN SELECT RAISE(ABORT, 'injected backfill failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err == nil {
		t.Fatal("Migrate() succeeded despite the injected backfill failure")
	}
	assertUsageBackfillMarkerCount(t, db, 0)
	if err := db.Exec("DROP TRIGGER fail_stored_bytes").Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() retry error = %v", err)
	}
	assertUsageBackfillMarkerCount(t, db, 1)
}
