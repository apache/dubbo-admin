package memory

import (
	"context"
	"dubbo-admin-ai/utils"
	"errors"
	"fmt"
	"sync"

	"github.com/firebase/genkit/go/ai"
)

type HistoryKey string
type SessionKey string

const (
	TurnLimit int = 10

	ChatHistoryKey  HistoryKey = "chat_history"
	SystemMemoryKey HistoryKey = "system_memory"
	CoreMemoryKey   HistoryKey = "core_memory"
	SessionIDKey    SessionKey = "session"
)

type Turn struct {
	UserMessages   []*ai.Message
	ModelMessages  []*ai.Message
	SystemMessages []*ai.Message
}

func NewTurn() *Turn {
	return &Turn{
		UserMessages:   make([]*ai.Message, 0),
		ModelMessages:  make([]*ai.Message, 0),
		SystemMessages: make([]*ai.Message, 0),
	}
}

type History struct {
	mu     sync.RWMutex
	memory map[string]*utils.Window[*Turn]
}

func NewMemoryContext(key HistoryKey) context.Context {
	return context.WithValue(
		context.Background(),
		key,
		&History{
			memory: make(map[string]*utils.Window[*Turn]),
		},
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

	if h.memory == nil {
		h.memory = make(map[string]*utils.Window[*Turn])
	}

	var systemMsgs, userMsgs, modelMsgs []*ai.Message
	for _, msg := range message {
		if msg == nil {
			continue
		}
		switch msg.Role {
		case ai.RoleSystem:
			systemMsgs = append(systemMsgs, msg)
		case ai.RoleUser:
			userMsgs = append(userMsgs, msg)
		case ai.RoleModel:
			modelMsgs = append(modelMsgs, msg)
		}
	}

	if h.memory[sessionID] == nil {
		h.memory[sessionID] = utils.NewWindow[*Turn](TurnLimit)
	}
	if h.memory[sessionID].IsEmpty() {
		h.memory[sessionID].Push(NewTurn())
	}

	turn := h.memory[sessionID].GetCurData()
	turn.SystemMessages = append(turn.SystemMessages, systemMsgs...)
	turn.UserMessages = append(turn.UserMessages, userMsgs...)
	turn.ModelMessages = append(turn.ModelMessages, modelMsgs...)
}

func (h *History) IsEmpty(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 检查该 session 的窗口是否为空
	return h.memory == nil || h.memory[sessionID] == nil || h.memory[sessionID].IsEmpty()
}

func (h *History) AllHistory(sessionID string) []*ai.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.memory == nil || h.memory[sessionID] == nil {
		return nil
	}

	var result []*ai.Message
	for _, turn := range h.memory[sessionID].GetWindow() {
		if turn != nil {
			result = append(result, turn.SystemMessages...)
			result = append(result, turn.UserMessages...)
			result = append(result, turn.ModelMessages...)
		}
	}

	return result
}

func (h *History) Clear(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.memory != nil {
		delete(h.memory, sessionID)
	}
}

func (h *History) ClearAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.memory = make(map[string]*utils.Window[*Turn])
}

func (h *History) GetAllSessions() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.memory == nil {
		return make([]string, 0)
	}

	sessions := make([]string, 0, len(h.memory))
	for sessionID := range h.memory {
		sessions = append(sessions, sessionID)
	}
	return sessions
}

func (h *History) SystemMemory(sessionID string) []*ai.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.memory == nil || h.memory[sessionID] == nil {
		return make([]*ai.Message, 0)
	}

	var result []*ai.Message
	window := h.memory[sessionID]
	allTurns := window.GetWindow()
	for _, turn := range allTurns {
		result = append(result, turn.SystemMessages...)
	}
	return result
}

func (h *History) UserMemory(sessionID string) []*ai.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.memory == nil || h.memory[sessionID] == nil {
		return make([]*ai.Message, 0)
	}

	var result []*ai.Message
	window := h.memory[sessionID]
	allTurns := window.GetWindow()
	for _, turn := range allTurns {
		result = append(result, turn.UserMessages...)
	}
	return result
}

// ModelMemory 获取指定 session 的模型消息历史
func (h *History) ModelMemory(sessionID string) []*ai.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.memory == nil || h.memory[sessionID] == nil {
		return make([]*ai.Message, 0)
	}

	var result []*ai.Message
	window := h.memory[sessionID]
	allTurns := window.GetWindow()
	for _, turn := range allTurns {
		result = append(result, turn.ModelMessages...)
	}
	return result
}

func (h *History) NextTurn(sessionID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.memory == nil {
		return errors.New("memory is nil")
	}

	if h.memory[sessionID] == nil {
		return errors.New("session not found")
	}

	if h.memory[sessionID].IsFull() {
		return errors.New("current session's context is full, please create a new session")
	}

	if !h.memory[sessionID].Pop() {
		return errors.New("failed to pop the context window")
	}

	return nil
}
