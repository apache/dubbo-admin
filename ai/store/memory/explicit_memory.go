/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
 */
package memory

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
)

var _ store.ExplicitMemoryStore = (*MemoryStore)(nil)

func (m *MemoryStore) memoryLedgerLocked(a security.AccessContext) store.MemoryLedger {
	l, ok := m.explicitMemory[store.MemoryOwnerKey(a.Subject.TenantID, a.Subject.UserID)]
	if !ok {
		l = store.NewMemoryLedger()
		l.TenantID = a.Subject.TenantID
		l.OwnerID = a.Subject.UserID
	}
	data, _ := json.Marshal(l)
	var cloned store.MemoryLedger
	_ = json.Unmarshal(data, &cloned)
	return cloned
}
func (m *MemoryStore) memorySourceLiveLocked(a security.AccessContext, id string) bool {
	s, ok := m.sessions[id]
	return ok && s.TenantID == a.Subject.TenantID && s.OwnerID == a.Subject.UserID && m.validateSessionLocked(a, id) == nil
}
func memoryAccess(ctx context.Context, a security.AccessContext) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	return a.ValidateBusiness()
}
func (m *MemoryStore) GetMemorySettings(ctx context.Context, a security.AccessContext) (store.MemorySettings, error) {
	if err := memoryAccess(ctx, a); err != nil {
		return store.MemorySettings{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.memoryLedgerLocked(a).MemorySettings, nil
}
func (m *MemoryStore) ListMemories(ctx context.Context, a security.AccessContext, cursor string, limit int) (store.MemoryPage, error) {
	if err := memoryAccess(ctx, a); err != nil {
		return store.MemoryPage{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return store.MemoryList(m.memoryLedgerLocked(a), cursor, limit, time.Now(), func(id string) bool { return m.memorySourceLiveLocked(a, id) })
}
func (m *MemoryStore) putMemoryLedgerLocked(a security.AccessContext, l store.MemoryLedger) error {
	key := store.MemoryOwnerKey(a.Subject.TenantID, a.Subject.UserID)
	old, exists := m.explicitMemory[key]
	beforeBytes := store.MemoryContentBytes(old)
	after, _ := json.Marshal(l)
	afterBytes := store.MemoryContentBytes(l)
	additional := len(l.Records) - len(old.Records)
	if !exists {
		additional++
	}
	if m.tenantBytesLocked(a.Subject.TenantID)+afterBytes-beforeBytes > m.quota.MaxTenantBytes || m.entityCountLocked()+additional > m.quota.MaxInMemoryEntities {
		return store.ErrResourceExhausted
	}
	if m.explicitMemory == nil {
		m.explicitMemory = map[string]store.MemoryLedger{}
	}
	// The caller's expiry pointer and returned record must not alias storage.
	var persisted store.MemoryLedger
	_ = json.Unmarshal(after, &persisted)
	m.explicitMemory[key] = persisted
	return nil
}
func (m *MemoryStore) SaveMemory(ctx context.Context, a security.AccessContext, w store.MemoryWrite) (*store.ExplicitMemory, error) {
	if err := memoryAccess(ctx, a); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if w.SourceSessionID != "" && !m.memorySourceLiveLocked(a, w.SourceSessionID) {
		return nil, security.ErrResourceNotAccessible
	}
	l := m.memoryLedgerLocked(a)
	r, err := l.Save(a, w, m.quota.ExplicitMemory, time.Now())
	if err != nil {
		return nil, err
	}
	if err = m.putMemoryLedgerLocked(a, l); err != nil {
		return nil, err
	}
	if r.SourceSessionID != "" {
		id := store.MemoryArtifactID(r.ID)
		now := time.Now()
		m.derivedArtifacts[id] = store.DerivedArtifact{ID: id, TenantID: r.TenantID, CanonicalSourceID: "session:" + r.SourceSessionID, Kind: store.MemoryArtifactKind, Locator: r.OwnerID + ":" + r.ID, Status: "active", CreatedAt: now, UpdatedAt: now}
	}
	return r, nil
}
func (m *MemoryStore) DeleteMemory(ctx context.Context, a security.AccessContext, id string, mutation store.MemoryMutation) error {
	if err := memoryAccess(ctx, a); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	l := m.memoryLedgerLocked(a)
	if err := l.Delete(id, mutation); err != nil {
		return err
	}
	if err := m.putMemoryLedgerLocked(a, l); err != nil {
		return err
	}
	delete(m.derivedArtifacts, store.MemoryArtifactID(id))
	return nil
}
func (m *MemoryStore) ChangeMemorySettings(ctx context.Context, a security.AccessContext, mutation store.MemoryMutation, clear bool) (store.MemorySettings, error) {
	if err := memoryAccess(ctx, a); err != nil {
		return store.MemorySettings{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	l := m.memoryLedgerLocked(a)
	previous := append([]store.ExplicitMemory(nil), l.Records...)
	if err := l.Change(mutation, clear); err != nil {
		return store.MemorySettings{}, err
	}
	err := m.putMemoryLedgerLocked(a, l)
	if err == nil && clear {
		for _, r := range previous {
			delete(m.derivedArtifacts, store.MemoryArtifactID(r.ID))
		}
	}
	return l.MemorySettings, err
}
func (m *MemoryStore) RecallMemories(ctx context.Context, a security.AccessContext, match store.MatchContext, limit, bytes int) ([]store.ExplicitMemory, store.MemorySettings, error) {
	if err := memoryAccess(ctx, a); err != nil {
		return nil, store.MemorySettings{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	l := m.memoryLedgerLocked(a)
	r, err := store.MemoryRecall(l, match, limit, bytes, time.Now(), func(id string) bool { return m.memorySourceLiveLocked(a, id) })
	return r, l.MemorySettings, err
}
func (m *MemoryStore) validateMemoryDependenciesLocked(a security.AccessContext, id string, deps []store.MemoryDependency) error {
	if len(deps) == 0 {
		return nil
	}
	s, ok := m.sessions[id]
	if !ok || !a.CanAccessOwner(s.TenantID, s.OwnerID) {
		return security.ErrResourceNotAccessible
	}
	// Memory ownership is stricter than the Session administrator permission.
	owner := security.MustAccessContext(security.Subject{TenantID: s.TenantID, UserID: s.OwnerID, AuthMethod: "internal-owner"})
	l := m.memoryLedgerLocked(owner)
	for _, d := range deps {
		if !store.MemoryDependencyValid(l, d, time.Now(), func(source string) bool { return m.memorySourceLiveLocked(owner, source) }) {
			return store.ErrMemoryDependency
		}
	}
	return nil
}
func (m *MemoryStore) ValidateMemoryDependencies(ctx context.Context, a security.AccessContext, id string, deps []store.MemoryDependency) error {
	if err := memoryAccess(ctx, a); err != nil {
		return err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.validateSessionLocked(a, id); err != nil {
		return err
	}
	return m.validateMemoryDependenciesLocked(a, id, deps)
}
func (m *MemoryStore) RecordTurnMemoryDependencies(ctx context.Context, a security.AccessContext, id string, turnID uint64, deps []store.MemoryDependency) error {
	if err := memoryAccess(ctx, a); err != nil {
		return err
	}
	if len(deps) > 256 {
		return store.ErrResourceExhausted
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(a, id); err != nil {
		return err
	}
	h := m.history[id]
	if h == nil || h.active[turnID] == nil {
		return store.ErrTurnNotFound
	}
	// Persist the actual old dependencies even if they changed during Provider IO.
	merged := store.MergeMemoryDependencies(h.active[turnID].dependencies, deps)
	if len(merged) > 256 {
		return store.ErrResourceExhausted
	}
	old, _ := json.Marshal(h.active[turnID].dependencies)
	next, _ := json.Marshal(merged)
	delta := int64(len(next)-len(old)) * 2
	if len(h.active[turnID].dependencies) == 0 {
		delta = int64(len(next)) * 2
	}
	if m.sessionBytesLocked(id)+delta > m.quota.MaxSessionBytes || m.tenantBytesLocked(a.Subject.TenantID)+delta > m.quota.MaxTenantBytes {
		return store.ErrResourceExhausted
	}
	h.active[turnID].dependencies = merged
	return nil
}
func (m *MemoryStore) DeleteSourceMemory(ctx context.Context, a security.AccessContext, artifact store.DerivedArtifact) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if !a.IsSystem() || artifact.Kind != store.MemoryArtifactKind {
		return security.ErrResourceNotAccessible
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	source, err := store.SessionIDFromCanonicalSource(artifact.CanonicalSourceID)
	if err != nil {
		return err
	}
	if s, ok := m.sessions[source]; ok && s.Status == store.SessionStatusActive && !(s.ExpiresAt != nil && !s.ExpiresAt.After(time.Now())) {
		return security.ErrResourceNotAccessible
	}
	split := strings.LastIndex(artifact.Locator, ":")
	if split < 1 {
		return store.ErrMemoryInvalid
	}
	key := store.MemoryOwnerKey(artifact.TenantID, artifact.Locator[:split])
	l := m.explicitMemory[key]
	if l.TenantID == artifact.TenantID {
		for i, r := range l.Records {
			if store.MemoryArtifactID(r.ID) == artifact.ID && r.SourceSessionID == source {
				l.Records = append(l.Records[:i:i], l.Records[i+1:]...)
				m.explicitMemory[key] = l
				return nil
			}
		}
	}
	return nil
}

func (m *MemoryStore) deleteSessionMemoriesLocked(session store.Session) {
	key := store.MemoryOwnerKey(session.TenantID, session.OwnerID)
	l, ok := m.explicitMemory[key]
	if !ok {
		return
	}
	kept := make([]store.ExplicitMemory, 0, len(l.Records))
	for _, r := range l.Records {
		if r.SourceSessionID != session.ID {
			kept = append(kept, r)
		}
	}
	l.Records = kept
	m.explicitMemory[key] = l
}
