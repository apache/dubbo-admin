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
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"

	"github.com/firebase/genkit/go/ai"
	"github.com/google/uuid"
)

// DefaultMaxTurns matches MemorySpec's default conversation limit.
const DefaultMaxTurns = 100

type turn struct {
	dependencies    []conversationstore.MemoryDependency
	id              uint64
	interactionID   string
	status          string
	createdAt       time.Time
	orderedMessages []*ai.Message
}

func (t *turn) messages() []*ai.Message {
	return t.orderedMessages
}

type sessionHistory struct {
	active  map[uint64]*turn
	history []*turn
	nextID  uint64
}

// MemoryStore is the in-process implementation of the conversation Store.
type MemoryStore struct {
	summaryTurns       map[string]map[uint64]uint64
	summaryChecks      map[string]map[string]conversationstore.SummaryInspection
	summaryHistory     map[string][]conversationstore.HistoryEntry
	summaries          map[string]conversationstore.Summary
	toolBatches        map[string]conversationstore.ToolBatch
	mu                 sync.RWMutex
	limit              int
	quota              conversationstore.QuotaPolicy
	lifecycle          conversationstore.LifecyclePolicy
	usagePolicy        conversationstore.UsageAuditPolicy
	sessions           map[string]conversationstore.Session
	history            map[string]*sessionHistory
	interactions       map[string]conversationstore.Interaction
	idempotency        map[string]string
	activeInteractions map[string]string
	derivedArtifacts   map[string]conversationstore.DerivedArtifact
	cleanupJobs        map[string]conversationstore.CleanupJob
	outbox             map[string]conversationstore.OutboxEvent
	usageEvents        list.List
	auditEvents        list.List
	usageCleanupCursor *list.Element
	auditCleanupCursor *list.Element
}

var _ conversationstore.BackendStore = (*MemoryStore)(nil)
var _ conversationstore.UsageAuditMaintenanceStore = (*MemoryStore)(nil)

// NewMemoryStore creates a MemoryStore. The optional limit exists for runtime
// configuration and tests; the default matches MemorySpec.
func NewMemoryStore(limits ...int) *MemoryStore {
	quota := conversationstore.DefaultQuotaPolicy()
	limit := DefaultMaxTurns
	if len(limits) > 0 && limits[0] > 0 {
		limit = limits[0]
	}
	quota.MaxTurnsPerSession = limit
	return NewMemoryStoreWithPolicy(quota)
}

func NewMemoryStoreWithPolicy(quota conversationstore.QuotaPolicy) *MemoryStore {
	return NewMemoryStoreWithPolicies(quota, conversationstore.DefaultLifecyclePolicy())
}

func NewMemoryStoreWithPolicies(quota conversationstore.QuotaPolicy, lifecycle conversationstore.LifecyclePolicy) *MemoryStore {
	quota = quota.WithDefaults()
	lifecycle = lifecycle.WithDefaults()
	usagePolicy := conversationstore.DefaultUsageAuditPolicy()
	return &MemoryStore{
		toolBatches:        make(map[string]conversationstore.ToolBatch),
		limit:              quota.MaxTurnsPerSession,
		quota:              quota,
		lifecycle:          lifecycle,
		usagePolicy:        usagePolicy,
		sessions:           make(map[string]conversationstore.Session),
		history:            make(map[string]*sessionHistory),
		interactions:       make(map[string]conversationstore.Interaction),
		idempotency:        make(map[string]string),
		activeInteractions: make(map[string]string),
		derivedArtifacts:   make(map[string]conversationstore.DerivedArtifact),
		cleanupJobs:        make(map[string]conversationstore.CleanupJob),
		outbox:             make(map[string]conversationstore.OutboxEvent),
	}
}

func (m *MemoryStore) SetUsageAuditPolicy(policy conversationstore.UsageAuditPolicy) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.usagePolicy = policy.WithDefaults()
	m.mu.Unlock()
}

func (m *MemoryStore) Create(ctx context.Context, access security.AccessContext, session *conversationstore.Session) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := access.ValidateBusiness(); err != nil {
		return err
	}
	if err := conversationstore.ValidateSession(session); err != nil {
		return err
	}
	if !access.CanAccessOwner(session.TenantID, session.OwnerID) {
		return security.ErrResourceNotAccessible
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	sessionCount := 0
	for _, existing := range m.sessions {
		if existing.TenantID == session.TenantID && existing.Status != conversationstore.SessionStatusDeleted {
			sessionCount++
		}
	}
	if sessionCount >= m.quota.MaxSessionsPerTenant || m.entityCountLocked() >= m.quota.MaxInMemoryEntities {
		return conversationstore.ErrResourceExhausted
	}
	if _, exists := m.sessions[session.ID]; exists {
		return fmt.Errorf("session %q already exists", session.ID)
	}
	stored := cloneSession(*session)
	if stored.ExpiresAt == nil {
		expiresAt := stored.UpdatedAt.Add(m.lifecycle.Retention)
		stored.ExpiresAt = &expiresAt
	}
	m.sessions[session.ID] = stored
	return nil
}

func (m *MemoryStore) Get(ctx context.Context, access security.AccessContext, sessionID string) (*conversationstore.Session, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if err := access.ValidateBusiness(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	session, exists := m.sessions[sessionID]
	if !exists || !access.CanAccessOwner(session.TenantID, session.OwnerID) {
		return nil, conversationstore.ErrSessionNotFound
	}
	if session.Status != conversationstore.SessionStatusActive {
		return nil, conversationstore.ErrSessionNotFound
	}
	if m.isExpired(session, time.Now()) {
		return nil, conversationstore.ErrSessionExpired
	}
	copy := cloneSession(session)
	return &copy, nil
}

func (m *MemoryStore) List(ctx context.Context, access security.AccessContext) ([]*conversationstore.Session, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if err := access.ValidateBusiness(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*conversationstore.Session, 0, len(m.sessions))
	now := time.Now()
	for _, session := range m.sessions {
		if !access.CanAccessOwner(session.TenantID, session.OwnerID) || session.Status != "active" || m.isExpired(session, now) {
			continue
		}
		copy := cloneSession(session)
		result = append(result, &copy)
	}
	return result, nil
}

func (m *MemoryStore) Touch(ctx context.Context, access security.AccessContext, sessionID string, updatedAt time.Time, retentions ...time.Duration) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := access.ValidateBusiness(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	session, exists := m.sessions[sessionID]
	if !exists || !access.CanAccessOwner(session.TenantID, session.OwnerID) {
		return conversationstore.ErrSessionNotFound
	}
	if session.Status != "active" {
		return fmt.Errorf("session %q is not active", sessionID)
	}
	if m.isExpired(session, time.Now()) {
		return conversationstore.ErrSessionExpired
	}
	session.UpdatedAt = updatedAt
	retention := m.lifecycle.Retention
	if len(retentions) > 0 && retentions[0] > 0 {
		retention = retentions[0]
	}
	expiresAt := updatedAt.Add(retention)
	session.ExpiresAt = &expiresAt
	session.Version++
	m.sessions[sessionID] = session
	return nil
}

func (m *MemoryStore) Delete(ctx context.Context, access security.AccessContext, sessionID string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := access.ValidateBusiness(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	session, exists := m.sessions[sessionID]
	if !exists || !access.CanAccessOwner(session.TenantID, session.OwnerID) {
		return conversationstore.ErrSessionNotFound
	}
	if session.Status != conversationstore.SessionStatusActive {
		return conversationstore.ErrSessionNotFound
	}
	session.Status = conversationstore.SessionStatusDeletionPending
	session.Version++
	session.UpdatedAt = time.Now()
	m.sessions[sessionID] = session
	delete(m.activeInteractions, sessionID)
	m.enqueueCleanupLocked(session)
	return nil
}

func (m *MemoryStore) DeleteExpired(ctx context.Context, access security.AccessContext, now time.Time, batchSizes ...int) (int, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	if err := access.Validate(); err != nil {
		return 0, err
	}
	if !access.IsSystem() {
		return 0, security.ErrResourceNotAccessible
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	batchLimit := 100
	if len(batchSizes) > 0 && batchSizes[0] > 0 {
		batchLimit = batchSizes[0]
	}
	type cleanupCandidate struct {
		sessionID string
		tenantID  string
		updatedAt time.Time
	}
	candidates := make([]cleanupCandidate, 0)
	for sessionID, session := range m.sessions {
		if (session.Status == conversationstore.SessionStatusActive && m.isExpired(session, now)) || session.Status == conversationstore.SessionStatusDeletionPending || session.Status == conversationstore.SessionStatusDerivedCleanup {
			candidates = append(candidates, cleanupCandidate{sessionID: sessionID, tenantID: session.TenantID, updatedAt: session.UpdatedAt})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].updatedAt.Equal(candidates[j].updatedAt) {
			return candidates[i].updatedAt.Before(candidates[j].updatedAt)
		}
		if candidates[i].tenantID != candidates[j].tenantID {
			return candidates[i].tenantID < candidates[j].tenantID
		}
		return candidates[i].sessionID < candidates[j].sessionID
	})
	if len(candidates) > batchLimit {
		candidates = candidates[:batchLimit]
	}
	deleted := 0
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		sessionID := candidate.sessionID
		session := m.sessions[sessionID]
		if session.Status == conversationstore.SessionStatusActive && !m.isExpired(session, now) {
			continue
		}
		if session.Status == conversationstore.SessionStatusDeleted {
			continue
		}
		if session.Status == conversationstore.SessionStatusActive {
			session.Status = conversationstore.SessionStatusDeletionPending
			session.Version++
			m.enqueueCleanupLocked(session)
		}
		session.Status = conversationstore.SessionStatusDerivedCleanup
		canonicalID := "session:" + sessionID
		blocked := false
		for _, artifact := range m.derivedArtifacts {
			if artifact.CanonicalSourceID == canonicalID && artifact.Status != "deleted" {
				blocked = true
				break
			}
		}
		job := m.cleanupJobs[canonicalID]
		job.Attempts++
		job.UpdatedAt = now
		if blocked {
			job.Status = "cleanup_failed"
			job.LastErrorCode = "HANDLER_REQUIRED"
			m.cleanupJobs[canonicalID] = job
			m.sessions[sessionID] = session
			continue
		}
		delete(m.history, sessionID)
		delete(m.summaryHistory, sessionID)
		delete(m.summaryTurns, sessionID)
		delete(m.summaryChecks, sessionID)
		delete(m.summaries, sessionID)
		m.deleteToolBatchesLocked(sessionID)
		for id, interaction := range m.interactions {
			if interaction.SessionID == sessionID {
				delete(m.interactions, id)
			}
		}
		for key, id := range m.idempotency {
			if _, exists := m.interactions[id]; !exists {
				delete(m.idempotency, key)
			}
		}
		delete(m.activeInteractions, sessionID)
		session.Status = conversationstore.SessionStatusDeleted
		session.UpdatedAt = now
		m.sessions[sessionID] = session
		job.Status = "completed"
		job.LastErrorCode = ""
		job.NextRetryAt = nil
		m.cleanupJobs[canonicalID] = job
		deleted++
	}
	return deleted, nil
}

func (m *MemoryStore) DeleteExpiredPage(ctx context.Context, access security.AccessContext, now time.Time, cursor conversationstore.CleanupCursor, limit int) (int, conversationstore.CleanupCursor, error) {
	if err := checkContext(ctx); err != nil {
		return 0, cursor, err
	}
	if err := access.Validate(); err != nil {
		return 0, cursor, err
	}
	if !access.IsSystem() {
		return 0, cursor, security.ErrResourceNotAccessible
	}
	if limit <= 0 {
		limit = 100
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	type candidate struct {
		id, tenant string
		at         time.Time
	}
	items := make([]candidate, 0)
	for id, session := range m.sessions {
		if (session.Status == conversationstore.SessionStatusActive && m.isExpired(session, now)) || session.Status == conversationstore.SessionStatusDeletionPending || session.Status == conversationstore.SessionStatusDerivedCleanup {
			if cursor.Valid && !afterCleanupCursor(session.UpdatedAt, session.TenantID, id, cursor) {
				continue
			}
			items = append(items, candidate{id, session.TenantID, session.UpdatedAt})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return cleanupCandidateBefore(items[i].at, items[i].tenant, items[i].id, items[j].at, items[j].tenant, items[j].id)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	next := conversationstore.CleanupCursor{}
	deleted := 0
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return deleted, next, err
		}
		next = conversationstore.CleanupCursor{SortTime: item.at, TenantID: item.tenant, ID: item.id, Valid: true}
		session := m.sessions[item.id]
		if session.Status == conversationstore.SessionStatusActive && !m.isExpired(session, now) {
			continue
		}
		if session.Status == conversationstore.SessionStatusActive {
			session.Status = conversationstore.SessionStatusDeletionPending
			session.Version++
			m.enqueueCleanupLocked(session)
		}
		session.Status = conversationstore.SessionStatusDerivedCleanup
		canonicalID := "session:" + item.id
		blocked := false
		for _, artifact := range m.derivedArtifacts {
			if artifact.CanonicalSourceID == canonicalID && artifact.Status != "deleted" {
				blocked = true
				break
			}
		}
		job := m.cleanupJobs[canonicalID]
		job.Attempts++
		job.UpdatedAt = now
		if blocked {
			job.Status = "cleanup_failed"
			job.LastErrorCode = "HANDLER_REQUIRED"
			m.cleanupJobs[canonicalID] = job
			session.UpdatedAt = now
			m.sessions[item.id] = session
			continue
		}
		delete(m.history, item.id)
		delete(m.summaryHistory, item.id)
		delete(m.summaryTurns, item.id)
		delete(m.summaryChecks, item.id)
		delete(m.summaries, item.id)
		m.deleteToolBatchesLocked(item.id)
		for id, interaction := range m.interactions {
			if interaction.SessionID == item.id {
				delete(m.interactions, id)
			}
		}
		for key, id := range m.idempotency {
			if _, exists := m.interactions[id]; !exists {
				delete(m.idempotency, key)
			}
		}
		delete(m.activeInteractions, item.id)
		session.Status = conversationstore.SessionStatusDeleted
		session.UpdatedAt = now
		m.sessions[item.id] = session
		job.Status = "completed"
		job.LastErrorCode = ""
		job.NextRetryAt = nil
		m.cleanupJobs[canonicalID] = job
		deleted++
	}
	return deleted, next, nil
}

func cleanupCandidateBefore(at time.Time, tenant, id string, otherAt time.Time, otherTenant, otherID string) bool {
	if !at.Equal(otherAt) {
		return at.Before(otherAt)
	}
	if tenant != otherTenant {
		return tenant < otherTenant
	}
	return id < otherID
}

func afterCleanupCursor(at time.Time, tenant, id string, cursor conversationstore.CleanupCursor) bool {
	return cleanupCandidateBefore(cursor.SortTime, cursor.TenantID, cursor.ID, at, tenant, id)
}

func (m *MemoryStore) enqueueCleanupLocked(session conversationstore.Session) {
	canonicalID := "session:" + session.ID
	if _, exists := m.cleanupJobs[canonicalID]; exists {
		return
	}
	now := time.Now()
	m.cleanupJobs[canonicalID] = conversationstore.CleanupJob{ID: uuid.NewString(), TenantID: session.TenantID, CanonicalSourceID: canonicalID, Status: "pending", CreatedAt: now, UpdatedAt: now}
	event := conversationstore.OutboxEvent{ID: uuid.NewString(), TenantID: session.TenantID, AggregateID: canonicalID, EventType: "session.deletion_requested", CreatedAt: now}
	m.outbox[event.ID] = event
}

func (m *MemoryStore) RegisterDerivedArtifact(ctx context.Context, access security.AccessContext, artifact conversationstore.DerivedArtifact) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := access.Validate(); err != nil {
		return err
	}
	if !access.IsSystem() || artifact.ID == "" || artifact.TenantID == "" || artifact.CanonicalSourceID == "" || artifact.Kind == "" || artifact.Locator == "" {
		return security.ErrResourceNotAccessible
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	sessionID, err := conversationstore.SessionIDFromCanonicalSource(artifact.CanonicalSourceID)
	if err != nil {
		return err
	}
	session, ok := m.sessions[sessionID]
	if !ok || session.TenantID != artifact.TenantID || session.Status != conversationstore.SessionStatusActive {
		return conversationstore.ErrCanonicalSourceNotWritable
	}
	if _, exists := m.derivedArtifacts[artifact.ID]; exists {
		return fmt.Errorf("derived artifact %q already exists", artifact.ID)
	}
	now := time.Now()
	artifact.Status = "active"
	artifact.CleanupAttempts = 0
	artifact.LastErrorCode = ""
	artifact.NextRetryAt = nil
	artifact.CreatedAt = now
	artifact.UpdatedAt = now
	m.derivedArtifacts[artifact.ID] = artifact
	return nil
}

func (m *MemoryStore) ListDerivedArtifactsForCleanup(ctx context.Context, access security.AccessContext, now time.Time, limit int) ([]conversationstore.DerivedArtifact, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if err := access.Validate(); err != nil {
		return nil, err
	}
	if !access.IsSystem() {
		return nil, security.ErrResourceNotAccessible
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]conversationstore.DerivedArtifact, 0)
	for _, artifact := range m.derivedArtifacts {
		if artifact.Status == "deleted" {
			continue
		}
		if artifact.NextRetryAt != nil && artifact.NextRetryAt.After(now) {
			continue
		}
		sessionID, err := conversationstore.SessionIDFromCanonicalSource(artifact.CanonicalSourceID)
		if err == nil {
			session, ok := m.sessions[sessionID]
			if ok && session.TenantID == artifact.TenantID && session.Status == conversationstore.SessionStatusActive {
				continue
			}
		}
		result = append(result, artifact)
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i].CreatedAt
		if result[i].NextRetryAt != nil {
			left = *result[i].NextRetryAt
		}
		right := result[j].CreatedAt
		if result[j].NextRetryAt != nil {
			right = *result[j].NextRetryAt
		}
		if !left.Equal(right) {
			return left.Before(right)
		}
		if result[i].TenantID != result[j].TenantID {
			return result[i].TenantID < result[j].TenantID
		}
		return result[i].ID < result[j].ID
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *MemoryStore) ListDerivedArtifactsForCleanupPage(ctx context.Context, access security.AccessContext, now time.Time, cursor conversationstore.CleanupCursor, limit int) ([]conversationstore.DerivedArtifact, conversationstore.CleanupCursor, error) {
	if err := checkContext(ctx); err != nil {
		return nil, cursor, err
	}
	if err := access.Validate(); err != nil {
		return nil, cursor, err
	}
	if !access.IsSystem() {
		return nil, cursor, security.ErrResourceNotAccessible
	}
	if limit <= 0 {
		limit = 100
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]conversationstore.DerivedArtifact, 0)
	for _, artifact := range m.derivedArtifacts {
		if artifact.Status == "deleted" || (artifact.NextRetryAt != nil && artifact.NextRetryAt.After(now)) {
			continue
		}
		sessionID, err := conversationstore.SessionIDFromCanonicalSource(artifact.CanonicalSourceID)
		if err == nil {
			if session, ok := m.sessions[sessionID]; ok && session.TenantID == artifact.TenantID && session.Status == conversationstore.SessionStatusActive {
				continue
			}
		}
		at := artifact.CreatedAt
		if artifact.NextRetryAt != nil {
			at = *artifact.NextRetryAt
		}
		if cursor.Valid && !afterCleanupCursor(at, artifact.TenantID, artifact.ID, cursor) {
			continue
		}
		result = append(result, artifact)
	}
	sort.Slice(result, func(i, j int) bool {
		li := result[i].CreatedAt
		if result[i].NextRetryAt != nil {
			li = *result[i].NextRetryAt
		}
		lj := result[j].CreatedAt
		if result[j].NextRetryAt != nil {
			lj = *result[j].NextRetryAt
		}
		return cleanupCandidateBefore(li, result[i].TenantID, result[i].ID, lj, result[j].TenantID, result[j].ID)
	})
	if len(result) > limit {
		result = result[:limit]
	}
	next := conversationstore.CleanupCursor{}
	if len(result) > 0 {
		last := result[len(result)-1]
		at := last.CreatedAt
		if last.NextRetryAt != nil {
			at = *last.NextRetryAt
		}
		next = conversationstore.CleanupCursor{SortTime: at, TenantID: last.TenantID, ID: last.ID, Valid: true}
	}
	return result, next, nil
}

func (m *MemoryStore) MarkDerivedArtifactDeleted(ctx context.Context, access security.AccessContext, artifactID string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := access.Validate(); err != nil {
		return err
	}
	if !access.IsSystem() {
		return security.ErrResourceNotAccessible
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	artifact, ok := m.derivedArtifacts[artifactID]
	if !ok {
		return conversationstore.ErrDerivedArtifactNotFound
	}
	artifact.Status = "deleted"
	artifact.LastErrorCode = ""
	artifact.NextRetryAt = nil
	now := time.Now()
	artifact.UpdatedAt = now
	m.derivedArtifacts[artifactID] = artifact
	m.refreshCleanupJobRetryLocked(artifact.CanonicalSourceID, now)
	return nil
}

func (m *MemoryStore) MarkDerivedArtifactCleanupFailed(ctx context.Context, access security.AccessContext, artifactID, errorCode string, failedAt time.Time) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := access.Validate(); err != nil {
		return err
	}
	if !access.IsSystem() {
		return security.ErrResourceNotAccessible
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	artifact, ok := m.derivedArtifacts[artifactID]
	if !ok {
		return conversationstore.ErrDerivedArtifactNotFound
	}
	if artifact.Status == "deleted" {
		return nil
	}
	artifact.CleanupAttempts++
	artifact.LastErrorCode = errorCode
	nextRetryAt := conversationstore.DerivedArtifactRetryAt(failedAt, artifact.CleanupAttempts)
	artifact.NextRetryAt = &nextRetryAt
	artifact.UpdatedAt = failedAt
	m.derivedArtifacts[artifactID] = artifact
	job, ok := m.cleanupJobs[artifact.CanonicalSourceID]
	if !ok {
		job = conversationstore.CleanupJob{ID: uuid.NewString(), TenantID: artifact.TenantID, CanonicalSourceID: artifact.CanonicalSourceID, CreatedAt: failedAt}
	}
	job.Status = "cleanup_failed"
	job.Attempts++
	job.LastErrorCode = errorCode
	job.UpdatedAt = failedAt
	m.cleanupJobs[artifact.CanonicalSourceID] = job
	m.refreshCleanupJobRetryLocked(artifact.CanonicalSourceID, failedAt)
	return nil
}

// refreshCleanupJobRetryLocked projects per-artifact retry eligibility onto
// the canonical-source job. Artifact retry fields remain the scheduler's
// authority; the job value is an aggregate for progress observation.
func (m *MemoryStore) refreshCleanupJobRetryLocked(canonicalSourceID string, updatedAt time.Time) {
	job, ok := m.cleanupJobs[canonicalSourceID]
	if !ok {
		return
	}
	var nextRetryAt *time.Time
	for _, artifact := range m.derivedArtifacts {
		if artifact.CanonicalSourceID != canonicalSourceID || artifact.Status == "deleted" {
			continue
		}
		if artifact.NextRetryAt == nil {
			nextRetryAt = nil
			break
		}
		if nextRetryAt == nil || artifact.NextRetryAt.Before(*nextRetryAt) {
			next := *artifact.NextRetryAt
			nextRetryAt = &next
		}
	}
	job.NextRetryAt = nextRetryAt
	job.UpdatedAt = updatedAt
	m.cleanupJobs[canonicalSourceID] = job
}

func (m *MemoryStore) AdoptLegacySessions(ctx context.Context, access security.AccessContext, tenantID, ownerID string) (int, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	if err := access.Validate(); err != nil {
		return 0, err
	}
	if !access.IsSystem() {
		return 0, security.ErrResourceNotAccessible
	}
	if tenantID == "" || ownerID == "" {
		return 0, errors.New("legacy session target tenant_id and owner_id are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	updated := 0
	for id, session := range m.sessions {
		if session.TenantID != "" || session.OwnerID != "" {
			continue
		}
		session.TenantID = tenantID
		session.OwnerID = ownerID
		if session.Version == 0 {
			session.Version = 1
		}
		if session.ExpiresAt == nil {
			expiresAt := session.UpdatedAt.Add(m.lifecycle.Retention)
			session.ExpiresAt = &expiresAt
		}
		m.sessions[id] = session
		updated++
	}
	return updated, nil
}

func idempotencyLookupKey(access security.AccessContext, sessionID, key string) string {
	return access.Subject.TenantID + "\x00" + access.Subject.UserID + "\x00" + sessionID + "\x00" + key
}

func (m *MemoryStore) PrepareInteraction(ctx context.Context, access security.AccessContext, sessionID, idempotencyKey, requestHash string, leaseDuration time.Duration) (*conversationstore.PreparedInteraction, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if idempotencyKey == "" || requestHash == "" || leaseDuration <= 0 {
		return nil, errors.New("idempotency key, request hash, and positive lease are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return nil, err
	}
	now := time.Now()
	lookup := idempotencyLookupKey(access, sessionID, idempotencyKey)
	if interactionID := m.idempotency[lookup]; interactionID != "" {
		interaction := m.interactions[interactionID]
		if interaction.RequestHash != requestHash {
			return nil, conversationstore.ErrIdempotencyConflict
		}
		if interaction.Status == conversationstore.InteractionStatusCompleted {
			return &conversationstore.PreparedInteraction{InteractionID: interaction.ID, TurnID: interaction.TurnID, Replayed: true, Result: interaction.Result}, nil
		}
		if interaction.Status == conversationstore.InteractionStatusRunning && interaction.LeaseExpiresAt.After(now) {
			return nil, conversationstore.ErrInteractionInProgress
		}
		history := m.ensureHistoryLocked(sessionID)
		terminalGrowth := 0
		if history.active[interaction.TurnID] != nil {
			terminalGrowth = 1
		}
		if len(m.summaryHistory[sessionID])+terminalGrowth >= conversationstore.MaxHistoryEntriesPerSession {
			return nil, conversationstore.ErrResourceExhausted
		}
		if len(history.history) >= m.limit {
			return nil, conversationstore.ErrResourceExhausted
		}
		additionalEntities := 1 // released active Turns are replaced by terminal metadata
		if m.entityCountLocked()+additionalEntities > m.quota.MaxInMemoryEntities {
			return nil, conversationstore.ErrResourceExhausted
		}
		if history.active[interaction.TurnID] != nil {
			m.recordSummaryHistoryLocked(sessionID, interaction.TurnID, conversationstore.InteractionStatusFailed)
		}
		delete(history.active, interaction.TurnID)
		history.nextID++
		interaction.TurnID = history.nextID
		interaction.Status = conversationstore.InteractionStatusRunning
		interaction.Attempt++
		interaction.UpdatedAt = now
		interaction.LeaseExpiresAt = now.Add(leaseDuration)
		interaction.Result = ""
		interaction.CompletedAt = nil
		history.active[interaction.TurnID] = &turn{id: interaction.TurnID, interactionID: interaction.ID, status: conversationstore.InteractionStatusRunning, createdAt: now}
		m.interactions[interaction.ID] = interaction
		m.activeInteractions[sessionID] = interaction.ID
		return &conversationstore.PreparedInteraction{InteractionID: interaction.ID, TurnID: interaction.TurnID}, nil
	}
	if activeID := m.activeInteractions[sessionID]; activeID != "" {
		active := m.interactions[activeID]
		if active.Status == conversationstore.InteractionStatusRunning && active.LeaseExpiresAt.After(now) {
			return nil, conversationstore.ErrSessionBusy
		}
		if history := m.history[sessionID]; history != nil {
			if history.active[active.TurnID] != nil {
				m.recordSummaryHistoryLocked(sessionID, active.TurnID, conversationstore.InteractionStatusFailed)
			}
			delete(history.active, active.TurnID)
		}
		active.Status = conversationstore.InteractionStatusFailed
		active.UpdatedAt = now
		m.interactions[activeID] = active
		delete(m.activeInteractions, sessionID)
	}
	history := m.ensureHistoryLocked(sessionID)
	if len(history.history) >= m.limit {
		return nil, conversationstore.ErrResourceExhausted
	}
	if len(m.summaryHistory[sessionID]) >= conversationstore.MaxHistoryEntriesPerSession {
		return nil, conversationstore.ErrResourceExhausted
	}
	if m.entityCountLocked()+2 > m.quota.MaxInMemoryEntities {
		return nil, conversationstore.ErrResourceExhausted
	}
	history.nextID++
	interaction := conversationstore.Interaction{
		ID: uuid.NewString(), TenantID: access.Subject.TenantID, OwnerID: access.Subject.UserID,
		SessionID: sessionID, IdempotencyKey: idempotencyKey, RequestHash: requestHash,
		TurnID: history.nextID, Status: conversationstore.InteractionStatusRunning, Attempt: 1,
		LeaseExpiresAt: now.Add(leaseDuration), CreatedAt: now, UpdatedAt: now,
	}
	history.active[interaction.TurnID] = &turn{id: interaction.TurnID, interactionID: interaction.ID, status: conversationstore.InteractionStatusRunning, createdAt: now}
	m.interactions[interaction.ID] = interaction
	m.idempotency[lookup] = interaction.ID
	m.activeInteractions[sessionID] = interaction.ID
	return &conversationstore.PreparedInteraction{InteractionID: interaction.ID, TurnID: interaction.TurnID}, nil
}

func (m *MemoryStore) CompleteInteraction(ctx context.Context, access security.AccessContext, sessionID, interactionID string, turnID uint64, result string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return err
	}
	interaction, ok := m.interactions[interactionID]
	if !ok || interaction.SessionID != sessionID || interaction.TurnID != turnID {
		return conversationstore.ErrTurnNotFound
	}
	history := m.history[sessionID]
	active, ok := history.active[turnID]
	if !ok {
		return conversationstore.ErrTurnNotFound
	}
	resultBytes := int64(len([]byte(result)))
	if m.entityCountLocked()+1 > m.quota.MaxInMemoryEntities {
		return conversationstore.ErrResourceExhausted
	}
	if m.sessionBytesLocked(sessionID)+resultBytes > m.quota.MaxSessionBytes || m.tenantBytesLocked(access.Subject.TenantID)+resultBytes > m.quota.MaxTenantBytes {
		return conversationstore.ErrResourceExhausted
	}
	now := time.Now()
	active.status = conversationstore.InteractionStatusCompleted
	m.recordSummaryHistoryLocked(sessionID, turnID, conversationstore.InteractionStatusCompleted)
	delete(history.active, turnID)
	history.history = append(history.history, active)
	interaction.Status = conversationstore.InteractionStatusCompleted
	interaction.Result = result
	interaction.UpdatedAt = now
	interaction.CompletedAt = &now
	m.interactions[interactionID] = interaction
	if m.activeInteractions[sessionID] == interactionID {
		delete(m.activeInteractions, sessionID)
	}
	return nil
}

func (m *MemoryStore) FailInteraction(ctx context.Context, access security.AccessContext, sessionID, interactionID string, turnID uint64) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return err
	}
	interaction, ok := m.interactions[interactionID]
	if !ok || interaction.SessionID != sessionID || interaction.TurnID != turnID {
		return conversationstore.ErrTurnNotFound
	}
	if history := m.history[sessionID]; history != nil {
		if history.active[turnID] != nil {
			m.recordSummaryHistoryLocked(sessionID, turnID, conversationstore.InteractionStatusFailed)
		}
		delete(history.active, turnID)
	}
	interaction.Status = conversationstore.InteractionStatusFailed
	interaction.UpdatedAt = time.Now()
	m.interactions[interactionID] = interaction
	if m.activeInteractions[sessionID] == interactionID {
		delete(m.activeInteractions, sessionID)
	}
	return nil
}

func (m *MemoryStore) BeginTurn(ctx context.Context, access security.AccessContext, sessionID string) (uint64, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return 0, err
	}
	history := m.ensureHistoryLocked(sessionID)
	if len(m.summaryHistory[sessionID]) >= conversationstore.MaxHistoryEntriesPerSession || m.entityCountLocked()+2 > m.quota.MaxInMemoryEntities {
		return 0, conversationstore.ErrResourceExhausted
	}
	if len(history.active) > 0 {
		return 0, conversationstore.ErrSessionBusy
	}
	if len(history.history)+len(history.active) >= m.limit {
		return 0, fmt.Errorf("%w: current session's context is full, please create a new session", conversationstore.ErrTurnLimitReached)
	}
	history.nextID++
	turnID := history.nextID
	history.active[turnID] = &turn{id: turnID, createdAt: time.Now()}
	return turnID, nil
}

func (m *MemoryStore) AddHistoryToTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64, messages ...*ai.Message) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return err
	}
	history := m.history[sessionID]
	if history == nil || history.active == nil {
		return conversationstore.ErrTurnNotFound
	}
	active, ok := history.active[turnID]
	if !ok {
		return conversationstore.ErrTurnNotFound
	}
	events, bytes, err := messageBatchUsage(messages)
	if err != nil {
		return err
	}
	if len(active.messages())+events > m.quota.MaxEventsPerTurn ||
		m.sessionBytesLocked(sessionID)+bytes > m.quota.MaxSessionBytes ||
		m.tenantBytesLocked(access.Subject.TenantID)+bytes > m.quota.MaxTenantBytes ||
		m.entityCountLocked()+events > m.quota.MaxInMemoryEntities {
		return conversationstore.ErrResourceExhausted
	}
	if err := appendMessages(active, messages...); err != nil {
		if len(active.messages()) == 0 {
			delete(history.active, turnID)
		}
		return err
	}
	return nil
}

func messageBatchUsage(messages []*ai.Message) (int, int64, error) {
	count := 0
	var bytes int64
	for _, message := range messages {
		if message == nil || !supportedRole(message.Role) {
			continue
		}
		payload, err := json.Marshal(message)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to encode message: %w", err)
		}
		count++
		bytes += int64(len(payload))
	}
	return count, bytes, nil
}

func (m *MemoryStore) sessionBytesLocked(sessionID string) int64 {
	total := int64(len(m.summaryChecks[sessionID])) * conversationstore.SummaryInspectionBytes
	if v, ok := m.summaries[sessionID]; ok {
		b, _ := json.Marshal(v)
		total += int64(len(b))
	}
	for _, batch := range m.toolBatches {
		if batch.SessionID == sessionID {
			data, _ := json.Marshal(batch)
			total += int64(len(data))
		}
	}
	history := m.history[sessionID]
	if history == nil {
		return total
	}
	turns := append([]*turn(nil), history.history...)
	for _, active := range history.active {
		turns = append(turns, active)
	}
	for _, current := range turns {
		for _, message := range current.messages() {
			payload, _ := json.Marshal(message)
			total += int64(len(payload))
		}
	}
	for _, interaction := range m.interactions {
		if interaction.SessionID == sessionID {
			total += int64(len([]byte(interaction.Result)))
		}
	}
	return total
}

func (m *MemoryStore) tenantBytesLocked(tenantID string) int64 {
	var total int64
	for id, session := range m.sessions {
		if session.TenantID == tenantID {
			total += m.sessionBytesLocked(id)
		}
	}
	return total
}

func (m *MemoryStore) entityCountLocked() int {
	total := len(m.sessions) + len(m.interactions) + len(m.toolBatches)
	total += len(m.summaries)
	for _, checks := range m.summaryChecks {
		total += len(checks)
	}
	for _, entries := range m.summaryHistory {
		total += len(entries)
	}
	for _, history := range m.history {
		total += len(history.active) + len(history.history)
		for _, current := range history.active {
			total += len(current.messages())
		}
		for _, current := range history.history {
			total += len(current.messages())
		}
	}
	return total
}

func (m *MemoryStore) IsTurnEmpty(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) (bool, error) {
	if err := checkContext(ctx); err != nil {
		return false, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return false, err
	}
	history := m.history[sessionID]
	if history == nil || history.active == nil {
		return false, conversationstore.ErrTurnNotFound
	}
	_, ok := history.active[turnID]
	if !ok {
		return false, conversationstore.ErrTurnNotFound
	}
	// An active Turn exists even when AddHistory received only nil or
	// unsupported messages; this preserves the historical IsEmpty contract.
	return false, nil
}

func (m *MemoryStore) WindowMemoryForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) ([]*ai.Message, error) {
	return m.ContextWindowForTurn(ctx, access, sessionID, turnID, 0)
}

// ContextWindowForTurn returns a stable copy of the most recent completed
// turns followed by the selected active turn. Other active turns are excluded.
func (m *MemoryStore) ContextWindowForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64, completedLimit int) ([]*ai.Message, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if completedLimit < 0 {
		return nil, fmt.Errorf("completed turn limit must not be negative")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return nil, err
	}
	history := m.history[sessionID]
	if history == nil || history.active == nil {
		return nil, conversationstore.ErrTurnNotFound
	}
	active, ok := history.active[turnID]
	if !ok {
		return nil, conversationstore.ErrTurnNotFound
	}
	start := len(history.history) - completedLimit
	if start < 0 {
		start = 0
	}
	turns := make([]*turn, 0, len(history.history)-start+1)
	turns = append(turns, history.history[start:]...)
	turns = append(turns, active)
	return cloneTurnList(turns)
}

func (m *MemoryStore) AllMemory(ctx context.Context, access security.AccessContext, sessionID string) ([]*ai.Message, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return nil, err
	}
	history := m.history[sessionID]
	if history == nil || history.active == nil {
		return nil, nil
	}
	activeIDs := make([]uint64, 0, len(history.active))
	for turnID := range history.active {
		activeIDs = append(activeIDs, turnID)
	}
	sort.Slice(activeIDs, func(i, j int) bool { return activeIDs[i] < activeIDs[j] })
	turns := make([]*turn, 0, len(activeIDs)+len(history.history))
	for _, turnID := range activeIDs {
		turns = append(turns, history.active[turnID])
	}
	turns = append(turns, history.history...)
	return cloneTurnList(turns)
}

func (m *MemoryStore) NextTurnForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return err
	}
	history := m.history[sessionID]
	if history == nil || history.active == nil {
		return conversationstore.ErrTurnNotFound
	}
	completed, ok := history.active[turnID]
	if !ok {
		return conversationstore.ErrTurnNotFound
	}
	if m.entityCountLocked()+1 > m.quota.MaxInMemoryEntities {
		return conversationstore.ErrResourceExhausted
	}
	if len(history.history) >= m.limit {
		return fmt.Errorf("%w: current session's context is full, please create a new session", conversationstore.ErrTurnLimitReached)
	}
	m.recordSummaryHistoryLocked(sessionID, turnID, conversationstore.InteractionStatusCompleted)
	delete(history.active, turnID)
	history.history = append(history.history, completed)
	return nil
}

// AbortTurnForTurn removes an unfinished interaction and its messages.
// Aborted Turns do not count toward the configured conversation limit.
func (m *MemoryStore) AbortTurnForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return err
	}
	history := m.history[sessionID]
	if history == nil || history.active == nil {
		return conversationstore.ErrTurnNotFound
	}
	if _, ok := history.active[turnID]; !ok {
		return conversationstore.ErrTurnNotFound
	}
	m.recordSummaryHistoryLocked(sessionID, turnID, conversationstore.InteractionStatusFailed)
	delete(history.active, turnID)
	return nil
}

// AddHistory is retained for callers of the pre-Store API. Production code
// must use BeginTurn and AddHistoryToTurn so concurrent interactions cannot
// share an active Turn.
func (m *MemoryStore) AddHistory(ctx context.Context, access security.AccessContext, sessionID string, messages ...*ai.Message) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		return err
	}
	history := m.ensureHistoryLocked(sessionID)
	if len(history.active) == 0 {
		if len(history.history) >= m.limit {
			return fmt.Errorf("%w: current session's context is full, please create a new session", conversationstore.ErrTurnLimitReached)
		}
		history.nextID++
		history.active[history.nextID] = &turn{id: history.nextID, createdAt: time.Now()}
	}
	if len(history.active) != 1 {
		return fmt.Errorf("session %q has multiple active turns; use AddHistoryToTurn", sessionID)
	}
	var active *turn
	for _, candidate := range history.active {
		active = candidate
	}
	events, bytes, err := messageBatchUsage(messages)
	if err != nil {
		return err
	}
	if len(active.messages())+events > m.quota.MaxEventsPerTurn || m.sessionBytesLocked(sessionID)+bytes > m.quota.MaxSessionBytes || m.tenantBytesLocked(access.Subject.TenantID)+bytes > m.quota.MaxTenantBytes || m.entityCountLocked()+events > m.quota.MaxInMemoryEntities {
		return conversationstore.ErrResourceExhausted
	}
	return appendMessages(active, messages...)
}

// IsEmpty is retained for compatibility with the pre-Store API.
func (m *MemoryStore) IsEmpty(ctx context.Context, access security.AccessContext, sessionID string) (bool, error) {
	if err := checkContext(ctx); err != nil {
		return false, err
	}
	m.mu.RLock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		m.mu.RUnlock()
		return false, err
	}
	history := m.history[sessionID]
	activeCount := 0
	if history != nil {
		activeCount = len(history.active)
	}
	m.mu.RUnlock()
	return activeCount == 0, nil
}

// WindowMemory is retained for compatibility with the pre-Store API. It is
// only unambiguous when one active Turn exists.
func (m *MemoryStore) WindowMemory(ctx context.Context, access security.AccessContext, sessionID string) ([]*ai.Message, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		m.mu.RUnlock()
		return nil, err
	}
	history := m.history[sessionID]
	var turnID uint64
	if history != nil && len(history.active) == 1 {
		for id := range history.active {
			turnID = id
		}
	}
	m.mu.RUnlock()
	if turnID == 0 {
		return nil, nil
	}
	return m.WindowMemoryForTurn(ctx, access, sessionID, turnID)
}

// NextTurn is retained for compatibility with the pre-Store API. Production
// code must finalize a specific Turn with NextTurnForTurn.
func (m *MemoryStore) NextTurn(ctx context.Context, access security.AccessContext, sessionID string) error {
	m.mu.RLock()
	if err := m.validateSessionLocked(access, sessionID); err != nil {
		m.mu.RUnlock()
		return err
	}
	history := m.history[sessionID]
	var turnID uint64
	if history != nil && len(history.active) == 1 {
		for id := range history.active {
			turnID = id
		}
	}
	m.mu.RUnlock()
	if turnID == 0 {
		return conversationstore.ErrNoActiveTurn
	}
	return m.NextTurnForTurn(ctx, access, sessionID, turnID)
}

func (m *MemoryStore) validateSessionLocked(access security.AccessContext, sessionID string) error {
	if err := access.ValidateBusiness(); err != nil {
		return err
	}
	session, exists := m.sessions[sessionID]
	if !exists || !access.CanAccessOwner(session.TenantID, session.OwnerID) {
		return conversationstore.ErrSessionNotFound
	}
	return conversationstore.ValidateSessionActive(sessionID, session.Status, m.isExpired(session, time.Now()))
}

func (m *MemoryStore) ensureHistoryLocked(sessionID string) *sessionHistory {
	history := m.history[sessionID]
	if history == nil {
		history = &sessionHistory{active: make(map[uint64]*turn)}
		m.history[sessionID] = history
	}
	return history
}

func cloneTurnList(turns []*turn) ([]*ai.Message, error) {
	var result []*ai.Message
	for _, current := range turns {
		if current == nil {
			continue
		}
		for _, message := range current.messages() {
			copy, err := cloneMessage(message)
			if err != nil {
				return nil, fmt.Errorf("failed to copy stored message: %w", err)
			}
			result = append(result, copy)
		}
	}
	return result, nil
}

func cloneSession(session conversationstore.Session) conversationstore.Session {
	if session.ExpiresAt != nil {
		expiresAt := *session.ExpiresAt
		session.ExpiresAt = &expiresAt
	}
	return session
}

func appendMessages(active *turn, messages ...*ai.Message) error {
	ordered := make([]*ai.Message, 0, len(messages))
	for _, message := range messages {
		if message == nil || !supportedRole(message.Role) {
			continue
		}
		copy, err := cloneMessage(message)
		if err != nil {
			return fmt.Errorf("failed to copy message: %w", err)
		}
		ordered = append(ordered, copy)
	}
	// Mutate the Turn only after the complete input batch has been cloned.
	active.orderedMessages = append(active.orderedMessages, ordered...)
	return nil
}

func cloneMessage(message *ai.Message) (*ai.Message, error) {
	if message == nil {
		return nil, errors.New("message is nil")
	}
	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	var copy ai.Message
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	return &copy, nil
}

func supportedRole(role ai.Role) bool {
	return role == ai.RoleSystem || role == ai.RoleUser || role == ai.RoleModel
}

func (m *MemoryStore) isExpired(session conversationstore.Session, now time.Time) bool {
	return conversationstore.IsSessionExpired(session.ExpiresAt, session.UpdatedAt, now, m.lifecycle.Retention)
}

func (m *MemoryStore) RecordUsage(ctx context.Context, event conversationstore.UsageEvent) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	access, err := security.AccessContextFromContext(ctx)
	if err != nil || access.ValidateBusiness() != nil || event.TenantID == "" || event.TenantID != access.Subject.TenantID || (event.UserID != "" && event.UserID != access.Subject.UserID) {
		return security.ErrResourceNotAccessible
	}
	event.UserID = access.Subject.UserID
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	m.mu.Lock()
	if err := checkContext(ctx); err != nil {
		m.mu.Unlock()
		return err
	}
	if m.usageEvents.Len() >= m.usagePolicy.MaxUsageEvents {
		m.mu.Unlock()
		return conversationstore.ErrResourceExhausted
	}
	m.usageEvents.PushBack(event)
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) RecordAudit(ctx context.Context, event conversationstore.AuditEvent) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	access, err := security.AccessContextFromContext(ctx)
	if err != nil || access.ValidateBusiness() != nil || event.TenantID == "" || event.TenantID != access.Subject.TenantID || (event.UserID != "" && event.UserID != access.Subject.UserID) {
		return security.ErrResourceNotAccessible
	}
	event.UserID = access.Subject.UserID
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	m.mu.Lock()
	if err := checkContext(ctx); err != nil {
		m.mu.Unlock()
		return err
	}
	if m.auditEvents.Len() >= m.usagePolicy.MaxAuditEvents {
		m.mu.Unlock()
		return conversationstore.ErrResourceExhausted
	}
	m.auditEvents.PushBack(event)
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) CleanupUsage(ctx context.Context, access security.AccessContext, cutoff time.Time, limit int) (int, error) {
	if err := access.Validate(); err != nil {
		return 0, err
	}
	if !access.IsSystem() {
		return 0, security.ErrResourceNotAccessible
	}
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff = conversationstore.UsageAuditCleanupCutoff(cutoff, time.Now(), m.usagePolicy)
	return cleanupEvents(ctx, &m.usageEvents, &m.usageCleanupCursor, cutoff, cleanupScanLimit(limit, m.usagePolicy.CleanupBatchSize))
}

func (m *MemoryStore) CleanupAudit(ctx context.Context, access security.AccessContext, cutoff time.Time, limit int) (int, error) {
	if err := access.Validate(); err != nil {
		return 0, err
	}
	if !access.IsSystem() {
		return 0, security.ErrResourceNotAccessible
	}
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff = conversationstore.UsageAuditCleanupCutoff(cutoff, time.Now(), m.usagePolicy)
	return cleanupEvents(ctx, &m.auditEvents, &m.auditCleanupCursor, cutoff, cleanupScanLimit(limit, m.usagePolicy.CleanupBatchSize))
}

func cleanupScanLimit(requested, configured int) int {
	if requested <= 0 || requested > configured {
		return configured
	}
	return requested
}

// A linked list permits bounded scanning and O(1) removals without copying the
// unscanned tail. The cursor advances even when no expired records are found.
func cleanupEvents(ctx context.Context, events *list.List, cursor **list.Element, cutoff time.Time, limit int) (int, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	count := min(limit, events.Len())
	next := *cursor
	if next == nil {
		next = events.Front()
	}
	remove := make([]*list.Element, 0, count)
	for i := 0; i < count; i++ {
		if err := checkContext(ctx); err != nil {
			return 0, err
		}
		var createdAt time.Time
		switch event := next.Value.(type) {
		case conversationstore.UsageEvent:
			createdAt = event.CreatedAt
		case conversationstore.AuditEvent:
			createdAt = event.CreatedAt
		}
		if createdAt.Before(cutoff) {
			remove = append(remove, next)
		}
		next = next.Next()
		if next == nil {
			next = events.Front()
		}
	}
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	// The entire list was visited: next may point to an element being removed.
	if count == events.Len() {
		next = nil
	}
	for _, element := range remove {
		events.Remove(element)
	}
	*cursor = next
	return len(remove), nil
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
