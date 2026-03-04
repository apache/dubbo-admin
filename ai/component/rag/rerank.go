package rag

import (
	"context"
	
	"fmt"
	"os"

	"github.com/cloudwego/eino/schema"
	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereClient "github.com/cohere-ai/cohere-go/v2/client"
)

type cohereReranker struct {
	cfg *cohereRerankerConfig
}

type cohereRerankerConfig struct {
	APIKey string
	Model  string
	TopN   int
}

func (r *cohereReranker) Rerank(ctx context.Context, query string, docs any, opts ...RerankOption) ([]*RetrieveResult, error) {
	if r == nil || r.cfg == nil {
		return nil, fmt.Errorf("rerank config is nil")
	}
	if query == "" {
		return nil, fmt.Errorf("query is empty")
	}

	// Convert docs to []*schema.Document
	var schemaDocs []*schema.Document
	switch v := docs.(type) {
	case []*schema.Document:
		schemaDocs = v
	case []any:
		schemaDocs = make([]*schema.Document, 0, len(v))
		for _, item := range v {
			if doc, ok := item.(*schema.Document); ok {
				schemaDocs = append(schemaDocs, doc)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported docs type: %T", docs)
	}

	if len(schemaDocs) == 0 {
		return []*RetrieveResult{}, nil
	}

	apiKey := r.cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("COHERE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("COHERE_API_KEY is not set")
	}

	var co CallOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&co)
		}
	}

	topN := r.cfg.TopN
	if co.TopN != nil {
		topN = *co.TopN
	}
	if topN <= 0 {
		topN = 3
	}

	texts := make([]*string, 0, len(schemaDocs))
	for _, d := range schemaDocs {
		c := d.Content
		texts = append(texts, &c)
	}

	res, err := rerank(apiKey, r.cfg.Model, query, texts, topN)
	if err != nil {
		return nil, err
	}

	out := make([]*RetrieveResult, 0, len(res))
	for _, item := range res {
		if item.Index < 0 || item.Index >= len(schemaDocs) {
			continue
		}
		out = append(out, &RetrieveResult{Content: schemaDocs[item.Index].Content, RelevanceScore: item.RelevanceScore})
	}

	return out, nil
}

func rerank(apiKey, model, query string, documents []*string, topN int) ([]*cohere.RerankResponseResultsItem, error) {
	client := cohereClient.NewClient(cohereClient.WithToken(apiKey))

	var rerankDocs []*cohere.RerankRequestDocumentsItem
	for _, doc := range documents {
		rerankDoc := &cohere.RerankRequestDocumentsItem{}
		rerankDoc.String = *doc
		rerankDocs = append(rerankDocs, rerankDoc)
	}

	rerankResponse, err := client.Rerank(
		context.Background(),
		&cohere.RerankRequest{
			Query:     query,
			Documents: rerankDocs,
			TopN:      &topN,
			Model:     &model,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call rerank API: %w", err)
	}

	return rerankResponse.Results, nil
}
