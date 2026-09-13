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

package storetest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	conversationstore "dubbo-admin-ai/store"

	"github.com/firebase/genkit/go/ai"
)

// RunStoreContractTests verifies behavior shared by every Store backend.
// Backend-specific tests should call this function with their own constructor.
func RunStoreContractTests(t *testing.T, newStore func() conversationstore.Store, limit int) {
	t.Helper()
	if limit <= 0 {
		t.Fatalf("contract test limit must be positive, got %d", limit)
	}
	begin := func(t *testing.T, s conversationstore.Store, ctx context.Context, sessionID string) uint64 {
		t.Helper()
		turnID, err := s.BeginTurn(ctx, sessionID)
		if err != nil {
			t.Fatalf("BeginTurn() error = %v", err)
		}
		return turnID
	}

	t.Run("session lifecycle and history deletion", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		session := &conversationstore.Session{
			ID:        "contract-session",
			CreatedAt: now,
			UpdatedAt: now,
			Status:    "active",
		}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserMessage(ai.NewTextPart("hello"))); err != nil {
			t.Fatalf("AddHistory() error = %v", err)
		}
		if err := s.Delete(ctx, session.ID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if _, err := s.Get(ctx, session.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
			t.Fatalf("Get() after Delete() error = %v, want ErrSessionNotFound", err)
		}
		messages, err := s.AllMemory(ctx, session.ID)
		if err != nil {
			t.Fatalf("AllMemory() after Delete() error = %v", err)
		}
		if len(messages) != 0 {
			t.Fatalf("AllMemory() after Delete() = %d messages, want 0", len(messages))
		}
	})

	t.Run("message roles and turn order", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		session := &conversationstore.Session{ID: "order-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID,
			ai.NewUserMessage(ai.NewTextPart("first")),
			ai.NewSystemMessage(ai.NewTextPart("system")),
			ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart("model")),
			nil,
			ai.NewMessage(ai.RoleTool, nil, ai.NewTextPart("ignored")),
		); err != nil {
			t.Fatalf("AddHistory() error = %v", err)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
			t.Fatalf("NextTurn() error = %v", err)
		}
		turnID = begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserMessage(ai.NewTextPart("second"))); err != nil {
			t.Fatalf("second AddHistory() error = %v", err)
		}
		messages, err := s.AllMemory(ctx, session.ID)
		if err != nil {
			t.Fatalf("AllMemory() error = %v", err)
		}
		if len(messages) != 4 || messages[0].Content[0].Text != "second" || messages[1].Role != ai.RoleSystem || messages[2].Role != ai.RoleUser || messages[3].Role != ai.RoleModel {
			t.Fatalf("AllMemory() = %#v, want active turn followed by system/user/model", messages)
		}
	})

	t.Run("concurrent turns remain isolated", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		session := &conversationstore.Session{ID: "concurrent-turn-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		first, err := s.BeginTurn(ctx, session.ID)
		if err != nil {
			t.Fatalf("BeginTurn(first) error = %v", err)
		}
		second, err := s.BeginTurn(ctx, session.ID)
		if err != nil {
			t.Fatalf("BeginTurn(second) error = %v", err)
		}
		if first == second {
			t.Fatalf("BeginTurn() returned duplicate IDs: %d", first)
		}
		if err := s.AddHistoryToTurn(ctx, session.ID, first, ai.NewUserTextMessage("first")); err != nil {
			t.Fatalf("AddHistoryToTurn(first) error = %v", err)
		}
		if err := s.AddHistoryToTurn(ctx, session.ID, second, ai.NewUserTextMessage("second")); err != nil {
			t.Fatalf("AddHistoryToTurn(second) error = %v", err)
		}
		firstMessages, err := s.WindowMemoryForTurn(ctx, session.ID, first)
		if err != nil || len(firstMessages) != 1 || firstMessages[0].Content[0].Text != "first" {
			t.Fatalf("first turn messages = %#v, error = %v", firstMessages, err)
		}
		secondMessages, err := s.WindowMemoryForTurn(ctx, session.ID, second)
		if err != nil || len(secondMessages) != 1 || secondMessages[0].Content[0].Text != "second" {
			t.Fatalf("second turn messages = %#v, error = %v", secondMessages, err)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, first); err != nil {
			t.Fatalf("NextTurnForTurn(first) error = %v", err)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, second); err != nil {
			t.Fatalf("NextTurnForTurn(second) error = %v", err)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, first); !errors.Is(err, conversationstore.ErrTurnNotFound) {
			t.Fatalf("repeated NextTurnForTurn(first) error = %v, want ErrTurnNotFound", err)
		}
	})

	t.Run("aborting a turn removes its partial history", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		session := &conversationstore.Session{ID: "abort-turn-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserTextMessage("partial")); err != nil {
			t.Fatalf("AddHistoryToTurn() error = %v", err)
		}
		if err := s.AbortTurnForTurn(ctx, session.ID, turnID); err != nil {
			t.Fatalf("AbortTurnForTurn() error = %v", err)
		}
		if _, err := s.WindowMemoryForTurn(ctx, session.ID, turnID); !errors.Is(err, conversationstore.ErrTurnNotFound) {
			t.Fatalf("WindowMemoryForTurn() after abort error = %v, want ErrTurnNotFound", err)
		}
		if messages, err := s.AllMemory(ctx, session.ID); err != nil || len(messages) != 0 {
			t.Fatalf("AllMemory() after abort = %#v, error = %v, want empty history", messages, err)
		}
		if nextID, err := s.BeginTurn(ctx, session.ID); err != nil || nextID == 0 {
			t.Fatalf("BeginTurn() after abort = %d, error = %v", nextID, err)
		}
	})

	t.Run("returned messages are snapshots", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		session := &conversationstore.Session{ID: "snapshot-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		message := ai.NewUserMessage(ai.NewTextPart("original"))
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, message); err != nil {
			t.Fatalf("AddHistory() error = %v", err)
		}
		message.Content[0].Text = "caller mutation"
		stored, err := s.WindowMemoryForTurn(ctx, session.ID, turnID)
		if err != nil {
			t.Fatalf("WindowMemory() error = %v", err)
		}
		if len(stored) != 1 || stored[0].Content[0].Text != "original" {
			t.Fatalf("stored message = %#v, want original snapshot", stored)
		}
	})

	t.Run("message batch failure does not expose partial writes", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		session := &conversationstore.Session{ID: "batch-failure-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		invalid := &ai.Message{
			Role:     ai.RoleUser,
			Metadata: map[string]any{"unsupported": func() {}},
		}
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserTextMessage("valid"), invalid); err == nil {
			t.Fatal("AddHistoryToTurn() succeeded with an unencodable message")
		}
		messages, err := s.WindowMemoryForTurn(ctx, session.ID, turnID)
		if errors.Is(err, conversationstore.ErrTurnNotFound) {
			return
		}
		if err != nil {
			t.Fatalf("WindowMemoryForTurn() error = %v", err)
		}
		if len(messages) != 0 {
			t.Fatalf("WindowMemoryForTurn() = %#v, want no partial messages", messages)
		}
	})

	t.Run("empty and unsupported history still creates an active turn", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		session := &conversationstore.Session{ID: "empty-turn-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, nil, ai.NewMessage(ai.RoleTool, nil, ai.NewTextPart("ignored"))); err != nil {
			t.Fatalf("AddHistory() error = %v", err)
		}
		empty, err := s.IsTurnEmpty(ctx, session.ID, turnID)
		if err != nil {
			t.Fatalf("IsEmpty() error = %v", err)
		}
		if empty {
			t.Fatal("IsEmpty() = true after an empty AddHistory(), want false")
		}
		messages, err := s.WindowMemoryForTurn(ctx, session.ID, turnID)
		if err != nil {
			t.Fatalf("WindowMemory() error = %v", err)
		}
		if len(messages) != 0 {
			t.Fatalf("WindowMemory() = %#v, want empty", messages)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
			t.Fatalf("NextTurn() on empty active turn error = %v", err)
		}
		if err := s.NextTurnForTurn(ctx, session.ID, turnID); !errors.Is(err, conversationstore.ErrTurnNotFound) {
			t.Fatalf("NextTurnForTurn() without active turn = %v, want ErrTurnNotFound", err)
		}
	})

	t.Run("closed session rejects history writes", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		session := &conversationstore.Session{ID: "closed-session", CreatedAt: now, UpdatedAt: now, Status: "closed"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if _, err := s.BeginTurn(ctx, session.ID); err == nil || !strings.Contains(err.Error(), "not active") {
			t.Fatalf("BeginTurn() error = %v, want inactive-session error", err)
		}
	})

	t.Run("expiration is consistent across session operations", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		expired := &conversationstore.Session{
			ID:        "expired-contract-session",
			CreatedAt: now.Add(-25 * time.Hour),
			UpdatedAt: now.Add(-25 * time.Hour),
			Status:    "active",
		}
		active := &conversationstore.Session{ID: "active-contract-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		for _, session := range []*conversationstore.Session{expired, active} {
			if err := s.Create(ctx, session); err != nil {
				t.Fatalf("Create(%s) error = %v", session.ID, err)
			}
		}
		if _, err := s.Get(ctx, expired.ID); !errors.Is(err, conversationstore.ErrSessionExpired) {
			t.Fatalf("Get(expired) error = %v, want ErrSessionExpired", err)
		}
		if err := s.Touch(ctx, expired.ID, now); !errors.Is(err, conversationstore.ErrSessionExpired) {
			t.Fatalf("Touch(expired) error = %v, want ErrSessionExpired", err)
		}
		listed, err := s.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(listed) != 1 || listed[0].ID != active.ID {
			t.Fatalf("List() = %#v, want only active session", listed)
		}
		deleted, err := s.DeleteExpired(ctx, now)
		if err != nil {
			t.Fatalf("DeleteExpired() error = %v", err)
		}
		if deleted != 1 {
			t.Fatalf("DeleteExpired() = %d, want 1", deleted)
		}
		if _, err := s.Get(ctx, expired.ID); !errors.Is(err, conversationstore.ErrSessionNotFound) {
			t.Fatalf("Get(expired) after DeleteExpired error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("turn limit and failed transition preserve active turn", func(t *testing.T) {
		ctx := context.Background()
		s := newStore()
		now := time.Now()
		session := &conversationstore.Session{ID: "limit-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
		if err := s.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		turnID := begin(t, s, ctx, session.ID)
		if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserTextMessage("turn-0")); err != nil {
			t.Fatalf("initial AddHistory() error = %v", err)
		}
		for i := 1; i < limit; i++ {
			if err := s.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
				t.Fatalf("NextTurnForTurn() at turn %d error = %v", i-1, err)
			}
			turnID = begin(t, s, ctx, session.ID)
			if err := s.AddHistoryToTurn(ctx, session.ID, turnID, ai.NewUserTextMessage(fmt.Sprintf("turn-%d", i))); err != nil {
				t.Fatalf("AddHistory() at turn %d error = %v", i, err)
			}
		}
		if err := s.NextTurnForTurn(ctx, session.ID, turnID); err != nil {
			t.Fatalf("final NextTurn() error = %v, want success", err)
		}
		if _, err := s.BeginTurn(ctx, session.ID); !errors.Is(err, conversationstore.ErrTurnLimitReached) {
			t.Fatalf("overflow BeginTurn() error = %v, want ErrTurnLimitReached", err)
		}
		messages, err := s.AllMemory(ctx, session.ID)
		if err != nil {
			t.Fatalf("AllMemory() after rejected turn error = %v", err)
		}
		if len(messages) != limit {
			t.Fatalf("AllMemory() after rejected turn = %d messages, want %d", len(messages), limit)
		}
	})
}
