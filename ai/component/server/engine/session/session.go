package session

import (
	"errors"
	"sync"
	"time"

	rt "dubbo-admin-ai/runtime"

	"github.com/google/uuid"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// Session is a simplified session instance, does not manage history
type Session struct {
	ID        string       `json:"id"`         // Session ID
	CreatedAt time.Time    `json:"created_at"` // Creation time
	UpdatedAt time.Time    `json:"updated_at"` // Last update time
	Status    string       `json:"status"`     // Session status: "active", "closed"
	mu        sync.RWMutex `json:"-"`          // Read-write lock
}

// UpdateActivity updates session activity time
func (s *Session) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.UpdatedAt = time.Now()
}

// Close closes the session
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = "closed"
}

// IsExpired checks if session is expired (24 hours)
func (s *Session) IsExpired() bool {
	return time.Since(s.UpdatedAt) > 24*time.Hour
}

// ToSessionInfo converts to API response format
func (s *Session) ToSessionInfo() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]any{
		"session_id": s.ID,
		"created_at": s.CreatedAt,
		"updated_at": s.UpdatedAt,
		"status":     s.Status,
	}
}

// Manager manages sessions
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewManager creates a session manager
func NewManager() *Manager {
	m := &Manager{
		sessions: make(map[string]*Session),
	}

	// Start goroutine to periodically cleanup expired sessions
	go m.cleanupExpiredSessions()

	return m
}

func (m *Manager) CreateMockSession() *Session {
	session := &Session{
		ID:        "session_test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "active",
	}
	m.sessions[session.ID] = session
	rt.GetLogger().Info("Session created", "session_id", session.ID)
	return session
}

// CreateSession creates a new session
func (m *Manager) CreateSession() *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := generateSessionID()
	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "active",
	}

	m.sessions[sessionID] = session

	rt.GetLogger().Info("Session created", "session_id", sessionID)
	return session
}

// GetSession retrieves a session
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	if session.IsExpired() {
		return nil, ErrSessionExpired
	}

	return session, nil
}

// DeleteSession deletes a session
func (m *Manager) DeleteSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	session.Close()
	delete(m.sessions, sessionID)

	rt.GetLogger().Info("Session deleted", "session_id", sessionID)
	return nil
}

// ListSessions lists all active sessions
func (m *Manager) ListSessions() []map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []map[string]any
	for _, session := range m.sessions {
		if session.Status == "active" && !session.IsExpired() {
			sessions = append(sessions, session.ToSessionInfo())
		}
	}

	return sessions
}

// cleanupExpiredSessions periodically cleans up expired sessions
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		var expiredSessions []string

		for sessionID, session := range m.sessions {
			if session.IsExpired() {
				expiredSessions = append(expiredSessions, sessionID)
			}
		}

		for _, sessionID := range expiredSessions {
			m.sessions[sessionID].Close()
			delete(m.sessions, sessionID)
		}

		if len(expiredSessions) > 0 {
			rt.GetLogger().Info("Cleaned up expired sessions", "count", len(expiredSessions))
		}

		m.mu.Unlock()
	}
}

// generateSessionID generates a session ID
func generateSessionID() string {
	return "session_" + uuid.New().String()
}
