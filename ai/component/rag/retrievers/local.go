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
	"sync"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/localvec"
)

// Retriever types supported
const (
	RetrieverTypeLocal   = "local"
	RetrieverTypePinecone = "pinecone"
	// RetrieverTypeMilvus is defined in milvus_const.go with build tag
)

// CommonRetrieverOptions are per-call retrieval options.
type CommonRetrieverOptions struct {
	Namespace   string
	TargetIndex *string
}

// LocalRetriever provides local vector retrieval using localvec.
type LocalRetriever struct {
	g         *genkit.Genkit
	embedder  string
	target    string
	defaultK  int
	mu        sync.Mutex
	retriever map[string]ai.Retriever // keyed by target index
}

// NewLocalRetriever creates a new LocalRetriever.
func NewLocalRetriever(g *genkit.Genkit, embedderModel string, targetIndex string, topK int) *LocalRetriever {
	return &LocalRetriever{
		g:        g,
		embedder: embedderModel,
		target:   targetIndex,
		defaultK: topK,
	}
}

func (r *LocalRetriever) getRetriever(ctx context.Context, targetIndex string) (ai.Retriever, error) {
	if targetIndex == "" {
		targetIndex = "default"
	}

	r.mu.Lock()
	if r.retriever == nil {
		r.retriever = make(map[string]ai.Retriever)
	}
	ret := r.retriever[targetIndex]
	r.mu.Unlock()
	if ret != nil {
		return ret, nil
	}

	embedder := genkit.LookupEmbedder(r.g, r.embedder)
	if embedder == nil {
		return nil, fmt.Errorf("failed to find embedder %s", r.embedder)
	}

	if err := localvec.Init(); err != nil {
		return nil, fmt.Errorf("failed to init localvec: %w", err)
	}

	localvecConfig := localvec.Config{Embedder: embedder}

	var err error
	if localvec.IsDefinedRetriever(r.g, targetIndex) {
		ret = localvec.Retriever(r.g, targetIndex)
	} else {
		_, ret, err = localvec.DefineRetriever(r.g, targetIndex, localvecConfig, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to define localvec retriever: %w", err)
	}

	r.mu.Lock()
	if r.retriever == nil {
		r.retriever = make(map[string]ai.Retriever)
	}
	if existing := r.retriever[targetIndex]; existing != nil {
		ret = existing
	} else {
		r.retriever[targetIndex] = ret
	}
	r.mu.Unlock()

	return ret, nil
}

func (r *LocalRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
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
	defaultK := r.defaultK
	// Apply Eino common options
	commonOpts := retriever.GetCommonOptions(&retriever.Options{
		TopK: &defaultK,
	}, opts...)

	k := defaultK
	if commonOpts.TopK != nil {
		k = *commonOpts.TopK
	}

	// Retrieve
	retrieverReq := &ai.RetrieverRequest{
		Query: ai.DocumentFromText(query, nil),
		Options: &localvec.RetrieverOptions{
			K: k,
		},
	}
	resp, err := ret.Retrieve(ctx, retrieverReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve: %w", err)
	}

	docs := utils.ToEinoDocuments(resp.Documents)

	return docs, nil
}
