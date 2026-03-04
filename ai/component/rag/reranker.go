package rag

import (
	"context"
	"dubbo-admin-ai/runtime"
	"fmt"
	"os"

	"github.com/cloudwego/eino/schema"
	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereClient "github.com/cohere-ai/cohere-go/v2/client"
)

// Reranker 重排序器接口
type Reranker interface {
	Rerank(ctx context.Context, query string, docs any, opts ...Option) ([]*RetrieveResult, error)
}

// rerankerComponent Reranker 组件包装器
type rerankerComponent struct {
	rerankerType string
	enabled      bool
	model        string
	apiKey       string
	reranker     Reranker
}

func NewRerankerComponent(rerankerType string, enabled bool, model, apiKey string) (runtime.Component, error) {
	return &rerankerComponent{
		rerankerType: rerankerType,
		enabled:      enabled,
		model:        model,
		apiKey:       apiKey,
	}, nil
}

func (c *rerankerComponent) Name() string { return "reranker" }

func (c *rerankerComponent) Validate() error { return nil }

func (c *rerankerComponent) Init(rt *runtime.Runtime) error {
	if !c.enabled {
		rt.GetLogger().Info("Reranker component disabled")
		return nil
	}

	reranker, err := newRerankerByType(c.rerankerType, c.enabled, c.model, c.apiKey)
	if err != nil {
		return fmt.Errorf("failed to create reranker: %w", err)
	}
	c.reranker = reranker

	rt.GetLogger().Info("Reranker component initialized", "type", c.rerankerType, "model", c.model)
	return nil
}

func (c *rerankerComponent) Start() error { return nil }

func (c *rerankerComponent) Stop() error { return nil }

func (c *rerankerComponent) get() Reranker {
	return c.reranker
}

type cohereReranker struct {
	cfg *cohereRerankerConfig
}

type cohereRerankerConfig struct {
	APIKey string
	Model  string
	TopN   int
}

func (r *cohereReranker) Rerank(ctx context.Context, query string, docs any, opts ...Option) ([]*RetrieveResult, error) {
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

	// TODO: Transfer the API key management to component initialization phase, and support multiple reranker types with different API keys
	apiKey := r.cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("COHERE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("COHERE_API_KEY is not set")
	}

	var co RAGOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&co)
		}
	}

	topN := r.cfg.TopN
	if co.RerankTopN != nil {
		topN = *co.RerankTopN
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

func newRerankerByType(rerankerType string, enabled bool, model, apiKey string) (Reranker, error) {
	if !enabled {
		return nil, nil
	}
	if model == "" {
		model = DefaultRerankerModel
	}
	switch rerankerType {
	case "cohere":
		return &cohereReranker{cfg: &cohereRerankerConfig{APIKey: apiKey, Model: model, TopN: 3}}, nil
	default:
		return nil, fmt.Errorf("unsupported reranker type: %s", rerankerType)
	}
}
