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

package indexers

import (
	"context"
	"dubbo-admin-ai/utils"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/pinecone"
)

// PineconeIndexer provides Pinecone vector storage.
type PineconeIndexer struct {
	g        *genkit.Genkit
	embedder string
	target   string
	batchSz  int
	mu       sync.Mutex
	docstore map[string]*pinecone.Docstore // keyed by target index
}

// NewPineconeIndexer creates a new PineconeIndexer.
func NewPineconeIndexer(g *genkit.Genkit, embedderModel string, targetIndex string, batchSize int) *PineconeIndexer {
	return &PineconeIndexer{
		g:        g,
		embedder: embedderModel,
		target:   targetIndex,
		batchSz:  batchSize,
	}
}

func (idx *PineconeIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	// Handle options
	implOpts := indexer.GetImplSpecificOptions(&CommonIndexerOptions{}, opts...)
	namespace := implOpts.Namespace
	effectiveTarget := idx.target
	if implOpts.TargetIndex != nil && *implOpts.TargetIndex != "" {
		effectiveTarget = *implOpts.TargetIndex
	}

	// Initialize indexer docstore for this target if not already done
	idx.mu.Lock()
	if idx.docstore == nil {
		idx.docstore = make(map[string]*pinecone.Docstore)
	}
	docstore := idx.docstore[effectiveTarget]
	idx.mu.Unlock()
	if docstore == nil {
		embedder := genkit.LookupEmbedder(idx.g, idx.embedder)
		if embedder == nil {
			return nil, fmt.Errorf("failed to find embedder %s", idx.embedder)
		}

		// Configure Pinecone connection
		pineconeConfig := pinecone.Config{
			IndexID:  effectiveTarget,
			Embedder: embedder,
		}

		newDocstore, _, err := pinecone.DefineRetriever(ctx, idx.g,
			pineconeConfig,
			&ai.RetrieverOptions{
				Label:        effectiveTarget,
				ConfigSchema: core.InferSchemaMap(pinecone.PineconeRetrieverOptions{}),
			})
		if err != nil {
			return nil, fmt.Errorf("failed to setup retriever for indexer: %w", err)
		}

		idx.mu.Lock()
		if idx.docstore == nil {
			idx.docstore = make(map[string]*pinecone.Docstore)
		}
		if idx.docstore[effectiveTarget] == nil {
			idx.docstore[effectiveTarget] = newDocstore
		}
		docstore = idx.docstore[effectiveTarget]
		idx.mu.Unlock()
	}

	// Convert to Genkit documents
	genkitDocs := utils.ToGenkitDocuments(docs)

	// Index in batches
	batchSize := idx.batchSz
	if implOpts.BatchSize != nil && *implOpts.BatchSize > 0 {
		batchSize = *implOpts.BatchSize
	}
	if batchSize <= 0 {
		return nil, fmt.Errorf("batch size must be positive")
	}
	for i := 0; i < len(genkitDocs); i += batchSize {
		end := min(i+batchSize, len(genkitDocs))
		batch := genkitDocs[i:end]
		if err := pinecone.Index(ctx, batch, docstore, namespace); err != nil {
			return nil, fmt.Errorf("failed to index documents batch %d-%d: %w", i+1, end, err)
		}
	}

	return nil, nil
}
