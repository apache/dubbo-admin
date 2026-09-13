package toolstest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	compMemory "dubbo-admin-ai/component/memory"
	compRag "dubbo-admin-ai/component/rag"
	toolEngine "dubbo-admin-ai/component/tools/engine"
	"dubbo-admin-ai/runtime"
	conversationstore "dubbo-admin-ai/store"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func TestNewInternalToolManager_ValidateArgs(t *testing.T) {
	tests := []struct {
		name    string
		rt      *runtime.Runtime
		errLike string
	}{
		{name: "nil_runtime", rt: nil, errLike: "runtime is nil"},
		{name: "nil_genkit", rt: runtime.NewRuntime(), errLike: "genkit registry is nil"},
		{name: "missing_memory", rt: newRuntimeWithRegistryForEngine(t), errLike: "memory component not found"},
		{name: "missing_rag", rt: newRuntimeWithMemoryForEngine(t), errLike: "rag component not found"},
		{name: "rag_not_initialized", rt: newRuntimeWithMemoryAndEmptyRAGForEngine(t), errLike: "rag system is not initialized"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := toolEngine.NewInternalToolManager(tt.rt)
			if err == nil || !strings.Contains(err.Error(), tt.errLike) {
				t.Fatalf("expected %q, got %v", tt.errLike, err)
			}
		})
	}
}

func newRuntimeWithRegistryForEngine(t *testing.T) *runtime.Runtime {
	t.Helper()
	rt := runtime.NewRuntime()
	rt.SetGenkitRegistry(genkit.Init(context.Background()))
	return rt
}

func newRuntimeWithMemoryForEngine(t *testing.T) *runtime.Runtime {
	t.Helper()
	rt := newRuntimeWithRegistryForEngine(t)
	memCompRaw, err := compMemory.NewMemoryComponent(compMemory.ChatHistoryKey)
	if err != nil {
		t.Fatalf("NewMemoryComponent() error: %v", err)
	}
	memComp := memCompRaw.(*compMemory.MemoryComponent)
	if err := memComp.Init(rt); err != nil {
		t.Fatalf("MemoryComponent.Init() error: %v", err)
	}
	rt.RegisterComponent(memComp)
	return rt
}

func newRuntimeWithMemoryAndEmptyRAGForEngine(t *testing.T) *runtime.Runtime {
	t.Helper()
	rt := newRuntimeWithMemoryForEngine(t)
	rt.RegisterComponent(&compRag.RAGComponent{})
	return rt
}

func TestNewInternalToolManager_Success(t *testing.T) {
	rt := newRuntimeWithMemoryForEngine(t)
	rt.RegisterComponent(&compRag.RAGComponent{
		Rag: &compRag.RAG{},
	})

	mgr, err := toolEngine.NewInternalToolManager(rt)
	if err != nil {
		t.Fatalf("NewInternalToolManager() error: %v", err)
	}
	if len(mgr.ToolRefs()) == 0 {
		t.Fatalf("expected non-empty tool refs")
	}
}

func TestMemoryToolReadsInjectedStore(t *testing.T) {
	rt := newRuntimeWithMemoryForEngine(t)
	rt.RegisterComponent(&compRag.RAGComponent{Rag: &compRag.RAG{}})

	memoryComponentRaw, err := rt.GetComponent("memory")
	if err != nil {
		t.Fatalf("GetComponent(memory) error = %v", err)
	}
	memoryComponent := memoryComponentRaw.(*compMemory.MemoryComponent)
	store, err := memoryComponent.GetStore()
	if err != nil {
		t.Fatalf("GetStore() error = %v", err)
	}
	now := time.Now()
	session := &conversationstore.Session{ID: "memory-tool-session", CreatedAt: now, UpdatedAt: now, Status: "active"}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	turnID, err := store.BeginTurn(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("BeginTurn() error = %v", err)
	}
	if err := store.AddHistoryToTurn(context.Background(), session.ID, turnID, ai.NewUserTextMessage("from shared store")); err != nil {
		t.Fatalf("AddHistory() error = %v", err)
	}

	if _, err := toolEngine.NewInternalToolManager(rt); err != nil {
		t.Fatalf("NewInternalToolManager() error = %v", err)
	}
	output, err := toolEngine.Call(context.Background(), rt.GetGenkitRegistry(), toolEngine.GetAllMemoryTool, map[string]any{
		"session_id": session.ID,
	})
	if err != nil {
		t.Fatalf("Call(memory tool) error = %v", err)
	}
	result, ok := output.Result.([]interface{})
	if !ok || len(result) != 1 {
		t.Fatalf("memory tool result = %#v, want shared-store message", output.Result)
	}
	message, ok := result[0].(map[string]interface{})
	if !ok {
		t.Fatalf("memory tool result = %#v, want shared-store message", output.Result)
	}
	content, ok := message["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("memory tool content = %#v, want one content item", message["content"])
	}
	part, ok := content[0].(map[string]interface{})
	if !ok || fmt.Sprint(part["text"]) != "from shared store" {
		t.Fatalf("memory tool content = %#v, want shared-store message", content[0])
	}
}
