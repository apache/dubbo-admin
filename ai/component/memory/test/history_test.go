package memorytest

import (
	"fmt"
	"sync"
	"testing"
	"time"

	compMemory "dubbo-admin-ai/component/memory"
)

func newChatMemory(t *testing.T, max int) *compMemory.ChatMemory {
	t.Helper()
	return compMemory.NewChatMemory(max)
}

func TestChatMemory_BasicFlow(t *testing.T) {
	m := newChatMemory(t, 20)
	sid := "session-1"

	m.Add(sid, compMemory.RoleUser, "hello")
	m.Add(sid, compMemory.RoleAssistant, "hi there!")
	m.Add(sid, compMemory.RoleUser, "how are you?")

	msgs := m.Get(sid)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	if msgs[0].Role != compMemory.RoleUser {
		t.Fatalf("expected first message role 'user', got '%s'", msgs[0].Role)
	}
	if msgs[0].Content != "hello" {
		t.Fatalf("expected first message content 'hello', got '%s'", msgs[0].Content)
	}
	if msgs[1].Role != compMemory.RoleAssistant {
		t.Fatalf("expected second message role 'assistant', got '%s'", msgs[1].Role)
	}
	if msgs[2].Content != "how are you?" {
		t.Fatalf("expected third message content 'how are you?', got '%s'", msgs[2].Content)
	}
}

func TestChatMemory_CircularOverflow(t *testing.T) {
	m := newChatMemory(t, 3) // Small buffer for testing
	sid := "session-overflow"

	// Add 5 messages (buffer only holds 3)
	for i := 0; i < 5; i++ {
		m.Add(sid, compMemory.RoleUser, fmt.Sprintf("msg%d", i))
	}

	msgs := m.Get(sid)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages after overflow, got %d", len(msgs))
	}

	// Oldest messages should be overwritten
	if msgs[0].Content != "msg2" {
		t.Fatalf("expected first message 'msg2', got '%s'", msgs[0].Content)
	}
	if msgs[1].Content != "msg3" {
		t.Fatalf("expected second message 'msg3', got '%s'", msgs[1].Content)
	}
	if msgs[2].Content != "msg4" {
		t.Fatalf("expected third message 'msg4', got '%s'", msgs[2].Content)
	}
}

func TestChatMemory_GetLast(t *testing.T) {
	m := newChatMemory(t, 10)
	sid := "session-last"

	for i := 0; i < 5; i++ {
		m.Add(sid, compMemory.RoleUser, fmt.Sprintf("msg%d", i))
	}

	// Get last 2 messages
	lastMsgs := m.GetLast(sid, 2)
	if len(lastMsgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(lastMsgs))
	}
	if lastMsgs[0].Content != "msg3" {
		t.Fatalf("expected first of last 2 'msg3', got '%s'", lastMsgs[0].Content)
	}
	if lastMsgs[1].Content != "msg4" {
		t.Fatalf("expected second of last 2 'msg4', got '%s'", lastMsgs[1].Content)
	}

	// Get more than available
	lastMsgs = m.GetLast(sid, 10)
	if len(lastMsgs) != 5 {
		t.Fatalf("expected 5 messages when requesting more than available, got %d", len(lastMsgs))
	}
}

func TestChatMemory_Clear(t *testing.T) {
	m := newChatMemory(t, 10)
	sid := "session-clear"

	m.Add(sid, compMemory.RoleUser, "test")
	m.Add(sid, compMemory.RoleAssistant, "response")

	if m.IsEmpty(sid) {
		t.Fatal("expected session to have messages before clear")
	}

	m.Clear(sid)

	if !m.IsEmpty(sid) {
		t.Fatal("expected session to be empty after clear")
	}

	msgs := m.Get(sid)
	if msgs != nil {
		t.Fatalf("expected nil messages after clear, got %d messages", len(msgs))
	}
}

func TestChatMemory_IsEmpty(t *testing.T) {
	m := newChatMemory(t, 10)
	sid := "session-empty"

	if !m.IsEmpty(sid) {
		t.Fatal("expected new session to be empty")
	}

	m.Add(sid, compMemory.RoleUser, "test")

	if m.IsEmpty(sid) {
		t.Fatal("expected session to not be empty after adding message")
	}

	// Non-existent session should be empty
	if !m.IsEmpty("non-existent") {
		t.Fatal("expected non-existent session to be empty")
	}
}

func TestChatMemory_MultipleSessions(t *testing.T) {
	m := newChatMemory(t, 10)

	m.Add("s1", compMemory.RoleUser, "session 1 message")
	m.Add("s2", compMemory.RoleUser, "session 2 message")

	msgs1 := m.Get("s1")
	if len(msgs1) != 1 {
		t.Fatalf("expected 1 message in s1, got %d", len(msgs1))
	}
	if msgs1[0].Content != "session 1 message" {
		t.Fatalf("unexpected content in s1: %s", msgs1[0].Content)
	}

	msgs2 := m.Get("s2")
	if len(msgs2) != 1 {
		t.Fatalf("expected 1 message in s2, got %d", len(msgs2))
	}
	if msgs2[0].Content != "session 2 message" {
		t.Fatalf("unexpected content in s2: %s", msgs2[0].Content)
	}
}

func TestChatMemory_GetAllSessions(t *testing.T) {
	m := newChatMemory(t, 10)

	m.Add("s1", compMemory.RoleUser, "test")
	m.Add("s2", compMemory.RoleUser, "test")
	m.Add("s3", compMemory.RoleUser, "test")

	sessions := m.GetAllSessions()
	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}

	// Check that all session IDs are present
	sessionMap := make(map[string]bool)
	for _, sid := range sessions {
		sessionMap[sid] = true
	}

	if !sessionMap["s1"] || !sessionMap["s2"] || !sessionMap["s3"] {
		t.Fatal("expected all session IDs to be present")
	}
}

func TestChatMemory_ClearAll(t *testing.T) {
	m := newChatMemory(t, 10)

	m.Add("s1", compMemory.RoleUser, "test")
	m.Add("s2", compMemory.RoleUser, "test")

	if m.IsEmpty("s1") || m.IsEmpty("s2") {
		t.Fatal("expected sessions to have messages")
	}

	m.ClearAll()

	if !m.IsEmpty("s1") || !m.IsEmpty("s2") {
		t.Fatal("expected all sessions to be empty after ClearAll")
	}

	if len(m.GetAllSessions()) != 0 {
		t.Fatal("expected no sessions after ClearAll")
	}
}

func TestChatMemory_ConcurrentAccess(t *testing.T) {
	m := newChatMemory(t, 100)
	sid := "session-concurrent"

	var wg sync.WaitGroup
	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				m.Add(sid, compMemory.RoleUser, fmt.Sprintf("msg-%d-%d", i, j))
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = m.Get(sid)
				_ = m.GetLast(sid, 5)
				_ = m.IsEmpty(sid)
			}
		}()
	}

	wg.Wait()

	// Verify we have some messages (exact count may vary due to concurrency)
	msgs := m.Get(sid)
	if len(msgs) == 0 {
		t.Fatal("expected some messages after concurrent writes")
	}
	if len(msgs) > 100 {
		t.Fatalf("expected at most 100 messages, got %d", len(msgs))
	}
}

func TestChatMemory_GetMessagesAsMap(t *testing.T) {
	m := newChatMemory(t, 10)
	sid := "session-map"

	m.Add(sid, compMemory.RoleUser, "hello")
	m.Add(sid, compMemory.RoleAssistant, "hi there!")

	maps := m.GetMessagesAsMap(sid)
	if len(maps) != 2 {
		t.Fatalf("expected 2 message maps, got %d", len(maps))
	}

	if maps[0]["role"] != compMemory.RoleUser {
		t.Fatalf("expected first map role 'user', got '%v'", maps[0]["role"])
	}
	if maps[0]["content"] != "hello" {
		t.Fatalf("expected first map content 'hello', got '%v'", maps[0]["content"])
	}
	if maps[1]["role"] != compMemory.RoleAssistant {
		t.Fatalf("expected second map role 'assistant', got '%v'", maps[1]["role"])
	}

	// Check timestamp exists
	if _, ok := maps[0]["timestamp"]; !ok {
		t.Fatal("expected timestamp in message map")
	}
}

func TestChatMemory_Timestamp(t *testing.T) {
	m := newChatMemory(t, 10)
	sid := "session-timestamp"

	before := time.Now().Unix()
	m.Add(sid, compMemory.RoleUser, "test")
	after := time.Now().Unix()

	msgs := m.Get(sid)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Timestamp < before || msgs[0].Timestamp > after {
		t.Fatalf("timestamp out of expected range: got %d, expected [%d, %d]",
			msgs[0].Timestamp, before, after)
	}
}

func TestMemoryComponent_Validate(t *testing.T) {
	// Test with default config (0 max_messages becomes 20 default)
	comp, err := compMemory.NewMemoryComponent(compMemory.MemoryConfig{})
	if err != nil {
		t.Fatalf("NewMemoryComponent() error: %v", err)
	}
	if err := comp.Validate(); err != nil {
		t.Fatalf("unexpected validation error for default config: %v", err)
	}

	// Test with valid config
	comp, err = compMemory.NewMemoryComponent(compMemory.MemoryConfig{MaxMessages: 20})
	if err != nil {
		t.Fatalf("NewMemoryComponent() error: %v", err)
	}
	if err := comp.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestSession_Size(t *testing.T) {
	s := compMemory.NewSession(5)

	if s.Size() != 0 {
		t.Fatalf("expected size 0 for new session, got %d", s.Size())
	}

	for i := 0; i < 3; i++ {
		s.Add(compMemory.RoleUser, fmt.Sprintf("msg%d", i))
	}

	if s.Size() != 3 {
		t.Fatalf("expected size 3 after adding 3 messages, got %d", s.Size())
	}

	// Add more to trigger overflow
	for i := 3; i < 10; i++ {
		s.Add(compMemory.RoleUser, fmt.Sprintf("msg%d", i))
	}

	// Size should not exceed max
	if s.Size() != 5 {
		t.Fatalf("expected size 5 (max) after overflow, got %d", s.Size())
	}
}
