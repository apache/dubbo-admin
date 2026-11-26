/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
 */
package gormstore_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dubbo-admin-ai/store"
	gormstore "dubbo-admin-ai/store/gorm"
	storetest "dubbo-admin-ai/store/test"
	"dubbo-admin-ai/testutils"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExplicitMemoryContract(t *testing.T) {
	storetest.RunExplicitMemoryContractTests(t, func() store.BackendStore { return newSQLiteStore(t, filepath.Join(t.TempDir(), "memory.db"), 20) })
}

func TestDisableMemoryWhenTenantQuotaIsFull(t *testing.T) {
	quota := store.DefaultQuotaPolicy()
	quota.MaxTenantBytes = 1
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "disable-quota.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	s, err := gormstore.NewGormStoreWithPolicy(db, quota)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	a := testutils.TestAccess()
	if err := s.DB().Model(&gormstore.TenantUsageModel{}).Where("tenant_id = ?", a.Subject.TenantID).Update("stored_bytes", int64(1)).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		user := a
		user.Subject.UserID = fmt.Sprintf("full-quota-user-%d", i)
		off := false
		if _, err := s.ChangeMemorySettings(ctx, user, store.MemoryMutation{Generation: 1, StateVersion: 1, Enabled: &off}, false); err != nil {
			t.Fatalf("user %d disable at full quota: %v", i, err)
		}
	}
}

func TestExplicitMemoryZeroByteChanges(t *testing.T) {
	s := newSQLiteStore(t, filepath.Join(t.TempDir(), "zero-byte.db"), 20)
	// Reject no-op quota writes so SQLite also catches the MySQL regression.
	if err := s.DB().Exec(`CREATE TRIGGER reject_zero_byte_quota BEFORE UPDATE OF stored_bytes ON ai_tenant_usage
WHEN NEW.stored_bytes = OLD.stored_bytes BEGIN SELECT RAISE(ABORT, 'zero-byte quota update'); END`).Error; err != nil {
		t.Fatal(err)
	}
	runExplicitMemoryZeroByteChanges(t, s)
}

// Run against a dedicated test database, e.g. AI_TEST_MYSQL_DSN with parseTime=true.
func TestExplicitMemoryZeroByteChangesMySQL(t *testing.T) {
	dsn := os.Getenv("AI_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("AI_TEST_MYSQL_DSN is not set")
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid MySQL test DSN")
	}
	// Exercise default changed-row semantics regardless of supplied parameters.
	cfg.ClientFoundRows = false
	db, err := gormstore.Open("mysql", cfg.FormatDSN(), nil)
	if err != nil {
		t.Fatal("cannot open MySQL test database")
	}
	s, err := gormstore.NewGormStore(db, 20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	runExplicitMemoryZeroByteChanges(t, s)
	unchanged := s.DB().Exec("UPDATE ai_tenant_usage SET stored_bytes = stored_bytes + 0 WHERE tenant_id = ?", testutils.TestAccess().Subject.TenantID)
	if unchanged.Error != nil || unchanged.RowsAffected != 0 {
		t.Fatal("expected MySQL changed-row semantics", unchanged.RowsAffected, unchanged.Error)
	}
}

func runExplicitMemoryZeroByteChanges(t *testing.T, s *gormstore.GormStore) {
	t.Helper()
	ctx := context.Background()
	a := testutils.TestAccess()
	// Isolate runs without deleting existing test database contents.
	a.Subject.UserID = "zero-byte-" + time.Now().Format("20060102150405.000000000")
	w := store.MemoryWrite{Generation: 1, StateVersion: 1, ScopeType: "user", MemoryType: "preference", FactKey: "response.language", Value: "Chinese"}
	r, err := s.SaveMemory(ctx, a, w)
	if err != nil {
		t.Fatal(err)
	}
	check := func(t *testing.T, operation func()) {
		t.Helper()
		var before, after gormstore.MemoryLedgerModel
		var usageBefore, usageAfter gormstore.TenantUsageModel
		if err := s.DB().Where("owner_id = ?", a.Subject.UserID).First(&before).Error; err != nil {
			t.Fatal(err)
		}
		if err := s.DB().Where("tenant_id = ?", a.Subject.TenantID).First(&usageBefore).Error; err != nil {
			t.Fatal(err)
		}
		operation()
		if err := s.DB().Where("id = ?", before.ID).First(&after).Error; err != nil {
			t.Fatal(err)
		}
		if err := s.DB().Where("tenant_id = ?", a.Subject.TenantID).First(&usageAfter).Error; err != nil {
			t.Fatal(err)
		}
		if after.Revision != before.Revision+1 || after.StoredBytes != before.StoredBytes || usageAfter.StoredBytes != usageBefore.StoredBytes {
			t.Fatal("zero-byte change must persist ledger CAS without changing byte accounting")
		}
	}
	t.Run("duplicate save", func(t *testing.T) {
		check(t, func() {
			got, err := s.SaveMemory(ctx, a, w)
			if err != nil || got.ID != r.ID || got.Version != r.Version {
				t.Fatal("duplicate save changed identity/version or failed", got, err)
			}
		})
	})
	t.Run("unchanged enabled state", func(t *testing.T) {
		check(t, func() {
			enabled := true
			got, err := s.ChangeMemorySettings(ctx, a, store.MemoryMutation{Generation: 1, StateVersion: 1, Enabled: &enabled}, false)
			if err != nil || !got.Enabled || got.StateVersion != 1 {
				t.Fatal(got, err)
			}
		})
	})
	settings, err := s.ChangeMemorySettings(ctx, a, store.MemoryMutation{Generation: 1, StateVersion: 1}, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("clear empty ledger", func(t *testing.T) {
		check(t, func() {
			got, err := s.ChangeMemorySettings(ctx, a, store.MemoryMutation{Generation: settings.Generation, StateVersion: settings.StateVersion}, true)
			if err != nil || got.Generation != settings.Generation+1 {
				t.Fatal(got, err)
			}
			persisted, err := s.GetMemorySettings(ctx, a)
			if err != nil || persisted != got {
				t.Fatal("generation not persisted", persisted, err)
			}
		})
	})
}
func TestExplicitMemoryRestartAndArtifactRollback(t *testing.T) {
	ctx := context.Background()
	a := testutils.TestAccess()
	path := filepath.Join(t.TempDir(), "restart.db")
	s := newSQLiteStore(t, path, 20)
	if err := s.Create(ctx, a, testutils.TestSession("source", time.Now())); err != nil {
		t.Fatal(err)
	}
	w := store.MemoryWrite{Generation: 1, StateVersion: 1, ScopeType: "user", MemoryType: "preference", FactKey: "response.language", Value: "Chinese", SourceSessionID: "source"}
	if err := s.DB().Exec("CREATE TRIGGER reject_memory_artifact BEFORE INSERT ON ai_derived_artifacts BEGIN SELECT RAISE(ABORT, 'fixture'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveMemory(ctx, a, w); err == nil {
		t.Fatal("expected artifact failure")
	}
	page, err := s.ListMemories(ctx, a, "", 32)
	if err != nil || len(page.Records) != 0 {
		t.Fatal("orphan Memory after rollback", page, err)
	}
	if err := s.DB().Exec("DROP TRIGGER reject_memory_artifact").Error; err != nil {
		t.Fatal(err)
	}
	r, err := s.SaveMemory(ctx, a, w)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = newSQLiteStore(t, path, 20)
	page, err = s.ListMemories(ctx, a, "", 32)
	if err != nil || len(page.Records) != 1 || page.Records[0].ID != r.ID {
		t.Fatal("Memory lost on restart", page, err)
	}
	if _, err := s.ChangeMemorySettings(ctx, a, store.MemoryMutation{Generation: 1, StateVersion: 1}, true); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = newSQLiteStore(t, path, 20)
	settings, err := s.GetMemorySettings(ctx, a)
	if err != nil || settings.Generation != 2 {
		t.Fatal("generation lost on restart", settings, err)
	}
}

func TestMigrateRebuildsMemoryLedgerReservation(t *testing.T) {
	ctx := context.Background()
	a := testutils.TestAccess()
	s := newSQLiteStore(t, filepath.Join(t.TempDir(), "memory-migration.db"), 20)

	r, err := s.SaveMemory(ctx, a, store.MemoryWrite{Generation: 1, StateVersion: 1, ScopeType: "user", MemoryType: "preference", FactKey: "response.language", Value: "Chinese"})
	if err != nil {
		t.Fatal(err)
	}

	// Simulate pre-fix accounting, where each ledger reserved its whole
	// serialized payload instead of only its Memory records.
	var row gormstore.MemoryLedgerModel
	if err := s.DB().Where("id = ?", store.MemoryOwnerKey(a.Subject.TenantID, a.Subject.UserID)).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	var ledger store.MemoryLedger
	if err := json.Unmarshal(row.Payload, &ledger); err != nil {
		t.Fatal(err)
	}
	contentBytes := store.MemoryContentBytes(ledger)
	legacy := int64(len(row.Payload))
	if legacy <= contentBytes {
		t.Fatalf("fixture does not exercise divergent accounting: legacy=%d content=%d", legacy, contentBytes)
	}
	if err := s.DB().Model(&gormstore.MemoryLedgerModel{}).Where("id = ?", row.ID).Update("stored_bytes", legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB().Model(&gormstore.TenantUsageModel{}).Where("tenant_id = ?", a.Subject.TenantID).Update("stored_bytes", legacy).Error; err != nil {
		t.Fatal(err)
	}

	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	var rebuilt gormstore.MemoryLedgerModel
	if err := s.DB().Where("id = ?", row.ID).First(&rebuilt).Error; err != nil {
		t.Fatal(err)
	}
	var usage gormstore.TenantUsageModel
	if err := s.DB().Where("tenant_id = ?", a.Subject.TenantID).First(&usage).Error; err != nil {
		t.Fatal(err)
	}
	if rebuilt.StoredBytes != contentBytes || usage.StoredBytes != contentBytes {
		t.Fatalf("migration must rebuild ledger and tenant together: ledger=%d tenant=%d want=%d", rebuilt.StoredBytes, usage.StoredBytes, contentBytes)
	}

	// A mutation after migration must release exactly the rebuilt reservation.
	if err := s.DeleteMemory(ctx, a, r.ID, store.MemoryMutation{Generation: 1, StateVersion: 1, Version: r.Version}); err != nil {
		t.Fatal(err)
	}
	if err := s.DB().Where("tenant_id = ?", a.Subject.TenantID).First(&usage).Error; err != nil {
		t.Fatal(err)
	}
	if usage.StoredBytes != 0 {
		t.Fatalf("tenant stored_bytes after deleting the only Memory = %d, want 0", usage.StoredBytes)
	}
}
