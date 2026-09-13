/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package engine

import (
	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/runtime"
	conversationstore "dubbo-admin-ai/store"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

const (
	GetAllMemoryTool                   string = "memory_all_by_session_id"
	RetrieveBasicConceptFromK8SDocTool string = "retrieve_basic_concept_from_k8s_doc"
	QueryKnowledgeBaseTool             string = "query_knowledge_base"
	// Default RAG settings
	DefaultRAGNamespace    string = "dubbo" // Default namespace for Dubbo docs
	DefaultRAGRetrieveTopK int    = 5
	DefaultRAGRerankTopN   int    = 3
	// K8s specific defaults (for backward compatibility)
	DefaultK8SDocTargetIndex  string = "kube-docs"
	DefaultK8SDocNamespace    string = "concepts"
	DefaultK8SDocRetrieveTopK int    = 5
	DefaultK8SDocRerankTopN   int    = 2
)

type MemoryToolInput struct {
	SessionID string `json:"session_id"`
}

// getAllMemoryBySession creates a tool for retrieving conversation history.
// Use when: User references previous context or current question lacks prior context.
// Tool: memory_all_by_session_id
func getAllMemoryBySession(rt *runtime.Runtime, messageStore conversationstore.MessageStore) ai.Tool {
	g := rt.GetGenkitRegistry()
	return genkit.DefineTool(
		g, GetAllMemoryTool, "Retrieve conversation history for a specific session. Use this when user references previous context (e.g., 'previous', 'earlier', 'that config', 'my setup') or when current question cannot be answered without missing prior context.",
		func(ctx *ai.ToolContext, input MemoryToolInput) (ToolOutput, error) {
			if input.SessionID == "" {
				return ToolOutput{}, fmt.Errorf("sessionID is required")
			}

			history, err := messageStore.AllMemory(ctx, input.SessionID)
			if err != nil {
				return ToolOutput{}, fmt.Errorf("failed to load conversation history: %w", err)
			}
			if len(history) == 0 {
				return ToolOutput{
					ToolName: GetAllMemoryTool,
					Summary:  "No memory available",
				}, nil
			}
			return ToolOutput{
				ToolName: GetAllMemoryTool,
				Result:   history,
				Summary:  "",
			}, nil
		},
	)
}

type K8SRAGQueryInput struct {
	Queries []string `json:"queries"`
}

// Generic RAG input structures
type RAGRetrieveInput struct {
	Query      string `json:"query"`                // Single query string
	TopK       *int   `json:"topK,omitempty"`       // Optional: number of results to retrieve
	RerankTopN *int   `json:"rerankTopN,omitempty"` // Optional: number of results after reranking
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

func defineMemoryTools(rt *runtime.Runtime, messageStore conversationstore.MessageStore) []ai.Tool {
	tools := []ai.Tool{
		getAllMemoryBySession(rt, messageStore),
		queryKnowledgeBase(rt),
		RetrieveBasicConceptFromK8SDoc(rt),
	}
	return tools
}

// queryKnowledgeBase creates a tool for retrieving domain documentation from knowledge base.
// Use when: Questions about product features, concepts, configuration, or 'how-to' guidance.
// NOT for: Historical incidents or troubleshooting solutions.
// Tool: query_knowledge_base
func queryKnowledgeBase(rt *runtime.Runtime) ai.Tool {
	g := rt.GetGenkitRegistry()
	return genkit.DefineTool(
		g, QueryKnowledgeBaseTool, "Retrieve domain documentation from knowledge base (Dubbo/K8s official docs, technical guides, API references). Use this for questions about product features, concepts, configuration, or 'how-to' guidance. NOT for historical incidents or troubleshooting solutions.",
		func(ctx *ai.ToolContext, input RAGRetrieveInput) (ToolOutput, error) {
			if input.Query == "" {
				return ToolOutput{}, fmt.Errorf("query is required")
			}

			ragCompRaw, err := rt.GetComponent("rag")
			if err != nil {
				return ToolOutput{}, fmt.Errorf("rag component not found: %w", err)
			}
			ragComp, ok := ragCompRaw.(*compRag.RAGComponent)
			if !ok {
				return ToolOutput{}, fmt.Errorf("invalid rag component type")
			}
			ragSys := ragComp.GetRAG()
			if ragSys == nil {
				return ToolOutput{}, fmt.Errorf("rag system is not initialized")
			}

			// Set defaults
			topK := DefaultRAGRetrieveTopK
			if input.TopK != nil && *input.TopK > 0 {
				topK = *input.TopK
			}
			rerankTopN := DefaultRAGRerankTopN
			if input.RerankTopN != nil && *input.RerankTopN > 0 {
				rerankTopN = *input.RerankTopN
			}

			// Use RetrieveV2 for multi-path retrieval and reranking
			req := &compRag.RetrieveRequest{
				Query: input.Query,
				TopK:  topK,
			}
			resp, err := ragSys.RetrieveV2(ctx, req)
			if err != nil {
				return ToolOutput{}, fmt.Errorf("failed to retrieve from RAG: %w", err)
			}

			// Apply reranking to get topN results
			results := resp.Results
			if rerankTopN < len(results) {
				results = results[:rerankTopN]
			}

			return ToolOutput{
				ToolName: QueryKnowledgeBaseTool,
				Result:   results,
				Summary:  fmt.Sprintf("Retrieved %d results for query: %s", len(results), input.Query),
			}, nil
		},
	)
}

// RetrieveBasicConceptFromK8SDoc creates a tool for retrieving K8s concepts from RAG.
// Use when: K8s-specific questions about Pods, Services, Deployments, Namespaces, networking.
// Note: For general documentation queries, prefer query_knowledge_base.
// Tool: retrieve_basic_concept_from_k8s_doc
func RetrieveBasicConceptFromK8SDoc(rt *runtime.Runtime) ai.Tool {
	g := rt.GetGenkitRegistry()
	return genkit.DefineTool(
		g, RetrieveBasicConceptFromK8SDocTool, "Retrieve Kubernetes concepts and fundamentals from RAG. Use this for K8s-specific questions about Pods, Services, Deployments, Namespaces, networking, or other K8s core concepts. For general documentation queries, prefer query_knowledge_base.",
		func(ctx *ai.ToolContext, input K8SRAGQueryInput) (ToolOutput, error) {
			if len(input.Queries) == 0 {
				return ToolOutput{}, fmt.Errorf("queries is required")
			}

			ragCompRaw, err := rt.GetComponent("rag")
			if err != nil {
				return ToolOutput{}, fmt.Errorf("rag component not found: %w", err)
			}
			ragComp, ok := ragCompRaw.(*compRag.RAGComponent)
			if !ok {
				return ToolOutput{}, fmt.Errorf("invalid rag component type")
			}
			ragSys := ragComp.GetRAG()
			if ragSys == nil {
				return ToolOutput{}, fmt.Errorf("rag system is not initialized")
			}

			defaults := (K8SRAGToolOptions{}).Default()
			retrieveOpts := []compRag.RetrieveOption{
				compRag.WithTopN(defaults.RerankTopN),
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
