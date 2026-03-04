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
	"dubbo-admin-ai/runtime"
	"fmt"

	"gopkg.in/yaml.v3"
)

type ragAssemblySpec struct {
	embedderModel string

	loaderType string

	splitterType      string
	splitterChunkSize int
	splitterOverlap   int
	markdownHeaders   map[string]string
	markdownTrim      bool

	indexerType        string
	indexerTargetIndex string
	indexerBatchSize   int

	retrieverType        string
	retrieverTargetIndex string
	retrieverDefaultTopK int

	rerankerType    string
	rerankerEnabled bool
	rerankerModel   string
	rerankerAPIKey  string
}

func decodeRAGAssemblySpec(cfg *RAGSpec) (*ragAssemblySpec, error) {
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

	assembly := &ragAssemblySpec{
		embedderModel:        embedderSpec.Model,
		loaderType:           cfg.Loader.Type,
		splitterType:         cfg.Splitter.Type,
		indexerType:          cfg.Indexer.Type,
		indexerTargetIndex:   DefaultIndexerTargetIndex,
		indexerBatchSize:     DefaultIndexerBatchSize,
		retrieverType:        cfg.Retriever.Type,
		retrieverTargetIndex: DefaultRetrieverTargetIndex,
		retrieverDefaultTopK: DefaultRetrieverTopK,
		rerankerModel:        DefaultRerankerModel,
	}

	if assembly.loaderType == "" {
		assembly.loaderType = "local"
	}

	if assembly.splitterType == "" {
		assembly.splitterType = "recursive"
	}
	switch assembly.splitterType {
	case "recursive":
		splitterSpec := DefaultSplitterSpec()
		if err := cfg.Splitter.Spec.Decode(splitterSpec); err != nil {
			return nil, fmt.Errorf("failed to decode recursive splitter spec: %w", err)
		}
		assembly.splitterChunkSize = splitterSpec.ChunkSize
		assembly.splitterOverlap = splitterSpec.OverlapSize
	case "markdown_header":
		var markdownSpec MarkdownHeaderSplitterSpec
		if err := cfg.Splitter.Spec.Decode(&markdownSpec); err != nil {
			return nil, fmt.Errorf("failed to decode markdown splitter spec: %w", err)
		}
		assembly.markdownHeaders = markdownSpec.Headers
		assembly.markdownTrim = markdownSpec.TrimHeaders
	}

	if cfg.Reranker != nil {
		assembly.rerankerType = cfg.Reranker.Type
		var rerankerSpec RerankerSpec
		if err := cfg.Reranker.Spec.Decode(&rerankerSpec); err != nil {
			return nil, fmt.Errorf("failed to decode reranker spec: %w", err)
		}
		assembly.rerankerEnabled = rerankerSpec.Enabled
		if rerankerSpec.Model != "" {
			assembly.rerankerModel = rerankerSpec.Model
		}
		assembly.rerankerAPIKey = rerankerSpec.APIKey
	}

	return assembly, nil
}

func buildRAGComponentFromAssembly(assembly *ragAssemblySpec) (runtime.Component, error) {
	loaderComp, err := NewLoaderComponent(assembly.loaderType)
	if err != nil {
		return nil, err
	}
	splitterComp, err := NewSplitterComponent(
		assembly.splitterType,
		assembly.splitterChunkSize,
		assembly.splitterOverlap,
		assembly.markdownHeaders,
		assembly.markdownTrim,
	)
	if err != nil {
		return nil, err
	}
	indexerComp, err := NewIndexerComponent(
		assembly.indexerType,
		assembly.embedderModel,
		assembly.indexerTargetIndex,
		assembly.indexerBatchSize,
	)
	if err != nil {
		return nil, err
	}
	retrieverComp, err := NewRetrieverComponent(
		assembly.retrieverType,
		assembly.embedderModel,
		assembly.retrieverTargetIndex,
		assembly.retrieverDefaultTopK,
	)
	if err != nil {
		return nil, err
	}
	rerankerComp, err := NewRerankerComponent(
		assembly.rerankerType,
		assembly.rerankerEnabled,
		assembly.rerankerModel,
		assembly.rerankerAPIKey,
	)
	if err != nil {
		return nil, err
	}

	return NewRAGComponent(
		assembly.embedderModel,
		loaderComp,
		splitterComp,
		indexerComp,
		retrieverComp,
		rerankerComp,
	)
}

func RAGFactory(spec *yaml.Node) (runtime.Component, error) {
	var cfg RAGSpec
	if err := spec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode rag spec: %w", err)
	}

	assembly, err := decodeRAGAssemblySpec(&cfg)
	if err != nil {
		return nil, err
	}

	return buildRAGComponentFromAssembly(assembly)
}
