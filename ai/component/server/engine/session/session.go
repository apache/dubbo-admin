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
	"dubbo-admin-ai/security"
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
	store       conversationstore.SessionStore
	maintenance conversationstore.MaintenanceStore
	policy      security.PolicyEnforcer
	lifecycle   conversationstore.LifecyclePolicy

	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
	cleanupWG     sync.WaitGroup
	cleanupCursor conversationstore.CleanupCursor
	closeOnce     sync.Once
}

// NewManager creates a session manager with separate business and privileged
// lifecycle Store dependencies.
func NewManager(sessionStore conversationstore.SessionStore, maintenanceStore conversationstore.MaintenanceStore) *Manager {
	return NewManagerWithLifecycle(sessionStore, maintenanceStore, security.OwnershipPolicy{}, conversationstore.DefaultLifecyclePolicy())
}

func NewManagerWithPolicy(sessionStore conversationstore.SessionStore, maintenanceStore conversationstore.MaintenanceStore, policy security.PolicyEnforcer) *Manager {
	return NewManagerWithLifecycle(sessionStore, maintenanceStore, policy, conversationstore.DefaultLifecyclePolicy())
}

func NewManagerWithLifecycle(sessionStore conversationstore.SessionStore, maintenanceStore conversationstore.MaintenanceStore, policy security.PolicyEnforcer, lifecycle conversationstore.LifecyclePolicy) *Manager {
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	m := &Manager{
		store:         sessionStore,
		maintenance:   maintenanceStore,
		policy:        policy,
		lifecycle:     lifecycle,
		cleanupCtx:    cleanupCtx,
		cleanupCancel: cleanupCancel,
	}
	if maintenanceStore != nil {
		m.cleanupWG.Add(1)
		go m.cleanupExpiredSessions()
	}
	return m
}

// CreateSession creates a new active session in the configured Store.
func (m *Manager) CreateSession(ctx context.Context, access security.AccessContext, templateRefs ...string) (*Session, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	now := time.Now()
	expiresAt := now.Add(m.lifecycle.Retention)
	session := &Session{
		ID:        generateSessionID(),
		TenantID:  access.Subject.TenantID,
		OwnerID:   access.Subject.UserID,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: &expiresAt,
		Status:    "active",
	}
	if len(templateRefs) > 1 {
		return nil, fmt.Errorf("only one template reference is allowed")
	}
	if len(templateRefs) == 1 {
		if err := conversationstore.ValidateTemplateRef(templateRefs[0]); err != nil {
			return nil, err
		}
		session.TemplateRef = templateRefs[0]
	}
	if err := m.policy.AuthorizeOwner(access, session.TenantID, session.OwnerID); err != nil {
		return nil, err
	}
	if err := m.store.Create(ctx, access, session); err != nil {
		return nil, err
	}
	rt.GetLogger().Info("Session created", "session_id", session.ID)
	return session, nil
}

// GetSession retrieves a session snapshot from the configured Store.
func (m *Manager) GetSession(ctx context.Context, access security.AccessContext, sessionID string) (*Session, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	session, err := m.store.Get(ctx, access, sessionID)
	if err != nil {
		return nil, err
	}
	if err := m.policy.AuthorizeOwner(access, session.TenantID, session.OwnerID); err != nil {
		return nil, err
	}
	return session, nil
}

// TouchSession records activity for an active session.
func (m *Manager) TouchSession(ctx context.Context, access security.AccessContext, sessionID string) error {
	if err := m.requireStore(); err != nil {
		return err
	}
	if _, err := m.GetSession(ctx, access, sessionID); err != nil {
		return err
	}
	return m.store.Touch(ctx, access, sessionID, time.Now(), m.lifecycle.Retention)
}

// DeleteSession removes a session and its conversation history.
func (m *Manager) DeleteSession(ctx context.Context, access security.AccessContext, sessionID string) error {
	if err := m.requireStore(); err != nil {
		return err
	}
	if _, err := m.GetSession(ctx, access, sessionID); err != nil {
		return err
	}
	if err := m.store.Delete(ctx, access, sessionID); err != nil {
		return err
	}
	rt.GetLogger().Info("Session deleted", "session_id", sessionID)
	return nil
}

// ListSessions returns active, non-expired session snapshots.
func (m *Manager) ListSessions(ctx context.Context, access security.AccessContext) ([]*Session, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	sessions, err := m.store.List(ctx, access)
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		if err := m.policy.AuthorizeOwner(access, session.TenantID, session.OwnerID); err != nil {
			return nil, err
		}
	}
	return sessions, nil
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
	ticker := time.NewTicker(m.lifecycle.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			deleted, next, err := m.maintenance.DeleteExpiredPage(m.cleanupCtx, security.SystemAccessContext(), time.Now(), m.cleanupCursor, m.lifecycle.CleanupBatchSize)
			if err != nil {
				rt.GetLogger().Error("Failed to clean expired sessions", "error", err)
				continue
			}
			previousCursor := m.cleanupCursor
			m.cleanupCursor = next
			if !next.Valid || (deleted == 0 && next == previousCursor) {
				m.cleanupCursor = conversationstore.CleanupCursor{}
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
	if m.policy == nil {
		return fmt.Errorf("session policy is not configured")
	}
	return nil
}

// generateSessionID generates a session ID.
func generateSessionID() string {
	return "session_" + uuid.New().String()
}
