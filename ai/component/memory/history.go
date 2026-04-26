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
	"fmt"
	"sync"
	"time"

	"github.com/firebase/genkit/go/ai"
)

type HistoryKey string
type SessionKey string

const (
	ChatHistoryKey  HistoryKey = "chat_history"
	SystemMemoryKey HistoryKey = "system_memory"
	CoreMemoryKey   HistoryKey = "core_memory"
	SessionIDKey    SessionKey = "session"
)

const (
	RoleUser      string = "user"
	RoleAssistant string = "assistant"
	RoleSystem    string = "system"
)

// Message represents a single message in the conversation
type Message struct {
	Role      string // "user" | "assistant" | "system"
	Content   string // Plain text content
	Timestamp int64  // Unix timestamp
}

// Session is a circular buffer for storing messages
type Session struct {
	messages []*Message
	max      int     // Maximum number of messages
	head     int     // Write position (circular)
	count    int     // Current number of messages
	mu       sync.RWMutex
}

// NewSession creates a new session with circular buffer
func NewSession(max int) *Session {
	if max <= 0 {
		max = 20 // default
	}
	return &Session{
		messages: make([]*Message, max),
		max:      max,
		head:     0,
		count:    0,
	}
}

// Add adds a message to the session
func (s *Session) Add(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages[s.head] = &Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	// Move head forward, wrap around if needed
	s.head = (s.head + 1) % s.max

	// If buffer is not full, increment count
	if s.count < s.max {
		s.count++
	}
}

// Get returns all messages in chronological order
func (s *Session) Get() []*Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return nil
	}

	result := make([]*Message, s.count)
	for i := 0; i < s.count; i++ {
		// Calculate the actual index accounting for wrap-around
		// If buffer is full (count == max), oldest message is at head
		// If buffer is not full, oldest message is at index 0
		var idx int
		if s.count == s.max {
			idx = (s.head + i) % s.max
		} else {
			idx = i
		}
		result[i] = s.messages[idx]
	}
	return result
}

// GetLast returns the last n messages
func (s *Session) GetLast(n int) []*Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 || n <= 0 {
		return nil
	}

	if n > s.count {
		n = s.count
	}

	result := make([]*Message, n)
	for i := 0; i < n; i++ {
		// Calculate the index from the end
		// The last message is at (head - 1 + max) % max
		pos := (s.head - 1 - i + s.max) % s.max
		result[n-1-i] = s.messages[pos]
	}
	return result
}

// IsEmpty returns true if the session has no messages
func (s *Session) IsEmpty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count == 0
}

// Clear removes all messages from the session
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.head = 0
	s.count = 0
	// Clear references for GC
	for i := range s.messages {
		s.messages[i] = nil
	}
}

// Size returns the current number of messages
func (s *Session) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

// ChatMemory manages multiple sessions
type ChatMemory struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	max      int // Default max messages per session
}

// NewChatMemory creates a new ChatMemory with default max messages
func NewChatMemory(max int) *ChatMemory {
	if max <= 0 {
		max = 20 // default
	}
	return &ChatMemory{
		sessions: make(map[string]*Session),
		max:      max,
	}
}

// getOrCreateSession gets or creates a session for the given sessionID
func (m *ChatMemory) getOrCreateSession(sessionID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sessions[sessionID] == nil {
		m.sessions[sessionID] = NewSession(m.max)
	}
	return m.sessions[sessionID]
}

// Add adds a message to the specified session
func (m *ChatMemory) Add(sessionID, role, content string) {
	session := m.getOrCreateSession(sessionID)
	session.Add(role, content)
}

// Get returns all messages for the specified session in chronological order
func (m *ChatMemory) Get(sessionID string) []*Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.sessions[sessionID] == nil {
		return nil
	}
	return m.sessions[sessionID].Get()
}

// GetLast returns the last n messages for the specified session
func (m *ChatMemory) GetLast(sessionID string, n int) []*Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.sessions[sessionID] == nil {
		return nil
	}
	return m.sessions[sessionID].GetLast(n)
}

// IsEmpty returns true if the session has no messages
func (m *ChatMemory) IsEmpty(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.sessions[sessionID] == nil {
		return true
	}
	return m.sessions[sessionID].IsEmpty()
}

// Clear removes all messages from the specified session
func (m *ChatMemory) Clear(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sessions[sessionID] != nil {
		m.sessions[sessionID].Clear()
	}
}

// ClearAll removes all sessions
func (m *ChatMemory) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]*Session)
}

// GetAllSessions returns all session IDs
func (m *ChatMemory) GetAllSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]string, 0, len(m.sessions))
	for sessionID := range m.sessions {
		sessions = append(sessions, sessionID)
	}
	return sessions
}

// GetMessagesAsMap converts messages to map format for tools
func (m *ChatMemory) GetMessagesAsMap(sessionID string) []map[string]any {
	messages := m.Get(sessionID)
	if messages == nil {
		return nil
	}

	result := make([]map[string]any, len(messages))
	for i, msg := range messages {
		result[i] = map[string]any{
			"role":      msg.Role,
			"content":   msg.Content,
			"timestamp": msg.Timestamp,
		}
	}
	return result
}

// NewMemoryContext creates a context with ChatMemory
func NewMemoryContext(key HistoryKey) context.Context {
	return context.WithValue(
		context.Background(),
		key,
		NewChatMemory(20), // default 20 messages per session
	)
}

// GetHistoryMemory retrieves ChatMemory from context
func GetHistoryMemory(ctx context.Context, key HistoryKey) (*ChatMemory, error) {
	memory, ok := ctx.Value(key).(*ChatMemory)
	if !ok {
		return nil, fmt.Errorf("failed to get history from context")
	}
	return memory, nil
}

// ===== Compatibility methods (deprecated) =====

// Deprecated: Use Add instead
func (m *ChatMemory) AddHistory(sessionID string, message ...*ai.Message) {
	for _, msg := range message {
		if msg == nil {
			continue
		}
		var role string
		switch msg.Role {
		case ai.RoleSystem:
			role = RoleSystem
		case ai.RoleUser:
			role = RoleUser
		case ai.RoleModel:
			role = RoleAssistant
		default:
			continue
		}

		// Extract text content from parts
		var content string
		for _, part := range msg.Content {
			if part.IsText() {
				content += part.Text
			}
		}
		// Call Add which handles locking internally
		m.Add(sessionID, role, content)
	}
}

// Deprecated: Use Get instead
func (m *ChatMemory) AllMemory(sessionID string) []*ai.Message {
	messages := m.Get(sessionID)
	if messages == nil {
		return nil
	}

	result := make([]*ai.Message, len(messages))
	for i, msg := range messages {
		var role ai.Role
		switch msg.Role {
		case RoleUser:
			role = ai.RoleUser
		case RoleAssistant:
			role = ai.RoleModel
		case RoleSystem:
			role = ai.RoleSystem
		default:
			role = ai.RoleUser
		}
		result[i] = ai.NewMessage(role, nil, ai.NewTextPart(msg.Content))
	}
	return result
}

// Deprecated: Use Get instead
func (m *ChatMemory) WindowMemory(sessionID string) []*ai.Message {
	return m.AllMemory(sessionID)
}

// Deprecated: Circular buffer doesn't need NextTurn
func (m *ChatMemory) NextTurn(sessionID string) error {
	// No-op: circular buffer automatically handles overflow
	return nil
}

// Deprecated: Use GetMessagesAsMap instead
func (m *ChatMemory) SystemMemory(sessionID string) []*ai.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.sessions[sessionID] == nil {
		return make([]*ai.Message, 0)
	}

	messages := m.sessions[sessionID].Get()
	var result []*ai.Message
	for _, msg := range messages {
		if msg.Role == RoleSystem {
			result = append(result, ai.NewMessage(ai.RoleSystem, nil, ai.NewTextPart(msg.Content)))
		}
	}
	return result
}

// Deprecated: Use GetMessagesAsMap instead
func (m *ChatMemory) UserMemory(sessionID string) []*ai.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.sessions[sessionID] == nil {
		return make([]*ai.Message, 0)
	}

	messages := m.sessions[sessionID].Get()
	var result []*ai.Message
	for _, msg := range messages {
		if msg.Role == RoleUser {
			result = append(result, ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart(msg.Content)))
		}
	}
	return result
}

// Deprecated: Use GetMessagesAsMap instead
func (m *ChatMemory) ModelMemory(sessionID string) []*ai.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.sessions[sessionID] == nil {
		return make([]*ai.Message, 0)
	}

	messages := m.sessions[sessionID].Get()
	var result []*ai.Message
	for _, msg := range messages {
		if msg.Role == RoleAssistant {
			result = append(result, ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart(msg.Content)))
		}
	}
	return result
}

// HistoryMemory is an alias for ChatMemory for backward compatibility
type HistoryMemory = ChatMemory
