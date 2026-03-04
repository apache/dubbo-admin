package engine

import (
	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/runtime"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

const (
	GetAllMemoryTool                   string = "memory_all_by_session_id"
	RetrieveBasicConceptFromK8SDocTool string = "retrieve_basic_concept_from_k8s_doc"
	DefaultK8SDocTargetIndex           string = "kube-docs"
	DefaultK8SDocRetrieveTopK          int    = 5
	DefaultK8SDocNamespace             string = "concepts"
	DefaultK8SDocRerankTopN            int    = 2
)

type MemoryToolInput struct {
	SessionID string `json:"session_id"`
}

func getAllMemoryBySession(rt *runtime.Runtime) ai.Tool {
	g := rt.GetGenkitRegistry()
	return genkit.DefineTool(
		g, GetAllMemoryTool, "Get all history memory messages of a session by input `session_id`",
		func(ctx *ai.ToolContext, input MemoryToolInput) (ToolOutput, error) {
			if input.SessionID == "" {
				return ToolOutput{}, fmt.Errorf("sessionID is required")
			}

			memoryComp, err := rt.GetComponent("memory")
			if err != nil {
				return ToolOutput{}, fmt.Errorf("memory component not found: %w", err)
			}
			memComp, ok := memoryComp.(*memory.MemoryComponent)
			if !ok {
				return ToolOutput{}, fmt.Errorf("invalid memory component type")
			}
			history, err := memComp.GetMemory()
			if err != nil {
				return ToolOutput{}, fmt.Errorf("failed to get history from memory component: %w", err)
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
	Queries []string `json:"queries"`
}

type K8SRAGToolOptions struct {
	RetrieveTopK int
	Namespace    string
	TargetIndex  string
	RerankTopN   int
}

func (K8SRAGToolOptions) Default() K8SRAGToolOptions {
	return K8SRAGToolOptions{
		RetrieveTopK: DefaultK8SDocRetrieveTopK,
		Namespace:    DefaultK8SDocNamespace,
		TargetIndex:  DefaultK8SDocTargetIndex,
		RerankTopN:   DefaultK8SDocRerankTopN,
	}
}

func defineMemoryTools(rt *runtime.Runtime) []ai.Tool {
	tools := []ai.Tool{
		getAllMemoryBySession(rt),
		RetrieveBasicConceptFromK8SDoc(rt),
	}
	return tools
}

func RetrieveBasicConceptFromK8SDoc(rt *runtime.Runtime) ai.Tool {
	g := rt.GetGenkitRegistry()
	return genkit.DefineTool(
		g, RetrieveBasicConceptFromK8SDocTool, "Retrieve the basic kubernetes concepts from RAG",
		func(ctx *ai.ToolContext, input K8SRAGQueryInput) (ToolOutput, error) {
			if len(input.Queries) == 0 {
				return ToolOutput{}, fmt.Errorf("queries is required")
			}

			ragCompRaw, err := rt.GetComponent("rag")
			if err != nil {
				return ToolOutput{}, fmt.Errorf("rag component not found: %w", err)
			}
			ragComp, ok := ragCompRaw.(*rag.RAGComponent)
			if !ok {
				return ToolOutput{}, fmt.Errorf("invalid rag component type")
			}
			ragSys := ragComp.Rag
			if ragSys == nil {
				return ToolOutput{}, fmt.Errorf("rag system is not initialized")
			}

			defaults := (K8SRAGToolOptions{}).Default()
			retrieveOpts := []rag.Option{
				rag.WithRetrieveTopK(defaults.RetrieveTopK),
				rag.WithTargetIndex(defaults.TargetIndex),
				rag.WithRerankTopN(defaults.RerankTopN),
			}

			results, err := ragSys.Retrieve(ctx, defaults.Namespace, input.Queries, retrieveOpts...)
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
