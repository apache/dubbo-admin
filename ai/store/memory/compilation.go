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
	"sort"

	"dubbo-admin-ai/security"
	store "dubbo-admin-ai/store"
)

var _ store.CompilationStore = (*MemoryStore)(nil)

func (m *MemoryStore) RecoveryToolBatches(ctx context.Context, a security.AccessContext, id string, active uint64, limit, maxBytes int) (store.ToolRecovery, error) {
	var result store.ToolRecovery
	if err := checkContext(ctx); err != nil {
		return result, err
	}
	if limit <= 0 || limit > 64 || maxBytes <= 0 {
		return result, store.ErrResourceExhausted
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.validateSessionLocked(a, id); err != nil {
		return result, err
	}
	h := m.history[id]
	if h == nil || h.active[active] == nil {
		return result, store.ErrTurnNotFound
	}
	live := map[uint64]bool{}
	for _, t := range h.history {
		live[t.id] = true
	}
	for tid := range h.active {
		live[tid] = true
	}
	// Keep only a bounded set of references while scanning the quota-bounded ledger.
	var recent []store.ToolBatch
	for _, b := range m.toolBatches {
		if err := checkContext(ctx); err != nil {
			return result, err
		}
		if b.SessionID != id || b.TurnID >= active || live[b.TurnID] {
			continue
		}
		recent = append(recent, b)
		sort.Slice(recent, func(i, j int) bool {
			if recent[i].TurnID != recent[j].TurnID {
				return recent[i].TurnID > recent[j].TurnID
			}
			return recent[i].Sequence > recent[j].Sequence
		})
		if len(recent) > limit {
			result.Omitted = true
			recent = recent[:limit]
		}
	}
	size := 0
	result.TerminalSequences = make(map[uint64]uint64)
	for _, b := range recent {
		data, err := json.Marshal(b)
		if err != nil {
			return result, err
		}
		if size+len(data) > maxBytes {
			result.Omitted = true
			break
		}
		size += len(data)
		result.Batches = append(result.Batches, cloneBatch(b))
		if sequence := m.summaryTurns[id][b.TurnID]; sequence != 0 {
			result.TerminalSequences[b.TurnID] = sequence
		}
	}
	for i, j := 0, len(result.Batches)-1; i < j; i, j = i+1, j-1 {
		result.Batches[i], result.Batches[j] = result.Batches[j], result.Batches[i]
	}
	return result, nil
}

func (m *MemoryStore) PinTemplate(ctx context.Context, a security.AccessContext, id, ref string) (string, error) {
	if err := store.ValidateTemplateRef(ref); err != nil {
		return "", err
	}
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(a, id); err != nil {
		return "", err
	}
	s := m.sessions[id]
	if s.TemplateRef == "" {
		s.TemplateRef = ref
		m.sessions[id] = s
	}
	return s.TemplateRef, nil
}

func cloneBatch(b store.ToolBatch) store.ToolBatch {
	data, _ := json.Marshal(b)
	var result store.ToolBatch
	_ = json.Unmarshal(data, &result)
	return result
}

func (m *MemoryStore) deleteToolBatchesLocked(id string) {
	for key, b := range m.toolBatches {
		if b.SessionID == id {
			delete(m.toolBatches, key)
		}
	}
}

func (m *MemoryStore) batchesLocked(id string, turn uint64) []store.ToolBatch {
	var result []store.ToolBatch
	for _, b := range m.toolBatches {
		if b.SessionID == id && b.TurnID == turn {
			result = append(result, cloneBatch(b))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence })
	return result
}

func (m *MemoryStore) ToolBatches(ctx context.Context, a security.AccessContext, id string, turn uint64, limit int) ([]store.ToolBatch, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.validateSessionLocked(a, id); err != nil {
		return nil, err
	}
	result := m.batchesLocked(id, turn)
	if limit <= 0 || len(result) > limit {
		return nil, store.ErrResourceExhausted
	}
	return result, nil
}

func (m *MemoryStore) CreateToolBatch(ctx context.Context, a security.AccessContext, b store.ToolBatch) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := store.ValidateToolBatch(a, b); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(a, b.SessionID); err != nil {
		return err
	}
	session := m.sessions[b.SessionID]
	if session.TenantID != b.TenantID || session.OwnerID != b.OwnerID {
		return security.ErrResourceNotAccessible
	}
	h := m.history[b.SessionID]
	if h == nil || h.active[b.TurnID] == nil {
		return store.ErrTurnNotFound
	}
	if h.active[b.TurnID].interactionID != b.InteractionID {
		return store.ErrContextConflict
	}
	if _, ok := m.toolBatches[b.ID]; ok {
		return store.ErrContextConflict
	}
	for _, old := range m.batchesLocked(b.SessionID, b.TurnID) {
		if old.Sequence == b.Sequence {
			return store.ErrContextConflict
		}
	}
	data, _ := json.Marshal(b)
	if len(data) > store.MaxToolBatchBytes || len(m.batchesLocked(b.SessionID, b.TurnID)) >= m.quota.MaxEventsPerTurn || m.sessionBytesLocked(b.SessionID)+int64(len(data)) > m.quota.MaxSessionBytes || m.tenantBytesLocked(b.TenantID)+int64(len(data)) > m.quota.MaxTenantBytes || m.entityCountLocked()+1 > m.quota.MaxInMemoryEntities {
		return store.ErrResourceExhausted
	}
	m.toolBatches[b.ID] = cloneBatch(b)
	return nil
}

func (m *MemoryStore) CompleteToolCall(ctx context.Context, a security.AccessContext, id, batch string, version uint64, index int, call store.ToolCall) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(a, id); err != nil {
		return err
	}
	b, ok := m.toolBatches[batch]
	if !ok || b.SessionID != id {
		return store.ErrContextConflict
	}
	// Fences stale workers after a failed/expired attempt has been released.
	if h := m.history[id]; h == nil || h.active[b.TurnID] == nil {
		return store.ErrContextConflict
	}
	old, _ := json.Marshal(b)
	b = cloneBatch(b)
	if err := store.UpdateToolCall(&b, version, index, call); err != nil {
		return err
	}
	data, _ := json.Marshal(b)
	delta := int64(len(data) - len(old))
	if len(data) > store.MaxToolBatchBytes || m.sessionBytesLocked(id)+delta > m.quota.MaxSessionBytes || m.tenantBytesLocked(b.TenantID)+delta > m.quota.MaxTenantBytes {
		return store.ErrResourceExhausted
	}
	m.toolBatches[batch] = b
	return nil
}

func (m *MemoryStore) ContextTurns(ctx context.Context, a security.AccessContext, id string, active uint64, limit, maxBytes int) ([]store.TurnContext, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if limit < 0 || maxBytes <= 0 {
		return nil, store.ErrResourceExhausted
	}
	if limit > m.quota.MaxTurnsPerSession {
		limit = m.quota.MaxTurnsPerSession
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.validateSessionLocked(a, id); err != nil {
		return nil, err
	}
	h := m.history[id]
	if h == nil || h.active[active] == nil {
		return nil, store.ErrTurnNotFound
	}
	start := len(h.history) - limit
	if start < 0 {
		start = 0
	}
	turns := append([]*turn(nil), h.history[start:]...)
	turns = append(turns, h.active[active])
	var result []store.TurnContext
	size := 0
	blocked := false
	for i := len(turns) - 1; i >= 0; i-- {
		t := turns[i]
		session := m.sessions[id]
		item := store.TurnContext{TenantID: session.TenantID, OwnerID: session.OwnerID, ID: t.id, Status: t.status}
		item.Dependencies = append([]store.MemoryDependency(nil), t.dependencies...)
		if position := m.summaryTurns[id][t.id]; position > 0 {
			item.Sequence = m.summaryHistory[id][position-1].Sequence
			if position > 1 {
				item.PreviousSequence = m.summaryHistory[id][position-2].Sequence
			}
		}
		if item.Status == "" {
			item.Status = store.InteractionStatusCompleted
			if t.id == active {
				item.Status = store.InteractionStatusRunning
			}
		}
		if blocked {
			item.ByteLimited = true
			result = append(result, item)
			continue
		}
		item.Messages = t.messages()
		item.Batches = m.batchesLocked(id, t.id)
		data, _ := json.Marshal(item)
		if size+len(data) > maxBytes {
			if t.id == active {
				return nil, store.ErrResourceExhausted
			}
			blocked = true
			item.ByteLimited = true
			item.Messages = nil
			item.Batches = nil
		} else {
			size += len(data)
			var err error
			item.Messages, err = cloneTurnList([]*turn{t})
			if err != nil {
				return nil, err
			}
		}
		result = append(result, item)
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}
