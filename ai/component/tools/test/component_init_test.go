package toolstest

import (
	"context"
	"strings"
	"testing"

	compMemory "dubbo-admin-ai/component/memory"
	compRag "dubbo-admin-ai/component/rag"
	compTools "dubbo-admin-ai/component/tools"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/runtime"

	"github.com/firebase/genkit/go/genkit"
	"gopkg.in/yaml.v3"
)

type stubComponent struct {
	name string
}

func (s *stubComponent) Name() string                { return s.name }
func (s *stubComponent) Validate() error             { return nil }
func (s *stubComponent) Init(*runtime.Runtime) error { return nil }
func (s *stubComponent) Start() error                { return nil }
func (s *stubComponent) Stop() error                 { return nil }

func toYAMLNode(t *testing.T, v any) yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		t.Fatalf("encode yaml node error: %v", err)
	}
	return n
}

func newRuntimeWithMemory(t *testing.T) (*runtime.Runtime, *genkit.Genkit) {
	t.Helper()
	rt := runtime.NewRuntime()
	g := genkit.Init(context.Background())
	rt.SetGenkitRegistry(g)

	memCompRaw, err := compMemory.NewMemoryComponent(compMemory.ChatHistoryKey)
	if err != nil {
		t.Fatalf("NewMemoryComponent() error: %v", err)
	}
	memComp := memCompRaw.(*compMemory.MemoryComponent)
	if err := memComp.Init(rt); err != nil {
		t.Fatalf("MemoryComponent.Init() error: %v", err)
	}
	rt.RegisterComponent(memComp)
	return rt, g
}

func newInitializedRAGComponent(t *testing.T, rt *runtime.Runtime) *compRag.RAGComponent {
	t.Helper()

	spec := compRag.RAGSpec{
		Embedder: &config.Config{
			Type: "genkit",
			Spec: toYAMLNode(t, &compRag.EmbedderSpec{Model: "test-embedding"}),
		},
		Loader: &config.Config{
			Type: "local",
			Spec: toYAMLNode(t, &compRag.LoaderSpec{}),
		},
		Splitter: &config.Config{
			Type: "recursive",
			Spec: toYAMLNode(t, &compRag.SplitterSpec{ChunkSize: 100, OverlapSize: 10}),
		},
		Indexer: &config.Config{
			Type: "dev",
			Spec: toYAMLNode(t, &compRag.IndexerSpec{}),
		},
		Retriever: &config.Config{
			Type: "dev",
			Spec: toYAMLNode(t, &compRag.RetrieverSpec{}),
		},
	}
	specNode := toYAMLNode(t, &spec)

	ragCompRaw, err := compRag.RAGFactory(&specNode)
	if err != nil {
		t.Fatalf("RAGFactory() error: %v", err)
	}
	ragComp := ragCompRaw.(*compRag.RAGComponent)
	if err := ragComp.Init(rt); err != nil {
		t.Fatalf("RAGComponent.Init() error: %v", err)
	}
	return ragComp
}

func newToolsComponent(t *testing.T) *compTools.ToolsComponent {
	t.Helper()
	compRaw, err := compTools.NewToolsComponent(compTools.ToolConfig{
		EnableMockTools:     false,
		EnableInternalTools: true,
		EnableMCPTools:      false,
		MCPTimeout:          30,
		MCPMaxRetries:       0,
	})
	if err != nil {
		t.Fatalf("NewToolsComponent() error: %v", err)
	}
	return compRaw.(*compTools.ToolsComponent)
}

func TestToolsComponent_Init_RAGDependency(t *testing.T) {
	t.Run("missing_rag_component", func(t *testing.T) {
		rt, _ := newRuntimeWithMemory(t)
		comp := newToolsComponent(t)

		err := comp.Init(rt)
		if err == nil || !strings.Contains(err.Error(), "rag component not found") {
			t.Fatalf("expected rag component not found error, got %v", err)
		}
	})

	t.Run("rag_component_type_mismatch", func(t *testing.T) {
		rt, _ := newRuntimeWithMemory(t)
		rt.RegisterComponent(&stubComponent{name: "rag"})
		comp := newToolsComponent(t)

		err := comp.Init(rt)
		if err == nil || !strings.Contains(err.Error(), "invalid rag component type") {
			t.Fatalf("expected invalid rag component type error, got %v", err)
		}
	})

	t.Run("rag_not_initialized", func(t *testing.T) {
		rt, _ := newRuntimeWithMemory(t)
		rt.RegisterComponent(&compRag.RAGComponent{})
		comp := newToolsComponent(t)

		err := comp.Init(rt)
		if err == nil || !strings.Contains(err.Error(), "rag system is not initialized") {
			t.Fatalf("expected rag system nil error, got %v", err)
		}
	})

	t.Run("success_with_injected_rag", func(t *testing.T) {
		rt, g := newRuntimeWithMemory(t)
		rt.RegisterComponent(newInitializedRAGComponent(t, rt))
		comp := newToolsComponent(t)

		if err := comp.Init(rt); err != nil {
			t.Fatalf("Init() error: %v", err)
		}

		if len(comp.GetToolRefs()) == 0 {
			t.Fatalf("expected non-empty tool refs")
		}

		if genkit.LookupTool(g, "retrieve_basic_concept_from_k8s_doc") == nil {
			t.Fatalf("expected retrieve_basic_concept_from_k8s_doc to be registered")
		}
	})
}
