package memory

import (
	"context"

	"github.com/firebase/genkit/go/ai"
)

type HistoryKey string

const (
	ChatHistoryKey  HistoryKey = "chat_history"
	SystemMemoryKey HistoryKey = "system_memory"
	CoreMemoryKey   HistoryKey = "core_memory"
)

type History struct {
	history []*ai.Message
}

func NewMemoryContext(key HistoryKey) context.Context {
	return context.WithValue(
		context.Background(),
		key,
		&History{history: make([]*ai.Message, 0)},
	)
}

func (h *History) AddHistory(message ...*ai.Message) {
	for _, msg := range message {
		if msg == nil {
			continue
		}
		h.history = append(h.history, msg)
	}
}

func (h *History) IsEmpty() bool {
	return len(h.history) == 0
}

func (h *History) AllHistory() []*ai.Message {
	return h.history
}

func (h *History) Clear() {
	h.history = make([]*ai.Message, 0)
}
