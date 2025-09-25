package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"dubbo-admin-ai/utils"

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

func (t *Turn) Messages() []*ai.Message {
	var msgs []*ai.Message
	msgs = append(msgs, t.SystemMessages...)
	msgs = append(msgs, t.UserMessages...)
	msgs = append(msgs, t.ModelMessages...)
	return msgs
}

type History struct {
	mu            sync.RWMutex
	windowMemory  map[string]*utils.Window[*Turn]
	historyMemory map[string][]*Turn
}

func NewMemoryContext(key HistoryKey) context.Context {
	return context.WithValue(
		context.Background(),
		key,
		&History{
			windowMemory:  make(map[string]*utils.Window[*Turn]),
			historyMemory: make(map[string][]*Turn),
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

	if h.windowMemory == nil {
		h.windowMemory = make(map[string]*utils.Window[*Turn])
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

	if h.windowMemory[sessionID] == nil {
		h.windowMemory[sessionID] = utils.NewWindow[*Turn](TurnLimit)
	}
	if h.windowMemory[sessionID].IsEmpty() {
		h.windowMemory[sessionID].Push(NewTurn())
	}

	turn := h.windowMemory[sessionID].GetCurData()
	turn.SystemMessages = append(turn.SystemMessages, systemMsgs...)
	turn.UserMessages = append(turn.UserMessages, userMsgs...)
	turn.ModelMessages = append(turn.ModelMessages, modelMsgs...)
}

func (h *History) IsEmpty(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 检查该 session 的窗口是否为空
	return h.windowMemory == nil || h.windowMemory[sessionID] == nil || h.windowMemory[sessionID].IsEmpty()
}

func (h *History) AllMemory(sessionID string) []*ai.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.windowMemory == nil || h.windowMemory[sessionID] == nil {
		return nil
	}

	var result []*ai.Message
	for _, turn := range h.windowMemory[sessionID].GetWindow() {
		if turn != nil {
			result = append(result, turn.Messages()...)
		}
	}
	for _, turn := range h.historyMemory[sessionID] {
		if turn != nil {
			result = append(result, turn.Messages()...)
		}
	}
	return result
}

func (h *History) WindowMemory(sessionID string) []*ai.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.windowMemory == nil || h.windowMemory[sessionID] == nil {
		return nil
	}

	var result []*ai.Message
	for _, turn := range h.windowMemory[sessionID].GetWindow() {
		if turn != nil {
			result = append(result, turn.Messages()...)
		}
	}

	return result
}

func (h *History) Clear(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.windowMemory != nil {
		delete(h.windowMemory, sessionID)
	}
}

func (h *History) ClearAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.windowMemory = make(map[string]*utils.Window[*Turn])
}

func (h *History) GetAllSessions() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.windowMemory == nil {
		return make([]string, 0)
	}

	sessions := make([]string, 0, len(h.windowMemory))
	for sessionID := range h.windowMemory {
		sessions = append(sessions, sessionID)
	}
	return sessions
}

func (h *History) SystemMemory(sessionID string) []*ai.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.windowMemory == nil || h.windowMemory[sessionID] == nil {
		return make([]*ai.Message, 0)
	}

	var result []*ai.Message
	window := h.windowMemory[sessionID]
	allTurns := window.GetWindow()
	for _, turn := range allTurns {
		result = append(result, turn.SystemMessages...)
	}
	return result
}

func (h *History) UserMemory(sessionID string) []*ai.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.windowMemory == nil || h.windowMemory[sessionID] == nil {
		return make([]*ai.Message, 0)
	}

	var result []*ai.Message
	window := h.windowMemory[sessionID]
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

	if h.windowMemory == nil || h.windowMemory[sessionID] == nil {
		return make([]*ai.Message, 0)
	}

	var result []*ai.Message
	window := h.windowMemory[sessionID]
	allTurns := window.GetWindow()
	for _, turn := range allTurns {
		result = append(result, turn.ModelMessages...)
	}
	return result
}

func (h *History) NextTurn(sessionID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.windowMemory == nil {
		return errors.New("memory is nil")
	}

	if h.windowMemory[sessionID] == nil {
		return errors.New("session not found")
	}

	if h.windowMemory[sessionID].IsFull() {
		return errors.New("current session's context is full, please create a new session")
	}

	poppedMemory := h.windowMemory[sessionID].Pop()
	h.historyMemory[sessionID] = append(h.historyMemory[sessionID], poppedMemory)

	return nil
}
