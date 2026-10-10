package react

import (
	"context"
	"testing"
	"time"

	"dubbo-admin-ai/schema"
	memorystore "dubbo-admin-ai/store/memory"
	"dubbo-admin-ai/testutils"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func TestInteractPersistsAndFinalizesTurn(t *testing.T) {
	store := memorystore.NewMemoryStore(2)
	now := time.Now()
	access := testutils.TestAccess()
	if err := store.Create(context.Background(), access, testutils.TestSession("store-agent-session", now)); err != nil {
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

	ctx := testutils.ContextWithTestAccess(context.Background())
	channels := ra.Interact(ctx, &schema.UserInput{Content: "hello"}, "store-agent-session")
	for {
		select {
		case <-channels.UserRespChan:
		case <-channels.Done():
			goto done
		}
	}

done:
	messages, err := store.AllMemory(context.Background(), access, "store-agent-session")
	if err != nil {
		t.Fatalf("AllMemory() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("persisted message count = %d, want user and model messages", len(messages))
	}
}
