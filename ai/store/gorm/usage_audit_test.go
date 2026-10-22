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
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
	gormstore "dubbo-admin-ai/store/gorm"
	"dubbo-admin-ai/testutils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUsageAuditComponentPersistsCanceledEndEventsWithVersions(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(cause.Error(), func(t *testing.T) {
			s := newSQLiteStore(t, filepath.Join(t.TempDir(), "recorder.db"))
			component := hooks.NewComponent(hooks.Spec{Hooks: []hooks.HookSpec{}})
			if err := component.SetUsageAuditStore(s); err != nil {
				t.Fatal(err)
			}
			if err := component.Init(runtime.NewRuntime()); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = component.Stop() })
			// Server updates only the rollout flags after hooks Init.
			component.SetUsageAuditFeatures(false, false)
			component.GetManager().Emit(testutils.ContextWithTestAccess(context.Background()), hooks.State{Event: hooks.EventModelCallEnd})
			var count int64
			if err := s.DB().Model(&gormstore.AuditEventModel{}).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("disabled audit count = %d, %v", count, err)
			}
			if err := s.DB().Model(&gormstore.UsageEventModel{}).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("disabled usage count = %d, %v", count, err)
			}
			component.SetUsageAuditFeatures(true, true)
			parent := testutils.ContextWithTestAccess(context.Background())
			var ctx context.Context
			if cause == context.Canceled {
				canceled, cancel := context.WithCancel(parent)
				cancel()
				ctx = canceled
			} else {
				expired, cancel := context.WithDeadline(parent, time.Now().Add(-time.Second))
				defer cancel()
				ctx = expired
			}
			for _, event := range []hooks.Event{hooks.EventModelCallEnd, hooks.EventToolCallEnd} {
				state := hooks.State{Event: event, ToolName: "tool", SessionID: "session", InteractionID: "interaction", Error: "private error body", ErrorType: "timeout"}
				if event == hooks.EventModelCallEnd {
					state.CallID = "call"
				} else {
					state.ToolCallID = "tool-call"
				}
				returned := component.GetManager().Emit(ctx, state)
				if !errors.Is(returned.Err(), cause) {
					t.Fatalf("hook changed caller cancellation: %v", returned.Err())
				}
			}
			var audits []gormstore.AuditEventModel
			if err := s.DB().Find(&audits).Error; err != nil {
				t.Fatal(err)
			}
			if len(audits) != 2 {
				t.Fatalf("audit count = %d", len(audits))
			}
			for _, audit := range audits {
				if audit.SchemaVersion != hooks.UsageAuditSchemaVersion || audit.PolicyVersion != hooks.UsageAuditPolicyVersion || audit.ComponentVersion != hooks.UsageAuditComponentVersion || audit.ReasonCode != "timeout" || audit.Status != "error" || audit.TenantID != testutils.TestAccess().Subject.TenantID || audit.UserID != testutils.TestAccess().Subject.UserID {
					t.Fatalf("audit provenance = %+v", audit)
				}
				if audit.EventType == string(hooks.EventModelCallEnd) {
					if audit.CallID != "call" || audit.ToolCallID != "" {
						t.Fatalf("model audit identity = %+v", audit)
					}
				} else if audit.ToolCallID != "tool-call" || audit.CallID != "" {
					t.Fatalf("tool audit identity = %+v", audit)
				}
			}
			var usage []gormstore.UsageEventModel
			if err := s.DB().Find(&usage).Error; err != nil {
				t.Fatal(err)
			}
			if len(usage) != 1 || usage[0].SchemaVersion != hooks.UsageAuditSchemaVersion || usage[0].PolicyVersion != hooks.UsageAuditPolicyVersion || usage[0].CallID != "call" || usage[0].InputTokens != nil || usage[0].ReasonCode != "timeout" || usage[0].Status != "error" {
				t.Fatalf("usage = %+v", usage)
			}
		})
	}
}

type legacyAuditWithoutCallIDs struct {
	ID            string `gorm:"primaryKey;size:64"`
	TenantID      string
	UserID        string
	InteractionID string
	EventType     string
	CreatedAt     time.Time
}

func (legacyAuditWithoutCallIDs) TableName() string { return "ai_audit_events" }

func TestAuditCallIdentifierMigrationPreservesExistingEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-migration.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&legacyAuditWithoutCallIDs{}); err != nil {
		t.Fatal(err)
	}
	legacy := legacyAuditWithoutCallIDs{ID: "legacy", TenantID: testutils.TestAccess().Subject.TenantID, UserID: testutils.TestAccess().Subject.UserID, InteractionID: "interaction", EventType: "interaction.start", CreatedAt: time.Now()}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	s := newSQLiteStore(t, path)
	ctx := testutils.ContextWithTestAccess(context.Background())
	events := []conversationstore.AuditEvent{
		{ID: "model-1", CallID: "call-1", EventType: "model_call.end"},
		{ID: "model-2", CallID: "call-2", EventType: "model_call.end"},
		{ID: "tool-1", ToolCallID: "tool-call-1", EventType: "tool_call.end"},
		{ID: "tool-2", ToolCallID: "tool-call-2", EventType: "tool_call.end"},
	}
	for _, event := range events {
		event.TenantID, event.InteractionID = legacy.TenantID, legacy.InteractionID
		if err := s.RecordAudit(ctx, event); err != nil {
			t.Fatal(err)
		}
		var stored gormstore.AuditEventModel
		if err := s.DB().First(&stored, "id = ?", event.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.InteractionID != event.InteractionID || stored.CallID != event.CallID || stored.ToolCallID != event.ToolCallID {
			t.Fatalf("call identity lost: %+v", stored)
		}
	}
	var stored gormstore.AuditEventModel
	if err := s.DB().First(&stored, "id = ?", legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.InteractionID != legacy.InteractionID || stored.EventType != legacy.EventType || stored.CallID != "" || stored.ToolCallID != "" {
		t.Fatalf("legacy audit changed: %+v", stored)
	}
}

func TestGormEventCapacityAcrossInstancesAndRollback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capacity.db")
	s := newSQLiteStore(t, path)
	// A separate connection pool represents another process sharing the database.
	db, err := gorm.Open(sqlite.Open(path+"?_busy_timeout=10000"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	other, err := gormstore.NewGormStore(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = other.Close() })
	for _, store := range []*gormstore.GormStore{s, other} {
		store.SetUsageAuditPolicy(conversationstore.UsageAuditPolicy{MaxUsageEvents: 3})
	}
	ctx := testutils.ContextWithTestAccess(context.Background())
	old := time.Now().Add(-40 * 24 * time.Hour)
	record := func(store *gormstore.GormStore, id string) error {
		return store.RecordUsage(ctx, conversationstore.UsageEvent{ID: id, TenantID: testutils.TestAccess().Subject.TenantID, CreatedAt: old})
	}
	if err := record(s, "duplicate"); err != nil {
		t.Fatal(err)
	}
	if err := record(other, "duplicate"); err == nil {
		t.Fatal("duplicate insert succeeded")
	}
	results := make(chan error, 10)
	for i := 0; i < cap(results); i++ {
		go func(i int) {
			store := s
			if i%2 == 1 {
				store = other
			}
			results <- record(store, fmt.Sprintf("concurrent-%d", i))
		}(i)
	}
	success := 0
	for i := 0; i < cap(results); i++ {
		err := <-results
		if err == nil {
			success++
		} else if !errors.Is(err, conversationstore.ErrResourceExhausted) {
			t.Error(err)
		}
	}
	if success != 2 {
		t.Fatalf("successful reservations = %d, want 2", success)
	}
	var counter gormstore.EventCapacityModel
	if err := db.First(&counter, "kind = ?", "usage").Error; err != nil || counter.EventCount != 3 {
		t.Fatalf("counter = %+v, %v", counter, err)
	}
	// Concurrent cleaners share the capacity lock as well.
	cleaned := make(chan int, 2)
	for _, store := range []*gormstore.GormStore{s, other} {
		go func(store *gormstore.GormStore) {
			n, err := store.CleanupUsage(ctx, security.SystemAccessContext(), time.Now(), 2)
			if err != nil {
				t.Error(err)
			}
			cleaned <- n
		}(store)
	}
	if n := <-cleaned + <-cleaned; n != 3 {
		t.Fatalf("cleanup total = %d", n)
	}
	if err := record(other, "after-cleanup"); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&counter, "kind = ?", "usage").Error; err != nil || counter.EventCount != 1 {
		t.Fatalf("counter after cleanup = %+v, %v", counter, err)
	}
}

func TestGormEventCapacityMigrationAdoptsExistingRows(t *testing.T) {
	s := newSQLiteStore(t, filepath.Join(t.TempDir(), "existing.db"))
	if err := s.DB().Create(&gormstore.UsageEventModel{ID: "existing", TenantID: "tenant", CreatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB().Where("kind = ?", "usage").Delete(&gormstore.EventCapacityModel{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.SetUsageAuditPolicy(conversationstore.UsageAuditPolicy{MaxUsageEvents: 1})
	ctx := testutils.ContextWithTestAccess(context.Background())
	if err := s.RecordUsage(ctx, conversationstore.UsageEvent{ID: "new", TenantID: testutils.TestAccess().Subject.TenantID}); !errors.Is(err, conversationstore.ErrResourceExhausted) {
		t.Fatalf("existing rows not counted: %v", err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	var counter gormstore.EventCapacityModel
	if err := s.DB().First(&counter, "kind = ?", "usage").Error; err != nil || counter.EventCount != 1 {
		t.Fatalf("repeat migration changed count = %+v, %v", counter, err)
	}
}
