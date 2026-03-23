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

package retrievers

import (
	"context"
	"dubbo-admin-ai/utils"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

const (
	// RetrieverTypeMilvus is the retriever type for Milvus vector database
	RetrieverTypeMilvus = "milvus"

	// DefaultMilvusHostEnv is the default environment variable for Milvus host
	DefaultMilvusHostEnv = "MILVUS_HOST"
	// DefaultMilvusTokenEnv is the default environment variable for Milvus token
	DefaultMilvusTokenEnv = "MILVUS_TOKEN"
)

// MilvusConfig defines configuration for Milvus retriever.
type MilvusConfig struct {
	// Connection
	Address  string // Milvus server address
	Token    string // Authentication token (for Zilliz Cloud)
	Username string // Username (for Milvus with auth)
	Password string // Password (for Milvus with auth)

	// Collection
	Collection string // Collection name

	// Embedder
	Embedder string // Embedder model name

	// Search type: "dense" (vector), "sparse" (BM25), or "hybrid" (both)
	SearchType string

	// Dense search configuration
	DenseField string // Dense vector field name
	DenseTopK  int
	MetricType string // Metric type: "L2", "IP", "COSINE" (default: "COSINE")

	// Sparse search configuration (BM25)
	SparseField string // Sparse vector field name
	SparseTopK  int
	EnableBM25  bool   // Use built-in BM25 function (Milvus 2.5+)
	TextField   string // Text content field name

	// Metadata field names (must match indexer config)
	SourceField     string // Document source path
	TitleField      string // Document title
	PageField       string // PDF page number
	UpdatedAtField  string // Last update time
	ChunkIndexField string // Chunk index
	ChunkSizeField  string // Chunk size
	HeaderPathField string // Markdown header path

	// Metadata retrieval
	EnableMetadata bool // Enable metadata field retrieval

	// Hybrid search configuration (reserved for future use)
	HybridRanker string  // "rrf" | "weighted_rank" | "nnf"
	DenseWeight  float64 // Weight for dense results
	SparseWeight float64 // Weight for sparse results
}

// MilvusRetriever provides Milvus vector retrieval.
type MilvusRetriever struct {
	g      *genkit.Genkit
	config *MilvusConfig
	client *milvusclient.Client
}

// defaultMilvusConfig returns the default configuration.
func defaultMilvusConfig() *MilvusConfig {
	return &MilvusConfig{
		SearchType:   "dense",
		DenseField:   "vector",
		SparseField:  "sparse",
		TextField:    "text",
		DenseTopK:    10,
		MetricType:   "COSINE",
		SparseTopK:   10,
		HybridRanker: "rrf",
		DenseWeight:  0.7,
		SparseWeight: 0.3,
		EnableBM25:   false,
		// Metadata fields
		SourceField:     "source",
		TitleField:      "title",
		PageField:       "page",
		UpdatedAtField:  "updated_at",
		ChunkIndexField: "chunk_index",
		ChunkSizeField:  "chunk_size",
		HeaderPathField: "header_path",
		EnableMetadata:  true,
	}
}

// applyDefaults applies default values to the Milvus configuration.
func applyDefaults(cfg *MilvusConfig) *MilvusConfig {
	if cfg == nil {
		cfg = &MilvusConfig{}
	}
	defaults := defaultMilvusConfig()

	result := *defaults // Copy defaults as base

	// Override with provided values
	if cfg.Address != "" {
		result.Address = cfg.Address
	}
	if cfg.Token != "" {
		result.Token = cfg.Token
	}
	if cfg.Username != "" {
		result.Username = cfg.Username
	}
	if cfg.Password != "" {
		result.Password = cfg.Password
	}
	if cfg.Collection != "" {
		result.Collection = cfg.Collection
	}
	if cfg.Embedder != "" {
		result.Embedder = cfg.Embedder
	}
	if cfg.SearchType != "" {
		result.SearchType = cfg.SearchType
	}
	if cfg.DenseField != "" {
		result.DenseField = cfg.DenseField
	}
	if cfg.DenseTopK > 0 {
		result.DenseTopK = cfg.DenseTopK
	}
	if cfg.MetricType != "" {
		result.MetricType = cfg.MetricType
	}
	if cfg.SparseField != "" {
		result.SparseField = cfg.SparseField
	}
	if cfg.TextField != "" {
		result.TextField = cfg.TextField
	}
	if cfg.SparseTopK > 0 {
		result.SparseTopK = cfg.SparseTopK
	}
	if cfg.HybridRanker != "" {
		result.HybridRanker = cfg.HybridRanker
	}
	if cfg.DenseWeight != 0 {
		result.DenseWeight = cfg.DenseWeight
	}
	if cfg.SparseWeight != 0 {
		result.SparseWeight = cfg.SparseWeight
	}
	if cfg.EnableBM25 {
		result.EnableBM25 = cfg.EnableBM25
	}

	return &result
}

// NewMilvusRetriever creates a new MilvusRetriever.
func NewMilvusRetriever(g *genkit.Genkit, config *MilvusConfig) (*MilvusRetriever, error) {
	// Apply defaults and load from environment
	cfg := applyDefaults(config)

	// Load address/token from environment if not provided
	if cfg.Address == "" {
		cfg.Address = os.Getenv(DefaultMilvusHostEnv)
	}
	if cfg.Token == "" {
		cfg.Token = os.Getenv(DefaultMilvusTokenEnv)
	}

	if cfg.Address == "" {
		return nil, fmt.Errorf("milvus address is required (set %s env or config.Address)", DefaultMilvusHostEnv)
	}

	// Create Milvus client config
	clientCfg := &milvusclient.ClientConfig{
		Address: cfg.Address,
	}

	// Use token-based auth for Zilliz Cloud
	if cfg.Token != "" {
		clientCfg.APIKey = cfg.Token
	} else if cfg.Username != "" && cfg.Password != "" {
		clientCfg.Username = cfg.Username
		clientCfg.Password = cfg.Password
	}

	ctx := context.Background()
	cli, err := milvusclient.New(ctx, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %w", err)
	}

	// Load collection
	loadOption := milvusclient.NewLoadCollectionOption(cfg.Collection)
	if _, err := cli.LoadCollection(ctx, loadOption); err != nil {
		log.Printf("Warning: failed to load collection: %v", err)
	}

	return &MilvusRetriever{
		g:      g,
		config: cfg,
		client: cli,
	}, nil
}

func (r *MilvusRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	switch r.config.SearchType {
	case "dense":
		return r.denseSearch(ctx, query, opts)
	case "sparse":
		return r.sparseSearch(ctx, query, opts)
	case "hybrid":
		return r.hybridSearch(ctx, query, opts)
	default:
		return r.denseSearch(ctx, query, opts)
	}
}

// denseSearch performs vector-only search.
func (r *MilvusRetriever) denseSearch(ctx context.Context, query string, opts []retriever.Option) ([]*schema.Document, error) {
	if r.client == nil {
		return nil, fmt.Errorf("milvus client is not initialized")
	}

	// Generate query embedding
	embedding, err := r.embedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Get TopK from options
	topK := r.config.DenseTopK
	commonOpts := retriever.GetCommonOptions(&retriever.Options{}, opts...)
	if commonOpts.TopK != nil && *commonOpts.TopK > 0 {
		topK = *commonOpts.TopK
	}

	// Convert to float32
	vector := make([]float32, len(embedding))
	for i, v := range embedding {
		vector[i] = float32(v)
	}

	// Build output fields - always include text and metadata if enabled
	outputFields := []string{r.config.TextField}
	if r.config.EnableMetadata {
		outputFields = append(outputFields,
			r.config.SourceField,
			r.config.TitleField,
			r.config.PageField,
			r.config.UpdatedAtField,
			r.config.ChunkIndexField,
			r.config.ChunkSizeField,
			r.config.HeaderPathField,
		)
	}

	// Create search option with new SDK API
	searchOption := milvusclient.NewSearchOption(
		r.config.Collection,
		topK,
		[]entity.Vector{entity.FloatVector(vector)},
	).WithANNSField(r.config.DenseField). // Specify dense field for search
		WithOutputFields(outputFields...) // Request text and metadata fields in results

	// Execute search
	resultSets, err := r.client.Search(ctx, searchOption)
	if err != nil {
		return nil, fmt.Errorf("milvus search failed: %w", err)
	}

	return r.toDocuments(resultSets)
}

// parseMetricType converts string metric type to entity.MetricType.
func parseMetricType(metricType string) entity.MetricType {
	switch metricType {
	case "L2":
		return entity.L2
	case "IP":
		return entity.IP
	case "COSINE":
		return entity.COSINE
	default:
		return entity.COSINE // Default to COSINE for Zilliz Cloud
	}
}

// sparseSearch performs BM25 search using sparse vectors.
func (r *MilvusRetriever) sparseSearch(ctx context.Context, query string, opts []retriever.Option) ([]*schema.Document, error) {
	if r.client == nil {
		return nil, fmt.Errorf("milvus client is not initialized")
	}

	// Get TopK from options
	topK := r.config.SparseTopK
	commonOpts := retriever.GetCommonOptions(&retriever.Options{}, opts...)
	if commonOpts.TopK != nil && *commonOpts.TopK > 0 {
		topK = *commonOpts.TopK
	}

	var searchOption milvusclient.SearchOption

	// Build output fields - always include text and metadata if enabled
	outputFields := []string{r.config.TextField}
	if r.config.EnableMetadata {
		outputFields = append(outputFields,
			r.config.SourceField,
			r.config.TitleField,
			r.config.PageField,
			r.config.UpdatedAtField,
			r.config.ChunkIndexField,
			r.config.ChunkSizeField,
			r.config.HeaderPathField,
		)
	}

	// Use built-in BM25 function (Milvus 2.5+)
	if r.config.EnableBM25 {
		// With BM25 function, use raw text query
		searchOption = milvusclient.NewSearchOption(
			r.config.Collection,
			topK,
			[]entity.Vector{entity.Text(query)}, // entity.Text for BM25
		).WithANNSField(r.config.SparseField). // Specify sparse field for search
			WithOutputFields(outputFields...) // Request text and metadata fields in results
	} else {
		// Fallback to sparse embedding generation
		sparseEmbedding, err := r.sparseEmbedQuery(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to generate sparse embedding: %w", err)
		}

		searchOption = milvusclient.NewSearchOption(
			r.config.Collection,
			topK,
			[]entity.Vector{sparseEmbedding},
		).WithANNSField(r.config.SparseField).
			WithOutputFields(outputFields...) // Request text and metadata fields in results
	}

	// Execute search
	resultSets, err := r.client.Search(ctx, searchOption)
	if err != nil {
		return nil, fmt.Errorf("milvus sparse search failed: %w", err)
	}

	return r.toDocuments(resultSets)
}

// hybridSearch performs combined dense + sparse search with result fusion.
func (r *MilvusRetriever) hybridSearch(ctx context.Context, query string, opts []retriever.Option) ([]*schema.Document, error) {
	if r.client == nil {
		return nil, fmt.Errorf("milvus client is not initialized")
	}

	// When BM25 is enabled, hybrid search with text match requires different API.
	// For simplicity, fall back to dense search in this case.
	// TODO: Implement proper HybridSearch API for BM25 + dense combination
	if r.config.EnableBM25 {
		return r.denseSearch(ctx, query, opts)
	}

	// Get TopK from options
	topK := r.config.DenseTopK
	commonOpts := retriever.GetCommonOptions(&retriever.Options{}, opts...)
	if commonOpts.TopK != nil && *commonOpts.TopK > 0 {
		topK = *commonOpts.TopK
	}

	// Generate both dense and sparse embeddings
	denseEmbedding, err := r.embedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate dense embedding: %w", err)
	}

	// Convert dense embedding to float32
	denseVector := make([]float32, len(denseEmbedding))
	for i, v := range denseEmbedding {
		denseVector[i] = float32(v)
	}

	sparseEmbedding, err := r.sparseEmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sparse embedding: %w", err)
	}

	// Build output fields - always include text and metadata if enabled
	outputFields := []string{r.config.TextField}
	if r.config.EnableMetadata {
		outputFields = append(outputFields,
			r.config.SourceField,
			r.config.TitleField,
			r.config.PageField,
			r.config.UpdatedAtField,
			r.config.ChunkIndexField,
			r.config.ChunkSizeField,
			r.config.HeaderPathField,
		)
	}

	// Create search option for hybrid search
	searchOption := milvusclient.NewSearchOption(
		r.config.Collection,
		topK,
		[]entity.Vector{entity.FloatVector(denseVector), sparseEmbedding},
	).WithANNSField(r.config.DenseField). // Primary search field is dense
		WithOutputFields(outputFields...) // Request text and metadata fields in results

	// Execute hybrid search
	resultSets, err := r.client.Search(ctx, searchOption)
	if err != nil {
		return nil, fmt.Errorf("milvus hybrid search failed: %w", err)
	}

	return r.toDocuments(resultSets)
}

// Helper methods

// embedQuery generates a dense embedding for the query.
func (r *MilvusRetriever) embedQuery(ctx context.Context, query string) ([]float64, error) {
	if r.g == nil {
		return nil, fmt.Errorf("genkit registry is nil")
	}

	// Lookup embedder from genkit registry
	embedder := genkit.LookupEmbedder(r.g, r.config.Embedder)
	if embedder == nil {
		return nil, fmt.Errorf("embedder '%s' not found in registry", r.config.Embedder)
	}

	// Create document with query using utils
	doc := utils.ToGenkitDocument(&schema.Document{Content: query})

	// Call embedder
	resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
		Input: []*ai.Document{doc},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert from float32 to float64
	embedding := make([]float64, len(resp.Embeddings[0].Embedding))
	for i, v := range resp.Embeddings[0].Embedding {
		embedding[i] = float64(v)
	}

	return embedding, nil
}

// sparseEmbedQuery generates a sparse embedding for the query using simple BM25-like encoding.
// This is a fallback when BM25 function is not enabled.
func (r *MilvusRetriever) sparseEmbedQuery(ctx context.Context, query string) (entity.SparseEmbedding, error) {
	// Tokenize query
	words := tokenizeText(query)

	// Calculate term frequencies
	termFreq := make(map[string]int)
	for _, word := range words {
		termFreq[word]++
	}

	// Convert to sparse embedding
	positions := make([]uint32, 0, len(termFreq))
	weights := make([]float32, 0, len(termFreq))

	for term, tf := range termFreq {
		// Use hash for position
		pos := uint32(hashString(term) % 100000)
		// Weight based on term frequency
		weight := float32(tf) * 0.1

		positions = append(positions, pos)
		weights = append(weights, weight)
	}

	return entity.NewSliceSparseEmbedding(positions, weights)
}

// toDocuments converts Milvus search results to Eino documents.
func (r *MilvusRetriever) toDocuments(resultSets []milvusclient.ResultSet) ([]*schema.Document, error) {
	if len(resultSets) == 0 {
		return []*schema.Document{}, nil
	}

	docs := make([]*schema.Document, 0)

	for _, resultSet := range resultSets {
		if resultSet.ResultCount == 0 {
			continue
		}

		// Get text column
		textCol := resultSet.GetColumn(r.config.TextField)
		if textCol == nil {
			continue
		}

		// Get metadata columns if enabled
		var sourceCol, titleCol, updatedAtCol, headerPathCol *column.ColumnVarChar
		var pageCol, chunkIndexCol, chunkSizeCol *column.ColumnInt64

		if r.config.EnableMetadata {
			if col := resultSet.GetColumn(r.config.SourceField); col != nil {
				sourceCol = col.(*column.ColumnVarChar)
			}
			if col := resultSet.GetColumn(r.config.TitleField); col != nil {
				titleCol = col.(*column.ColumnVarChar)
			}
			if col := resultSet.GetColumn(r.config.PageField); col != nil {
				pageCol = col.(*column.ColumnInt64)
			}
			if col := resultSet.GetColumn(r.config.UpdatedAtField); col != nil {
				updatedAtCol = col.(*column.ColumnVarChar)
			}
			if col := resultSet.GetColumn(r.config.ChunkIndexField); col != nil {
				chunkIndexCol = col.(*column.ColumnInt64)
			}
			if col := resultSet.GetColumn(r.config.ChunkSizeField); col != nil {
				chunkSizeCol = col.(*column.ColumnInt64)
			}
			if col := resultSet.GetColumn(r.config.HeaderPathField); col != nil {
				headerPathCol = col.(*column.ColumnVarChar)
			}
		}

		// Extract text and metadata from columns
		for i := 0; i < resultSet.ResultCount; i++ {
			if textData, ok := textCol.(*column.ColumnVarChar); ok && len(textData.Data()) > i {
				doc := &schema.Document{
					Content: string(textData.Data()[i]),
				}

				// Add score from result
				scores := resultSet.Scores
				if len(scores) > i {
					if doc.MetaData == nil {
						doc.MetaData = make(map[string]any)
					}
					doc.MetaData["score"] = float64(scores[i])
				}

				// Add metadata if enabled
				if r.config.EnableMetadata {
					if doc.MetaData == nil {
						doc.MetaData = make(map[string]any)
					}

					if sourceCol != nil && len(sourceCol.Data()) > i {
						doc.MetaData["source"] = sourceCol.Data()[i]
					}
					if titleCol != nil && len(titleCol.Data()) > i {
						doc.MetaData["title"] = titleCol.Data()[i]
					}
					if pageCol != nil && len(pageCol.Data()) > i {
						doc.MetaData["page"] = int(pageCol.Data()[i])
					}
					if updatedAtCol != nil && len(updatedAtCol.Data()) > i {
						doc.MetaData["updated_at"] = updatedAtCol.Data()[i]
					}
					if chunkIndexCol != nil && len(chunkIndexCol.Data()) > i {
						doc.MetaData["chunk_index"] = int(chunkIndexCol.Data()[i])
					}
					if chunkSizeCol != nil && len(chunkSizeCol.Data()) > i {
						doc.MetaData["chunk_size"] = int(chunkSizeCol.Data()[i])
					}
					if headerPathCol != nil && len(headerPathCol.Data()) > i {
						// Parse JSON string back to []string
						headerPathStr := headerPathCol.Data()[i]
						if headerPathStr != "" && headerPathStr != "[]" {
							// Simple JSON parsing for ["a", "b"] format
							headerPath := parseJSONStringSlice(headerPathStr)
							doc.MetaData["header_path"] = headerPath
						}
					}
				}

				docs = append(docs, doc)
			}
		}
	}

	return docs, nil
}

// Close closes the Milvus client connection.
func (r *MilvusRetriever) Close() error {
	if r.client != nil {
		return r.client.Close(context.Background())
	}
	return nil
}

// Client returns the underlying Milvus client.
func (r *MilvusRetriever) Client() *milvusclient.Client {
	return r.client
}

// ValidateConfig validates Milvus retriever configuration without creating a client.
func ValidateConfig(config *MilvusConfig) error {
	cfg := defaultMilvusConfig()

	// Override with provided values
	if config.Address != "" {
		cfg.Address = config.Address
	}
	if config.Token != "" {
		cfg.Token = config.Token
	}
	if config.Collection != "" {
		cfg.Collection = config.Collection
	}
	if config.SearchType != "" {
		cfg.SearchType = config.SearchType
	}
	if config.Embedder != "" {
		cfg.Embedder = config.Embedder
	}

	// Load from environment
	if cfg.Address == "" {
		cfg.Address = os.Getenv(DefaultMilvusHostEnv)
	}
	if cfg.Token == "" {
		cfg.Token = os.Getenv(DefaultMilvusTokenEnv)
	}

	if cfg.Address == "" {
		return fmt.Errorf("milvus address is required (set %s env or config.Address)", DefaultMilvusHostEnv)
	}

	if cfg.Collection == "" {
		return fmt.Errorf("milvus collection is required")
	}

	// Validate search type
	validSearchTypes := map[string]bool{
		"dense":  true,
		"sparse": true,
		"hybrid": true,
	}
	if !validSearchTypes[cfg.SearchType] {
		return fmt.Errorf("invalid search type: %s, must be one of: dense, sparse, hybrid", cfg.SearchType)
	}

	// Sparse search requires text field
	if (config.SearchType == "sparse" || config.SearchType == "hybrid") && config.TextField == "" {
		return fmt.Errorf("text field is required for sparse/hybrid search")
	}

	return nil
}

// parseJSONStringSlice parses a simple JSON string array like ["a", "b"] into []string.
// This is a lightweight parser for the header_path field.
func parseJSONStringSlice(jsonStr string) []string {
	jsonStr = strings.TrimSpace(jsonStr)
	if len(jsonStr) < 2 || jsonStr[0] != '[' || jsonStr[len(jsonStr)-1] != ']' {
		return nil
	}

	inner := jsonStr[1 : len(jsonStr)-1]
	if inner == "" {
		return []string{}
	}

	// Simple parsing: split by "," and trim quotes
	result := []string{}
	current := strings.Builder{}
	inQuotes := false
	escaped := false

	for _, ch := range inner {
		switch ch {
		case '\\':
			if inQuotes {
				escaped = true
			}
		case '"':
			if inQuotes && !escaped {
				inQuotes = false
			} else {
				inQuotes = true
			}
			escaped = false
		case ',':
			if !inQuotes {
				result = append(result, current.String())
				current.Reset()
				continue
			}
			fallthrough
		default:
			current.WriteRune(ch)
			escaped = false
		}
	}

	// Add the last element
	if current.Len() > 0 || len(result) > 0 {
		result = append(result, current.String())
	}

	return result
}

// tokenizeText splits text into words for sparse encoding.
func tokenizeText(text string) []string {
	// Simple tokenization: lowercase and split by non-alphanumeric characters
	words := make([]string, 0)
	currentWord := make([]rune, 0)

	for _, ch := range text {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			currentWord = append(currentWord, ch)
		} else {
			if len(currentWord) > 0 {
				words = append(words, string(currentWord))
				currentWord = currentWord[:0]
			}
		}
	}
	if len(currentWord) > 0 {
		words = append(words, string(currentWord))
	}

	return words
}

// hashString creates a simple hash of a string for sparse vector positions.
func hashString(s string) uint32 {
	// Simple djb2 hash
	h := uint32(5381)
	for _, ch := range s {
		h = ((h << 5) + h) + uint32(ch)
	}
	return h
}
