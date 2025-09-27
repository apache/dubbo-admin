package tools

import (
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/memory"
	"dubbo-admin-ai/utils"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

const (
	GetAllMemoryTool                   string = "memory_all_by_session_id"
	RetrieveBasicConceptFromK8SDocTool string = "retrieve_basic_concept_from_k8s_doc"
)

type MemoryToolInput struct {
	SessionID string `json:"session_id"`
}

func defineMemoryTools(g *genkit.Genkit, history *memory.History) []ai.Tool {
	tools := []ai.Tool{
		getAllMemoryBySession(g, history),
		RetrieveBasicConceptFromK8SDoc(g, config.EMBEDDING_MODEL.Key(), config.K8S_RAG_INDEX, config.RAG_TOP_K, config.RERANK_TOP_N),
	}
	return tools
}

func getAllMemoryBySession(g *genkit.Genkit, history *memory.History) ai.Tool {
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

func RetrieveBasicConceptFromK8SDoc(g *genkit.Genkit, embedder, indexName string, topK, topN int) ai.Tool {
	return genkit.DefineTool(
		g, RetrieveBasicConceptFromK8SDocTool, "Retrieve the basic kubernetes concepts from RAG",
		func(ctx *ai.ToolContext, input K8SRAGQueryInput) (ToolOutput, error) {
			if input.Querys == nil {
				return ToolOutput{}, fmt.Errorf("query is required")
			}

			results, err := utils.RetrieveFromPinecone(g, embedder, indexName, K8S_CONCEPTS_NAMESPACE, input.Querys, topK, config.RERANK_ENABLE, topN)
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
