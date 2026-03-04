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

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
)

// Reranker 重排序器接口
type Reranker interface {
	Rerank(ctx context.Context, query string, docs any, opts ...RerankOption) ([]*RetrieveResult, error)
}

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

	idx, err := newIndexer(registry, c.cfg.Type, c.embedderName)
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

	rtv, err := newRetriever(registry, c.cfg.Type, c.embedderName)
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

	reranker, err := newReranker(c.enabled, c.model, c.apiKey)
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

// ============= RAGComponent 主组件 =============

// RAGComponent RAG 系统组件
type RAGComponent struct {
	cfg          *RAGSpec
	embedderName string
	loader       *loaderComponent
	splitter     *splitterComponent
	indexer      *indexerComponent
	retriever    *retrieverComponent
	reranker     *rerankerComponent
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

	rt.GetLogger().Info("RAG component initialized",
		"embedder", r.embedderName,
		"indexer", r.cfg.Indexer.Type,
		"retriever", r.cfg.Retriever.Type,
		"splitter", r.cfg.Splitter.Type,
		"reranker_enabled", r.cfg.Reranker != nil)

	return nil
}

func (r *RAGComponent) Start() error {
	components := []runtime.Component{r.loader, r.splitter, r.indexer, r.retriever, r.reranker}
	for _, comp := range components {
		if err := comp.Start(); err != nil {
			return fmt.Errorf("failed to start %s: %w", comp.Name(), err)
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
