package memorytest

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	compMemory "dubbo-admin-ai/component/memory"

	"github.com/firebase/genkit/go/ai"
)

func newHistoryMemory(t *testing.T) *compMemory.HistoryMemory {
	t.Helper()
	ctx := compMemory.NewMemoryContext(compMemory.ChatHistoryKey)
	h, err := compMemory.GetHistoryMemory(ctx, compMemory.ChatHistoryKey)
	if err != nil {
		t.Fatalf("GetHistoryMemory() error: %v", err)
	}
	return h
}

func TestHistoryMemory_AddHistory(t *testing.T) {
	h := newHistoryMemory(t)
	sid := "session-1"
	h.AddHistory(sid, ai.NewUserMessage(ai.NewTextPart("hello")))

	got := h.UserMemory(sid)
	if len(got) != 1 {
		t.Fatalf("user memory len = %d, want 1", len(got))
	}
}

func TestHistoryMemory_NextTurn(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, h *compMemory.HistoryMemory)
	}{
		{
			name: "archives_current_turn",
			run: func(t *testing.T, h *compMemory.HistoryMemory) {
				sid := "session-1"
				h.AddHistory(sid, ai.NewUserMessage(ai.NewTextPart("first")))
				if err := h.NextTurn(sid); err != nil {
					t.Fatalf("NextTurn() error: %v", err)
				}
				if len(h.WindowMemory(sid)) != 0 {
					t.Fatalf("window memory should be empty after archive")
				}
				if len(h.AllMemory(sid)) == 0 {
					t.Fatalf("history memory should contain archived turn")
				}
			},
		},
		{
			name: "session_full",
			run: func(t *testing.T, h *compMemory.HistoryMemory) {
				sid := "session-full"
				h.AddHistory(sid, ai.NewUserMessage(ai.NewTextPart("seed")))
				for i := 0; i < compMemory.TurnLimit; i++ {
					if err := h.NextTurn(sid); err != nil {
						if strings.Contains(err.Error(), "context is full") {
							return
						}
						t.Fatalf("unexpected error at step %d: %v", i, err)
					}
					h.AddHistory(sid, ai.NewUserMessage(ai.NewTextPart(fmt.Sprintf("turn-%d", i))))
				}
				if err := h.NextTurn(sid); err == nil || !strings.Contains(err.Error(), "context is full") {
					t.Fatalf("expected context full error, got %v", err)
				}
			},
		},
		{
			name: "empty_window_safety",
			run: func(t *testing.T, h *compMemory.HistoryMemory) {
				sid := "session-empty"
				h.AddHistory(sid, ai.NewUserMessage(ai.NewTextPart("seed")))
				if err := h.NextTurn(sid); err != nil {
					t.Fatalf("first NextTurn() error: %v", err)
				}
				defer func() {
					r := recover()
					if r == nil {
						t.Fatalf("expected panic on empty window")
					}
					if !strings.Contains(fmt.Sprint(r), "window is empty") {
						t.Fatalf("unexpected panic: %v", r)
					}
				}()
				_ = h.NextTurn(sid)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, newHistoryMemory(t))
		})
	}
}

func TestHistoryMemory_Concurrency(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, h *compMemory.HistoryMemory)
	}{
		{
			name: "concurrent_add_history",
			run: func(t *testing.T, h *compMemory.HistoryMemory) {
				sid := "session-concurrent"
				var wg sync.WaitGroup
				for i := 0; i < 100; i++ {
					wg.Add(1)
					go func(i int) {
						defer wg.Done()
						h.AddHistory(sid, ai.NewUserMessage(ai.NewTextPart(fmt.Sprintf("m-%d", i))))
					}(i)
				}
				wg.Wait()
				if len(h.UserMemory(sid)) == 0 {
					t.Fatalf("expected user messages after concurrent writes")
				}
			},
		},
		{
			name: "concurrent_read_write",
			run: func(t *testing.T, h *compMemory.HistoryMemory) {
				sid := "session-rw"
				var wg sync.WaitGroup
				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func(i int) {
						defer wg.Done()
						for j := 0; j < 50; j++ {
							h.AddHistory(sid, ai.NewUserMessage(ai.NewTextPart(fmt.Sprintf("w-%d-%d", i, j))))
						}
					}(i)
				}
				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for j := 0; j < 50; j++ {
							_ = h.AllMemory(sid)
							_ = h.WindowMemory(sid)
						}
					}()
				}
				wg.Wait()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, newHistoryMemory(t))
		})
	}
}

func TestMemoryComponent_Validate(t *testing.T) {
	comp, err := compMemory.NewMemoryComponent(compMemory.ChatHistoryKey, 0)
	if err != nil {
		t.Fatalf("NewMemoryComponent() error: %v", err)
	}
	if err := comp.Validate(); err == nil || !strings.Contains(err.Error(), "max_turns") {
		t.Fatalf("expected max_turns validation error, got %v", err)
	}
}
