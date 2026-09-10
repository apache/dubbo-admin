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
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	conversationstore "dubbo-admin-ai/store"

	"github.com/firebase/genkit/go/ai"
)

// DefaultMaxTurns matches MemorySpec's default conversation limit.
const DefaultMaxTurns = 100

const sessionExpiration = 24 * time.Hour

type turn struct {
	id             uint64
	createdAt      time.Time
	userMessages   []*ai.Message
	modelMessages  []*ai.Message
	systemMessages []*ai.Message
}

func (t *turn) messages() []*ai.Message {
	messages := make([]*ai.Message, 0,
		len(t.systemMessages)+len(t.userMessages)+len(t.modelMessages))
	messages = append(messages, t.systemMessages...)
	messages = append(messages, t.userMessages...)
	messages = append(messages, t.modelMessages...)
	return messages
}

type sessionHistory struct {
	active  map[uint64]*turn
	history []*turn
	nextID  uint64
}

// MemoryStore is the in-process implementation of the conversation Store.
type MemoryStore struct {
	mu       sync.RWMutex
	limit    int
	sessions map[string]conversationstore.Session
	history  map[string]*sessionHistory
}

var _ conversationstore.Store = (*MemoryStore)(nil)

// NewMemoryStore creates a MemoryStore. The optional limit exists for runtime
// configuration and tests; the default matches MemorySpec.
func NewMemoryStore(limits ...int) *MemoryStore {
	limit := DefaultMaxTurns
	if len(limits) > 0 && limits[0] > 0 {
		limit = limits[0]
	}
	return &MemoryStore{
		limit:    limit,
		sessions: make(map[string]conversationstore.Session),
		history:  make(map[string]*sessionHistory),
	}
}

func (m *MemoryStore) Create(ctx context.Context, session *conversationstore.Session) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if session == nil {
		return errors.New("session is nil")
	}
	if session.ID == "" {
		return errors.New("session id is required")
	}
	if session.CreatedAt.IsZero() {
		return errors.New("session created_at is required")
	}
	if session.UpdatedAt.IsZero() {
		return errors.New("session updated_at is required")
	}
	if session.Status == "" {
		return errors.New("session status is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[session.ID]; exists {
		return fmt.Errorf("session %q already exists", session.ID)
	}
	m.sessions[session.ID] = *session
	return nil
}

func (m *MemoryStore) Get(ctx context.Context, sessionID string) (*conversationstore.Session, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, conversationstore.ErrSessionNotFound
	}
	if isExpired(session.UpdatedAt, time.Now()) {
		return nil, conversationstore.ErrSessionExpired
	}
	copy := session
	return &copy, nil
}

func (m *MemoryStore) List(ctx context.Context) ([]*conversationstore.Session, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*conversationstore.Session, 0, len(m.sessions))
	now := time.Now()
	for _, session := range m.sessions {
		if session.Status != "active" || isExpired(session.UpdatedAt, now) {
			continue
		}
		copy := session
		result = append(result, &copy)
	}
	return result, nil
}

func (m *MemoryStore) Touch(ctx context.Context, sessionID string, updatedAt time.Time) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	session, exists := m.sessions[sessionID]
	if !exists {
		return conversationstore.ErrSessionNotFound
	}
	if session.Status != "active" {
		return fmt.Errorf("session %q is not active", sessionID)
	}
	if isExpired(session.UpdatedAt, time.Now()) {
		return conversationstore.ErrSessionExpired
	}
	session.UpdatedAt = updatedAt
	m.sessions[sessionID] = session
	return nil
}

func (m *MemoryStore) Delete(ctx context.Context, sessionID string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[sessionID]; !exists {
		return conversationstore.ErrSessionNotFound
	}
	delete(m.sessions, sessionID)
	delete(m.history, sessionID)
	return nil
}

func (m *MemoryStore) DeleteExpired(ctx context.Context, now time.Time) (int, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	deleted := 0
	for sessionID, session := range m.sessions {
		if !isExpired(session.UpdatedAt, now) {
			continue
		}
		delete(m.sessions, sessionID)
		delete(m.history, sessionID)
		deleted++
	}
	return deleted, nil
}

func (m *MemoryStore) BeginTurn(ctx context.Context, sessionID string) (uint64, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(sessionID); err != nil {
		return 0, err
	}
	history := m.ensureHistoryLocked(sessionID)
	if len(history.history)+len(history.active) >= m.limit {
		return 0, fmt.Errorf("%w: current session's context is full, please create a new session", conversationstore.ErrTurnLimitReached)
	}
	history.nextID++
	turnID := history.nextID
	history.active[turnID] = &turn{id: turnID, createdAt: time.Now()}
	return turnID, nil
}

func (m *MemoryStore) AddHistoryToTurn(ctx context.Context, sessionID string, turnID uint64, messages ...*ai.Message) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(sessionID); err != nil {
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
	if err := appendMessages(active, messages...); err != nil {
		if len(active.messages()) == 0 {
			delete(history.active, turnID)
		}
		return err
	}
	return nil
}

func (m *MemoryStore) IsTurnEmpty(ctx context.Context, sessionID string, turnID uint64) (bool, error) {
	if err := checkContext(ctx); err != nil {
		return false, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
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

func (m *MemoryStore) WindowMemoryForTurn(ctx context.Context, sessionID string, turnID uint64) ([]*ai.Message, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	history := m.history[sessionID]
	if history == nil || history.active == nil {
		return nil, conversationstore.ErrTurnNotFound
	}
	active, ok := history.active[turnID]
	if !ok {
		return nil, conversationstore.ErrTurnNotFound
	}
	return cloneTurnList([]*turn{active})
}

func (m *MemoryStore) AllMemory(ctx context.Context, sessionID string) ([]*ai.Message, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
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

func (m *MemoryStore) NextTurnForTurn(ctx context.Context, sessionID string, turnID uint64) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(sessionID); err != nil {
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
	if len(history.history) >= m.limit {
		return fmt.Errorf("%w: current session's context is full, please create a new session", conversationstore.ErrTurnLimitReached)
	}
	delete(history.active, turnID)
	history.history = append(history.history, completed)
	return nil
}

// AbortTurnForTurn removes an unfinished interaction and its messages.
// Aborted Turns do not count toward the configured conversation limit.
func (m *MemoryStore) AbortTurnForTurn(ctx context.Context, sessionID string, turnID uint64) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	history := m.history[sessionID]
	if history == nil || history.active == nil {
		return conversationstore.ErrTurnNotFound
	}
	if _, ok := history.active[turnID]; !ok {
		return conversationstore.ErrTurnNotFound
	}
	delete(history.active, turnID)
	return nil
}

// AddHistory is retained for callers of the pre-Store API. Production code
// must use BeginTurn and AddHistoryToTurn so concurrent interactions cannot
// share an active Turn.
func (m *MemoryStore) AddHistory(ctx context.Context, sessionID string, messages ...*ai.Message) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.validateSessionLocked(sessionID); err != nil {
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
	return appendMessages(active, messages...)
}

// IsEmpty is retained for compatibility with the pre-Store API.
func (m *MemoryStore) IsEmpty(ctx context.Context, sessionID string) (bool, error) {
	if err := checkContext(ctx); err != nil {
		return false, err
	}
	m.mu.RLock()
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
func (m *MemoryStore) WindowMemory(ctx context.Context, sessionID string) ([]*ai.Message, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
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
	return m.WindowMemoryForTurn(ctx, sessionID, turnID)
}

// NextTurn is retained for compatibility with the pre-Store API. Production
// code must finalize a specific Turn with NextTurnForTurn.
func (m *MemoryStore) NextTurn(ctx context.Context, sessionID string) error {
	m.mu.RLock()
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
	return m.NextTurnForTurn(ctx, sessionID, turnID)
}

func (m *MemoryStore) validateSessionLocked(sessionID string) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return conversationstore.ErrSessionNotFound
	}
	if session.Status != "active" {
		return fmt.Errorf("session %q is not active", sessionID)
	}
	if isExpired(session.UpdatedAt, time.Now()) {
		return conversationstore.ErrSessionExpired
	}
	return nil
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

func appendMessages(active *turn, messages ...*ai.Message) error {
	var system, user, model []*ai.Message
	for _, message := range messages {
		if message == nil || !supportedRole(message.Role) {
			continue
		}
		copy, err := cloneMessage(message)
		if err != nil {
			return fmt.Errorf("failed to copy message: %w", err)
		}
		switch copy.Role {
		case ai.RoleSystem:
			system = append(system, copy)
		case ai.RoleUser:
			user = append(user, copy)
		case ai.RoleModel:
			model = append(model, copy)
		}
	}
	// Mutate the Turn only after the complete input batch has been cloned.
	active.systemMessages = append(active.systemMessages, system...)
	active.userMessages = append(active.userMessages, user...)
	active.modelMessages = append(active.modelMessages, model...)
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

func isExpired(updatedAt, now time.Time) bool {
	return now.Sub(updatedAt) > sessionExpiration
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
