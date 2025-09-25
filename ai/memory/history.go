package memory

import (
	"context"
	"fmt"
	"sync"

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

type History struct {
	mu      sync.RWMutex
	history map[string][]*ai.Message
}

func NewMemoryContext(key HistoryKey) context.Context {
	return context.WithValue(
		context.Background(),
		key,
		&History{history: make(map[string][]*ai.Message)},
	)
}

// GetHistory 从上下文中获取 History
func GetHistory(ctx context.Context, key HistoryKey) (*History, error) {
	history, ok := ctx.Value(key).(*History)
	if !ok {
		return nil, fmt.Errorf("failed to get history from context")
	}
	return history, nil
}

func (h *History) AddHistory(sessionID string, message ...*ai.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.history == nil {
		h.history = make(map[string][]*ai.Message)
	}

	if h.history[sessionID] == nil {
		h.history[sessionID] = make([]*ai.Message, 0)
	}

	for _, msg := range message {
		if msg == nil {
			continue
		}
		h.history[sessionID] = append(h.history[sessionID], msg)
	}
}

func (h *History) IsEmpty(sessionID string) bool {
	if h.history == nil {
		return true
	}

	messages, exists := h.history[sessionID]
	return !exists || len(messages) == 0
}

func (h *History) AllHistory(sessionID string) []*ai.Message {
	if h.history == nil {
		return make([]*ai.Message, 0)
	}

	messages, exists := h.history[sessionID]
	if !exists {
		return make([]*ai.Message, 0)
	}

	// 返回副本以避免并发修改
	result := make([]*ai.Message, len(messages))
	copy(result, messages)
	return result
}

func (h *History) Clear(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.history == nil {
		return
	}

	h.history[sessionID] = make([]*ai.Message, 0)
}

func (h *History) ClearAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.history = make(map[string][]*ai.Message)
}

// GetAllSessions 返回所有活跃的 session ID
func (h *History) GetAllSessions() []string {
	if h.history == nil {
		return make([]string, 0)
	}

	sessions := make([]string, 0, len(h.history))
	for sessionID := range h.history {
		sessions = append(sessions, sessionID)
	}
	return sessions
}

// RemoveSessionHistory 删除指定 session 的所有历史记录
func (h *History) RemoveSessionHistory(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.history == nil {
		return
	}

	delete(h.history, sessionID)
}
