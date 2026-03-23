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

package retrievers

import (
	"context"
	"dubbo-admin-ai/utils"
	"fmt"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/pinecone"
)

// PineconeRetriever provides Pinecone vector retrieval.
type PineconeRetriever struct {
	g         *genkit.Genkit
	embedder  string
	target    string
	defaultK  int
	retriever map[string]ai.Retriever // keyed by target index
}

// NewPineconeRetriever creates a new PineconeRetriever.
func NewPineconeRetriever(g *genkit.Genkit, embedderModel string, targetIndex string, topK int) *PineconeRetriever {
	return &PineconeRetriever{
		g:        g,
		embedder: embedderModel,
		target:   targetIndex,
		defaultK: topK,
	}
}

func (r *PineconeRetriever) getRetriever(ctx context.Context, targetIndex string) (ai.Retriever, error) {
	if targetIndex == "" {
		targetIndex = "default"
	}

	if r.retriever == nil {
		r.retriever = make(map[string]ai.Retriever)
	}
	ret := r.retriever[targetIndex]
	if ret != nil {
		return ret, nil
	}

	embedder := genkit.LookupEmbedder(r.g, r.embedder)
	if embedder == nil {
		return nil, fmt.Errorf("failed to find embedder %s", r.embedder)
	}

	var err error
	if !pinecone.IsDefinedRetriever(r.g, targetIndex) {
		_, ret, err = pinecone.DefineRetriever(ctx, r.g,
			pinecone.Config{
				IndexID:  targetIndex,
				Embedder: embedder,
			},
			&ai.RetrieverOptions{
				Label:        targetIndex,
				ConfigSchema: core.InferSchemaMap(pinecone.PineconeRetrieverOptions{}),
			})
	} else {
		ret = pinecone.Retriever(r.g, targetIndex)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to define retriever: %w", err)
	}

	if r.retriever == nil {
		r.retriever = make(map[string]ai.Retriever)
	}
	if existing := r.retriever[targetIndex]; existing != nil {
		ret = existing
	} else {
		r.retriever[targetIndex] = ret
	}

	return ret, nil
}

func (r *PineconeRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	impl := retriever.GetImplSpecificOptions(&CommonRetrieverOptions{}, opts...)
	effectiveTarget := r.target
	if impl.TargetIndex != nil && *impl.TargetIndex != "" {
		effectiveTarget = *impl.TargetIndex
	}
	ret, err := r.getRetriever(ctx, effectiveTarget)
	if err != nil {
		return nil, err
	}

	// Options handling
	// Default options
	defaultK := r.defaultK
	pineconeOpts := &pinecone.PineconeRetrieverOptions{
		K: defaultK, // Default TopK
	}

	// Apply Eino common options
	commonOpts := retriever.GetCommonOptions(&retriever.Options{
		TopK: &defaultK,
	}, opts...)

	if commonOpts.TopK != nil {
		pineconeOpts.K = *commonOpts.TopK
	}

	// Apply implementation specific options (for Namespace)
	if impl.Namespace != "" {
		pineconeOpts.Namespace = impl.Namespace
	}

	// Retrieve
	resp, err := ret.Retrieve(ctx, &ai.RetrieverRequest{
		Query:   ai.DocumentFromText(query, nil),
		Options: pineconeOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve: %w", err)
	}

	docs := utils.ToEinoDocuments(resp.Documents)

	return docs, nil
}
