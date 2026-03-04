package rag

import (
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
)

// RetrieveOption defines per-call retrieve/rerank options.
type RetrieveOption = RerankOption

func WithTopK(topK int) RetrieveOption {
	return func(o *CallOptions) { o.TopK = &topK }
}

func WithTopN(topN int) RetrieveOption {
	return func(o *CallOptions) { o.TopN = &topN }
}

func WithTargetIndex(index string) RetrieveOption {
	return func(o *CallOptions) { o.TargetIndex = &index }
}

func WithRetrieverTargetIndex(index string) RetrieveOption {
	return WithTargetIndex(index)
}

func WithRetrieverNamespace(namespace string) RetrieveOption {
	return func(o *CallOptions) { o.Namespace = &namespace }
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
