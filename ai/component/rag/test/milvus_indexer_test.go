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
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

// truncateString truncates a string to a maximum length and adds "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// TestMilvusIndexerConfig tests Milvus indexer configuration validation.
func TestMilvusIndexerConfig(t *testing.T) {
	// Save and restore env vars for missing_address test
	oldHost := os.Getenv("MILVUS_HOST")
	oldToken := os.Getenv("MILVUS_TOKEN")

	tests := []struct {
		name    string
		config  *indexers.MilvusConfig
		wantErr bool
		setup   func() // setup function before test
		cleanup func() // cleanup function after test
	}{
		{
			name: "valid_config",
			config: &indexers.MilvusConfig{
				Address:      "localhost:19530",
				Collection:   "test",
				Dimension:    1536,
				Embedder:     "text-embedding-v4",
				BatchSize:    100,
				EnableBM25:   false,
				IDField:      "id",
				DenseField:   "vector",
				TextField:    "text",
				SourceField:  "source",
				TitleField:   "title",
			},
			wantErr: false,
		},
		{
			name: "missing_address",
			config: &indexers.MilvusConfig{
				Collection: "test",
				Dimension:  1536,
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
			name: "missing_collection",
			config: &indexers.MilvusConfig{
				Address:   "localhost:19530",
				Dimension: 1536,
				Embedder:  "text-embedding-v4",
			},
			wantErr: true,
		},
		{
			name: "invalid_dimension",
			config: &indexers.MilvusConfig{
				Address:    "localhost:19530",
				Collection: "test",
				Dimension:  0,
				Embedder:   "text-embedding-v4",
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
			err := indexers.ValidateConfig(tt.config)
			if (err != nil) && !tt.wantErr {
				t.Errorf("ValidateConfig() unexpected error = %v", err)
			}
			if tt.wantErr && err == nil {
				t.Error("ValidateConfig() expected error, got nil")
			}
		})
	}
}

// TestMilvusIndexer_Store tests Milvus indexer functionality.
// Requires MILVUS_HOST and MILVUS_TOKEN environment variables.
func TestMilvusIndexer_Store(t *testing.T) {
	// Load .env file
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Skip("Skipping test: .env file not found")
	}

	host := os.Getenv("MILVUS_HOST")
	token := os.Getenv("MILVUS_TOKEN")
	if host == "" || token == "" {
		t.Skip("Skipping test: MILVUS_HOST and MILVUS_TOKEN environment variables are required")
	}

	ctx := context.Background()
	// Use the same collection as multipath tests to avoid exceeding limit
	collectionName := "dubbo_rag_test"

	// Check if collection exists first
	cli, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: host,
		APIKey:  token,
	})
	if err != nil {
		t.Fatalf("Failed to create Milvus client: %v", err)
	}
	defer cli.Close(ctx)

	hasCollection, _ := cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(collectionName))

	cfg := &indexers.MilvusConfig{
		Address:        host,
		Token:          token,
		Collection:     collectionName,
		Dimension:      1024,
		Embedder:       "dashscope/text-embedding-v4",
		BatchSize:      100,
		EnableMetadata: true,
		EnableBM25:     true,
	}

	idx, err := indexers.NewMilvusIndexer(nil, cfg)
	if err != nil {
		t.Fatalf("Failed to create Milvus indexer: %v", err)
	}
	defer idx.Close()

	// Only store documents if collection is new
	if !hasCollection {
		// Create test documents
		docs := []*schema.Document{
			{
				ID:      "doc1",
				Content: "Dubbo is a high-performance RPC framework",
				MetaData: map[string]any{
					"source": "test.md",
					"title":  "Introduction",
				},
			},
			{
				ID:      "doc2",
				Content: "It provides service discovery and load balancing",
				MetaData: map[string]any{
					"source": "test.md",
					"title":  "Features",
				},
			},
			{
				ID:      "doc3",
				Content: "Kubernetes is a container orchestration platform",
				MetaData: map[string]any{
					"source": "k8s.md",
					"title":  "K8s Basics",
				},
			},
		}

		// Store documents
		ids, err := idx.Store(ctx, docs)
		if err != nil {
			t.Fatalf("Store() error = %v", err)
		}

		if len(ids) != 3 {
			t.Errorf("Store() returned %d ids, want 3", len(ids))
		}

		t.Logf("Stored documents with IDs: %v", ids)

		// Flush
		if err := idx.Flush(ctx); err != nil {
			t.Logf("Flush() warning: %v", err)
		}

		t.Logf("Created new collection and stored test documents")
	} else {
		t.Logf("Collection already exists, reusing existing data")
	}
}
