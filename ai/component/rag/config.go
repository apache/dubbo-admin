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
	"dubbo-admin-ai/config"
	"fmt"
)

const (
	DefaultIndexerTargetIndex   = "default"
	DefaultIndexerBatchSize     = 100
	DefaultRetrieverTargetIndex = "default"
	DefaultRetrieverTopK        = 3
	DefaultRerankerModel        = "rerank-english-v3.0"
)

// RAGSpec defines RAG component configuration with recursive structure
// Each subcomponent uses the standard Config pattern (type + spec)
type RAGSpec struct {
	Embedder  *config.Config `yaml:"embedder"`
	Loader    *config.Config `yaml:"loader"`
	Splitter  *config.Config `yaml:"splitter"`
	Indexer   *config.Config `yaml:"indexer"`
	Retriever *config.Config `yaml:"retriever"`
	Reranker  *config.Config `yaml:"reranker,omitempty"`
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

// MarkdownHeaderSplitterSpec defines markdown header splitter specific parameters.
type MarkdownHeaderSplitterSpec struct {
	Headers     map[string]string `yaml:"headers"`
	TrimHeaders bool              `yaml:"trim_headers"`
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

// DefaultEmbedderSpec returns default embedder configuration
func DefaultEmbedderSpec() *EmbedderSpec {
	return &EmbedderSpec{Model: "dashscope/text-embedding-v4"}
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

// Validate validates RAG configuration
func (c *RAGSpec) Validate() error {
	if c == nil {
		return fmt.Errorf("rag config is nil")
	}
	if c.Embedder == nil {
		return fmt.Errorf("embedder config is required")
	}
	if c.Loader == nil {
		return fmt.Errorf("loader config is required")
	}
	if c.Splitter == nil {
		return fmt.Errorf("splitter config is required")
	}
	if c.Indexer == nil {
		return fmt.Errorf("indexer config is required")
	}
	if c.Retriever == nil {
		return fmt.Errorf("retriever config is required")
	}

	switch c.Loader.Type {
	case "", "local":
	default:
		return fmt.Errorf("unsupported loader type: %s", c.Loader.Type)
	}

	switch c.Splitter.Type {
	case "", "recursive", "markdown_header":
	default:
		return fmt.Errorf("unsupported splitter type: %s", c.Splitter.Type)
	}

	if c.Splitter.Type == "" || c.Splitter.Type == "recursive" {
		splitter := DefaultSplitterSpec()
		if err := c.Splitter.Spec.Decode(splitter); err != nil {
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
	switch c.Indexer.Type {
	case "dev", "pinecone":
	default:
		return fmt.Errorf("unsupported indexer type: %s", c.Indexer.Type)
	}

	switch c.Retriever.Type {
	case "dev", "pinecone":
	default:
		return fmt.Errorf("unsupported retriever type: %s", c.Retriever.Type)
	}

	if c.Reranker != nil {
		var rerankerSpec RerankerSpec
		if err := c.Reranker.Spec.Decode(&rerankerSpec); err != nil {
			return fmt.Errorf("failed to decode reranker spec: %w", err)
		}
		if rerankerSpec.Enabled {
			switch c.Reranker.Type {
			case "cohere":
			default:
				return fmt.Errorf("unsupported reranker type: %s", c.Reranker.Type)
			}
		}
	}

	return nil
}
