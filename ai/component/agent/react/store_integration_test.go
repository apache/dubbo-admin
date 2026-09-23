package react

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/schema"
	conversationstore "dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type trackingContextStore struct {
	conversationstore.Store
	calls atomic.Int32
	fail  error
}

type failingMessageStore struct {
	conversationstore.Store
	addCalls int
	failOn   int
}

func (s *failingMessageStore) AddHistoryToTurn(ctx context.Context, sessionID string, turnID uint64, messages ...*ai.Message) error {
	s.addCalls++
	if s.addCalls == s.failOn {
		return errors.New("injected history write failure")
	}
	return s.Store.AddHistoryToTurn(ctx, sessionID, turnID, messages...)
}

func (s *trackingContextStore) ContextWindowForTurn(ctx context.Context, sessionID string, turnID uint64, completedLimit int) ([]*ai.Message, error) {
	s.calls.Add(1)
	if s.fail != nil {
		return nil, s.fail
	}
	return s.Store.ContextWindowForTurn(ctx, sessionID, turnID, completedLimit)
}

func TestInteractPersistsAndFinalizesTurn(t *testing.T) {
	store := memorystore.NewMemoryStore(2)
	now := time.Now()
	if err := store.Create(context.Background(), &conversationstore.Session{
		ID: "store-agent-session", CreatedAt: now, UpdatedAt: now, Status: "active",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	script := &scriptPrompt{resps: []*ai.ModelResponse{textResp("persisted answer")}}
	ra := &ReActAgent{
		registry:      genkit.Init(context.Background()),
		messageStore:  store,
		actPrompt:     script,
		answerPrompt:  script,
		maxIterations: 1,
		bufferSize:    8,
	}

	channels := ra.Interact(context.Background(), &schema.UserInput{Content: "hello"}, "store-agent-session")
	for {
		select {
		case <-channels.UserRespChan:
		case <-channels.Done():
			goto done
		}
	}

done:
	messages, err := store.AllMemory(context.Background(), "store-agent-session")
	if err != nil {
		t.Fatalf("AllMemory() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("persisted message count = %d, want user and model messages", len(messages))
	}
}

func TestNewInteractionBuildsStableContextSnapshot(t *testing.T) {
	ctx := context.Background()
	store := memorystore.NewMemoryStore(10)
	createAgentSession(t, store, "snapshot-session")

	completed, err := store.BeginTurn(ctx, "snapshot-session")
	if err != nil {
		t.Fatalf("BeginTurn(completed) error = %v", err)
	}
	if err := store.AddHistoryToTurn(ctx, "snapshot-session", completed,
		ai.NewUserTextMessage("first question"), ai.NewModelTextMessage("first answer")); err != nil {
		t.Fatalf("AddHistoryToTurn(completed) error = %v", err)
	}
	if err := store.NextTurnForTurn(ctx, "snapshot-session", completed); err != nil {
		t.Fatalf("NextTurnForTurn(completed) error = %v", err)
	}

	ra := &ReActAgent{messageStore: store, contextWindowTurns: 5}
	_, state, err := ra.newInteraction(ctx, &schema.UserInput{Content: "follow-up"}, "snapshot-session")
	if err != nil {
		t.Fatalf("newInteraction() error = %v", err)
	}
	defer state.cancelPersistence()
	defer func() { _ = store.AbortTurnForTurn(context.Background(), "snapshot-session", state.turnID) }()

	if got, want := messageTextList(state.messages), []string{"first question", "first answer", "follow-up"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("context snapshot = %v, want %v", got, want)
	}

	other, err := store.BeginTurn(ctx, "snapshot-session")
	if err != nil {
		t.Fatalf("BeginTurn(other) error = %v", err)
	}
	if err := store.AddHistoryToTurn(ctx, "snapshot-session", other, ai.NewUserTextMessage("later turn")); err != nil {
		t.Fatalf("AddHistoryToTurn(other) error = %v", err)
	}
	if err := store.NextTurnForTurn(ctx, "snapshot-session", other); err != nil {
		t.Fatalf("NextTurnForTurn(other) error = %v", err)
	}
	if got := messageTextList(state.messages); strings.Contains(strings.Join(got, " "), "later turn") {
		t.Fatalf("interaction snapshot changed after another turn completed: %v", got)
	}
}

func TestRunAppendsToolOutputToContextSnapshotWithoutReloading(t *testing.T) {
	ctx := context.Background()
	baseStore := memorystore.NewMemoryStore(10)
	createAgentSession(t, baseStore, "tool-snapshot-session")
	store := &trackingContextStore{Store: baseStore}

	g := genkit.Init(ctx)
	genkit.DefineTool(g, "snapshot_tool", "returns snapshot evidence", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
		return map[string]any{"tool_name": "snapshot_tool", "summary": "evidence"}, nil
	})
	script := &scriptPrompt{resps: []*ai.ModelResponse{
		toolReqResp("snapshot_tool", map[string]any{}),
		textResp("answer"),
	}}
	ra := &ReActAgent{
		registry:           g,
		messageStore:       store,
		actPrompt:          script,
		answerPrompt:       script,
		toolTimeouts:       newToolTimeoutResolver(defaultToolTimeoutSeconds, nil),
		maxIterations:      2,
		contextWindowTurns: 5,
	}
	interactionCtx, state, err := ra.newInteraction(ctx, &schema.UserInput{Content: "use a tool"}, "tool-snapshot-session")
	if err != nil {
		t.Fatalf("newInteraction() error = %v", err)
	}
	defer state.cancelPersistence()
	defer func() { _ = baseStore.AbortTurnForTurn(context.Background(), "tool-snapshot-session", state.turnID) }()

	if _, err := ra.run(interactionCtx, nil, state); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if store.calls.Load() != 1 {
		t.Fatalf("context window loaded %d times, want once per interaction", store.calls.Load())
	}
	if len(script.messages) != 2 {
		t.Fatalf("captured %d model message sets, want 2", len(script.messages))
	}
	if got := messageTextList(script.messages[0]); fmt.Sprint(got) != fmt.Sprint([]string{"use a tool"}) {
		t.Fatalf("first model call messages = %v, want initial snapshot", got)
	}
	if got := messageTextList(script.messages[1]); len(got) != 2 || got[0] != "use a tool" || !strings.Contains(got[1], "snapshot_tool") {
		t.Fatalf("second model call messages = %v, want snapshot plus tool output", got)
	}
	if len(state.messages) != 2 || !strings.Contains(state.messages[1].Text(), "snapshot_tool") {
		t.Fatalf("tool output was not appended to context snapshot: %#v", state.messages)
	}
}

func TestNewInteractionAbortsTurnWhenContextLoadFails(t *testing.T) {
	ctx := context.Background()
	baseStore := memorystore.NewMemoryStore(10)
	createAgentSession(t, baseStore, "failed-context-session")
	store := &trackingContextStore{Store: baseStore, fail: errors.New("context unavailable")}
	ra := &ReActAgent{messageStore: store, contextWindowTurns: 5}

	if _, _, err := ra.newInteraction(ctx, &schema.UserInput{Content: "hello"}, "failed-context-session"); err == nil || !strings.Contains(err.Error(), "failed to load conversation context") {
		t.Fatalf("newInteraction() error = %v, want context load failure", err)
	}
	messages, err := baseStore.AllMemory(ctx, "failed-context-session")
	if err != nil {
		t.Fatalf("AllMemory() error = %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("failed context load left turn messages behind: %#v", messages)
	}
}

func TestInteractAbortsTurnWhenModelCallFails(t *testing.T) {
	ctx := context.Background()
	store := memorystore.NewMemoryStore(10)
	createAgentSession(t, store, "model-failure-session")
	script := &scriptPrompt{errs: []error{errors.New("model unavailable")}}
	ra := &ReActAgent{
		registry:           genkit.Init(ctx),
		messageStore:       store,
		actPrompt:          script,
		answerPrompt:       script,
		maxIterations:      1,
		contextWindowTurns: 5,
	}
	consumeInteraction(t, ra.Interact(ctx, &schema.UserInput{Content: "hello"}, "model-failure-session"))
	assertNoStoredMessages(t, store, "model-failure-session")
}

func TestInteractAbortsTurnWhenToolHistoryWriteFails(t *testing.T) {
	ctx := context.Background()
	baseStore := memorystore.NewMemoryStore(10)
	createAgentSession(t, baseStore, "tool-write-failure-session")
	store := &failingMessageStore{Store: baseStore, failOn: 2}
	g := genkit.Init(ctx)
	genkit.DefineTool(g, "failing_history_tool", "tool used to test persistence failure", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
		return map[string]any{"summary": "tool result"}, nil
	})
	script := &scriptPrompt{resps: []*ai.ModelResponse{
		toolReqResp("failing_history_tool", map[string]any{}),
	}}
	ra := &ReActAgent{
		registry:           g,
		messageStore:       store,
		actPrompt:          script,
		answerPrompt:       script,
		maxIterations:      2,
		contextWindowTurns: 5,
		toolTimeouts:       newToolTimeoutResolver(defaultToolTimeoutSeconds, nil),
	}
	consumeInteraction(t, ra.Interact(ctx, &schema.UserInput{Content: "hello"}, "tool-write-failure-session"))
	assertNoStoredMessages(t, baseStore, "tool-write-failure-session")
}

func createAgentSession(t *testing.T, store conversationstore.Store, sessionID string) {
	t.Helper()
	now := time.Now()
	if err := store.Create(context.Background(), &conversationstore.Session{
		ID: sessionID, CreatedAt: now, UpdatedAt: now, Status: "active",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func messageTextList(messages []*ai.Message) []string {
	result := make([]string, 0, len(messages))
	for _, message := range messages {
		result = append(result, message.Text())
	}
	return result
}

func consumeInteraction(t *testing.T, channels *agent.Channels) {
	t.Helper()
	for {
		select {
		case err := <-channels.ErrorChan:
			if err != nil {
				continue
			}
		case <-channels.UserRespChan:
		case <-channels.Done():
			return
		}
	}
}

func assertNoStoredMessages(t *testing.T, store conversationstore.Store, sessionID string) {
	t.Helper()
	messages, err := store.AllMemory(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("AllMemory() error = %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("failed interaction left messages behind: %#v", messages)
	}
}
