package react

import (
	"context"
	"testing"
	"time"

	"dubbo-admin-ai/schema"
	conversationstore "dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

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
