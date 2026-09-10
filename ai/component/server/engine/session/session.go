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

package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	rt "dubbo-admin-ai/runtime"
	conversationstore "dubbo-admin-ai/store"

	"github.com/google/uuid"
)

var (
	ErrSessionNotFound = conversationstore.ErrSessionNotFound
	ErrSessionExpired  = conversationstore.ErrSessionExpired
)

// Session is kept as an alias for callers that use the session package. The
// Store package owns the persisted session fields and snapshot semantics.
type Session = conversationstore.Session

// Manager coordinates SessionStore operations and periodic expiration cleanup.
type Manager struct {
	store conversationstore.SessionStore

	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
	cleanupWG     sync.WaitGroup
	closeOnce     sync.Once
}

// NewManager creates a session manager backed by the supplied SessionStore.
func NewManager(sessionStore conversationstore.SessionStore) *Manager {
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	m := &Manager{
		store:         sessionStore,
		cleanupCtx:    cleanupCtx,
		cleanupCancel: cleanupCancel,
	}
	if sessionStore != nil {
		m.cleanupWG.Add(1)
		go m.cleanupExpiredSessions()
	}
	return m
}

// CreateSession creates a new active session in the configured Store.
func (m *Manager) CreateSession(ctx context.Context) (*Session, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	now := time.Now()
	session := &Session{
		ID:        generateSessionID(),
		CreatedAt: now,
		UpdatedAt: now,
		Status:    "active",
	}
	if err := m.store.Create(ctx, session); err != nil {
		return nil, err
	}
	rt.GetLogger().Info("Session created", "session_id", session.ID)
	return session, nil
}

// GetSession retrieves a session snapshot from the configured Store.
func (m *Manager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	return m.store.Get(ctx, sessionID)
}

// TouchSession records activity for an active session.
func (m *Manager) TouchSession(ctx context.Context, sessionID string) error {
	if err := m.requireStore(); err != nil {
		return err
	}
	return m.store.Touch(ctx, sessionID, time.Now())
}

// DeleteSession removes a session and its conversation history.
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	if err := m.requireStore(); err != nil {
		return err
	}
	if err := m.store.Delete(ctx, sessionID); err != nil {
		return err
	}
	rt.GetLogger().Info("Session deleted", "session_id", sessionID)
	return nil
}

// ListSessions returns active, non-expired session snapshots.
func (m *Manager) ListSessions(ctx context.Context) ([]*Session, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	return m.store.List(ctx)
}

// Close stops expiration cleanup. It does not close the underlying Store.
func (m *Manager) Close() error {
	m.closeOnce.Do(func() {
		m.cleanupCancel()
		m.cleanupWG.Wait()
	})
	return nil
}

func (m *Manager) cleanupExpiredSessions() {
	defer m.cleanupWG.Done()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			deleted, err := m.store.DeleteExpired(m.cleanupCtx, time.Now())
			if err != nil {
				rt.GetLogger().Error("Failed to clean expired sessions", "error", err)
				continue
			}
			if deleted > 0 {
				rt.GetLogger().Info("Cleaned up expired sessions", "count", deleted)
			}
		case <-m.cleanupCtx.Done():
			return
		}
	}
}

func (m *Manager) requireStore() error {
	if m == nil || m.store == nil {
		return fmt.Errorf("session store is not configured")
	}
	return nil
}

// generateSessionID generates a session ID.
func generateSessionID() string {
	return "session_" + uuid.New().String()
}
