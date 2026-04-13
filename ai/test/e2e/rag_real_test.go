package e2e

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	appruntime "dubbo-admin-ai/runtime"
	compRag "dubbo-admin-ai/component/rag"
)

func TestRAGRealMethodCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	if os.Getenv("DASHSCOPE_API_KEY") == "" {
		t.Skip("DASHSCOPE_API_KEY not set")
	}
	if os.Getenv("MILVUS_HOST") == "" {
		t.Skip("MILVUS_HOST not set")
	}
	if os.Getenv("MILVUS_TOKEN") == "" {
		t.Skip("MILVUS_TOKEN not set")
	}

	t.Run("milvus_hybrid_with_query_enhancement", func(t *testing.T) {
		ctx := context.Background()
		_, file, _, _ := runtime.Caller(0)
		aiDir := filepath.Dir(filepath.Dir(filepath.Dir(file)))
		tmpDir := t.TempDir()

		toSlash := func(p string) string {
			return strings.ReplaceAll(p, "\\", "/")
		}

		// Get configuration from environment variables
		milvusHost := os.Getenv("MILVUS_HOST")
		milvusToken := os.Getenv("MILVUS_TOKEN")
		// Use the embedder name that's registered in models.yaml
		// Format: provider/key (from models.yaml: dashscope provider, key: text-embedding-v4)
		embedModel := "dashscope/text-embedding-v4"
		llmModel := os.Getenv("LLM_MODEL")
		if llmModel == "" {
			llmModel = "dashscope/qwen-max"
		}

		// Use existing Milvus test collection (created by milvus_e2e_test.go)
		testCollection := "dubbo_rag_test"

		ragConfigPath := filepath.Join(tmpDir, "rag_lifecycle.yaml")

		ragConfigBuilder := new(strings.Builder)
		ragConfigBuilder.WriteString("type: rag\n")
		ragConfigBuilder.WriteString("spec:\n")
		ragConfigBuilder.WriteString("  embedder:\n")
		ragConfigBuilder.WriteString("    type: genkit\n")
		ragConfigBuilder.WriteString("    spec:\n")
		ragConfigBuilder.WriteString("      model: " + embedModel + "\n")
		ragConfigBuilder.WriteString("  loader:\n")
		ragConfigBuilder.WriteString("    type: local\n")
		ragConfigBuilder.WriteString("    spec: {}\n")
		ragConfigBuilder.WriteString("  splitter:\n")
		ragConfigBuilder.WriteString("    type: recursive\n")
		ragConfigBuilder.WriteString("    spec:\n")
		ragConfigBuilder.WriteString("      chunk_size: 300\n")
		ragConfigBuilder.WriteString("      overlap_size: 50\n")
		// Milvus Indexer configuration
		ragConfigBuilder.WriteString("  indexer:\n")
		ragConfigBuilder.WriteString("    type: milvus\n")
		ragConfigBuilder.WriteString("    spec:\n")
		ragConfigBuilder.WriteString("      address: " + milvusHost + "\n")
		ragConfigBuilder.WriteString("      token: " + milvusToken + "\n")
		ragConfigBuilder.WriteString("      collection: " + testCollection + "\n")
		ragConfigBuilder.WriteString("      dimension: 1024\n")
		ragConfigBuilder.WriteString("      batch_size: 100\n")
		ragConfigBuilder.WriteString("      enable_sparse: false\n")
		// Milvus Retriever configuration - Hybrid search (dense + sparse)
		ragConfigBuilder.WriteString("  retriever:\n")
		ragConfigBuilder.WriteString("    type: milvus\n")
		ragConfigBuilder.WriteString("    spec:\n")
		ragConfigBuilder.WriteString("      address: " + milvusHost + "\n")
		ragConfigBuilder.WriteString("      token: " + milvusToken + "\n")
		ragConfigBuilder.WriteString("      collection: " + testCollection + "\n")
		// Use hybrid search for multi-path retrieval (dense + BM25)
		ragConfigBuilder.WriteString("      search_type: hybrid\n")
		ragConfigBuilder.WriteString("      metric_type: COSINE\n")
		ragConfigBuilder.WriteString("      dense_field: vector\n")
		ragConfigBuilder.WriteString("      dense_top_k: 10\n")
		ragConfigBuilder.WriteString("      sparse_field: sparse\n")
		ragConfigBuilder.WriteString("      sparse_top_k: 10\n")
		ragConfigBuilder.WriteString("      hybrid_ranker: rrf\n")
		ragConfigBuilder.WriteString("      dense_weight: 0.7\n")
		ragConfigBuilder.WriteString("      sparse_weight: 0.3\n")
		// Query Processor - Enable query enhancement/rewrite
		ragConfigBuilder.WriteString("  query_processor:\n")
		ragConfigBuilder.WriteString("    type: rewrite\n")
		ragConfigBuilder.WriteString("    spec:\n")
		ragConfigBuilder.WriteString("      enabled: true\n")
		ragConfigBuilder.WriteString("      model: " + llmModel + "\n")
		ragConfigBuilder.WriteString("      timeout: 10s\n")
		ragConfigBuilder.WriteString("      temperature: 0.3\n")
		ragConfigBuilder.WriteString("      fallback_on_error: true\n")
		// Reranker
		ragConfigBuilder.WriteString("  reranker:\n")
		ragConfigBuilder.WriteString("    type: cohere\n")
		ragConfigBuilder.WriteString("    spec:\n")
		ragConfigBuilder.WriteString("      enabled: false\n")

		_ = os.WriteFile(ragConfigPath, []byte(ragConfigBuilder.String()), 0644)

		agentConfigPath := filepath.Join(tmpDir, "agent.yaml")
		agentContent, _ := os.ReadFile(filepath.Join(aiDir, "component", "agent", "agent.yaml"))
		modifiedAgentContent := strings.Replace(string(agentContent), "prompt_base_path: \"./prompts\"",
			"prompt_base_path: \""+toSlash(filepath.Join(aiDir, "prompts"))+"\"", 1)
		_ = os.WriteFile(agentConfigPath, []byte(modifiedAgentContent), 0644)

		mainConfigBuilder := new(strings.Builder)
		mainConfigBuilder.WriteString("project: dubbo-admin-ai\n")
		mainConfigBuilder.WriteString("version: 1.0.0\n")
		mainConfigBuilder.WriteString("components:\n")
		mainConfigBuilder.WriteString("  logger: " + toSlash(filepath.Join(aiDir, "component", "logger", "logger.yaml")) + "\n")
		mainConfigBuilder.WriteString("  memory: " + toSlash(filepath.Join(aiDir, "component", "memory", "memory.yaml")) + "\n")
		mainConfigBuilder.WriteString("  models: " + toSlash(filepath.Join(aiDir, "component", "models", "models.yaml")) + "\n")
		mainConfigBuilder.WriteString("  tools: " + toSlash(filepath.Join(aiDir, "component", "tools", "tools.yaml")) + "\n")
		mainConfigBuilder.WriteString("  rag: " + toSlash(ragConfigPath) + "\n")
		mainConfigBuilder.WriteString("  agent: " + toSlash(agentConfigPath) + "\n")

		configPath := filepath.Join(tmpDir, "config.yaml")
		_ = os.WriteFile(configPath, []byte(mainConfigBuilder.String()), 0644)

		rt, err := appruntime.Bootstrap(configPath, registerFactories)
		if err != nil {
			t.Fatalf("Failed to bootstrap: %v", err)
		}
		defer rt.StopAll()

		ragComp, err := rt.GetComponent("rag")
		if err != nil {
			t.Fatalf("Failed to get RAG component: %v", err)
		}

		type ragComponentWithGetRAG interface {
			GetRAG() *compRag.RAG
		}
		ragC, ok := ragComp.(ragComponentWithGetRAG)
		if !ok {
			t.Fatalf("RAG component does not implement GetRAG()")
		}

		rag := ragC.GetRAG()
		if rag == nil {
			t.Fatal("RAG instance is nil")
		}

		t.Log("\n=== RAG Configuration ===")
		t.Logf("Milvus Address: %s", milvusHost)
		t.Logf("Collection: %s", testCollection)
		t.Logf("Search Type: hybrid (dense + sparse/RRF)")
		t.Logf("Query Processor: enabled (model: %s)", llmModel)

		t.Log("\n=== Step 1: SPLITTER - Test Document Splitting ===")
		testDocs := []*schema.Document{
			{ID: "doc1", Content: "Apache Dubbo is a high-performance Java-based RPC framework.", MetaData: map[string]any{"source": "test.txt"}},
			{ID: "doc2", Content: "Dubbo provides service governance with load balancing.", MetaData: map[string]any{"source": "test.txt"}},
			{ID: "doc3", Content: "The architecture uses Netty for communication.", MetaData: map[string]any{"source": "test.txt"}},
		}
		chunks, err := rag.Split(ctx, testDocs)
		if err != nil {
			t.Fatalf("Failed to split documents: %v", err)
		}
		t.Logf("✓ Split %d documents into %d chunks", len(testDocs), len(chunks))

		// Use a more ambiguous query that benefits from query enhancement
		// This query should trigger the rewrite step to expand/clarify terms
		complexQuery := "dubbo治理功能"

		t.Log("\n=== Step 2: RETRIEVE with Query Enhancement ===")
		t.Logf("Original Query: %s", complexQuery)

		retrieveReq := &compRag.RetrieveRequest{Query: complexQuery, TopK: 5}
		results, err := rag.RetrieveV2(ctx, retrieveReq)
		if err != nil {
			t.Fatalf("Failed to retrieve: %v", err)
		}

		t.Log("\n=== Query Processing Results ===")
		if results.QueryResult != nil {
			if results.QueryResult.Modified {
				t.Logf("Query was modified/enhanced:")
				t.Logf("  Processed: %s", results.QueryResult.Query)
			} else {
				t.Logf("Query was not modified (used as-is)")
			}
			if results.QueryResult.Intent != "" {
				t.Logf("  Intent: %s", results.QueryResult.Intent)
			}
			if results.QueryResult.Hypothetical != "" {
				t.Logf("  Hypothetical Document: %s", truncateString(results.QueryResult.Hypothetical, 100))
			}
			if len(results.QueryResult.Queries) > 0 {
				t.Logf("  Expanded Queries: %v", results.QueryResult.Queries)
			}
		}

		t.Log("\n=== Retrieval Results ===")
		t.Logf("Retrieved %d results", len(results.Results))
		for i, r := range results.Results {
			t.Logf("  Result %d:", i+1)
			t.Logf("    Score: %.4f", r.Score)
			t.Logf("    Content: %s", truncateString(r.Content, 100))
			if r.Source != "" {
				t.Logf("    Source: %s", r.Source)
			}
		}

		t.Log("\n=== RAG Multi-Path Test Complete ===")
	})
}

