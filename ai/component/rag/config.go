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

package rag

import (
	"dubbo-admin-ai/component/rag/query"
	"dubbo-admin-ai/config"
	"fmt"
)

// QueryProcessor is an alias for query.QueryProcessor (legacy interface)
type QueryProcessor = query.QueryProcessor

// QueryProcessorConfig is an alias for query.QueryProcessorConfig (legacy interface)
type QueryProcessorConfig = query.QueryProcessorConfig

// RAGSpec defines RAG component configuration with recursive structure
// Each subcomponent uses the standard Config pattern (type + spec)
type RAGSpec struct {
	Embedder        *config.Config `yaml:"embedder"`
	Loader          *config.Config `yaml:"loader"`
	Splitter        *config.Config `yaml:"splitter"`
	Indexer         *config.Config `yaml:"indexer"`
	Retriever       *config.Config `yaml:"retriever"`
	Reranker        *config.Config `yaml:"reranker,omitempty"`
	QueryProcessor  *config.Config `yaml:"query_processor,omitempty"`
}

// EmbedderSpec defines embedder specific parameters
type EmbedderSpec struct {
	Model string `yaml:"model"`
}

// LoaderSpec defines loader specific parameters
// Loader has no specific parameters, configuration is through spec
type LoaderSpec struct {
	// Loader has no concrete parameters
}

// SplitterSpec defines splitter specific parameters
type SplitterSpec struct {
	ChunkSize   int `yaml:"chunk_size"`
	OverlapSize int `yaml:"overlap_size"`
}

// IndexerSpec defines indexer specific parameters
type IndexerSpec struct {
	StoragePath string `yaml:"storage_path"`
	IndexFormat string `yaml:"index_format"`
	Dimension   int    `yaml:"dimension"`
}

// RetrieverSpec defines retriever specific parameters
type RetrieverSpec struct {
	StoragePath string `yaml:"storage_path"`
	IndexFormat string `yaml:"index_format"`
	Dimension   int    `yaml:"dimension"`
}

// RerankerSpec defines reranker specific parameters
type RerankerSpec struct {
	Enabled bool   `yaml:"enabled"`
	Model   string `yaml:"model"`
	APIKey  string `yaml:"api_key,omitempty"`
}

// QueryProcessorSpec defines query processor specific parameters
type QueryProcessorSpec struct {
	Enabled         bool    `yaml:"enabled"`
	Model           string  `yaml:"model"`
	Timeout         string  `yaml:"timeout"`        // Duration string like "5s"
	Temperature     float64 `yaml:"temperature"`
	FallbackOnError bool    `yaml:"fallback_on_error"`
}

// MilvusIndexerSpec defines Milvus indexer specific parameters
type MilvusIndexerSpec struct {
	Address     string `yaml:"address"`      // Milvus server address (env: MILVUS_HOST)
	Token       string `yaml:"token"`        // Auth token for Zilliz Cloud (env: MILVUS_TOKEN)
	Username    string `yaml:"username"`     // Optional username (for self-hosted)
	Password    string `yaml:"password"`     // Optional password (for self-hosted)
	Collection  string `yaml:"collection"`   // Collection name
	Dimension   int    `yaml:"dimension"`    // Vector dimension
	BatchSize   int    `yaml:"batch_size"`   // Insert batch size
	EnableSparse bool   `yaml:"enable_sparse"` // Enable sparse vector support for BM25

	// Field names
	DenseField  string `yaml:"dense_field"`  // Dense vector field (default: "vector")
	SparseField string `yaml:"sparse_field"` // Sparse vector field for BM25 (default: "sparse_vector")
	TextField   string `yaml:"text_field"`   // Text content field (default: "text")

	// Index configuration
	DenseIndexType  string `yaml:"dense_index_type"`  // IVF_FLAT, HNSW, etc.
	SparseIndexType string `yaml:"sparse_index_type"` // SPARSE_INVERTED_INDEX, etc.
}

// MilvusRetrieverSpec defines Milvus retriever specific parameters with hybrid search support
type MilvusRetrieverSpec struct {
	Address    string `yaml:"address"`     // Milvus server address (env: MILVUS_HOST)
	Token      string `yaml:"token"`       // Auth token for Zilliz Cloud (env: MILVUS_TOKEN)
	Username   string `yaml:"username"`    // Optional username (for self-hosted)
	Password   string `yaml:"password"`    // Optional password (for self-hosted)
	Collection string `yaml:"collection"`  // Collection name

	// Search type: "dense" (vector), "sparse" (BM25), or "hybrid" (both)
	SearchType string `yaml:"search_type"` // dense | sparse | hybrid

	// Dense search configuration
	DenseField string `yaml:"dense_field"` // Dense vector field (default: "vector")
	DenseTopK  int    `yaml:"dense_top_k"` // Default TopK for dense search
	MetricType string `yaml:"metric_type"` // Metric type: L2, IP, COSINE (default: "COSINE")

	// Sparse search configuration (BM25)
	SparseField string `yaml:"sparse_field"` // Sparse vector field (default: "sparse_vector")
	SparseTopK  int    `yaml:"sparse_top_k"` // Default TopK for sparse search

	// Hybrid search configuration
	HybridRanker string  `yaml:"hybrid_ranker"`  // rrf | weighted_rank | nnf
	DenseWeight  float64 `yaml:"dense_weight"`   // Weight for dense results (default: 0.7)
	SparseWeight float64 `yaml:"sparse_weight"`  // Weight for sparse results (default: 0.3)
}

// DefaultEmbedderSpec returns default embedder configuration
func DefaultEmbedderSpec() *EmbedderSpec {
	return &EmbedderSpec{Model: "dashscope/qwen3-embedding"}
}

// DefaultSplitterSpec returns default splitter configuration
func DefaultSplitterSpec() *SplitterSpec {
	return &SplitterSpec{ChunkSize: 1000, OverlapSize: 100}
}

// DefaultIndexerSpec returns default indexer configuration
func DefaultIndexerSpec() *IndexerSpec {
	return &IndexerSpec{
		StoragePath: "../../data/ai/index",
		IndexFormat: "sqlite",
		Dimension:   1536,
	}
}

// DefaultRetrieverSpec returns default retriever configuration
func DefaultRetrieverSpec() *RetrieverSpec {
	return &RetrieverSpec{
		StoragePath: "../../data/ai/index",
		IndexFormat: "sqlite",
		Dimension:   1536,
	}
}

// DefaultRerankerSpec returns default reranker configuration
func DefaultRerankerSpec() *RerankerSpec {
	return &RerankerSpec{
		Enabled: false,
		Model:   "rerank-english-v3.0",
	}
}

// DefaultQueryProcessorSpec returns default query processor configuration
func DefaultQueryProcessorSpec() *QueryProcessorSpec {
	return &QueryProcessorSpec{
		Enabled:         false,
		Model:           "dashscope/qwen-max",
		Timeout:         "5s",
		Temperature:     0.3,
		FallbackOnError: true,
	}
}

// Validate validates RAG configuration
func (c *RAGSpec) Validate() error {
	if c == nil {
		return fmt.Errorf("rag config is nil")
	}
	if c.Splitter != nil && c.Splitter.Type == "recursive" {
		var splitter SplitterSpec
		if err := c.Splitter.Spec.Decode(&splitter); err != nil {
			return fmt.Errorf("failed to decode splitter spec: %w", err)
		}
		if splitter.ChunkSize <= 0 {
			return fmt.Errorf("splitter.chunk_size must be greater than 0")
		}
		if splitter.OverlapSize < 0 {
			return fmt.Errorf("splitter.overlap_size must be >= 0")
		}
		if splitter.OverlapSize >= splitter.ChunkSize {
			return fmt.Errorf("splitter.overlap_size must be less than chunk_size")
		}
	}
	if c.Indexer != nil {
		switch c.Indexer.Type {
		case "local", "pinecone", "milvus":
		default:
			return fmt.Errorf("unsupported indexer type: %s", c.Indexer.Type)
		}
	}
	if c.Retriever != nil {
		switch c.Retriever.Type {
		case "local", "pinecone", "milvus":
		default:
			return fmt.Errorf("unsupported retriever type: %s", c.Retriever.Type)
		}
	}
	return nil
}

// --- Exported types for rag package ---

// RetrieveResult defines the unified result structure for RAG queries.
// High-frequency fields are flattened for direct access; low-frequency fields go into Metadata.
type RetrieveResult struct {
	Content  string         `json:"content"`
	Score    float64        `json:"score"`               // Final relevance score
	Source   string         `json:"source,omitempty"`    // Document source path
	Title    string         `json:"title,omitempty"`     // Document title
	Metadata map[string]any `json:"metadata,omitempty"` // Extended metadata (page, header_path, etc.)
}

// ========== New API Types ==========

// RetrieveRequest represents a retrieval request.
type RetrieveRequest struct {
	Query     string          // Original user query
	TopK      int             // Maximum results to return
	Namespace string          // Namespace/collection
	Options   map[string]any  // Additional options
}

// RetrieveResponse represents the response from retrieval.
type RetrieveResponse struct {
	Results       []*RetrieveResult    // Retrieved documents
	QueryResult   *QueryProcessResult   // Query processing results
	RetrievalMeta map[string]any        // Retrieval metadata
}

// QueryProcessResult represents the result of query processing.
// This is an alias for query.Result for external use.
type QueryProcessResult = query.Result

// DefaultRetrieveRequest returns a default retrieval request.
func DefaultRetrieveRequest() *RetrieveRequest {
	return &RetrieveRequest{
		TopK:    10,
		Options: make(map[string]any),
	}
}
