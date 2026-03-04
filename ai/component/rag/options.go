package rag

import (
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
)

// RAGOptions defines all per-call options used by rag package.
type RAGOptions struct {
	RetrieveTopK *int
	RerankTopN   *int
	Namespace   string
	TargetIndex *string
	BatchSize   *int
}

type Option func(*RAGOptions)

func WithRetrieveTopK(k int) Option {
	return func(o *RAGOptions) { o.RetrieveTopK = &k }
}

func WithRerankTopN(n int) Option {
	return func(o *RAGOptions) { o.RerankTopN = &n }
}

func WithTargetIndex(index string) Option {
	return func(o *RAGOptions) { o.TargetIndex = &index }
}

func WithRetrieverTargetIndex(index string) Option {
	return WithTargetIndex(index)
}

func WithRetrieverNamespace(namespace string) Option {
	return func(o *RAGOptions) { o.Namespace = namespace }
}

func WithIndexerNamespace(ns string) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *RAGOptions) {
		opts.Namespace = ns
	})
}

func WithIndexerBatchSize(batchSize int) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *RAGOptions) {
		opts.BatchSize = &batchSize
	})
}

func WithIndexerTargetIndex(targetIndex string) indexer.Option {
	return indexer.WrapImplSpecificOptFn(func(opts *RAGOptions) {
		opts.TargetIndex = &targetIndex
	})
}

func WithRetrieverImplNamespace(ns string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RAGOptions) {
		opts.Namespace = ns
	})
}

func WithRetrieverImplTargetIndex(targetIndex string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(opts *RAGOptions) {
		opts.TargetIndex = &targetIndex
	})
}
