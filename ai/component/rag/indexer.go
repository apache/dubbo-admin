package rag

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
	"github.com/firebase/genkit/go/plugins/localvec"
	"github.com/firebase/genkit/go/plugins/pinecone"
)

// --- Indexer ---
type PineconeIndexer struct {
	g        *genkit.Genkit
	embedder string
	target   string
	batchSz  int
	mu       sync.Mutex
	docstore map[string]*pinecone.Docstore // keyed by target index
}

func newPineconeIndexer(g *genkit.Genkit, embedderModel string, targetIndex string, batchSize int) *PineconeIndexer {
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

	// TODO(indexer, 2026-02-24): Validate namespace if needed for multi-tenancy support
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

// --- DevIndexer ---
type DevIndexer struct {
	g        *genkit.Genkit
	embedder string
	target   string
	batchSz  int
	mu       sync.Mutex
	docstore map[string]*localvec.DocStore // keyed by target index
}

func newDevIndexer(g *genkit.Genkit, embedderModel string, targetIndex string, batchSize int) *DevIndexer {
	return &DevIndexer{
		g:        g,
		embedder: embedderModel,
		target:   targetIndex,
		batchSz:  batchSize,
	}
}

func (idx *DevIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
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

		// Configure localvec with Dev-specific settings
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
