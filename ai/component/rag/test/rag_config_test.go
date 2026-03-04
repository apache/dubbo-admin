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
	_ = node.Encode(v)
	return node
}

func newValidRAGSpec() *compRag.RAGSpec {
	return &compRag.RAGSpec{
		Embedder:  &config.Config{Type: "genkit", Spec: encodeToYAMLNode(&compRag.EmbedderSpec{Model: "dashscope/text-embedding-v4"})},
		Loader:    &config.Config{Type: "local", Spec: encodeToYAMLNode(&compRag.LoaderSpec{})},
		Splitter:  &config.Config{Type: "recursive", Spec: encodeToYAMLNode(&compRag.SplitterSpec{ChunkSize: 100, OverlapSize: 10})},
		Indexer:   &config.Config{Type: "dev", Spec: encodeToYAMLNode(compRag.DefaultIndexerSpec())},
		Retriever: &config.Config{Type: "dev", Spec: encodeToYAMLNode(compRag.DefaultRetrieverSpec())},
	}
}

func TestRAGSpec_Validate(t *testing.T) {
	t.Run("splitter_semantic_validation", func(t *testing.T) {
		cfg := newValidRAGSpec()
		cfg.Splitter.Spec = encodeToYAMLNode(&compRag.SplitterSpec{ChunkSize: 100, OverlapSize: 100})
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "overlap_size") {
			t.Fatalf("expected splitter semantic validation error, got %v", err)
		}
	})

	t.Run("unsupported_loader_type", func(t *testing.T) {
		cfg := newValidRAGSpec()
		cfg.Loader.Type = "remote"
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "unsupported loader type") {
			t.Fatalf("expected unsupported loader type error, got %v", err)
		}
	})

	t.Run("unsupported_indexer_type", func(t *testing.T) {
		cfg := newValidRAGSpec()
		cfg.Indexer.Type = "unknown"
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "unsupported indexer type") {
			t.Fatalf("expected unsupported indexer type error, got %v", err)
		}
	})

	t.Run("unsupported_retriever_type", func(t *testing.T) {
		cfg := newValidRAGSpec()
		cfg.Retriever.Type = "unknown"
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "unsupported retriever type") {
			t.Fatalf("expected unsupported retriever type error, got %v", err)
		}
	})

	t.Run("unsupported_enabled_reranker_type", func(t *testing.T) {
		cfg := newValidRAGSpec()
		cfg.Reranker = &config.Config{
			Type: "unknown",
			Spec: encodeToYAMLNode(&compRag.RerankerSpec{
				Enabled: true,
				Model:   "rerank-english-v3.0",
			}),
		}
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "unsupported reranker type") {
			t.Fatalf("expected unsupported reranker type error, got %v", err)
		}
	})

	t.Run("cohere_reranker_enabled", func(t *testing.T) {
		cfg := newValidRAGSpec()
		cfg.Reranker = &config.Config{
			Type: "cohere",
			Spec: encodeToYAMLNode(&compRag.RerankerSpec{
				Enabled: true,
				Model:   "rerank-english-v3.0",
			}),
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected valid cohere reranker config, got %v", err)
		}
	})

	t.Run("required_subcomponents", func(t *testing.T) {
		cfg := newValidRAGSpec()
		cfg.Loader = nil
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "loader config is required") {
			t.Fatalf("expected missing loader error, got %v", err)
		}
	})

	t.Run("baseline_valid", func(t *testing.T) {
		cfg := newValidRAGSpec()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected valid base rag config, got %v", err)
		}
	})
}
