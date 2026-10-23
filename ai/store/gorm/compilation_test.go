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
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	"dubbo-admin-ai/testutils"
	"github.com/firebase/genkit/go/ai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCompilationLedgerAndPinSurviveRestartAndAreCleanedWithSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.db")
	s := newSQLiteStore(t, path, 10)
	ctx := context.Background()
	a := testutils.TestAccess()
	if err := s.Create(ctx, a, testutils.TestSession("restart", time.Now())); err != nil {
		t.Fatal(err)
	}
	p, err := s.PrepareInteraction(ctx, a, "restart", "request", "hash", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddHistoryToTurn(ctx, a, "restart", p.TurnID, ai.NewUserTextMessage("question")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PinTemplate(ctx, a, "restart", "react@1:content"); err != nil {
		t.Fatal(err)
	}
	b := store.ToolBatch{ID: "batch", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "restart", TurnID: p.TurnID, InteractionID: p.InteractionID, Version: 1, SchemaVersion: "v1", Sequence: 1, Calls: []store.ToolCall{{ID: "call", Name: "lookup", Input: json.RawMessage(`{}`), Status: "pending"}}}
	if err := s.CreateToolBatch(ctx, a, b); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := newSQLiteStore(t, path, 10)
	rows, err := reopened.ToolBatches(ctx, a, "restart", p.TurnID, 10)
	if err != nil || len(rows) != 1 || rows[0].Calls[0].Status != "pending" {
		t.Fatalf("pending execution lost after restart: %+v %v", rows, err)
	}
	ref, err := reopened.PinTemplate(ctx, a, "restart", "react@2:changed")
	if err != nil || ref != "react@1:content" {
		t.Fatalf("pin changed after restart: %q %v", ref, err)
	}
	if err := reopened.FailInteraction(ctx, a, "restart", p.InteractionID, p.TurnID); err != nil {
		t.Fatal(err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	reopened = newSQLiteStore(t, path, 10)
	next, err := reopened.PrepareInteraction(ctx, a, "restart", "next", "next", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := reopened.RecoveryToolBatches(ctx, a, "restart", next.TurnID, 16, 65536)
	if err != nil || len(recovery.Batches) != 1 || recovery.Batches[0].Calls[0].Status != "pending" {
		t.Fatalf("released record not recovered after restart: %+v %v", recovery, err)
	}
	if err := reopened.Delete(ctx, a, "restart"); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.DeleteExpired(ctx, security.SystemAccessContext(), time.Now(), 10); err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	var count int64
	if err := db.Table("ai_tool_batches").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("tool ledger survived physical session cleanup: %d %v", count, err)
	}
}
