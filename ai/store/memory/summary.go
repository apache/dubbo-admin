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
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"dubbo-admin-ai/security"
	store "dubbo-admin-ai/store"
)

var _ store.SummaryStore = (*MemoryStore)(nil)

func (m *MemoryStore) recordSummaryHistoryLocked(id string, turn uint64, status string) {
	if m.summaryHistory == nil {
		m.summaryHistory = map[string][]store.HistoryEntry{}
	}
	entries := m.summaryHistory[id]
	if m.summaryTurns == nil {
		m.summaryTurns = map[string]map[uint64]uint64{}
	}
	if m.summaryTurns[id] == nil {
		m.summaryTurns[id] = map[uint64]uint64{}
	}
	if m.summaryTurns[id][turn] != 0 {
		return
	}
	var deps []store.MemoryDependency
	if h := m.history[id]; h != nil {
		if t := h.active[turn]; t != nil {
			deps = append(deps, t.dependencies...)
		}
	}
	m.summaryHistory[id] = append(entries, store.HistoryEntry{Dependencies: deps, Sequence: uint64(len(entries) + 1), SessionID: id, TurnID: turn, Status: status})
	m.summaryTurns[id][turn] = uint64(len(entries) + 1)
}

func (m *MemoryStore) validateSummarySnapshotLocked(a security.AccessContext, snap store.HistorySnapshot) error {
	if err := m.validateSessionLocked(a, snap.SessionID); err != nil {
		return err
	}
	h := m.history[snap.SessionID]
	if h == nil || h.active[snap.ActiveTurn] == nil {
		return store.ErrContextConflict
	}
	if interaction := m.interactions[h.active[snap.ActiveTurn].interactionID]; interaction.ID != "" && !interaction.LeaseExpiresAt.After(time.Now()) {
		return store.ErrContextConflict
	}
	if snap.Upper > uint64(len(m.summaryHistory[snap.SessionID])) || snap.Target > snap.Upper {
		return store.ErrContextConflict
	}
	return nil
}

func (m *MemoryStore) OpenHistorySnapshot(ctx context.Context, a security.AccessContext, id string, active uint64, keep int) (store.HistorySnapshot, error) {
	var snap store.HistorySnapshot
	if err := checkContext(ctx); err != nil {
		return snap, err
	}
	if keep < 0 {
		return snap, store.ErrContextConflict
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(a, id); err != nil {
		return snap, err
	}
	h := m.history[id]
	if h == nil || h.active[active] == nil {
		return snap, store.ErrTurnNotFound
	}
	entries := m.summaryHistory[id]
	snap = store.HistorySnapshot{SessionID: id, ActiveTurn: active, Count: len(entries)}
	if len(entries) > 0 {
		snap.Upper = entries[len(entries)-1].Sequence
	}
	// Keep N completed Turns, while retaining failed entries in the coverage stream.
	remaining := keep
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Status == store.InteractionStatusCompleted {
			if remaining == 0 {
				snap.Target = entries[i].Sequence
				break
			}
			remaining--
		}
	}
	return snap, m.validateSummarySnapshotLocked(a, snap)
}

func (m *MemoryStore) readSummaryBatchLocked(ctx context.Context, a security.AccessContext, req store.SummaryReadRequest) (store.SummaryBatch, error) {
	if err := checkContext(ctx); err != nil {
		return store.SummaryBatch{}, err
	}
	if err := m.validateSummarySnapshotLocked(a, req.Snapshot); err != nil {
		return store.SummaryBatch{}, err
	}
	entries := m.summaryHistory[req.Snapshot.SessionID]
	index := sort.Search(len(entries), func(i int) bool { return entries[i].Sequence > req.Cursor.After })
	if index == len(entries) || entries[index].Sequence > req.Snapshot.Target {
		return store.SummaryBatch{}, store.ErrSummaryNotFound
	}
	entry := entries[index]
	// Marshal one record at a time; canonical in-memory messages are already resident.
	var records []any
	var classes []string
	if h := m.history[entry.SessionID]; h != nil {
		for _, t := range h.history {
			if t.id == entry.TurnID {
				for _, msg := range t.messages() {
					records = append(records, msg)
					classes = append(classes, string(msg.Role))
				}
				break
			}
		}
	}
	var batches []store.ToolBatch
	messageCount := len(records)
	for _, b := range m.toolBatches {
		if b.SessionID == entry.SessionID && b.TurnID == entry.TurnID {
			batches = append(batches, b)
		}
	}
	sort.Slice(batches, func(i, j int) bool { return batches[i].Sequence < batches[j].Sequence })
	for _, b := range batches {
		records = append(records, b)
		class := "tool_success"
		for _, call := range b.Calls {
			if call.Status != "succeeded" {
				class = "tool_uncertain"
				break
			}
		}
		classes = append(classes, class)
	}
	order := store.SummaryRecordOrder(classes[:messageCount], len(batches))
	return store.BuildSummaryBatch(req, entry, len(records), func(index, offset, limit int) (string, int, error) {
		index = order[index]
		var b []byte
		var err error
		version := uint64(1)
		key := fmt.Sprintf("message:%d:%d", entry.TurnID, index)
		if batch, ok := records[index].(store.ToolBatch); ok {
			version = batch.Version
			key = "batch:" + batch.ID
			b, err = store.SummaryToolEvidence(batch)
		} else {
			b, err = json.Marshal(records[index])
		}
		if err != nil {
			return "", 0, err
		}
		read := func(start, size int) (string, int, error) {
			if err := checkContext(ctx); err != nil {
				return "", 0, err
			}
			if start > len(b) {
				return "", len(b), store.ErrContextConflict
			}
			return string(b[start:min(start+size, len(b))]), len(b), nil
		}
		check := m.summaryChecks[entry.SessionID][key]
		if !check.Valid(version) {
			check, err = store.InspectSummaryRecord(version, read)
			if err != nil {
				return "", 0, err
			}
			if m.summaryChecks[entry.SessionID][key].Rule == "" && (m.sessionBytesLocked(entry.SessionID)+store.SummaryInspectionBytes > m.quota.MaxSessionBytes || m.tenantBytesLocked(a.Subject.TenantID)+store.SummaryInspectionBytes > m.quota.MaxTenantBytes || m.entityCountLocked()+1 > m.quota.MaxInMemoryEntities) {
				return "", 0, store.ErrResourceExhausted
			}
			if m.summaryChecks == nil {
				m.summaryChecks = map[string]map[string]store.SummaryInspection{}
			}
			if m.summaryChecks[entry.SessionID] == nil {
				m.summaryChecks[entry.SessionID] = map[string]store.SummaryInspection{}
			}
			m.summaryChecks[entry.SessionID][key] = check
		}
		return store.ReadInspectedSummaryFragment(check, version, offset, limit, read)
	}, func(index int) string { return classes[order[index]] })
}

func (m *MemoryStore) ReadSummaryBatch(ctx context.Context, a security.AccessContext, req store.SummaryReadRequest) (store.SummaryBatch, error) {
	if err := checkContext(ctx); err != nil {
		return store.SummaryBatch{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readSummaryBatchLocked(ctx, a, req)
}

func cloneSummary(v store.Summary) *store.Summary {
	b, _ := json.Marshal(v)
	var out store.Summary
	_ = json.Unmarshal(b, &out)
	return &out
}

func (m *MemoryStore) GetSummary(ctx context.Context, a security.AccessContext, id string) (*store.Summary, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.validateSessionLocked(a, id); err != nil {
		return nil, err
	}
	v, ok := m.summaries[id]
	if !ok {
		return nil, store.ErrSummaryNotFound
	}
	return cloneSummary(v), nil
}

func (m *MemoryStore) CommitSummary(ctx context.Context, a security.AccessContext, c store.SummaryCommit) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	batch, err := m.readSummaryBatchLocked(ctx, a, c.Read)
	if err != nil {
		return err
	}
	var old *store.Summary
	if v, ok := m.summaries[c.Value.SessionID]; ok {
		old = &v
	}
	if err := store.ValidateSummaryCommit(a, c, old, batch); err != nil {
		return err
	}
	data, _ := json.Marshal(c.Value)
	delta := int64(len(data))
	if old != nil {
		raw, _ := json.Marshal(old)
		delta -= int64(len(raw))
	}
	if m.sessionBytesLocked(c.Value.SessionID)+delta > m.quota.MaxSessionBytes || m.tenantBytesLocked(c.Value.TenantID)+delta > m.quota.MaxTenantBytes || (old == nil && m.entityCountLocked()+1 > m.quota.MaxInMemoryEntities) {
		return store.ErrResourceExhausted
	}
	if m.summaries == nil {
		m.summaries = map[string]store.Summary{}
	}
	m.summaries[c.Value.SessionID] = *cloneSummary(c.Value)
	id := store.SummaryArtifactID(c.Value.SessionID)
	if _, ok := m.derivedArtifacts[id]; !ok {
		now := time.Now()
		m.derivedArtifacts[id] = store.DerivedArtifact{ID: id, TenantID: c.Value.TenantID, CanonicalSourceID: "session:" + c.Value.SessionID, Kind: store.SummaryArtifactKind, Locator: c.Value.SessionID, Status: "active", CreatedAt: now, UpdatedAt: now}
	}
	return nil
}

func (m *MemoryStore) DeleteSummary(ctx context.Context, access security.AccessContext, artifact store.DerivedArtifact) error {
	if err := access.Validate(); err != nil {
		return err
	}
	if !access.IsSystem() {
		return security.ErrResourceNotAccessible
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	id, err := store.SessionIDFromCanonicalSource(artifact.CanonicalSourceID)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if summary, exists := m.summaries[id]; exists && summary.TenantID != artifact.TenantID {
		return security.ErrResourceNotAccessible
	}
	if artifact.Kind != store.SummaryArtifactKind || artifact.Locator != id || (ok && (session.TenantID != artifact.TenantID || session.Status == store.SessionStatusActive)) {
		return security.ErrResourceNotAccessible
	}
	delete(m.summaries, id)
	return nil
}
