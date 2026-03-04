package engine

import (
	"context"
	"dubbo-admin-ai/component/memory"
	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/config"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"gopkg.in/yaml.v3"
)

const (
	GetAllMemoryTool                   string = "memory_all_by_session_id"
	RetrieveBasicConceptFromK8SDocTool string = "retrieve_basic_concept_from_k8s_doc"
)

type MemoryToolInput struct {
	SessionID string `json:"session_id"`
}

func defineMemoryTools(g *genkit.Genkit, history *memory.HistoryMemory) []ai.Tool {
	tools := []ai.Tool{
		getAllMemoryBySession(g, history),
		RetrieveBasicConceptFromK8SDoc(g),
	}
	return tools
}

func getAllMemoryBySession(g *genkit.Genkit, history *memory.HistoryMemory) ai.Tool {
	return genkit.DefineTool(
		g, GetAllMemoryTool, "Get all history memory messages of a session by input `session_id`",
		func(ctx *ai.ToolContext, input MemoryToolInput) (ToolOutput, error) {
			if input.SessionID == "" {
				return ToolOutput{}, fmt.Errorf("sessionID is required")
			}

			if history.IsEmpty(input.SessionID) {
				return ToolOutput{
					ToolName: GetAllMemoryTool,
					Summary:  "No memory available",
				}, nil
			}

			return ToolOutput{
				ToolName: GetAllMemoryTool,
				Result:   history.AllMemory(input.SessionID),
				Summary:  "",
			}, nil
		},
	)
}

type K8SRAGQueryInput struct {
	Querys []string `json:"query"`
}

const (
	K8S_CONCEPTS_NAMESPACE string = "concepts"
)

func RetrieveBasicConceptFromK8SDoc(g *genkit.Genkit) ai.Tool {
	return genkit.DefineTool(
		g, RetrieveBasicConceptFromK8SDocTool, "Retrieve the basic kubernetes concepts from RAG",
		func(ctx *ai.ToolContext, input K8SRAGQueryInput) (ToolOutput, error) {
			if input.Querys == nil {
				return ToolOutput{}, fmt.Errorf("query is required")
			}

			// TODO(memory-tool, 2026-02-24): Get configuration from Runtime instead of hardcoded values
			// Current: backend="dev", indexName="k8s", topK=10
			// Should be: Read from runtime config
			backend := "dev"
			indexName := "k8s"
			topK := 10
			rerankEnabled := false
			rerankTopN := 2
			rerankModel := "rerank-v3.5"
			embeddingModel := "qwen3-embedding"

			// Build configuration using standard Config pattern
			cfg := &compRag.RAGSpec{
				Embedder: &config.Config{
					Type: "genkit",
					Spec: encodeToYAMLNode(&compRag.EmbedderSpec{Model: embeddingModel}),
				},
				Indexer: &config.Config{
					Type: backend,
					Spec: encodeToYAMLNode(&compRag.IndexerSpec{}),
				},
				Retriever: &config.Config{
					Type: backend,
					Spec: encodeToYAMLNode(&compRag.RetrieverSpec{}),
				},
			}
			if rerankEnabled {
				cfg.Reranker = &config.Config{
					Type: "cohere",
					Spec: encodeToYAMLNode(&compRag.RerankerSpec{
						Enabled: true,
						Model:   rerankModel,
					}),
				}
			}

			sys, err := compRag.BuildRAGFromSpec(context.Background(), g, cfg)
			if err != nil {
				return ToolOutput{}, fmt.Errorf("failed to build RAG system: %w", err)
			}

			retrieveOpts := []compRag.RetrieveOption{compRag.WithTopK(topK), compRag.WithTargetIndex(indexName)}
			if rerankEnabled {
				retrieveOpts = append(retrieveOpts, compRag.WithTopN(rerankTopN))
			}
			results, err := sys.Retrieve(context.Background(), K8S_CONCEPTS_NAMESPACE, input.Querys, retrieveOpts...)
			if err != nil {
				return ToolOutput{}, fmt.Errorf("failed to retrieve from RAG: %w", err)
			}

			return ToolOutput{
				ToolName: RetrieveBasicConceptFromK8SDocTool,
				Result:   results,
				Summary:  fmt.Sprintf("Retrieved %d results", len(results)),
			}, nil
		},
	)
}

// encodeToYAMLNode converts a struct to yaml.Node
func encodeToYAMLNode(v any) yaml.Node {
	var node yaml.Node
	node.Encode(v)
	return node
}
