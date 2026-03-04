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

type HistoryMemory struct {
	mu            sync.RWMutex
	windowMemory  map[string]*utils.Window[*Turn]
	historyMemory map[string][]*Turn
}

func NewMemoryContext(key HistoryKey) context.Context {
	return context.WithValue(
		context.Background(),
		key,
		&HistoryMemory{
			windowMemory:  make(map[string]*utils.Window[*Turn]),
			historyMemory: make(map[string][]*Turn),
		},
	)
}

// GetHistoryMemory retrieves HistoryMemory from context
func GetHistoryMemory(ctx context.Context, key HistoryKey) (*HistoryMemory, error) {
	history, ok := ctx.Value(key).(*HistoryMemory)
	if !ok {
		return nil, fmt.Errorf("failed to get history from context")
	}
	return history, nil
}

func (h *HistoryMemory) AddHistory(sessionID string, message ...*ai.Message) {
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

func (h *HistoryMemory) IsEmpty(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check if the window for this session is empty
	return h.windowMemory == nil || h.windowMemory[sessionID] == nil || h.windowMemory[sessionID].IsEmpty()
}

func (h *HistoryMemory) AllMemory(sessionID string) []*ai.Message {
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

func (h *HistoryMemory) WindowMemory(sessionID string) []*ai.Message {
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

func (h *HistoryMemory) Clear(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.windowMemory != nil {
		delete(h.windowMemory, sessionID)
	}
}

func (h *HistoryMemory) ClearAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.windowMemory = make(map[string]*utils.Window[*Turn])
}

func (h *HistoryMemory) GetAllSessions() []string {
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

func (h *HistoryMemory) SystemMemory(sessionID string) []*ai.Message {
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

func (h *HistoryMemory) UserMemory(sessionID string) []*ai.Message {
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

// ModelMemory retrieves the model message history for the specified session
func (h *HistoryMemory) ModelMemory(sessionID string) []*ai.Message {
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

func (h *HistoryMemory) NextTurn(sessionID string) error {
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
