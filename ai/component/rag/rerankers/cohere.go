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

package rerankers

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino/schema"
	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereClient "github.com/cohere-ai/cohere-go/v2/client"
)

// Reranker types supported
const (
	RerankerTypeCohere = "cohere"
	RerankerTypeNone   = ""
)

// Reranker defines the interface for reranking retrieved documents.
type Reranker interface {
	// Rerank reorders documents based on their relevance to the query.
	// Returns reranked results with relevance scores.
	Rerank(ctx context.Context, query string, docs any, opts ...Option) ([]*Result, error)
}

// Result represents a reranked document with its relevance score.
type Result struct {
	Content        string
	RelevanceScore float64
	Index          int    // Original index in input docs
	Metadata       map[string]any
}

// Option is a function that configures rerank behavior.
type Option func(*CallOptions)

// CallOptions holds per-call rerank options.
type CallOptions struct {
	TopN            *int
	MaxTokens       *int
	ReturnDocuments bool
}

// WithTopN sets the maximum number of results to return.
func WithTopN(topN int) Option {
	return func(o *CallOptions) { o.TopN = &topN }
}

// WithMaxTokens sets the maximum tokens per document (implementation-specific).
func WithMaxTokens(maxTokens int) Option {
	return func(o *CallOptions) { o.MaxTokens = &maxTokens }
}

// WithReturnDocuments sets whether to return document content.
func WithReturnDocuments(returnDocs bool) Option {
	return func(o *CallOptions) { o.ReturnDocuments = returnDocs }
}

// applyOptions applies options to call options.
func applyOptions(opts []Option) CallOptions {
	var co CallOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&co)
		}
	}
	return co
}

// ToSchemaDocuments converts various input types to []*schema.Document.
func ToSchemaDocuments(docs any) ([]*schema.Document, error) {
	switch v := docs.(type) {
	case []*schema.Document:
		return v, nil
	case []any:
		schemaDocs := make([]*schema.Document, 0, len(v))
		for _, item := range v {
			if doc, ok := item.(*schema.Document); ok {
				schemaDocs = append(schemaDocs, doc)
			}
		}
		return schemaDocs, nil
	case []*Result:
		// Convert Results back to Documents (for multi-stage reranking)
		schemaDocs := make([]*schema.Document, 0, len(v))
		for _, r := range v {
			schemaDocs = append(schemaDocs, &schema.Document{
				Content:  r.Content,
				MetaData: r.Metadata,
			})
		}
		return schemaDocs, nil
	default:
		return nil, &UnsupportedTypeError{Type: docs}
	}
}

// UnsupportedTypeError is returned when docs type is not supported.
type UnsupportedTypeError struct {
	Type any
}

func (e *UnsupportedTypeError) Error() string {
	return "unsupported docs type"
}

// CohereConfig holds configuration for Cohere reranker.
type CohereConfig struct {
	APIKey string // Defaults to COHERE_API_KEY env var
	Model  string // Defaults to "rerank-english-v3.0"
	TopN   int    // Defaults to 3
}

// DefaultCohereConfig returns default Cohere configuration.
func DefaultCohereConfig() *CohereConfig {
	return &CohereConfig{
		Model: "rerank-english-v3.0",
		TopN:  3,
	}
}

// NewCohereReranker creates a new Cohere reranker.
func NewCohereReranker(cfg *CohereConfig) (Reranker, error) {
	if cfg == nil {
		cfg = DefaultCohereConfig()
	}
	if cfg.Model == "" {
		cfg.Model = "rerank-english-v3.0"
	}
	if cfg.TopN <= 0 {
		cfg.TopN = 3
	}
	return &cohereReranker{cfg: cfg}, nil
}

type cohereReranker struct {
	cfg *CohereConfig
}

func (r *cohereReranker) Rerank(ctx context.Context, query string, docs any, opts ...Option) ([]*Result, error) {
	if query == "" {
		return nil, fmt.Errorf("query is empty")
	}

	schemaDocs, err := ToSchemaDocuments(docs)
	if err != nil {
		return nil, err
	}

	if len(schemaDocs) == 0 {
		return []*Result{}, nil
	}

	apiKey := r.cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("COHERE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("COHERE_API_KEY is not set")
	}

	co := applyOptions(opts)
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

	res, err := r.callCohereRerank(ctx, apiKey, query, texts, topN)
	if err != nil {
		return nil, err
	}

	out := make([]*Result, 0, len(res))
	for _, item := range res {
		if item.Index < 0 || item.Index >= len(schemaDocs) {
			continue
		}
		doc := schemaDocs[item.Index]
		out = append(out, &Result{
			Content:        doc.Content,
			RelevanceScore: item.RelevanceScore,
			Index:          int(item.Index),
			Metadata:       doc.MetaData,
		})
	}

	return out, nil
}

func (r *cohereReranker) callCohereRerank(ctx context.Context, apiKey, query string, documents []*string, topN int) ([]*cohere.RerankResponseResultsItem, error) {
	client := cohereClient.NewClient(cohereClient.WithToken(apiKey))

	var rerankDocs []*cohere.RerankRequestDocumentsItem
	for _, doc := range documents {
		rerankDocs = append(rerankDocs, &cohere.RerankRequestDocumentsItem{
			String: *doc,
		})
	}

	rerankResponse, err := client.Rerank(
		ctx,
		&cohere.RerankRequest{
			Query:     query,
			Documents: rerankDocs,
			TopN:      &topN,
			Model:     &r.cfg.Model,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call cohere rerank API: %w", err)
	}

	return rerankResponse.Results, nil
}
