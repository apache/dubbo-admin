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
	"testing"

	"dubbo-admin-ai/component/rag/indexers"
	"dubbo-admin-ai/component/rag/retrievers"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go/option"
)

// TestMilvusRetrieverConfig tests Milvus retriever configuration validation.
func TestMilvusRetrieverConfig(t *testing.T) {
	// Save and restore env vars for missing_address test
	oldHost := os.Getenv("MILVUS_HOST")
	oldToken := os.Getenv("MILVUS_TOKEN")

	tests := []struct {
		name    string
		config  *retrievers.MilvusConfig
		wantErr bool
		setup   func() // setup function before test
		cleanup func() // cleanup function after test
	}{
		{
			name: "valid_config",
			config: &retrievers.MilvusConfig{
				Address:      "localhost:19530",
				Collection:   "test",
				Embedder:     "text-embedding-v4",
				SearchType:   "dense",
				DenseField:   "vector",
				DenseTopK:    10,
				MetricType:   "COSINE",
				SourceField:  "source",
				TitleField:   "title",
			},
			wantErr: false,
		},
		{
			name: "missing_address",
			config: &retrievers.MilvusConfig{
				Collection: "test",
				Embedder:   "text-embedding-v4",
			},
			wantErr: true,
			setup: func() {
				// Clear env vars to test missing address scenario
				os.Unsetenv("MILVUS_HOST")
				os.Unsetenv("MILVUS_TOKEN")
			},
			cleanup: func() {
				// Restore env vars
				if oldHost != "" {
					os.Setenv("MILVUS_HOST", oldHost)
				}
				if oldToken != "" {
					os.Setenv("MILVUS_TOKEN", oldToken)
				}
			},
		},
		{
			name: "invalid_search_type",
			config: &retrievers.MilvusConfig{
				Address:    "localhost:19530",
				Collection: "test",
				Embedder:   "text-embedding-v4",
				SearchType: "invalid",
			},
			wantErr: true,
		},
		{
			name: "sparse_search_without_text_field",
			config: &retrievers.MilvusConfig{
				Address:    "localhost:19530",
				Collection: "test",
				Embedder:   "text-embedding-v4",
				SearchType: "sparse",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
				defer tt.cleanup()
			}
			err := retrievers.ValidateConfig(tt.config)
			if (err != nil) && !tt.wantErr {
				t.Errorf("ValidateConfig() unexpected error = %v", err)
			}
			if tt.wantErr && err == nil {
				t.Error("ValidateConfig() expected error, got nil")
			}
		})
	}
}

// TestMilvusRetriever_Retrieve tests Milvus retriever functionality.
// Requires MILVUS_HOST and MILVUS_TOKEN environment variables.
func TestMilvusRetriever_Retrieve(t *testing.T) {
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

	// Use the same collection as multipath tests
	collectionName := "dubbo_rag_test"

	// Check if collection exists
	cli, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: host,
		APIKey:  token,
	})
	if err != nil {
		t.Fatalf("Failed to create Milvus client: %v", err)
	}
	defer cli.Close(ctx)

	hasCollection, _ := cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(collectionName))

	// Setup: Create indexer and index some documents first (only if collection doesn't exist)
	indexerCfg := &indexers.MilvusConfig{
		Address:        host,
		Token:          token,
		Collection:     collectionName,
		Dimension:      1024,
		Embedder:       "dashscope/text-embedding-v4",
		BatchSize:      100,
		EnableMetadata: true,
		EnableBM25:     true,
	}

	idx, err := indexers.NewMilvusIndexer(g, indexerCfg)
	if err != nil {
		t.Fatalf("Failed to create Milvus indexer: %v", err)
	}
	defer idx.Close()

	if !hasCollection {
		// Index test documents
		testDocs := []*schema.Document{
			{
				ID:      "test1",
				Content: "Dubbo is a high-performance RPC framework for microservices",
				MetaData: map[string]any{
					"source": "dubbo_guide.md",
					"title":  "Introduction",
				},
			},
			{
				ID:      "test2",
				Content: "Apache Dubbo provides service discovery and load balancing",
				MetaData: map[string]any{
					"source": "dubbo_guide.md",
					"title":  "Features",
				},
			},
			{
				ID:      "test3",
				Content: "Kubernetes is a container orchestration platform",
				MetaData: map[string]any{
					"source": "k8s_guide.md",
					"title":  "K8s Basics",
				},
			},
		}

		_, err = idx.Store(ctx, testDocs)
		if err != nil {
			t.Fatalf("Failed to store test documents: %v", err)
		}

		// Flush
		if err := idx.Flush(ctx); err != nil {
			t.Logf("Flush warning: %v", err)
		}

		t.Logf("Created new collection and indexed test documents")
	} else {
		t.Logf("Reusing existing collection with test documents")
	}

	// Create retriever
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
	defer rtv.Close()

	// Test retrieval
	t.Run("basic_retrieve", func(t *testing.T) {
		docs, err := rtv.Retrieve(ctx, "Dubbo framework")
		if err != nil {
			t.Fatalf("Retrieve() error = %v", err)
		}

		t.Logf("Retrieved %d documents", len(docs))
		for i, doc := range docs {
			t.Logf("  [%d] Content: %s", i, truncateString(doc.Content, 60))
			if doc.MetaData != nil {
				t.Logf("      Source: %v, Title: %v",
					doc.MetaData["source"], doc.MetaData["title"])
			}
		}
	})

	t.Run("retrieve_with_topk", func(t *testing.T) {
		retrieverCfg.DenseTopK = 2
		rtv2, err := retrievers.NewMilvusRetriever(g, retrieverCfg)
		if err != nil {
			t.Fatalf("Failed to create Milvus retriever: %v", err)
		}
		defer rtv2.Close()

		docs, err := rtv2.Retrieve(ctx, "architecture")
		if err != nil {
			t.Fatalf("Retrieve() error = %v", err)
		}

		if len(docs) > 2 {
			t.Errorf("Retrieve() returned %d results, want at most 2", len(docs))
		}

		t.Logf("Retrieved %d documents (TopK=2)", len(docs))
	})
}
