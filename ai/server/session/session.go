package session

import (
	"errors"
	"sync"
	"time"

	"dubbo-admin-ai/manager"

	"github.com/google/uuid"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

// Session 简化的会话实例，不再管理history
type Session struct {
	ID        string       `json:"id"`         // 会话ID
	CreatedAt time.Time    `json:"created_at"` // 创建时间
	UpdatedAt time.Time    `json:"updated_at"` // 最后更新时间
	Status    string       `json:"status"`     // 会话状态: "active", "closed"
	mu        sync.RWMutex `json:"-"`          // 读写锁
}

// UpdateActivity 更新会话活动时间
func (s *Session) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.UpdatedAt = time.Now()
}

// Close 关闭会话
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = "closed"
}

// IsExpired 检查会话是否过期（24小时）
func (s *Session) IsExpired() bool {
	return time.Since(s.UpdatedAt) > 24*time.Hour
}

// ToSessionInfo 转换为API响应格式
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

// Manager 会话管理器
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewManager 创建会话管理器
func NewManager() *Manager {
	m := &Manager{
		sessions: make(map[string]*Session),
	}

	// 启动定期清理过期会话的goroutine
	go m.cleanupExpiredSessions()

	return m
}

// CreateSession 创建新会话
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

	manager.GetLogger().Info("Session created", "session_id", sessionID)
	return session
}

// GetSession 获取会话
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

// DeleteSession 删除会话
func (m *Manager) DeleteSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	session.Close()
	delete(m.sessions, sessionID)

	manager.GetLogger().Info("Session deleted", "session_id", sessionID)
	return nil
}

// ListSessions 列出所有活跃会话
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

// cleanupExpiredSessions 定期清理过期会话
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour) // 每小时清理一次
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
			manager.GetLogger().Info("Cleaned up expired sessions", "count", len(expiredSessions))
		}

		m.mu.Unlock()
	}
}

// generateSessionID 生成会话ID
func generateSessionID() string {
	return "session_" + uuid.New().String()
}
