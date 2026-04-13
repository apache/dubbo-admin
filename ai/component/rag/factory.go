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
	"dubbo-admin-ai/component/rag/indexers"
	"dubbo-admin-ai/component/rag/loaders"
	"dubbo-admin-ai/component/rag/mergers"
	"dubbo-admin-ai/component/rag/query"
	"dubbo-admin-ai/component/rag/rerankers"
	"dubbo-admin-ai/component/rag/retrievers"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/openai/openai-go"
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
		cfg = &config.Config{Type: loaders.LoaderTypeLocal}
	}
	ctx := context.Background()
	switch cfg.Type {
	case "", loaders.LoaderTypeLocal:
		return loaders.NewLocalFileLoader(ctx)
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
	const batchSize = 100
	switch indexerType {
	case indexers.IndexerTypeLocal:
		return indexers.NewLocalIndexer(g, embedderModel, targetIndex, batchSize), nil
	case indexers.IndexerTypePinecone:
		return indexers.NewPineconeIndexer(g, embedderModel, targetIndex, batchSize), nil
	default:
		return nil, fmt.Errorf("unsupported indexer type: %s", indexerType)
	}
}

// newIndexerWithConfig creates an indexer with full configuration support.
// This is used by RAGComponent which has access to the full config.
func newIndexerWithConfig(g *genkit.Genkit, cfg *config.Config, embedderModel string) (indexer.Indexer, error) {
	const targetIndex = "default"
	const batchSize = 100

	switch cfg.Type {
	case indexers.IndexerTypeLocal:
		return indexers.NewLocalIndexer(g, embedderModel, targetIndex, batchSize), nil
	case indexers.IndexerTypePinecone:
		return indexers.NewPineconeIndexer(g, embedderModel, targetIndex, batchSize), nil
	case indexers.IndexerTypeMilvus:
		var milvusCfg indexers.MilvusConfig
		if err := cfg.Spec.Decode(&milvusCfg); err != nil {
			return nil, fmt.Errorf("failed to decode milvus indexer spec: %w", err)
		}
		milvusCfg.Embedder = embedderModel
		milvusCfg.BatchSize = batchSize
		return indexers.NewMilvusIndexer(g, &milvusCfg)
	default:
		return nil, fmt.Errorf("unsupported indexer type: %s", cfg.Type)
	}
}

func newRetriever(g *genkit.Genkit, retrieverType, embedderModel string) (retriever.Retriever, error) {
	const targetIndex = "default"
	const defaultTopK = 3
	switch retrieverType {
	case retrievers.RetrieverTypeLocal:
		return retrievers.NewLocalRetriever(g, embedderModel, targetIndex, defaultTopK), nil
	case retrievers.RetrieverTypePinecone:
		return retrievers.NewPineconeRetriever(g, embedderModel, targetIndex, defaultTopK), nil
	default:
		return nil, fmt.Errorf("unsupported retriever type: %s", retrieverType)
	}
}

// newRetrieverWithConfig creates a retriever with full configuration support.
// This is used by RAGComponent which has access to the full config.
func newRetrieverWithConfig(g *genkit.Genkit, cfg *config.Config, embedderModel string) (retriever.Retriever, error) {
	const targetIndex = "default"
	const defaultTopK = 3

	switch cfg.Type {
	case retrievers.RetrieverTypeLocal:
		return retrievers.NewLocalRetriever(g, embedderModel, targetIndex, defaultTopK), nil
	case retrievers.RetrieverTypePinecone:
		return retrievers.NewPineconeRetriever(g, embedderModel, targetIndex, defaultTopK), nil
	case retrievers.RetrieverTypeMilvus:
		var milvusCfg retrievers.MilvusConfig
		if err := cfg.Spec.Decode(&milvusCfg); err != nil {
			return nil, fmt.Errorf("failed to decode milvus retriever spec: %w", err)
		}
		milvusCfg.Embedder = embedderModel
		return retrievers.NewMilvusRetriever(g, &milvusCfg)
	default:
		return nil, fmt.Errorf("unsupported retriever type: %s", cfg.Type)
	}
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

	idx, err := newIndexerWithConfig(g, cfg.Indexer, embedderSpec.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}

	rtv, err := newRetrieverWithConfig(g, cfg.Retriever, embedderSpec.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create retriever: %w", err)
	}

	var rr Reranker
	if getRerankerEnabled(cfg.Reranker) {
		rr, err = rerankers.NewCohereReranker(&rerankers.CohereConfig{
			APIKey: getRerankerAPIKey(cfg.Reranker),
			Model:  getRerankerModel(cfg.Reranker),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create reranker: %w", err)
		}
	}

	return &RAG{
		Loader:    loader,
		Splitter:  splitter,
		Indexer:   idx,
		Retriever: rtv,
		Reranker:  rr,
	}, nil
}

// ============= Helper Functions =============

// NewQueryProcessor creates a new query processor using the query package.
func NewQueryProcessor(g *genkit.Genkit, cfg *query.QueryProcessorConfig, promptBasePath string) (QueryProcessor, error) {
	// Use the new query package's factory function
	return query.NewQueryProcessor(g, cfg, promptBasePath)
}

// buildPrompt creates an AI prompt with the given parameters.
func buildPrompt(registry *genkit.Genkit, inType, outType any, tag, prompt string, temp float64, model string, extraPrompt string, tools ...ai.ToolRef) (ai.Prompt, error) {
	opts := []ai.PromptOption{
		ai.WithSystem(prompt),
		ai.WithConfig(&openai.ChatCompletionNewParams{
			Temperature: openai.Float(temp),
		}),
		ai.WithModelName(model),
	}
	if inType != nil {
		opts = append(opts, ai.WithInputType(inType))
	}
	if outType != nil {
		opts = append(opts, ai.WithOutputType(outType))
	}
	if extraPrompt != "" {
		opts = append(opts, ai.WithPrompt(extraPrompt))
	}
	if tools != nil {
		opts = append(opts, ai.WithTools(tools...), ai.WithReturnToolRequests(true))
	}

	return genkit.DefinePrompt(registry, tag, opts...), nil
}

// ============= New API: Build RAG with multi-path and query layer =============

// BuildRAGFromSpecV2 creates a new RAG instance with multi-path retrieval and query understanding.
func BuildRAGFromSpecV2(ctx context.Context, g *genkit.Genkit, cfg *RAGSpec) (*RAG, error) {
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

	// Create basic components
	loader, err := newLoader(cfg.Loader)
	if err != nil {
		return nil, fmt.Errorf("failed to create loader: %w", err)
	}

	splitter, err := newSplitter(cfg.Splitter)
	if err != nil {
		return nil, fmt.Errorf("failed to create splitter: %w", err)
	}

	idx, err := newIndexerWithConfig(g, cfg.Indexer, embedderSpec.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}

	// Create multi-path retrievers
	var retrievalPaths []*RetrievalPath
	var merger *mergers.MergeLayer

	rtv, err := newRetrieverWithConfig(g, cfg.Retriever, embedderSpec.Model)
	if err == nil && rtv != nil {
		// Single path (backward compatible)
		retrievalPaths = append(retrievalPaths, &RetrievalPath{
			Label:    "default",
			Retriever: rtv,
			Weight:   1.0,
		})
	}

	// Create merger if multi-path
	if len(retrievalPaths) > 0 {
		merger = mergers.NewMergeLayer(&mergers.MergeConfig{
			Strategy:        "rrf",
			NormalizeMethod: "minmax",
			EnableDedup:     true,
			TopK:            10,
		})
	}

	// Create query layer
	var queryLayer *query.Layer
	if cfg.QueryProcessor != nil {
		var qpSpec QueryProcessorSpec
		if err := cfg.QueryProcessor.Spec.Decode(&qpSpec); err == nil {
			if qpSpec.Enabled {
				querySpec := &query.LayerSpec{
					Model:       qpSpec.Model,
					Temperature: qpSpec.Temperature,
					Rewrite: &query.StepSpec{
						Enabled:     true,
						Model:       qpSpec.Model,
						Temperature: qpSpec.Temperature,
					},
				}
				promptBasePath := "./prompts"
				queryLayer, err = query.NewLayerFromSpec(g, querySpec, promptBasePath)
				if err != nil {
					return nil, fmt.Errorf("failed to create query layer: %w", err)
				}
			}
		}
	}

	// Create reranker
	var reranker rerankers.Reranker
	if getRerankerEnabled(cfg.Reranker) {
		reranker, err = rerankers.NewCohereReranker(&rerankers.CohereConfig{
			APIKey: getRerankerAPIKey(cfg.Reranker),
			Model:  getRerankerModel(cfg.Reranker),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create reranker: %w", err)
		}
	}

	// Build RAG
	return NewRAG(&Components{
		Loader:         loader,
		Splitter:       splitter,
		Indexer:        idx,
		Retriever:      rtv,
		RetrievalPaths: retrievalPaths,
		Merger:         merger,
		QueryLayer:     queryLayer,
		Reranker:       reranker,
	})
}
