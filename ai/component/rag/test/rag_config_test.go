package ragtest

import (
	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/config"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func encodeToYAMLNode(v any) yaml.Node {
	var node yaml.Node
	node.Encode(v)
	return node
}

func TestRAGComponent_Validate(t *testing.T) {
	cfg := &compRag.RAGSpec{
		Embedder:  &config.Config{Type: "genkit", Spec: encodeToYAMLNode(&compRag.EmbedderSpec{Model: "dashscope/qwen3-embedding"})},
		Loader:    &config.Config{Type: "local", Spec: encodeToYAMLNode(&compRag.LoaderSpec{})},
		Splitter:  &config.Config{Type: "recursive", Spec: encodeToYAMLNode(&compRag.SplitterSpec{ChunkSize: 100, OverlapSize: 100})},
		Indexer:   &config.Config{Type: "dev", Spec: encodeToYAMLNode(compRag.DefaultIndexerSpec())},
		Retriever: &config.Config{Type: "dev", Spec: encodeToYAMLNode(compRag.DefaultRetrieverSpec())},
	}

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "overlap_size") {
		t.Fatalf("expected splitter semantic validation error, got %v", err)
	}
}
