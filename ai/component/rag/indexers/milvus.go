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

package indexers

import (
	"context"
	"dubbo-admin-ai/utils"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

const (
	// IndexerTypeMilvus is the indexer type for Milvus vector database
	IndexerTypeMilvus = "milvus"

	// DefaultMilvusHostEnv is the default environment variable for Milvus host
	DefaultMilvusHostEnv = "MILVUS_HOST"
	// DefaultMilvusTokenEnv is the default environment variable for Milvus token
	DefaultMilvusTokenEnv = "MILVUS_TOKEN"
)

// MilvusConfig defines configuration for Milvus indexer.
type MilvusConfig struct {
	// Connection
	Address  string // Milvus server address
	Token    string // Authentication token (for Zilliz Cloud)
	Username string // Username (for Milvus with auth)
	Password string // Password (for Milvus with auth)

	// Collection
	Collection string // Collection name

	// Vector configuration
	Dimension  int
	Embedder   string // Embedder model name
	BatchSize  int

	// BM25 configuration (Milvus 2.5+)
	EnableBM25 bool   // Enable built-in BM25 function for full-text search
	BM25K1     float64 // BM25 k1 parameter (default: 1.2)
	BM25B      float64 // BM25 b parameter (default: 0.75)

	// Field names
	IDField     string // Primary key field name
	DenseField  string // Dense vector field name
	TextField   string // Text content field (for BM25)
	SparseField string // Sparse vector field (for BM25 output)

	// Metadata field names (for storing document metadata)
	SourceField     string // Document source path (VARCHAR)
	TitleField      string // Document title (VARCHAR)
	PageField       string // PDF page number (INT64)
	UpdatedAtField  string // Last update time (VARCHAR)
	ChunkIndexField string // Chunk index (INT64)
	ChunkSizeField  string // Chunk size (INT64)
	HeaderPathField string // Markdown header path (VARCHAR/JSON)

	// Metadata storage configuration
	EnableMetadata bool // Enable metadata field creation and storage

	// Index configuration
	DenseIndexType string // IVF_FLAT, HNSW, etc.
}

// MilvusIndexer provides Milvus vector storage with BM25 support.
type MilvusIndexer struct {
	g      *genkit.Genkit
	config *MilvusConfig
	client *milvusclient.Client
}

// defaultMilvusConfig returns the default configuration.
func defaultMilvusConfig() *MilvusConfig {
	return &MilvusConfig{
		IDField:        "id",
		DenseField:     "vector",
		TextField:      "text",
		SparseField:    "sparse",
		BatchSize:      100,
		EnableBM25:     false,
		BM25K1:         1.2,
		BM25B:          0.75,
		DenseIndexType: "IVF_FLAT",
		// Metadata fields
		SourceField:     "source",
		TitleField:      "title",
		PageField:       "page",
		UpdatedAtField:  "updated_at",
		ChunkIndexField: "chunk_index",
		ChunkSizeField:  "chunk_size",
		HeaderPathField: "header_path",
		EnableMetadata:  true, // Enable metadata by default
	}
}

// applyDefaults applies default values to the Milvus configuration.
// It preserves any non-empty values from the input config.
func applyDefaults(cfg *MilvusConfig, defaults *MilvusConfig) *MilvusConfig {
	if cfg == nil {
		cfg = &MilvusConfig{}
	}
	if defaults == nil {
		defaults = defaultMilvusConfig()
	}

	result := *defaults // Copy defaults as base

	// Override with provided values (only if non-empty)
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
	if cfg.Dimension > 0 {
		result.Dimension = cfg.Dimension
	}
	if cfg.Embedder != "" {
		result.Embedder = cfg.Embedder
	}
	if cfg.BatchSize > 0 {
		result.BatchSize = cfg.BatchSize
	}
	if cfg.IDField != "" {
		result.IDField = cfg.IDField
	}
	if cfg.DenseField != "" {
		result.DenseField = cfg.DenseField
	}
	if cfg.TextField != "" {
		result.TextField = cfg.TextField
	}
	if cfg.SparseField != "" {
		result.SparseField = cfg.SparseField
	}
	if cfg.DenseIndexType != "" {
		result.DenseIndexType = cfg.DenseIndexType
	}
	if cfg.EnableBM25 {
		result.EnableBM25 = cfg.EnableBM25
	}
	if cfg.BM25K1 > 0 {
		result.BM25K1 = cfg.BM25K1
	}
	if cfg.BM25B > 0 {
		result.BM25B = cfg.BM25B
	}

	return &result
}

// NewMilvusIndexer creates a new MilvusIndexer.
func NewMilvusIndexer(g *genkit.Genkit, config *MilvusConfig) (*MilvusIndexer, error) {
	// Apply defaults and load from environment
	cfg := applyDefaults(config, nil)

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

	idxer := &MilvusIndexer{
		g:      g,
		config: cfg,
		client: cli,
	}

	// Ensure collection exists
	if err := idxer.ensureCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return idxer, nil
}

func (idx *MilvusIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if idx.client == nil {
		return nil, fmt.Errorf("milvus client is not initialized")
	}

	_ = indexer.GetImplSpecificOptions(&CommonIndexerOptions{}, opts...)

	// Prepare data for insertion
	ids := make([]int64, len(docs))
	texts := make([]string, len(docs))

	for i, doc := range docs {
		ids[i] = int64(i)
		texts[i] = doc.Content
	}

	// Build columns for insertion
	columns := []column.Column{
		column.NewColumnInt64(idx.config.IDField, ids),
		column.NewColumnVarChar(idx.config.TextField, texts),
	}

	// Add metadata columns if enabled
	if idx.config.EnableMetadata {
		sources := make([]string, len(docs))
		titles := make([]string, len(docs))
		pages := make([]int64, len(docs))
		updatedAts := make([]string, len(docs))
		chunkIndices := make([]int64, len(docs))
		chunkSizes := make([]int64, len(docs))
		headerPaths := make([]string, len(docs))

		for i, doc := range docs {
			if doc.MetaData != nil {
				// Extract source
				if v, ok := doc.MetaData["source"].(string); ok {
					sources[i] = v
				}
				// Extract title
				if v, ok := doc.MetaData["title"].(string); ok {
					titles[i] = v
				}
				// Extract page
				if v, ok := doc.MetaData["page"].(int); ok {
					pages[i] = int64(v)
				} else if v, ok := doc.MetaData["page"].(int64); ok {
					pages[i] = v
				}
				// Extract updated_at
				if v, ok := doc.MetaData["updated_at"].(string); ok {
					updatedAts[i] = v
				}
				// Extract chunk_index
				if v, ok := doc.MetaData["chunk_index"].(int); ok {
					chunkIndices[i] = int64(v)
				} else if v, ok := doc.MetaData["chunk_index"].(int64); ok {
					chunkIndices[i] = v
				}
				// Extract chunk_size
				if v, ok := doc.MetaData["chunk_size"].(int); ok {
					chunkSizes[i] = int64(v)
				} else if v, ok := doc.MetaData["chunk_size"].(int64); ok {
					chunkSizes[i] = v
				}
				// Extract header_path (convert []string to JSON)
				if v, ok := doc.MetaData["header_path"].([]string); ok {
					headerPaths[i] = sliceToJSONString(v)
				} else if v, ok := doc.MetaData["header_path"].([]any); ok {
					// Handle []any format
					strSlice := make([]string, 0, len(v))
					for _, item := range v {
						if str, ok := item.(string); ok {
							strSlice = append(strSlice, str)
						}
					}
					headerPaths[i] = sliceToJSONString(strSlice)
				}
			}
		}

		columns = append(columns,
			column.NewColumnVarChar(idx.config.SourceField, sources),
			column.NewColumnVarChar(idx.config.TitleField, titles),
			column.NewColumnInt64(idx.config.PageField, pages),
			column.NewColumnVarChar(idx.config.UpdatedAtField, updatedAts),
			column.NewColumnInt64(idx.config.ChunkIndexField, chunkIndices),
			column.NewColumnInt64(idx.config.ChunkSizeField, chunkSizes),
			column.NewColumnVarChar(idx.config.HeaderPathField, headerPaths),
		)
	}

	// Add dense vectors if embedder is configured
	if idx.config.Embedder != "" {
		embeddings, err := idx.embedDocs(ctx, docs)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings: %w", err)
		}

		vectors := make([][]float32, len(docs))
		for i, emb := range embeddings {
			vectors[i] = float64SliceToFloat32(emb)
		}
		columns = append(columns, column.NewColumnFloatVector(idx.config.DenseField, idx.config.Dimension, vectors))
	}

	// Insert data - BM25 function will auto-generate sparse vectors if enabled
	insertOption := milvusclient.NewColumnBasedInsertOption(idx.config.Collection, columns...)
	_, err := idx.client.Insert(ctx, insertOption)
	if err != nil {
		return nil, fmt.Errorf("failed to insert into milvus: %w", err)
	}

	// Flush to ensure data is searchable
	flushOption := milvusclient.NewFlushOption(idx.config.Collection)
	_, err = idx.client.Flush(ctx, flushOption)
	if err != nil {
		return nil, fmt.Errorf("failed to flush collection: %w", err)
	}

	// Generate returned IDs
	resultIDs := make([]string, len(docs))
	for i, doc := range docs {
		if doc.ID != "" {
			resultIDs[i] = doc.ID
		} else {
			resultIDs[i] = fmt.Sprintf("%d", ids[i])
		}
	}

	return resultIDs, nil
}

// ensureCollection ensures the collection exists with proper schema.
func (idx *MilvusIndexer) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	describeOption := milvusclient.NewDescribeCollectionOption(idx.config.Collection)
	_, err := idx.client.DescribeCollection(ctx, describeOption)
	if err == nil {
		return nil // Collection exists
	}

	// Build schema with new SDK API
	schema := entity.NewSchema()

	// Primary key field
	schema.WithField(entity.NewField().
		WithName(idx.config.IDField).
		WithDataType(entity.FieldTypeInt64).
		WithIsPrimaryKey(true).
		WithIsAutoID(false),
	)

	// Text field with analyzer enabled for BM25
	schema.WithField(entity.NewField().
		WithName(idx.config.TextField).
		WithDataType(entity.FieldTypeVarChar).
		WithMaxLength(65535).
		WithEnableAnalyzer(true), // Enable analyzer for BM25
	)

	// Dense vector field (if embedder configured)
	if idx.config.Embedder != "" {
		schema.WithField(entity.NewField().
			WithName(idx.config.DenseField).
			WithDataType(entity.FieldTypeFloatVector).
			WithTypeParams("dim", fmt.Sprintf("%d", idx.config.Dimension)),
		)
	}

	// BM25 function for full-text search (Milvus 2.5+)
	if idx.config.EnableBM25 {
		// Add sparse vector field for BM25 output
		schema.WithField(entity.NewField().
			WithName(idx.config.SparseField).
			WithDataType(entity.FieldTypeSparseVector),
		)

		// Define BM25 function
		bm25Function := entity.NewFunction().
			WithName("text_bm25_emb").
			WithInputFields(idx.config.TextField).
			WithOutputFields(idx.config.SparseField).
			WithType(entity.FunctionTypeBM25)

		schema.WithFunction(bm25Function)
	}

	// Metadata fields (for document tracking and retrieval)
	if idx.config.EnableMetadata {
		// Source field - document file path
		schema.WithField(entity.NewField().
			WithName(idx.config.SourceField).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(512),
		)

		// Title field - document title
		schema.WithField(entity.NewField().
			WithName(idx.config.TitleField).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(256),
		)

		// Page field - PDF page number
		schema.WithField(entity.NewField().
			WithName(idx.config.PageField).
			WithDataType(entity.FieldTypeInt64),
		)

		// UpdatedAt field - last modification time
		schema.WithField(entity.NewField().
			WithName(idx.config.UpdatedAtField).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(64),
		)

		// ChunkIndex field - chunk index in document
		schema.WithField(entity.NewField().
			WithName(idx.config.ChunkIndexField).
			WithDataType(entity.FieldTypeInt64),
		)

		// ChunkSize field - chunk size in characters
		schema.WithField(entity.NewField().
			WithName(idx.config.ChunkSizeField).
			WithDataType(entity.FieldTypeInt64),
		)

		// HeaderPath field - Markdown section path (stored as JSON string)
		schema.WithField(entity.NewField().
			WithName(idx.config.HeaderPathField).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(1024),
		)
	}

	// Prepare index options
	var indexOptions []milvusclient.CreateIndexOption

	// Add index for dense vector
	if idx.config.Embedder != "" {
		denseIndexOption := milvusclient.NewCreateIndexOption(
			idx.config.Collection,
			idx.config.DenseField,
			index.NewAutoIndex(entity.MetricType(entity.COSINE)),
		)
		indexOptions = append(indexOptions, denseIndexOption)
	}

	// Add index for sparse vector (BM25)
	if idx.config.EnableBM25 {
		// BM25 sparse vectors must use BM25 metric type
		sparseIndex := index.NewSparseInvertedIndex(entity.BM25, 0.1)
		sparseIndexOption := milvusclient.NewCreateIndexOption(
			idx.config.Collection,
			idx.config.SparseField,
			sparseIndex,
		)
		indexOptions = append(indexOptions, sparseIndexOption)
	}

	// Create collection with schema and indexes
	createOption := milvusclient.NewCreateCollectionOption(idx.config.Collection, schema)
	if len(indexOptions) > 0 {
		createOption = createOption.WithIndexOptions(indexOptions...)
	}

	err = idx.client.CreateCollection(ctx, createOption)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Load collection
	loadOption := milvusclient.NewLoadCollectionOption(idx.config.Collection)
	_, err = idx.client.LoadCollection(ctx, loadOption)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	return nil
}

// embedDocs generates dense embeddings for documents using the configured embedder.
func (idx *MilvusIndexer) embedDocs(ctx context.Context, docs []*schema.Document) ([][]float64, error) {
	if idx.g == nil {
		return nil, fmt.Errorf("genkit registry is nil")
	}

	// Lookup embedder from genkit registry
	embedder := genkit.LookupEmbedder(idx.g, idx.config.Embedder)
	if embedder == nil {
		return nil, fmt.Errorf("embedder '%s' not found in registry", idx.config.Embedder)
	}

	// Convert eino documents to genkit documents using utils
	inputDocs := utils.ToGenkitDocuments(docs)

	// Call embedder
	resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
		Input: inputDocs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to embed documents: %w", err)
	}

	// Convert embeddings from float32 to float64
	embeddings := make([][]float64, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		embeddings[i] = make([]float64, len(emb.Embedding))
		for j, v := range emb.Embedding {
			embeddings[i][j] = float64(v)
		}
	}

	return embeddings, nil
}

// float64SliceToFloat32 converts []float64 to []float32.
func float64SliceToFloat32(input []float64) []float32 {
	result := make([]float32, len(input))
	for i, v := range input {
		result[i] = float32(v)
	}
	return result
}

// sliceToJSONString converts a string slice to a simple JSON-like string representation.
// This is used to store arrays like header_path in VARCHAR fields.
func sliceToJSONString(slice []string) string {
	if len(slice) == 0 {
		return "[]"
	}
	result := "["
	for i, s := range slice {
		if i > 0 {
			result += ","
		}
		// Simple escaping: replace " with \"
		escaped := strings.ReplaceAll(s, "\"", "\\\"")
		result += "\"" + escaped + "\""
	}
	result += "]"
	return result
}

// Close closes the Milvus client connection.
func (idx *MilvusIndexer) Close() error {
	if idx.client != nil {
		return idx.client.Close(context.Background())
	}
	return nil
}

// Client returns the underlying Milvus client.
func (idx *MilvusIndexer) Client() *milvusclient.Client {
	return idx.client
}

// ValidateConfig validates Milvus configuration without creating a client.
func ValidateConfig(config *MilvusConfig) error {
	cfg := applyDefaults(config, nil)

	// Load address/token from environment if not provided
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

	if cfg.Dimension <= 0 {
		return fmt.Errorf("milvus dimension must be greater than 0")
	}

	return nil
}

// Flush flushes the collection to ensure data is searchable.
func (idx *MilvusIndexer) Flush(ctx context.Context) error {
	if idx.client == nil {
		return fmt.Errorf("milvus client is not initialized")
	}
	flushOption := milvusclient.NewFlushOption(idx.config.Collection)
	_, err := idx.client.Flush(ctx, flushOption)
	return err
}
