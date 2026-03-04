package ragtest

import (
	"context"
	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/runtime"
	"fmt"
	"testing"

	"github.com/firebase/genkit/go/genkit"
	"gopkg.in/yaml.v3"
)

func toYAMLNode(t *testing.T, v any) yaml.Node {
	t.Helper()
	var node yaml.Node
	if err := node.Encode(v); err != nil {
		t.Fatalf("encode yaml node error: %v", err)
	}
	return node
}

func newRAGFactorySpec(t *testing.T, cfg *compRag.RAGSpec) *yaml.Node {
	t.Helper()
	node := toYAMLNode(t, cfg)
	return &node
}

func TestRAGFactory_Init_Success(t *testing.T) {
	rawCfg := &compRag.RAGSpec{
		Embedder: &config.Config{Type: "genkit", Spec: toYAMLNode(t, &compRag.EmbedderSpec{Model: "test-embedding"})},
		Loader:   &config.Config{Type: "local", Spec: toYAMLNode(t, &compRag.LoaderSpec{})},
		Splitter: &config.Config{Type: "recursive", Spec: toYAMLNode(t, &compRag.SplitterSpec{ChunkSize: 100, OverlapSize: 10})},
		Indexer:  &config.Config{Type: "dev", Spec: toYAMLNode(t, compRag.DefaultIndexerSpec())},
		Retriever: &config.Config{
			Type: "dev",
			Spec: toYAMLNode(t, compRag.DefaultRetrieverSpec()),
		},
	}

	compRaw, err := compRag.RAGFactory(newRAGFactorySpec(t, rawCfg))
	if err != nil {
		t.Fatalf("RAGFactory() error: %v", err)
	}

	rt := runtime.NewRuntime()
	rt.SetGenkitRegistry(genkit.Init(context.Background()))

	ragComp, ok := compRaw.(*compRag.RAGComponent)
	if !ok {
		t.Fatalf("unexpected component type: %T", compRaw)
	}

	if err := ragComp.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if err := ragComp.Init(rt); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if ragComp.Rag == nil {
		t.Fatalf("expected rag system initialized")
	}
}

func TestRAGFactory_Init_MarkdownPineconeCohere(t *testing.T) {
	rawCfg := &compRag.RAGSpec{
		Embedder: &config.Config{Type: "genkit", Spec: toYAMLNode(t, &compRag.EmbedderSpec{Model: "test-embedding"})},
		Loader:   &config.Config{Type: "local", Spec: toYAMLNode(t, &compRag.LoaderSpec{})},
		Splitter: &config.Config{Type: "markdown_header", Spec: toYAMLNode(t, &compRag.MarkdownHeaderSplitterSpec{Headers: map[string]string{"#": "h1"}, TrimHeaders: true})},
		Indexer:  &config.Config{Type: "pinecone", Spec: toYAMLNode(t, compRag.DefaultIndexerSpec())},
		Retriever: &config.Config{
			Type: "pinecone",
			Spec: toYAMLNode(t, compRag.DefaultRetrieverSpec()),
		},
		Reranker: &config.Config{
			Type: "cohere",
			Spec: toYAMLNode(t, &compRag.RerankerSpec{Enabled: true, Model: "rerank-english-v3.0"}),
		},
	}

	rt := runtime.NewRuntime()
	g := genkit.Init(context.Background())
	rt.SetGenkitRegistry(g)

	compRaw, err := compRag.RAGFactory(newRAGFactorySpec(t, rawCfg))
	if err != nil {
		t.Fatalf("RAGFactory() error: %v", err)
	}
	ragComp := compRaw.(*compRag.RAGComponent)
	if err := ragComp.Init(rt); err != nil {
		t.Fatalf("RAGComponent.Init() error: %v", err)
	}
	gotTypes := []string{
		fmt.Sprintf("%T", ragComp.Rag.Loader),
		fmt.Sprintf("%T", ragComp.Rag.Splitter),
		fmt.Sprintf("%T", ragComp.Rag.Indexer),
		fmt.Sprintf("%T", ragComp.Rag.Retriever),
		fmt.Sprintf("%T", ragComp.Rag.Reranker),
	}
	wantTypes := []string{
		"*file.FileLoader",
		"*markdown.headerSplitter",
		"*rag.PineconeIndexer",
		"*rag.PineconeRetriever",
		"*rag.cohereReranker",
	}

	for i := range gotTypes {
		if gotTypes[i] != wantTypes[i] {
			t.Fatalf("unexpected component type at idx=%d: got=%s want=%s", i, gotTypes[i], wantTypes[i])
		}
	}
}
