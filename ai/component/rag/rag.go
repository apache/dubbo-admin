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
	"log/slog"

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

	// Logger for operation logging
	logger *slog.Logger
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
		logger:         components.Logger,
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
	Logger         *slog.Logger

	// Legacy: for backward compatibility
	QueryProcessor QueryProcessor
}

// Split splits documents into chunks.
func (r *RAG) Split(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.Split: START",
			"input_documents", len(docs),
		)
	}
	if r.Splitter == nil {
		if r.logger != nil {
			r.logger.InfoContext(ctx, "RAG.Split: END - no splitter configured",
				"output_documents", len(docs),
			)
		}
		return docs, nil
	}
	result, err := r.Splitter.Transform(ctx, docs)
	if err != nil {
		if r.logger != nil {
			r.logger.ErrorContext(ctx, "RAG.Split: ERROR",
				"error", err,
			)
		}
		return nil, err
	}
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.Split: END",
			"input_documents", len(docs),
			"output_chunks", len(result),
		)
	}
	return result, nil
}

// Index indexes documents into the vector store.
func (r *RAG) Index(ctx context.Context, namespace string, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.Index: START",
			"namespace", namespace,
			"input_documents", len(docs),
		)
	}
	if r.Indexer == nil {
		err := fmt.Errorf("indexer is nil")
		if r.logger != nil {
			r.logger.ErrorContext(ctx, "RAG.Index: ERROR",
				"error", err,
			)
		}
		return nil, err
	}
	var ids []string
	var err error
	if namespace == "" {
		ids, err = r.Indexer.Store(ctx, docs, opts...)
	} else {
		all := append([]indexer.Option{WithIndexerNamespace(namespace)}, opts...)
		ids, err = r.Indexer.Store(ctx, docs, all...)
	}
	if err != nil {
		if r.logger != nil {
			r.logger.ErrorContext(ctx, "RAG.Index: ERROR",
				"error", err,
			)
		}
		return nil, err
	}
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.Index: END",
			"namespace", namespace,
			"input_documents", len(docs),
			"indexed_ids", len(ids),
		)
	}
	return ids, nil
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

	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.RetrieveV2: START",
			"query", req.Query,
			"top_k", req.TopK,
			"reranker_enabled", r.Reranker != nil,
		)
	}

	// Step 1: Query understanding
	queryResult, err := r.processQuery(ctx, req)
	if err != nil {
		if r.logger != nil {
			r.logger.ErrorContext(ctx, "RAG.RetrieveV2: query processing ERROR",
				"error", err,
				"query", req.Query,
			)
		}
		return nil, fmt.Errorf("query processing failed: %w", err)
	}

	// Step 2: Multi-path retrieval
	rawResults, err := r.retrieveMultiPath(ctx, queryResult, req)
	if err != nil {
		if r.logger != nil {
			r.logger.ErrorContext(ctx, "RAG.RetrieveV2: retrieval ERROR",
				"error", err,
				"query", req.Query,
			)
		}
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// Step 3: Reranking
	if r.Reranker != nil {
		rawResults, err = r.rerank(ctx, req.Query, rawResults, req.TopK)
		if err != nil {
			if r.logger != nil {
				r.logger.WarnContext(ctx, "RAG.RetrieveV2: rerank failed, returning unranked results",
					"error", err,
					"results_count", len(rawResults),
				)
			}
			// Return results without reranking
			return &RetrieveResponse{
				Results:       toRetrieveResults(rawResults),
				QueryResult:   toQueryProcessResult(queryResult),
				RetrievalMeta: buildRetrievalMeta(rawResults),
			}, nil
		}
	}

	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.RetrieveV2: END",
			"query", req.Query,
			"results_count", len(rawResults),
			"top_k", req.TopK,
		)
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
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.processQuery: START",
			"original_query", queryStr,
			"top_k", req.TopK,
		)
	}
	// Use new query.Layer if available
	var result *query.Result
	var err error
	if r.QueryLayer != nil {
		result, err = r.QueryLayer.Process(ctx, queryStr)
		if err != nil {
			if r.logger != nil {
				r.logger.ErrorContext(ctx, "RAG.processQuery: ERROR",
					"error", err,
				)
			}
			return nil, err
		}
	} else {
		// Fallback to legacy QueryProcessor
		result = &query.Result{Query: queryStr}
	}
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.processQuery: END",
			"original_query", queryStr,
			"processed_query", result.Query,
			"queries_count", len(result.Queries),
			"intent", result.Intent,
			"modified", result.Modified,
			"has_hypothetical", result.Hypothetical != "",
		)
	}
	return result, nil
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
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.retrieveFromPaths: START",
			"queries", queries,
			"top_k", topK,
			"paths_count", len(r.RetrievalPaths),
		)
	}
	// Collect results from all paths
	allPaths := make([]*mergers.MultiPathResult, 0)

	for _, path := range r.RetrievalPaths {
		if path.Retriever == nil {
			continue
		}
		if r.logger != nil {
			r.logger.InfoContext(ctx, "RAG.retrieveFromPaths: retrieving from path",
				"path_label", path.Label,
				"path_top_k", path.TopK,
				"path_weight", path.Weight,
			)
		}
		pathTopK := path.TopK
		if pathTopK <= 0 {
			pathTopK = topK
		}

		// Retrieve for each query (use first query for simplicity)
		docs, err := path.Retriever.Retrieve(ctx, queries[0], retriever.WithTopK(pathTopK))
		if err != nil {
			if r.logger != nil {
				r.logger.WarnContext(ctx, "RAG.retrieveFromPaths: path retrieval failed",
					"path_label", path.Label,
					"error", err,
				)
			}
			// Log and continue with other paths
			continue
		}

		if r.logger != nil {
			r.logger.InfoContext(ctx, "RAG.retrieveFromPaths: path retrieved",
				"path_label", path.Label,
				"results_count", len(docs),
			)
		}

		allPaths = append(allPaths, &mergers.MultiPathResult{
			Label:   mergers.SourceLabel(path.Label),
			Results: docs,
			Weight:  path.Weight,
		})
	}

	// Merge paths
	var merged []*schema.Document
	if r.Merger != nil {
		var err error
		merged, err = r.Merger.Merge(ctx, allPaths)
		if err != nil {
			if r.logger != nil {
				r.logger.ErrorContext(ctx, "RAG.retrieveFromPaths: merge ERROR",
					"error", err,
				)
			}
			return nil, err
		}
		if r.logger != nil {
			r.logger.InfoContext(ctx, "RAG.retrieveFromPaths: merged results",
				"input_paths", len(allPaths),
				"merged_count", len(merged),
			)
		}
	} else {
		// Fallback: concatenate without merge
		merged = r.concatenateResults(allPaths)
		if r.logger != nil {
			r.logger.InfoContext(ctx, "RAG.retrieveFromPaths: concatenated results (no merger)",
				"input_paths", len(allPaths),
				"output_count", len(merged),
			)
		}
	}

	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.retrieveFromPaths: END",
			"queries", queries,
			"top_k", topK,
			"final_results", len(merged),
		)
	}

	return merged, nil
}

// retrieveSingle executes single-path retrieval.
func (r *RAG) retrieveSingle(ctx context.Context, query string, req *RetrieveRequest) ([]*schema.Document, error) {
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.retrieveSingle: START",
			"query", query,
			"top_k", topK,
		)
	}
	docs, err := r.Retriever.Retrieve(ctx, query, retriever.WithTopK(topK))
	if err != nil {
		if r.logger != nil {
			r.logger.ErrorContext(ctx, "RAG.retrieveSingle: ERROR",
				"error", err,
				"query", query,
			)
		}
		return nil, err
	}
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.retrieveSingle: END",
			"query", query,
			"top_k", topK,
			"results_count", len(docs),
		)
	}
	return docs, nil
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
	if r.logger != nil {
		r.logger.InfoContext(ctx, "RAG.rerank: START",
			"query", query,
			"input_docs", len(docs),
			"top_k", topK,
		)
	}
	reranked, err := r.Reranker.Rerank(ctx, query, docs, rerankers.WithTopN(topK))
	if err != nil {
		if r.logger != nil {
			r.logger.ErrorContext(ctx, "RAG.rerank: ERROR",
				"error", err,
				"query", query,
			)
		}
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

	if r.logger != nil {
		scores := make([]float64, 0, len(result))
		for _, doc := range result {
			if s, ok := doc.MetaData["rerank_score"].(float64); ok {
				scores = append(scores, s)
			}
		}
		r.logger.InfoContext(ctx, "RAG.rerank: END",
			"query", query,
			"input_docs", len(docs),
			"output_docs", len(result),
			"top_k", topK,
			"rerank_scores", scores,
		)
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
