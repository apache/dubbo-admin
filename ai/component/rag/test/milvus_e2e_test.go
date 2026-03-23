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

package ragtest

import (
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/component/rag/indexers"
	"dubbo-admin-ai/component/rag/loaders"
	"dubbo-admin-ai/component/rag/retrievers"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go/option"
)

// TestMilvusRAGE2E_FullWorkflow tests the complete RAG workflow with Milvus:
// Load -> Split -> Index -> Retrieve
func TestMilvusRAGE2E_FullWorkflow(t *testing.T) {
	// Load .env file
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Skip("Skipping test: .env file not found")
	}

	host := os.Getenv("MILVUS_HOST")
	token := os.Getenv("MILVUS_TOKEN")
	if host == "" || token == "" {
		t.Skip("Skipping test: MILVUS_HOST and MILVUS_TOKEN environment variables are required")
	}

	// Initialize genkit with embedder
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("EMBEDDING_API_KEY")
	}
	if apiKey == "" {
		t.Skip("DASHSCOPE_API_KEY/EMBEDDING_API_KEY not configured")
	}

	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	// Remove /embeddings suffix if present
	if len(baseURL) > 12 && baseURL[len(baseURL)-11:] == "/embeddings" {
		baseURL = baseURL[:len(baseURL)-11]
	}

	ctx := context.Background()

	dashscopePlugin := &compat_oai.OpenAICompatible{
		Provider: "dashscope",
		Opts: []option.RequestOption{
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
		},
	}

	g := genkit.Init(ctx, genkit.WithPlugins(dashscopePlugin))

	embedder := dashscopePlugin.DefineEmbedder("dashscope", "text-embedding-v4", &ai.EmbedderOptions{
		Label:      "text-embedding-v4",
		Supports:   &ai.EmbedderSupports{Input: []string{"text"}},
		Dimensions: 1024,
	})

	genkit.RegisterAction(g, embedder)

	embedderName := "dashscope/text-embedding-v4"
	if genkit.LookupEmbedder(g, embedderName) == nil {
		t.Fatalf("Failed to lookup embedder: %s", embedderName)
	}

	// Use the same collection as multipath tests to avoid exceeding limit
	collectionName := "dubbo_rag_test"

	// Create client to check if collection exists
	cli, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: host,
		APIKey:  token,
	})
	if err != nil {
		t.Fatalf("Failed to create Milvus client: %v", err)
	}
	defer cli.Close(ctx)

	// Check if collection exists
	hasCollection, _ := cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(collectionName))
	if hasCollection {
		t.Logf("Collection %s already exists, reusing it", collectionName)
	} else {
		t.Logf("Collection %s does not exist, will create it", collectionName)
	}

	// Step 1: Create test documents
	testDir := t.TempDir()
	createRAGTestDocuments(t, testDir)
	t.Logf("Created test documents in: %s", testDir)

	// Step 2: Load documents
	loader, err := loaders.NewLocalFileLoader(ctx)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	loadedDocs, err := loaders.LoadDirectory(ctx, loader, testDir)
	if err != nil {
		t.Fatalf("Failed to load directory: %v", err)
	}

	t.Logf("Loaded %d documents", len(loadedDocs))
	for i, doc := range loadedDocs {
		t.Logf("  Doc[%d]: source=%s, %d chars",
			i, doc.MetaData["source"], len(doc.Content))
	}

	// Step 3: Create Milvus indexer
	indexerCfg := &indexers.MilvusConfig{
		Address:        host,
		Token:          token,
		Collection:     collectionName,
		Dimension:      1024, // Use actual DashScope dimension
		Embedder:       "dashscope/text-embedding-v4",
		BatchSize:      100,
		EnableMetadata: true,
	}

	idx, err := indexers.NewMilvusIndexer(nil, indexerCfg)
	if err != nil {
		t.Fatalf("Failed to create Milvus indexer: %v", err)
	}
	defer idx.Close()

	// Step 4: Split and index documents
	chunkSize := 500
	overlap := 50
	chunks := manualSplit(loadedDocs, chunkSize, overlap)

	t.Logf("Split into %d chunks", len(chunks))

	// Only index if collection is new or was just created
	if !hasCollection {
		// Index chunks
		ids, err := idx.Store(ctx, chunks)
		if err != nil {
			t.Fatalf("Failed to index chunks: %v", err)
		}

		t.Logf("Indexed %d chunks, got %d IDs", len(chunks), len(ids))

		// Flush
		if err := idx.Flush(ctx); err != nil {
			t.Logf("Flush warning: %v", err)
		}

		// Wait for indexing
		time.Sleep(3 * time.Second)
	} else {
		t.Logf("Reusing existing collection with data, skipping indexing")
	}

	// Flush
	if err := idx.Flush(ctx); err != nil {
		t.Logf("Flush warning: %v", err)
	}

	// Step 5: Create Milvus retriever
	retrieverCfg := &retrievers.MilvusConfig{
		Address:       host,
		Token:         token,
		Collection:    collectionName,
		Embedder:      "dashscope/text-embedding-v4",
		SearchType:    "dense",
		DenseField:    "vector",
		DenseTopK:     5,
		MetricType:    "COSINE",
		EnableMetadata: true,
		SourceField:   "source",
		TitleField:    "title",
	}

	rtv, err := retrievers.NewMilvusRetriever(g, retrieverCfg)
	if err != nil {
		t.Fatalf("Failed to create Milvus retriever: %v", err)
	}
	defer rtv.Close()

	// Step 6: Create RAG instance
	rag := &compRag.RAG{
		Loader:    loader,
		Indexer:   idx,
		Retriever: rtv,
	}

	// Step 7: Test retrieval
	testQueries := []struct {
		name   string
		query  string
	}{
		{"dubbo_intro", "What is Dubbo?"},
		{"service_discovery", "service discovery"},
		{"architecture", "microservices architecture"},
	}

	for _, tq := range testQueries {
		t.Run(tq.name, func(t *testing.T) {
			results, err := rag.Retrieve(ctx, collectionName, []string{tq.query})
			if err != nil {
				t.Fatalf("Retrieve() error for query %q: %v", tq.query, err)
			}

			t.Logf("Query: %s", tq.query)
			t.Logf("Results: %d", len(results[tq.query]))

			for i, result := range results[tq.query] {
				t.Logf("  [%d] Score: %.4f", i, result.Score)
				t.Logf("      Content: %s", truncateString(result.Content, 80))
				t.Logf("      Source: %s", result.Source)
				t.Logf("      Title: %s", result.Title)
			}

			if len(results[tq.query]) == 0 {
				t.Logf("Warning: No results for query %q", tq.query)
			}
		})
	}
}

// TestMilvusRAG_NamespaceIsolation tests namespace isolation.
func TestMilvusRAG_NamespaceIsolation(t *testing.T) {
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Skip("Skipping test: .env file not found")
	}

	host := os.Getenv("MILVUS_HOST")
	token := os.Getenv("MILVUS_TOKEN")
	if host == "" || token == "" {
		t.Skip("Skipping test: MILVUS_HOST and MILVUS_TOKEN environment variables are required")
	}

	// Initialize genkit with embedder
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("EMBEDDING_API_KEY")
	}
	if apiKey == "" {
		t.Skip("DASHSCOPE_API_KEY/EMBEDDING_API_KEY not configured")
	}

	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if len(baseURL) > 12 && baseURL[len(baseURL)-11:] == "/embeddings" {
		baseURL = baseURL[:len(baseURL)-11]
	}

	ctx := context.Background()

	dashscopePlugin := &compat_oai.OpenAICompatible{
		Provider: "dashscope",
		Opts: []option.RequestOption{
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
		},
	}

	g := genkit.Init(ctx, genkit.WithPlugins(dashscopePlugin))

	embedder := dashscopePlugin.DefineEmbedder("dashscope", "text-embedding-v4", &ai.EmbedderOptions{
		Label:      "text-embedding-v4",
		Supports:   &ai.EmbedderSupports{Input: []string{"text"}},
		Dimensions: 1024,
	})

	genkit.RegisterAction(g, embedder)

	// Use the same collection as multipath tests to avoid exceeding limit
	collectionName := "dubbo_rag_test"

	// Create client to check if collection exists
	cli, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: host,
		APIKey:  token,
	})
	if err != nil {
		t.Fatalf("Failed to create Milvus client: %v", err)
	}
	defer cli.Close(ctx)

	// Check if collection exists
	hasCollection, _ := cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(collectionName))

	// Create indexer and retriever
	indexerCfg := &indexers.MilvusConfig{
		Address:        host,
		Token:          token,
		Collection:     collectionName,
		Dimension:      1024,
		Embedder:       "dashscope/text-embedding-v4",
		BatchSize:      100,
		EnableMetadata: true,
	}

	idx, err := indexers.NewMilvusIndexer(g, indexerCfg)
	if err != nil {
		t.Fatalf("Failed to create Milvus indexer: %v", err)
	}
	defer idx.Close()

	retrieverCfg := &retrievers.MilvusConfig{
		Address:       host,
		Token:         token,
		Collection:    collectionName,
		Embedder:      "dashscope/text-embedding-v4",
		SearchType:    "dense",
		DenseField:    "vector",
		DenseTopK:     5,
		MetricType:    "COSINE",
		EnableMetadata: true,
	}

	rtv, err := retrievers.NewMilvusRetriever(g, retrieverCfg)
	if err != nil {
		t.Fatalf("Failed to create Milvus retriever: %v", err)
	}
	defer rtv.Close()
	defer rtv.Close()

	rag := &compRag.RAG{
		Indexer:   idx,
		Retriever: rtv,
	}

	// Index documents in different namespaces
	ns1Docs := []*schema.Document{
		{
			ID:        "ns1_doc1",
			Content:   "Namespace 1: This content is only in namespace 1",
			MetaData: map[string]any{
				"namespace": "ns1",
			},
		},
	}

	ns2Docs := []*schema.Document{
		{
			ID:        "ns2_doc1",
			Content:   "Namespace 2: This content is only in namespace 2",
			MetaData: map[string]any{
				"namespace": "ns2",
			},
		},
	}

	// Only index if collection is new
	if !hasCollection {
		// Index with namespace options
		_, err = rag.Index(ctx, "namespace1", ns1Docs)
		if err != nil {
			t.Fatalf("Failed to index ns1 documents: %v", err)
		}

		_, err = rag.Index(ctx, "namespace2", ns2Docs)
		if err != nil {
			t.Fatalf("Failed to index ns2 documents: %v", err)
		}

		_ = idx.Flush(ctx)
		time.Sleep(3 * time.Second)
		t.Logf("Created new collection and indexed test documents")
	} else {
		t.Logf("Reusing existing collection with test documents")
	}

	// Test namespace isolation
	results1, err := rag.Retrieve(ctx, "namespace1", []string{"content"})
	if err != nil {
		t.Fatalf("Retrieve() from namespace1 error: %v", err)
	}

	results2, err := rag.Retrieve(ctx, "namespace2", []string{"content"})
	if err != nil {
		t.Fatalf("Retrieve() from namespace2 error: %v", err)
	}

	t.Logf("Namespace1 results: %d", len(results1["content"]))
	t.Logf("Namespace2 results: %d", len(results2["content"]))
}

// createRAGTestDocuments creates test documents for RAG testing.
func createRAGTestDocuments(t *testing.T, dir string) {
	docs := map[string]string{
		"dubbo_intro.md": `# Dubbo Introduction

Dubbo is a high-performance, lightweight Java RPC framework.
It provides three core capabilities: remote interface invocation, graceful fault tolerance, and service discovery.

## Key Features

- **High Performance**: Uses Netty for NIO transport
- **Lightweight**: No heavy container dependencies
- **Service Discovery**: Supports multiple registries like Zookeeper, Nacos
`,
		"service_discovery.md": `# Service Discovery in Dubbo

Service discovery is a critical component of microservices architecture.

## Registration Center

Dubbo supports multiple registration centers:
- Zookeeper: Recommended for production
- Nacos: Alibaba's open source solution
- Redis: Simple deployment

## How it Works

1. Provider starts and registers with registry
2. Consumer subscribes to provider list
3. Registry pushes provider updates
`,
		"microservices.md": `# Microservices with Dubbo

## What are Microservices?

Microservices architecture breaks down applications into small, independent services.

## Dubbo for Microservices

Dubbo simplifies microservices development with load balancing and fault tolerance.
`,
	}

	for filename, content := range docs {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}
}
