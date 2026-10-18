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

package gormstore_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
	gormstore "dubbo-admin-ai/store/gorm"
	storetest "dubbo-admin-ai/store/test"
	"dubbo-admin-ai/testutils"

	"github.com/firebase/genkit/go/ai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSQLiteStore(t *testing.T, path string, limits ...int) *gormstore.GormStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	store, err := gormstore.NewGormStore(db, limits...)
	if err != nil {
		t.Fatalf("NewGormStore() error = %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

type legacySessionModel struct {
	ID        string `gorm:"primaryKey;size:64"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    string
}

func (legacySessionModel) TableName() string { return "ai_sessions" }

func TestGormStore_Contract(t *testing.T) {
	storetest.RunStoreContractTests(t, func() conversationstore.BackendStore {
		return newSQLiteStore(t, filepath.Join(t.TempDir(), "contract.db"), 3)
	}, 3)
}

func TestGormStore_MigratesAndAdoptsLegacySessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&legacySessionModel{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.Create(&legacySessionModel{ID: "legacy", CreatedAt: now, UpdatedAt: now, Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	store, err := gormstore.NewGormStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() legacy schema error = %v", err)
	}
	access := testutils.TestAccess()
	partialTenant := gormstore.SessionModel{ID: "partial-tenant", TenantID: "tenant-existing", CreatedAt: now, UpdatedAt: now, Status: "active", Version: 1}
	partialOwner := gormstore.SessionModel{ID: "partial-owner", OwnerID: "owner-existing", CreatedAt: now, UpdatedAt: now, Status: "active", Version: 1}
	if err := store.DB().Create(&partialTenant).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.DB().Create(&partialOwner).Error; err != nil {
		t.Fatal(err)
	}

	adopted, err := store.AdoptLegacySessions(context.Background(), security.SystemAccessContext(), access.Subject.TenantID, access.Subject.UserID)
	if err != nil || adopted != 1 {
		t.Fatalf("AdoptLegacySessions() = %d, %v", adopted, err)
	}
	got, err := store.Get(context.Background(), access, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	if got.TenantID != access.Subject.TenantID || got.OwnerID != access.Subject.UserID || got.Version != 1 || got.ExpiresAt == nil {
		t.Fatalf("adopted session = %+v", got)
	}
	for id, want := range map[string][2]string{
		"partial-tenant": {"tenant-existing", ""},
		"partial-owner":  {"", "owner-existing"},
	} {
		var row gormstore.SessionModel
		if err := store.DB().First(&row, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		if row.TenantID != want[0] || row.OwnerID != want[1] {
			t.Fatalf("partially owned session %q changed to tenant=%q owner=%q", id, row.TenantID, row.OwnerID)
		}
	}
	admin := security.MustAccessContext(security.Subject{
		TenantID: access.Subject.TenantID, UserID: "admin", Scopes: []string{security.ScopeSessionAdmin}, AuthMethod: security.AuthMethodStatic,
	})
	for _, id := range []string{"partial-tenant", "partial-owner"} {
		if _, err := store.Get(context.Background(), admin, id); !errors.Is(err, conversationstore.ErrSessionNotFound) {
			t.Fatalf("admin Get(%q) error = %v, want ErrSessionNotFound", id, err)
		}
	}
	listed, err := store.List(context.Background(), admin)
	if err != nil {
		t.Fatalf("admin List() error = %v", err)
	}
	for _, session := range listed {
		if session.ID == "partial-tenant" || session.ID == "partial-owner" {
			t.Fatalf("admin List() returned partially owned session %+v", session)
		}
	}
}

func TestGormStore_UsesExactIdentityComparison(t *testing.T) {
	store := newSQLiteStore(t, filepath.Join(t.TempDir(), "identity.db"))
	now := time.Now()
	expiresAt := now.Add(time.Hour)
	for _, session := range []gormstore.SessionModel{
		{ID: "case-tenant", TenantID: "TEST-TENANT", OwnerID: "test-user", Version: 1, CreatedAt: now, UpdatedAt: now, ExpiresAt: &expiresAt, Status: "active"},
		{ID: "case-owner", TenantID: "test-tenant", OwnerID: "TEST-USER", Version: 1, CreatedAt: now, UpdatedAt: now, ExpiresAt: &expiresAt, Status: "active"},
	} {
		if err := store.DB().Create(&session).Error; err != nil {
			t.Fatalf("seed case-variant session %q: %v", session.ID, err)
		}
	}
	access := testutils.TestAccess()
	for _, id := range []string{"case-tenant", "case-owner"} {
		if _, err := store.Get(context.Background(), access, id); !errors.Is(err, conversationstore.ErrSessionNotFound) {
			t.Fatalf("Get(%q) error = %v, want ErrSessionNotFound", id, err)
		}
	}
}

func TestGormStore_RestartRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conversation.db")
	ctx := context.Background()
	access := testutils.TestAccess()
	first := newSQLiteStore(t, path)
	session := testutils.TestSession("restart-session", time.Now())
	if err := first.Create(ctx, access, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := first.AddHistory(ctx, access, session.ID, ai.NewSystemTextMessage("system"), ai.NewUserTextMessage("hello")); err != nil {
		t.Fatalf("AddHistory() error = %v", err)
	}
	if err := first.NextTurn(ctx, access, session.ID); err != nil {
		t.Fatalf("NextTurn() error = %v", err)
	}
	if err := first.AddHistory(ctx, access, session.ID, ai.NewModelTextMessage("answer")); err != nil {
		t.Fatalf("second AddHistory() error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}

	second := newSQLiteStore(t, path)
	got, err := second.AllMemory(ctx, access, session.ID)
	if err != nil {
		t.Fatalf("AllMemory() after restart error = %v", err)
	}
	if len(got) != 3 || got[0].Text() != "answer" || got[1].Text() != "system" || got[2].Text() != "hello" {
		t.Fatalf("AllMemory() after restart = %#v, want active message followed by archived messages", got)
	}
}

func TestGormStore_MultiInstanceSharedDataAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.db")
	ctx := context.Background()
	access := testutils.TestAccess()
	first := newSQLiteStore(t, path)
	second := newSQLiteStore(t, path)
	session := testutils.TestSession("shared-session", time.Now())
	if err := first.Create(ctx, access, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := first.AddHistory(ctx, access, session.ID, ai.NewUserTextMessage("from-first")); err != nil {
		t.Fatalf("first AddHistory() error = %v", err)
	}
	got, err := second.WindowMemory(ctx, access, session.ID)
	if err != nil {
		t.Fatalf("second WindowMemory() error = %v", err)
	}
	if len(got) != 1 || got[0].Text() != "from-first" {
		t.Fatalf("second WindowMemory() = %#v, want first instance message", got)
	}
	if err := second.Delete(ctx, access, session.ID); err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if _, err := first.AllMemory(ctx, access, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
		t.Fatalf("first AllMemory() after shared delete error = %v, want ErrSessionNotFound", err)
	}
}

func TestGormStore_AddHistoryRollsBackWhenMessageWriteFails(t *testing.T) {
	store := newSQLiteStore(t, filepath.Join(t.TempDir(), "rollback.db"))
	ctx := context.Background()
	access := testutils.TestAccess()
	session := testutils.TestSession("rollback-session", time.Now())
	if err := store.Create(ctx, access, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Fail after the Message INSERT callback has run. AddHistory must roll back
	// both that write and the Turn created earlier in the same transaction.
	if err := store.DB().Callback().Create().After("gorm:create").Register(
		"test:fail_message_create",
		func(tx *gorm.DB) {
			if tx.Statement.Table == "ai_messages" {
				tx.AddError(errors.New("injected message write failure"))
			}
		},
	); err != nil {
		t.Fatalf("register failure callback error = %v", err)
	}

	if err := store.AddHistory(ctx, access, session.ID, ai.NewUserTextMessage("should rollback")); err == nil {
		t.Fatal("AddHistory() succeeded, want injected write failure")
	}

	var turnCount int64
	if err := store.DB().Model(&gormstore.TurnModel{}).
		Where("session_id = ?", session.ID).Count(&turnCount).Error; err != nil {
		t.Fatalf("count turns error = %v", err)
	}
	if turnCount != 0 {
		t.Fatalf("turn count after rollback = %d, want 0", turnCount)
	}

	var messageCount int64
	if err := store.DB().Model(&gormstore.MessageModel{}).Count(&messageCount).Error; err != nil {
		t.Fatalf("count messages error = %v", err)
	}
	if messageCount != 0 {
		t.Fatalf("message count after rollback = %d, want 0", messageCount)
	}
}

func TestGormStore_MessageSequenceIsUniquePerTurn(t *testing.T) {
	store := newSQLiteStore(t, filepath.Join(t.TempDir(), "sequence.db"))
	ctx := context.Background()
	access := testutils.TestAccess()
	session := testutils.TestSession("sequence-session", time.Now())
	if err := store.Create(ctx, access, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.AddHistory(ctx, access, session.ID, ai.NewUserTextMessage("first")); err != nil {
		t.Fatalf("AddHistory() error = %v", err)
	}

	var turn gormstore.TurnModel
	if err := store.DB().Where("session_id = ?", session.ID).First(&turn).Error; err != nil {
		t.Fatalf("find turn error = %v", err)
	}
	duplicate := &gormstore.MessageModel{
		TurnID:    turn.ID,
		Sequence:  0,
		Payload:   []byte(`{"role":"user","content":[{"text":"duplicate"}]}`),
		CreatedAt: time.Now(),
	}
	if err := store.DB().Create(duplicate).Error; err == nil {
		t.Fatal("duplicate (turn_id, sequence) insert succeeded, want unique constraint error")
	}

	var count int64
	if err := store.DB().Model(&gormstore.MessageModel{}).
		Where("turn_id = ?", turn.ID).Count(&count).Error; err != nil {
		t.Fatalf("count messages error = %v", err)
	}
	if count != 1 {
		t.Fatalf("message count after duplicate insert = %d, want 1", count)
	}
}

func TestGormStore_DeleteExpiredDoesNotDeleteRefreshedSession(t *testing.T) {
	store := newSQLiteStore(t, filepath.Join(t.TempDir(), "expired-race.db"))
	now := time.Now()
	access := testutils.TestAccess()
	session := testutils.TestSession("expired-race-session", now.Add(-25*time.Hour))
	session.ExpiresAt = nil
	if err := store.Create(context.Background(), access, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Simulate another instance refreshing the row after cleanup has selected it
	// but before cleanup revalidates it for deletion.
	refreshed := false
	if err := store.DB().Callback().Query().After("gorm:after_query").Register(
		"test:refresh_expired_session",
		func(tx *gorm.DB) {
			if refreshed {
				return
			}
			rows, ok := tx.Statement.Dest.(*[]gormstore.SessionModel)
			if !ok || len(*rows) == 0 {
				return
			}
			refreshed = true
			if err := tx.Session(&gorm.Session{NewDB: true}).Exec("UPDATE ai_sessions SET updated_at = ? WHERE id = ?", now, session.ID).Error; err != nil {
				tx.AddError(err)
			}
		},
	); err != nil {
		t.Fatalf("register refresh callback error = %v", err)
	}

	deleted, err := store.DeleteExpired(context.Background(), security.SystemAccessContext(), now)
	if err != nil {
		t.Fatalf("DeleteExpired() error = %v", err)
	}
	if deleted != 0 {
		t.Fatalf("DeleteExpired() = %d, want 0 for refreshed session", deleted)
	}
	if _, err := store.Get(context.Background(), access, session.ID); err != nil {
		t.Fatalf("Get() after refreshed cleanup error = %v", err)
	}
}

func TestGormStore_NoForeignKeysAndInvalidPayload(t *testing.T) {
	store := newSQLiteStore(t, filepath.Join(t.TempDir(), "schema.db"))
	var count int
	for _, table := range []string{"ai_sessions", "ai_turns", "ai_messages"} {
		if err := store.DB().Raw("SELECT count(*) FROM pragma_foreign_key_list(?)", table).Scan(&count).Error; err != nil {
			t.Fatalf("foreign key inspection for %s error = %v", table, err)
		}
		if count != 0 {
			t.Fatalf("table %s has %d foreign keys", table, count)
		}
	}

	ctx := context.Background()
	access := testutils.TestAccess()
	session := testutils.TestSession("invalid-payload", time.Now())
	if err := store.Create(ctx, access, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.AddHistory(ctx, access, session.ID, ai.NewUserTextMessage("valid")); err != nil {
		t.Fatalf("AddHistory() error = %v", err)
	}
	var turn gormstore.TurnModel
	if err := store.DB().Where("session_id = ?", session.ID).First(&turn).Error; err != nil {
		t.Fatalf("find turn error = %v", err)
	}
	if err := store.DB().Model(&gormstore.MessageModel{}).Where("turn_id = ?", turn.ID).Update("payload", []byte("{")).Error; err != nil {
		t.Fatalf("corrupt payload error = %v", err)
	}
	_, err := store.WindowMemory(ctx, access, session.ID)
	if !containsMessageDecodeError(err) {
		t.Fatalf("WindowMemory() error = %v, want decode error", err)
	}
}

func containsMessageDecodeError(err error) bool {
	return err != nil && len(err.Error()) >= len("failed to decode message") &&
		err.Error()[:len("failed to decode message")] == "failed to decode message"
}
