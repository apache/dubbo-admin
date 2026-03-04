package rag

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

// RAG provides runtime-facing document split, index and retrieve operations.
type RAG struct {
	Loader    document.Loader
	Splitter  document.Transformer
	Indexer   indexer.Indexer
	Retriever retriever.Retriever
	Reranker  Reranker
}

func (s *RAG) Split(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
	if s.Splitter == nil {
		return docs, nil
	}
	return s.Splitter.Transform(ctx, docs)
}

func (s *RAG) Index(ctx context.Context, namespace string, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if s.Indexer == nil {
		return nil, fmt.Errorf("indexer is nil")
	}
	if namespace == "" {
		return s.Indexer.Store(ctx, docs, opts...)
	}
	all := append([]indexer.Option{WithIndexerNamespace(namespace)}, opts...)
	return s.Indexer.Store(ctx, docs, all...)
}

func (s *RAG) Retrieve(ctx context.Context, namespace string, queries []string, opts ...RetrieveOption) (map[string][]*RetrieveResult, error) {
	if s.Retriever == nil {
		return nil, fmt.Errorf("retriever is nil")
	}
	if len(queries) == 0 {
		return map[string][]*RetrieveResult{}, nil
	}

	var co CallOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&co)
		}
	}

	retrieveOpts := make([]retriever.Option, 0, 2)
	if co.TopK != nil {
		retrieveOpts = append(retrieveOpts, retriever.WithTopK(*co.TopK))
	}
	if co.TargetIndex != nil && *co.TargetIndex != "" {
		retrieveOpts = append(retrieveOpts, WithRetrieverImplTargetIndex(*co.TargetIndex))
	}
	effectiveNamespace := namespace
	if co.Namespace != nil {
		effectiveNamespace = *co.Namespace
	}
	if effectiveNamespace != "" {
		retrieveOpts = append(retrieveOpts, WithRetrieverImplNamespace(effectiveNamespace))
	}

	resp := make(map[string][]*RetrieveResult, len(queries))
	for _, query := range queries {
		docs, err := s.Retriever.Retrieve(ctx, query, retrieveOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve for query %q: %w", query, err)
		}
		results := make([]*RetrieveResult, 0, len(docs))
		for _, doc := range docs {
			results = append(results, &RetrieveResult{Content: doc.Content, RelevanceScore: 0})
		}
		resp[query] = results
	}

	if s.Reranker == nil {
		return resp, nil
	}

	final := make(map[string][]*RetrieveResult, len(resp))
	for query, raw := range resp {
		docs := make([]*schema.Document, 0, len(raw))
		for _, r := range raw {
			docs = append(docs, &schema.Document{Content: r.Content})
		}
		reranked, err := s.Reranker.Rerank(ctx, query, docs, opts...)
		if err != nil {
			return nil, err
		}
		final[query] = reranked
	}

	return final, nil
}
