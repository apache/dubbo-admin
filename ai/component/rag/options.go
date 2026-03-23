package rag

import (
	"dubbo-admin-ai/component/rag/rerankers"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
)

// RetrieveOption defines per-call retrieve/rerank options.
// This is an alias for rerankers.Option for backward compatibility.
type RetrieveOption = rerankers.Option

// RerankOption is an alias for rerankers.Option.
type RerankOption = rerankers.Option

// Option is an alias for RetrieveOption for backward compatibility.
type Option = RetrieveOption

// RetrieveOptions holds RAG-specific retrieve options.
// These options are separate from reranker options.
type RetrieveOptions struct {
	TopK        int
	Namespace   string
	TargetIndex string
}

// RetrieveOptionFunc is a function that configures retrieve options.
type RetrieveOptionFunc func(*RetrieveOptions)

// WithRetrieveTopK sets the TopK for retrieval.
func WithRetrieveTopK(topK int) RetrieveOptionFunc {
	return func(o *RetrieveOptions) {
		o.TopK = topK
	}
}

// WithRetrieveNamespace sets the namespace for retrieval.
func WithRetrieveNamespace(namespace string) RetrieveOptionFunc {
	return func(o *RetrieveOptions) {
		o.Namespace = namespace
	}
}

// WithRetrieveTargetIndex sets the target index for retrieval.
func WithRetrieveTargetIndex(index string) RetrieveOptionFunc {
	return func(o *RetrieveOptions) {
		o.TargetIndex = index
	}
}

// ========== Legacy aliases for reranker options ==========

// WithTopN sets the TopN for reranking (reranker option).
func WithTopN(topN int) RetrieveOption {
	return rerankers.WithTopN(topN)
}

// WithRerankTopN is an alias for WithTopN for backward compatibility.
func WithRerankTopN(topN int) RetrieveOption {
	return WithTopN(topN)
}

// WithTargetIndex is an alias for WithRetrieveTargetIndex for backward compatibility.
func WithTargetIndex(index string) RetrieveOptionFunc {
	return WithRetrieveTargetIndex(index)
}

// ========== Legacy function names (deprecated) ==========
// These are aliases for backward compatibility with existing code.

// WithRetrieverTargetIndex is deprecated; use WithRetrieveTargetIndex instead.
// Kept for backward compatibility.
func WithRetrieverTargetIndex(index string) RetrieveOptionFunc {
	return WithRetrieveTargetIndex(index)
}

// WithRetrieverNamespace is deprecated; use WithRetrieveNamespace instead.
// Kept for backward compatibility.
func WithRetrieverNamespace(namespace string) RetrieveOptionFunc {
	return WithRetrieveNamespace(namespace)
}

// CommonIndexerOptions are per-call indexing options.
type CommonIndexerOptions struct {
	Namespace   string
	BatchSize   *int
	TargetIndex *string
}

func WithIndexerNamespace(ns string) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *CommonIndexerOptions) {
		opts.Namespace = ns
	})
}

func WithIndexerBatchSize(batchSize int) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *CommonIndexerOptions) {
		opts.BatchSize = &batchSize
	})
}

func WithIndexerTargetIndex(targetIndex string) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *CommonIndexerOptions) {
		opts.TargetIndex = &targetIndex
	})
}

// CommonRetrieverOptions are per-call retrieval options.
type CommonRetrieverOptions struct {
	Namespace   string
	TargetIndex *string
}

func WithRetrieverImplNamespace(ns string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *CommonRetrieverOptions) {
		opts.Namespace = ns
	})
}

func WithRetrieverImplTargetIndex(targetIndex string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *CommonRetrieverOptions) {
		opts.TargetIndex = &targetIndex
	})
}
