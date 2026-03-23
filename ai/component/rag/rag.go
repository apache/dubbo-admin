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

package rag

import (
	"context"
	"fmt"

	"dubbo-admin-ai/component/rag/loaders"
	"dubbo-admin-ai/component/rag/mergers"
	"dubbo-admin-ai/component/rag/query"
	"dubbo-admin-ai/component/rag/rerankers"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

// LoadDirectory loads all supported files from a directory recursively.
// This is a convenience wrapper around loaders.LoadDirectory.
func LoadDirectory(ctx context.Context, loader document.Loader, dirPath string, opts ...loaders.LoaderOption) ([]*schema.Document, error) {
	return loaders.LoadDirectory(ctx, loader, dirPath, opts...)
}

// RAG provides runtime-facing document split, index and retrieve operations.
type RAG struct {
	// Document processing
	Loader   document.Loader
	Splitter document.Transformer
	Indexer  indexer.Indexer

	// Retrieval (legacy single path, for backward compatibility)
	Retriever retriever.Retriever

	// Multi-path retrieval (new)
	RetrievalPaths []*RetrievalPath
	Merger         *mergers.MergeLayer

	// Query understanding
	QueryLayer *query.Layer

	// Reranking
	Reranker rerankers.Reranker
}

// RetrievalPath represents a single retrieval path with its configuration.
type RetrievalPath struct {
	Label    string              // Path identifier (e.g., "dense", "sparse")
	Retriever retriever.Retriever // The retriever for this path
	TopK     int                 // TopK for this path (0 = use default)
	Weight   float64             // Weight for weighted fusion (default: 1.0)
}

// NewRAG creates a new RAG instance with the given components.
func NewRAG(components *Components) (*RAG, error) {
	if components == nil {
		return nil, fmt.Errorf("components is nil")
	}

	return &RAG{
		Loader:         components.Loader,
		Splitter:       components.Splitter,
		Indexer:        components.Indexer,
		Retriever:      components.Retriever,
		RetrievalPaths: components.RetrievalPaths,
		Merger:         components.Merger,
		QueryLayer:     components.QueryLayer,
		Reranker:       components.Reranker,
	}, nil
}

// Components holds all RAG components.
type Components struct {
	Loader         document.Loader
	Splitter       document.Transformer
	Indexer        indexer.Indexer
	Retriever      retriever.Retriever
	RetrievalPaths []*RetrievalPath
	Merger         *mergers.MergeLayer
	QueryLayer     *query.Layer
	Reranker       rerankers.Reranker

	// Legacy: for backward compatibility
	QueryProcessor QueryProcessor
}

// Split splits documents into chunks.
func (r *RAG) Split(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
	if r.Splitter == nil {
		return docs, nil
	}
	return r.Splitter.Transform(ctx, docs)
}

// Index indexes documents into the vector store.
func (r *RAG) Index(ctx context.Context, namespace string, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if r.Indexer == nil {
		return nil, fmt.Errorf("indexer is nil")
	}
	if namespace == "" {
		return r.Indexer.Store(ctx, docs, opts...)
	}
	all := append([]indexer.Option{WithIndexerNamespace(namespace)}, opts...)
	return r.Indexer.Store(ctx, docs, all...)
}

// RetrieveV2 performs retrieval with query understanding, multi-path retrieval, and reranking.
//
// Flow:
// 1. Query Layer (intent, rewrite, expansion, HyDE)
// 2. Multi-path Retrieval (dense, sparse, etc.)
// 3. Merge (dedup, normalize, combine)
// 4. Rerank (optional)
func (r *RAG) RetrieveV2(ctx context.Context, req *RetrieveRequest) (*RetrieveResponse, error) {
	if req == nil {
		req = DefaultRetrieveRequest()
	}

	// Step 1: Query understanding
	queryResult, err := r.processQuery(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("query processing failed: %w", err)
	}

	// Step 2: Multi-path retrieval
	rawResults, err := r.retrieveMultiPath(ctx, queryResult, req)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// Step 3: Reranking
	if r.Reranker != nil {
		rawResults, err = r.rerank(ctx, req.Query, rawResults, req.TopK)
		if err != nil {
			// Log error but return results without reranking
			return &RetrieveResponse{
				Results:       toRetrieveResults(rawResults),
				QueryResult:   toQueryProcessResult(queryResult),
				RetrievalMeta: buildRetrievalMeta(rawResults),
			}, nil
		}
	}

	return &RetrieveResponse{
		Results:       toRetrieveResults(rawResults),
		QueryResult:   toQueryProcessResult(queryResult),
		RetrievalMeta: buildRetrievalMeta(rawResults),
	}, nil
}

// processQuery applies query understanding layer.
func (r *RAG) processQuery(ctx context.Context, req *RetrieveRequest) (*query.Result, error) {
	queryStr := req.Query

	// Use new query.Layer if available
	if r.QueryLayer != nil {
		return r.QueryLayer.Process(ctx, queryStr)
	}

	// Fallback to legacy QueryProcessor
	// This won't be used if QueryLayer is set
	return &query.Result{Query: queryStr}, nil
}

// retrieveMultiPath executes multi-path retrieval with merging.
func (r *RAG) retrieveMultiPath(ctx context.Context, queryResult *query.Result, req *RetrieveRequest) ([]*schema.Document, error) {
	// Determine which queries to use
	queries := queryResult.Queries
	if len(queries) == 0 {
		queries = []string{queryResult.Query}
	}

	// Use HyDE document if available
	hydeQuery := queryResult.Hypothetical
	if hydeQuery != "" {
		queries = append(queries, hydeQuery)
	}

	// Execute multi-path retrieval
	if len(r.RetrievalPaths) > 0 && r.Merger != nil {
		return r.retrieveFromPaths(ctx, queries, req)
	}

	// Fallback to single retriever
	if r.Retriever != nil {
		return r.retrieveSingle(ctx, queries[0], req)
	}

	return nil, fmt.Errorf("no retriever configured")
}

// retrieveFromPaths executes all retrieval paths and merges results.
func (r *RAG) retrieveFromPaths(ctx context.Context, queries []string, req *RetrieveRequest) ([]*schema.Document, error) {
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	// Collect results from all paths
	allPaths := make([]*mergers.MultiPathResult, 0)

	for _, path := range r.RetrievalPaths {
		if path.Retriever == nil {
			continue
		}

		pathTopK := path.TopK
		if pathTopK <= 0 {
			pathTopK = topK
		}

		// Retrieve for each query (use first query for simplicity)
		docs, err := path.Retriever.Retrieve(ctx, queries[0], retriever.WithTopK(pathTopK))
		if err != nil {
			// Log and continue with other paths
			continue
		}

		allPaths = append(allPaths, &mergers.MultiPathResult{
			Label:   mergers.SourceLabel(path.Label),
			Results: docs,
			Weight:  path.Weight,
		})
	}

	// Merge paths
	if r.Merger != nil {
		return r.Merger.Merge(ctx, allPaths)
	}

	// Fallback: concatenate without merge
	return r.concatenateResults(allPaths), nil
}

// retrieveSingle executes single-path retrieval.
func (r *RAG) retrieveSingle(ctx context.Context, query string, req *RetrieveRequest) ([]*schema.Document, error) {
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}
	return r.Retriever.Retrieve(ctx, query, retriever.WithTopK(topK))
}

// concatenateResults concatenates results from multiple paths.
func (r *RAG) concatenateResults(paths []*mergers.MultiPathResult) []*schema.Document {
	seen := make(map[string]bool)
	result := make([]*schema.Document, 0)

	for _, path := range paths {
		for _, doc := range path.Results {
			id := doc.ID
			if id == "" {
				id = doc.Content
			}
			if !seen[id] {
				seen[id] = true
				result = append(result, doc)
			}
		}
	}

	return result
}

// rerank applies reranking to the results.
func (r *RAG) rerank(ctx context.Context, query string, docs []*schema.Document, topK int) ([]*schema.Document, error) {
	reranked, err := r.Reranker.Rerank(ctx, query, docs, rerankers.WithTopN(topK))
	if err != nil {
		return nil, err
	}

	// Convert reranker.Result to schema.Document, preserving original IDs
	result := make([]*schema.Document, 0, len(reranked))
	for _, r := range reranked {
		// Find original document by Index
		if r.Index >= 0 && r.Index < len(docs) {
			original := docs[r.Index]
			doc := &schema.Document{
				ID:      original.ID,
				Content: r.Content,
				MetaData: make(map[string]any),
			}
			// Copy original metadata
			if original.MetaData != nil {
				for k, v := range original.MetaData {
					doc.MetaData[k] = v
				}
			}
			// Add rerank metadata
			doc.MetaData["rerank_score"] = r.RelevanceScore
			doc.MetaData["rerank_index"] = r.Index
			result = append(result, doc)
		}
	}

	return result, nil
}

// ========== Legacy: Retrieve method for backward compatibility ==========

// Retrieve performs retrieval with the legacy interface for backward compatibility.
func (r *RAG) Retrieve(ctx context.Context, namespace string, queries []string, opts ...RetrieveOption) (map[string][]*RetrieveResult, error) {
	if r.Retriever == nil {
		return nil, fmt.Errorf("retriever is nil")
	}
	if len(queries) == 0 {
		return map[string][]*RetrieveResult{}, nil
	}

	// Process options for reranker
	var rerankOpts []rerankers.Option
	for _, opt := range opts {
		if opt != nil {
			rerankOpts = append(rerankOpts, opt)
		}
	}

	// Build retriever options
	retrieveOpts := make([]retriever.Option, 0)
	if namespace != "" {
		retrieveOpts = append(retrieveOpts, WithRetrieverImplNamespace(namespace))
	}

	// Process queries
	resp := make(map[string][]*RetrieveResult)
	for _, originalQuery := range queries {
		// Apply query processing
		processedQuery := originalQuery
		if r.QueryLayer != nil {
			result, err := r.QueryLayer.Process(ctx, originalQuery)
			if err == nil && result.Query != "" {
				processedQuery = result.Query
			}
		}

		// Retrieve
		docs, err := r.Retriever.Retrieve(ctx, processedQuery, retrieveOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve for query %q: %w", originalQuery, err)
		}

		// Convert to results
		results := make([]*RetrieveResult, 0, len(docs))
		for _, doc := range docs {
			results = append(results, &RetrieveResult{
				Content:  doc.Content,
				Score:    extractScore(doc),
				Source:   extractMetadata(doc, "source"),
				Title:    extractMetadata(doc, "title"),
				Metadata: doc.MetaData,
			})
		}

		// Apply reranker
		if r.Reranker != nil && len(results) > 0 {
			reranked, err := r.applyReranker(ctx, originalQuery, results, opts...)
			if err == nil {
				results = reranked
			}
		}

		resp[originalQuery] = results
	}

	return resp, nil
}

// applyReranker applies reranker to results.
func (r *RAG) applyReranker(ctx context.Context, query string, results []*RetrieveResult, opts ...RetrieveOption) ([]*RetrieveResult, error) {
	docs := make([]*schema.Document, 0, len(results))
	for _, r := range results {
		docs = append(docs, &schema.Document{
			Content:  r.Content,
			MetaData: r.Metadata,
		})
	}

	reranked, err := r.Reranker.Rerank(ctx, query, docs, opts...)
	if err != nil {
		return nil, err
	}

	output := make([]*RetrieveResult, 0, len(reranked))
	for _, r := range reranked {
		output = append(output, &RetrieveResult{
			Content:  r.Content,
			Score:    r.RelevanceScore,
			Source:   extractMetadataFromMap(r.Metadata, "source"),
			Title:    extractMetadataFromMap(r.Metadata, "title"),
			Metadata: r.Metadata,
		})
	}

	return output, nil
}

// ========== Helper functions ==========

func extractScore(doc *schema.Document) float64 {
	if doc.MetaData == nil {
		return 0
	}
	if score, ok := doc.MetaData["score"].(float64); ok {
		return score
	}
	if score, ok := doc.MetaData["merge_score"].(float64); ok {
		return score
	}
	return 0
}

func extractMetadata(doc *schema.Document, key string) string {
	if doc.MetaData == nil {
		return ""
	}
	if val, ok := doc.MetaData[key].(string); ok {
		return val
	}
	return ""
}

func extractMetadataFromMap(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	if val, ok := meta[key].(string); ok {
		return val
	}
	return ""
}

func toRetrieveResults(docs []*schema.Document) []*RetrieveResult {
	results := make([]*RetrieveResult, 0, len(docs))
	for _, doc := range docs {
		results = append(results, &RetrieveResult{
			Content:  doc.Content,
			Score:    extractScore(doc),
			Source:   extractMetadata(doc, "source"),
			Title:    extractMetadata(doc, "title"),
			Metadata: doc.MetaData,
		})
	}
	return results
}

func toQueryProcessResult(qr *query.Result) *QueryProcessResult {
	if qr == nil {
		return &QueryProcessResult{}
	}
	return &QueryProcessResult{
		Query:        qr.Query,
		Queries:      qr.Queries,
		Intent:       qr.Intent,
		Hypothetical: qr.Hypothetical,
		Modified:     qr.Modified,
		Metadata:     qr.Metadata,
	}
}

func buildRetrievalMeta(docs []*schema.Document) map[string]any {
	meta := make(map[string]any)
	meta["total_results"] = len(docs)

	// Count sources
	sources := make(map[string]int)
	for _, doc := range docs {
		if srcs, ok := doc.MetaData["merge_sources"].([]string); ok {
			for _, src := range srcs {
				sources[src]++
			}
		}
	}
	meta["source_counts"] = sources

	return meta
}
