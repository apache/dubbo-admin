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
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/localvec"
)

// Indexer types supported
const (
	IndexerTypeLocal   = "local"
	IndexerTypePinecone = "pinecone"
	// IndexerTypeMilvus is defined in milvus_const.go with build tag
)

// CommonIndexerOptions are per-call indexing options.
type CommonIndexerOptions struct {
	Namespace   string
	BatchSize   *int
	TargetIndex *string
}

// LocalIndexer provides local vector storage using localvec.
type LocalIndexer struct {
	g        *genkit.Genkit
	embedder string
	target   string
	batchSz  int
	mu       sync.Mutex
	docstore map[string]*localvec.DocStore // keyed by target index
}

// NewLocalIndexer creates a new LocalIndexer.
func NewLocalIndexer(g *genkit.Genkit, embedderModel string, targetIndex string, batchSize int) *LocalIndexer {
	return &LocalIndexer{
		g:        g,
		embedder: embedderModel,
		target:   targetIndex,
		batchSz:  batchSize,
	}
}

func (idx *LocalIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	implOpts := indexer.GetImplSpecificOptions(&CommonIndexerOptions{}, opts...)
	_ = implOpts.Namespace
	effectiveTarget := idx.target
	if implOpts.TargetIndex != nil && *implOpts.TargetIndex != "" {
		effectiveTarget = *implOpts.TargetIndex
	}

	// Initialize indexer docstore for this target if not already done
	idx.mu.Lock()
	if idx.docstore == nil {
		idx.docstore = make(map[string]*localvec.DocStore)
	}
	docstore := idx.docstore[effectiveTarget]
	idx.mu.Unlock()
	if docstore == nil {
		embedder := genkit.LookupEmbedder(idx.g, idx.embedder)
		if embedder == nil {
			return nil, fmt.Errorf("failed to find embedder %s", idx.embedder)
		}

		// Initialize localvec if needed (idempotent)
		if err := localvec.Init(); err != nil {
			return nil, fmt.Errorf("failed to init localvec: %w", err)
		}

		// Configure localvec with Local-specific settings
		localvecConfig := localvec.Config{
			Embedder: embedder,
		}

		var err error
		docstore, _, err = localvec.DefineRetriever(idx.g, effectiveTarget, localvecConfig, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to define localvec retriever: %w", err)
		}

		idx.mu.Lock()
		if idx.docstore == nil {
			idx.docstore = make(map[string]*localvec.DocStore)
		}
		if existing := idx.docstore[effectiveTarget]; existing != nil {
			docstore = existing
		} else {
			idx.docstore[effectiveTarget] = docstore
		}
		idx.mu.Unlock()
	}

	// Convert to Genkit documents
	genkitDocs := utils.ToGenkitDocuments(docs)

	// Index documents in batches
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
		if err := localvec.Index(ctx, batch, docstore); err != nil {
			return nil, fmt.Errorf("failed to index documents batch %d-%d: %w", i+1, end, err)
		}
	}

	// Return IDs (localvec doesn't return IDs on Index, so we extract from docs)
	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}
	return ids, nil
}
