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
	"context"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/runtime"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/firebase/genkit/go/genkit"
	"gopkg.in/yaml.v3"
)

// RAGFactory creates a RAG component from configuration.
func RAGFactory(spec *yaml.Node) (runtime.Component, error) {
	var cfg RAGSpec
	if err := spec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode rag spec: %w", err)
	}
	return &RAGComponent{cfg: &cfg}, nil
}

// ============= 创建函数 =============

func newLoader(cfg *config.Config) (document.Loader, error) {
	if cfg == nil {
		cfg = &config.Config{Type: "local"}
	}
	ctx := context.Background()
	switch cfg.Type {
	case "", "local":
		return newLocalFileLoader(ctx)
	default:
		return nil, fmt.Errorf("unsupported loader type: %s", cfg.Type)
	}
}

func newSplitter(cfg *config.Config) (document.Transformer, error) {
	if cfg == nil {
		cfg = &config.Config{Type: "recursive"}
	}
	ctx := context.Background()
	switch cfg.Type {
	case "markdown_header":
		var spec struct {
			Headers     map[string]string `yaml:"headers"`
			TrimHeaders bool              `yaml:"trim_headers"`
		}
		if err := cfg.Spec.Decode(&spec); err != nil {
			return nil, fmt.Errorf("failed to decode markdown splitter spec: %w", err)
		}
		headers := spec.Headers
		if len(headers) == 0 {
			headers = map[string]string{"#": "h1", "##": "h2", "###": "h3", "####": "h4"}
		}
		return markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{Headers: headers, TrimHeaders: spec.TrimHeaders})
	case "", "recursive":
		var spec SplitterSpec
		if err := cfg.Spec.Decode(&spec); err != nil {
			spec = *DefaultSplitterSpec()
		}
		chunkSize := spec.ChunkSize
		if chunkSize <= 0 {
			chunkSize = 1000
		}
		overlap := spec.OverlapSize
		if overlap <= 0 {
			overlap = 100
		}
		return recursive.NewSplitter(ctx, &recursive.Config{ChunkSize: chunkSize, OverlapSize: overlap})
	default:
		return nil, fmt.Errorf("unsupported splitter type: %s", cfg.Type)
	}
}

func newIndexer(g *genkit.Genkit, indexerType, embedderModel string) (indexer.Indexer, error) {
	const targetIndex = "default"
	switch indexerType {
	case "dev":
		return newDevIndexer(g, embedderModel, targetIndex, 100), nil
	case "pinecone":
		return newPineconeIndexer(g, embedderModel, targetIndex, 100), nil
	default:
		return nil, fmt.Errorf("unsupported indexer type: %s", indexerType)
	}
}

func newRetriever(g *genkit.Genkit, retrieverType, embedderModel string) (retriever.Retriever, error) {
	const targetIndex = "default"
	const defaultTopK = 3
	switch retrieverType {
	case "dev":
		return newDevRetriever(g, embedderModel, targetIndex, defaultTopK), nil
	case "pinecone":
		return newPineconeRetriever(g, embedderModel, targetIndex, defaultTopK), nil
	default:
		return nil, fmt.Errorf("unsupported retriever type: %s", retrieverType)
	}
}

func newReranker(enabled bool, model, apiKey string) (Reranker, error) {
	if !enabled {
		return nil, nil
	}
	if model == "" {
		model = "rerank-english-v3.0"
	}
	return &cohereReranker{cfg: &cohereRerankerConfig{APIKey: apiKey, Model: model, TopN: 3}}, nil
}

// BuildRAGFromSpec 创建独立 RAG 实例（用于 CLI/工具）
func BuildRAGFromSpec(ctx context.Context, g *genkit.Genkit, cfg *RAGSpec) (*RAG, error) {
	if g == nil {
		return nil, fmt.Errorf("genkit registry is nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("rag config is nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var embedderSpec EmbedderSpec
	if err := cfg.Embedder.Spec.Decode(&embedderSpec); err != nil {
		return nil, fmt.Errorf("failed to parse embedder spec: %w", err)
	}

	loader, err := newLoader(cfg.Loader)
	if err != nil {
		return nil, fmt.Errorf("failed to create loader: %w", err)
	}

	splitter, err := newSplitter(cfg.Splitter)
	if err != nil {
		return nil, fmt.Errorf("failed to create splitter: %w", err)
	}

	idx, err := newIndexer(g, cfg.Indexer.Type, embedderSpec.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}

	rtv, err := newRetriever(g, cfg.Retriever.Type, embedderSpec.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create retriever: %w", err)
	}

	rr, err := newReranker(getRerankerEnabled(cfg.Reranker), getRerankerModel(cfg.Reranker), getRerankerAPIKey(cfg.Reranker))
	if err != nil {
		return nil, fmt.Errorf("failed to create reranker: %w", err)
	}

	return &RAG{
		Loader:    loader,
		Splitter:  splitter,
		Indexer:   idx,
		Retriever: rtv,
		Reranker:  rr,
	}, nil
}
