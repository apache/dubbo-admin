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
)

// RAGComponent RAG 系统组件
type RAGComponent struct {
	instanceName string
	Rag *RAG

	embedderModel   string
	loaderType      string
	splitterType    string
	indexerType     string
	retrieverType   string
	rerankerEnabled bool

	loaderComp    *loaderComponent
	splitterComp  *splitterComponent
	indexerComp   *indexerComponent
	retrieverComp *retrieverComponent
	rerankerComp  *rerankerComponent
}

func NewRAGComponent(
	embedderModel string,
	loader runtime.Component,
	splitter runtime.Component,
	indexer runtime.Component,
	retriever runtime.Component,
	reranker runtime.Component,
) (runtime.Component, error) {
	loaderComp, ok := loader.(*loaderComponent)
	if !ok {
		return nil, fmt.Errorf("invalid loader component type")
	}
	splitterComp, ok := splitter.(*splitterComponent)
	if !ok {
		return nil, fmt.Errorf("invalid splitter component type")
	}
	indexerComp, ok := indexer.(*indexerComponent)
	if !ok {
		return nil, fmt.Errorf("invalid indexer component type")
	}
	retrieverComp, ok := retriever.(*retrieverComponent)
	if !ok {
		return nil, fmt.Errorf("invalid retriever component type")
	}
	rerankerComp, ok := reranker.(*rerankerComponent)
	if !ok {
		return nil, fmt.Errorf("invalid reranker component type")
	}

	return &RAGComponent{
		embedderModel:   embedderModel,
		loaderType:      loaderComp.loaderType,
		splitterType:    splitterComp.splitterType,
		indexerType:     indexerComp.indexerType,
		retrieverType:   retrieverComp.retrieverType,
		rerankerEnabled: rerankerComp.enabled,
		loaderComp:      loaderComp,
		splitterComp:    splitterComp,
		indexerComp:     indexerComp,
		retrieverComp:   retrieverComp,
		rerankerComp:    rerankerComp,
	}, nil
}

func (r *RAGComponent) Name() string {
	if r.instanceName != "" {
		return r.instanceName
	}
	return "rag"
}

func (r *RAGComponent) SetName(name string) {
	r.instanceName = name
}

func (r *RAGComponent) Validate() error {
	if r.loaderComp == nil {
		return fmt.Errorf("loader component is required")
	}
	if r.splitterComp == nil {
		return fmt.Errorf("splitter component is required")
	}
	if r.indexerComp == nil {
		return fmt.Errorf("indexer component is required")
	}
	if r.retrieverComp == nil {
		return fmt.Errorf("retriever component is required")
	}
	if r.rerankerComp == nil {
		return fmt.Errorf("reranker component is required")
	}
	if r.embedderModel == "" {
		return fmt.Errorf("embedder model is required")
	}
	return nil
}

func (r *RAGComponent) Init(rt *runtime.Runtime) error {
	components := []runtime.Component{r.loaderComp, r.splitterComp, r.indexerComp, r.retrieverComp, r.rerankerComp}
	for _, comp := range components {
		if err := comp.Init(rt); err != nil {
			return fmt.Errorf("failed to init %s: %w", comp.Name(), err)
		}
	}

	r.Rag = &RAG{
		Loader:    r.loaderComp.get(),
		Splitter:  r.splitterComp.get(),
		Indexer:   r.indexerComp.get(),
		Retriever: r.retrieverComp.get(),
		Reranker:  r.rerankerComp.get(),
	}

	rt.GetLogger().Info("RAG component initialized",
		"embedder", r.embedderModel,
		"indexer", r.indexerType,
		"retriever", r.retrieverType,
		"splitter", r.splitterType,
		"reranker_enabled", r.rerankerEnabled)

	return nil
}

func (r *RAGComponent) Start() error {
	return nil
}

func (r *RAGComponent) Stop() error {
	return nil
}
