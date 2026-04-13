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
	"dubbo-admin-ai/component/rag/rerankers"
	"dubbo-admin-ai/runtime"
	"fmt"
t"dubbo-admin-ai/component/rag/query"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
)

// Reranker is an alias for rerankers.Reranker.
type Reranker = rerankers.Reranker

// ============= 子组件包装器 =============

// loaderComponent Loader 组件包装器
type loaderComponent struct {
	cfg    *config.Config
	loader document.Loader
}

func newLoaderComponent(cfg *config.Config) *loaderComponent {
	return &loaderComponent{cfg: cfg}
}

func (c *loaderComponent) Name() string { return "loader" }
func (c *loaderComponent) Validate() error {
	return nil
}

func (c *loaderComponent) Init(rt *runtime.Runtime) error {
	loader, err := newLoader(c.cfg)
	if err != nil {
		return fmt.Errorf("failed to create loader: %w", err)
	}
	c.loader = loader
	rt.GetLogger().Info("Loader component initialized", "type", c.cfg.Type)
	return nil
}

func (c *loaderComponent) Start() error { return nil }
func (c *loaderComponent) Stop() error  { return nil }
func (c *loaderComponent) get() document.Loader {
	return c.loader
}

// splitterComponent Splitter 组件包装器
type splitterComponent struct {
	cfg      *config.Config
	splitter document.Transformer
}

func newSplitterComponent(cfg *config.Config) *splitterComponent {
	return &splitterComponent{cfg: cfg}
}

func (c *splitterComponent) Name() string { return "splitter" }
func (c *splitterComponent) Validate() error {
	return nil
}

func (c *splitterComponent) Init(rt *runtime.Runtime) error {
	splitter, err := newSplitter(c.cfg)
	if err != nil {
		return fmt.Errorf("failed to create splitter: %w", err)
	}
	c.splitter = splitter

	var spec SplitterSpec
	if c.cfg.Spec.Decode(&spec) == nil {
		rt.GetLogger().Info("Splitter component initialized", "type", c.cfg.Type, "chunk_size", spec.ChunkSize, "overlap_size", spec.OverlapSize)
	} else {
		rt.GetLogger().Info("Splitter component initialized", "type", c.cfg.Type)
	}
	return nil
}

func (c *splitterComponent) Start() error { return nil }
func (c *splitterComponent) Stop() error  { return nil }
func (c *splitterComponent) get() document.Transformer {
	return c.splitter
}

// indexerComponent Indexer 组件包装器
type indexerComponent struct {
	cfg          *config.Config
	embedderName string
	indexer      indexer.Indexer
}

func newIndexerComponent(cfg *config.Config, embedderName string) *indexerComponent {
	return &indexerComponent{cfg: cfg, embedderName: embedderName}
}

func (c *indexerComponent) Name() string { return "indexer" }
func (c *indexerComponent) Validate() error {
	return nil
}

func (c *indexerComponent) Init(rt *runtime.Runtime) error {
	registry := rt.GetGenkitRegistry()
	if registry == nil {
		return fmt.Errorf("genkit registry not initialized")
	}

	idx, err := newIndexerWithConfig(registry, c.cfg, c.embedderName)
	if err != nil {
		return fmt.Errorf("failed to create indexer: %w", err)
	}
	c.indexer = idx

	rt.GetLogger().Info("Indexer component initialized", "type", c.cfg.Type, "embedder", c.embedderName)
	return nil
}

func (c *indexerComponent) Start() error { return nil }
func (c *indexerComponent) Stop() error  { return nil }
func (c *indexerComponent) get() indexer.Indexer {
	return c.indexer
}

// retrieverComponent Retriever 组件包装器
type retrieverComponent struct {
	cfg          *config.Config
	embedderName string
	retriever    retriever.Retriever
}

func newRetrieverComponent(cfg *config.Config, embedderName string) *retrieverComponent {
	return &retrieverComponent{cfg: cfg, embedderName: embedderName}
}

func (c *retrieverComponent) Name() string { return "retriever" }
func (c *retrieverComponent) Validate() error {
	return nil
}

func (c *retrieverComponent) Init(rt *runtime.Runtime) error {
	registry := rt.GetGenkitRegistry()
	if registry == nil {
		return fmt.Errorf("genkit registry not initialized")
	}

	rtv, err := newRetrieverWithConfig(registry, c.cfg, c.embedderName)
	if err != nil {
		return fmt.Errorf("failed to create retriever: %w", err)
	}
	c.retriever = rtv

	rt.GetLogger().Info("Retriever component initialized", "type", c.cfg.Type, "embedder", c.embedderName)
	return nil
}

func (c *retrieverComponent) Start() error { return nil }
func (c *retrieverComponent) Stop() error  { return nil }
func (c *retrieverComponent) get() retriever.Retriever {
	return c.retriever
}

// rerankerComponent Reranker 组件包装器
type rerankerComponent struct {
	enabled  bool
	model    string
	apiKey   string
	reranker Reranker
}

func newRerankerComponent(enabled bool, model, apiKey string) *rerankerComponent {
	return &rerankerComponent{enabled: enabled, model: model, apiKey: apiKey}
}

func (c *rerankerComponent) Name() string { return "reranker" }
func (c *rerankerComponent) Validate() error {
	return nil
}

func (c *rerankerComponent) Init(rt *runtime.Runtime) error {
	if !c.enabled {
		rt.GetLogger().Info("Reranker component disabled")
		return nil
	}

	cfg := &rerankers.CohereConfig{
		APIKey: c.apiKey,
		Model:  c.model,
	}
	reranker, err := rerankers.NewCohereReranker(cfg)
	if err != nil {
		return fmt.Errorf("failed to create reranker: %w", err)
	}
	c.reranker = reranker

	rt.GetLogger().Info("Reranker component initialized", "model", c.model)
	return nil
}

func (c *rerankerComponent) Start() error { return nil }
func (c *rerankerComponent) Stop() error  { return nil }
func (c *rerankerComponent) get() Reranker {
	return c.reranker
}

// queryProcessorComponent QueryProcessor 组件包装器
type queryProcessorComponent struct {
	cfg            *config.Config
	processor      QueryProcessor
	registry       *genkitRegistryHolder
	promptBasePath string
}

type genkitRegistryHolder struct {
	registry interface{}
}

func newQueryProcessorComponent(cfg *config.Config, registry interface{}, promptBasePath string) *queryProcessorComponent {
	return &queryProcessorComponent{
		cfg:            cfg,
		registry:       &genkitRegistryHolder{registry: registry},
		promptBasePath: promptBasePath,
	}
}

func (c *queryProcessorComponent) Name() string { return "query_processor" }
func (c *queryProcessorComponent) Validate() error {
	return nil
}

func (c *queryProcessorComponent) Init(rt *runtime.Runtime) error {
	if c.cfg == nil {
		rt.GetLogger().Info("QueryProcessor component not configured")
		return nil
	}

	var spec QueryProcessorSpec
	if err := c.cfg.Spec.Decode(&spec); err != nil {
		return fmt.Errorf("failed to parse query_processor spec: %w", err)
	}

	// Check if enabled
	if !spec.Enabled {
		rt.GetLogger().Info("QueryProcessor component disabled")
		return nil
	}

	// Build configuration
	cfg := &QueryProcessorConfig{
		Model:           spec.Model,
		Timeout:         spec.Timeout,
		Temperature:     spec.Temperature,
		FallbackOnError: spec.FallbackOnError,
		Enabled:         spec.Enabled,
	}

	// Get the genkit registry from runtime
	g := rt.GetRegistry()
	if g == nil {
		return fmt.Errorf("genkit registry not initialized")
	}

	// Create processor
	processor, err := NewQueryProcessor(g, cfg, c.promptBasePath)
	if err != nil {
		return fmt.Errorf("failed to create query processor: %w", err)
	}
	c.processor = processor

	rt.GetLogger().Info("QueryProcessor component initialized",
		"model", spec.Model, "timeout", spec.Timeout)

	return nil
}

func (c *queryProcessorComponent) Start() error { return nil }
func (c *queryProcessorComponent) Stop() error  { return nil }
func (c *queryProcessorComponent) get() QueryProcessor {
	return c.processor
}

// ============= RAGComponent 主组件 =============

// RAGComponent RAG 系统组件
type RAGComponent struct {
	cfg            *RAGSpec
	embedderName   string
	loader         *loaderComponent
	splitter       *splitterComponent
	indexer        *indexerComponent
	retriever      *retrieverComponent
	reranker       *rerankerComponent
	queryProcessor *queryProcessorComponent
	promptBasePath string

	// Rag is the RAG instance created after Init
	Rag *RAG
}

func (r *RAGComponent) Name() string {
	return "rag"
}

func (r *RAGComponent) Validate() error {
	return r.cfg.Validate()
}

func (r *RAGComponent) Init(rt *runtime.Runtime) error {
	// 获取 embedder 模型名称
	var embedderSpec EmbedderSpec
	if err := r.cfg.Embedder.Spec.Decode(&embedderSpec); err != nil {
		return fmt.Errorf("failed to parse embedder spec: %w", err)
	}
	r.embedderName = embedderSpec.Model

	// 设置 prompt 基础路径
	if r.promptBasePath == "" {
		r.promptBasePath = "./prompts"
	}

	// 创建子组件
	r.loader = newLoaderComponent(r.cfg.Loader)
	r.splitter = newSplitterComponent(r.cfg.Splitter)
	r.indexer = newIndexerComponent(r.cfg.Indexer, r.embedderName)
	r.retriever = newRetrieverComponent(r.cfg.Retriever, r.embedderName)
	r.reranker = newRerankerComponent(
		getRerankerEnabled(r.cfg.Reranker),
		getRerankerModel(r.cfg.Reranker),
		getRerankerAPIKey(r.cfg.Reranker),
	)

	// 初始化所有子组件
	components := []runtime.Component{r.loader, r.splitter, r.indexer, r.retriever, r.reranker}
	for _, comp := range components {
		if err := comp.Init(rt); err != nil {
			return fmt.Errorf("failed to init %s: %w", comp.Name(), err)
		}
	}

	// 初始化 QueryProcessor (需要 registry，放在最后)
	if r.cfg.QueryProcessor != nil {
		r.queryProcessor = newQueryProcessorComponent(r.cfg.QueryProcessor, rt.GetRegistry(), r.promptBasePath)
		if err := r.queryProcessor.Init(rt); err != nil {
			return fmt.Errorf("failed to init query_processor: %w", err)
		}
	}

	rt.GetLogger().Info("RAG component initialized",
		"embedder", r.embedderName,
		"indexer", r.cfg.Indexer.Type,
		"retriever", r.cfg.Retriever.Type,
		"splitter", r.cfg.Splitter.Type,
		"reranker_enabled", r.cfg.Reranker != nil,
		"query_processor_enabled", r.queryProcessor != nil && r.queryProcessor.get() != nil)
	// Extract QueryLayer from query processor if available
	var queryLayer *t.Layer
	if r.queryProcessor != nil {
		if qp := r.queryProcessor.get(); qp != nil {
			// Try to get the Layer from LegacyProcessor
			type legacyProcessor interface {
				GetLayer() *t.Layer
			}
			if lp, ok := qp.(legacyProcessor); ok {
				queryLayer = lp.GetLayer()
			}
		}
	}


	// Create RAG instance with logger
	r.Rag = &RAG{
		QueryLayer: queryLayer,
		Splitter:  r.splitter.get(),
		Indexer:   r.indexer.get(),
		Retriever: r.retriever.get(),
		Reranker:  r.reranker.get(),
		logger:    rt.GetLogger(),
	}

	return nil
}

func (r *RAGComponent) Start() error {
	components := []runtime.Component{r.loader, r.splitter, r.indexer, r.retriever, r.reranker}
	for _, comp := range components {
		if err := comp.Start(); err != nil {
			return fmt.Errorf("failed to start %s: %w", comp.Name(), err)
		}
	}
	if r.queryProcessor != nil {
		if err := r.queryProcessor.Start(); err != nil {
			return fmt.Errorf("failed to start query_processor: %w", err)
		}
	}
	return nil
}

func (r *RAGComponent) Stop() error {
	components := []runtime.Component{r.reranker, r.retriever, r.indexer, r.splitter, r.loader}
	for _, comp := range components {
		if err := comp.Stop(); err != nil {
			return fmt.Errorf("failed to stop %s: %w", comp.Name(), err)
		}
	}
	if r.queryProcessor != nil {
		if err := r.queryProcessor.Stop(); err != nil {
			return fmt.Errorf("failed to stop query_processor: %w", err)
		}
	}
	return nil
}

// Getter 方法
func (r *RAGComponent) GetLoader() document.Loader {
	if r.loader != nil {
		return r.loader.get()
	}
	return nil
}

func (r *RAGComponent) GetSplitter() document.Transformer {
	if r.splitter != nil {
		return r.splitter.get()
	}
	return nil
}

func (r *RAGComponent) GetIndexer() indexer.Indexer {
	if r.indexer != nil {
		return r.indexer.get()
	}
	return nil
}

func (r *RAGComponent) GetRetriever() retriever.Retriever {
	if r.retriever != nil {
		return r.retriever.get()
	}
	return nil
}

func (r *RAGComponent) GetReranker() Reranker {
	if r.reranker != nil {
		return r.reranker.get()
	}
	return nil
}

func (r *RAGComponent) GetEmbedderName() string {
	return r.embedderName
}

func (r *RAGComponent) GetQueryProcessor() QueryProcessor {
	if r.queryProcessor != nil {
		return r.queryProcessor.get()
	}
	return nil
}

// GetRAG returns the RAG instance for direct method access
func (r *RAGComponent) GetRAG() *RAG {
	return r.Rag
}

// ============= 辅助函数 =============

func getRerankerEnabled(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	var spec RerankerSpec
	if err := cfg.Spec.Decode(&spec); err != nil {
		return false
	}
	return spec.Enabled
}

func getRerankerModel(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	var spec RerankerSpec
	if err := cfg.Spec.Decode(&spec); err != nil {
		return ""
	}
	return spec.Model
}

func getRerankerAPIKey(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	var spec RerankerSpec
	if err := cfg.Spec.Decode(&spec); err != nil {
		return ""
	}
	return spec.APIKey
}
