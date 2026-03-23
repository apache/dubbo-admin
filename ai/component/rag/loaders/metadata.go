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

package loaders

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

// MetadataKey defines standard metadata keys used across the RAG pipeline.
type MetadataKey string

const (
	// Document-level metadata (injected by Loader)
	MetaSource    MetadataKey = "source"     // Original file path
	MetaTitle     MetadataKey = "title"      // Document title
	MetaUpdatedAt MetadataKey = "updated_at" // Last modification time (RFC3339)
	MetaFileSize  MetadataKey = "file_size"  // File size in bytes
	MetaFileType  MetadataKey = "file_type"  // File extension (pdf, md, txt, etc.)

	// Page-level metadata (injected by Parser, mainly for PDF)
	MetaPage MetadataKey = "page" // PDF page number (1-based)

	// Chunk-level metadata (injected by Splitter)
	MetaHeaderPath  MetadataKey = "header_path"  // Markdown section path
	MetaHeaderLevel MetadataKey = "header_level" // Current heading depth
	MetaChunkIndex  MetadataKey = "chunk_index"  // Index of this chunk in the document
	MetaChunkStart  MetadataKey = "chunk_start"  // Start position in original document
	MetaChunkSize   MetadataKey = "chunk_size"   // Size of this chunk in characters

	// Retrieval metadata (added during retrieval)
	MetaRetrieveScore MetadataKey = "retrieve_score" // Original retrieval score
	MetaRerankScore   MetadataKey = "rerank_score"   // Reranking score
	MetaRecallSource  MetadataKey = "recall_source"  // Retrieval method
)

func (k MetadataKey) String() string {
	return string(k)
}

// ExtractFileMetadata extracts metadata from a file path and info.
func ExtractFileMetadata(sourcePath string, fileInfo os.FileInfo) map[string]any {
	meta := make(map[string]any)

	meta[MetaSource.String()] = sourcePath
	meta[MetaUpdatedAt.String()] = fileInfo.ModTime().Format(time.RFC3339)
	meta[MetaFileSize.String()] = fileInfo.Size()

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(sourcePath)), ".")
	meta[MetaFileType.String()] = ext

	title := extractTitleFromPath(sourcePath)
	meta[MetaTitle.String()] = title

	return meta
}

// ExtractTitleFromContent attempts to extract title from document content.
func ExtractTitleFromContent(content string, fileType string, currentTitle string) string {
	if currentTitle != "" && len(currentTitle) > 3 && len(currentTitle) < 200 {
		return currentTitle
	}

	if fileType == "md" {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") {
				title := strings.TrimLeft(line, "#")
				title = strings.TrimSpace(title)
				if title != "" {
					return title
				}
			}
			if line != "" && !strings.HasPrefix(line, "#") {
				break
			}
		}
	}

	return currentTitle
}

func extractTitleFromPath(path string) string {
	filename := filepath.Base(path)
	title := strings.TrimSuffix(filename, filepath.Ext(filename))

	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.TrimSuffix(title, ".")
	title = strings.TrimSpace(title)

	return title
}

// InheritMetadata creates metadata for a chunk by inheriting from parent.
func InheritMetadata(parentMeta map[string]any, chunkSpecific map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range parentMeta {
		result[k] = v
	}
	for k, v := range chunkSpecific {
		result[k] = v
	}
	return result
}

// MergeMetadata merges multiple metadata maps.
func MergeMetadata(maps ...map[string]any) map[string]any {
	result := make(map[string]any)
	for _, m := range maps {
		if m != nil {
			for k, v := range m {
				result[k] = v
			}
		}
	}
	return result
}

// Builder helps build and manipulate document metadata.
type Builder struct {
	meta map[string]any
}

// NewBuilder creates a new Builder with optional initial metadata.
func NewBuilder(initial ...map[string]any) *Builder {
	b := &Builder{
		meta: make(map[string]any),
	}
	if len(initial) > 0 && initial[0] != nil {
		for k, v := range initial[0] {
			b.meta[k] = v
		}
	}
	return b
}

// Set sets a metadata key-value pair.
func (b *Builder) Set(key MetadataKey, value any) *Builder {
	if value != nil {
		b.meta[key.String()] = value
	}
	return b
}

// SetString sets a string metadata value.
func (b *Builder) SetString(key MetadataKey, value string) *Builder {
	if value != "" {
		b.meta[key.String()] = value
	}
	return b
}

// SetInt sets an int metadata value.
func (b *Builder) SetInt(key MetadataKey, value int) *Builder {
	b.meta[key.String()] = value
	return b
}

// SetAll sets multiple metadata key-value pairs.
func (b *Builder) SetAll(values map[string]any) *Builder {
	if values != nil {
		for k, v := range values {
			if v != nil {
				b.meta[k] = v
			}
		}
	}
	return b
}

// Get gets a metadata value by key.
func (b *Builder) Get(key MetadataKey) any {
	return b.meta[key.String()]
}

// Has checks if a metadata key exists.
func (b *Builder) Has(key MetadataKey) bool {
	_, exists := b.meta[key.String()]
	return exists
}

// Delete removes a metadata key.
func (b *Builder) Delete(key MetadataKey) *Builder {
	delete(b.meta, key.String())
	return b
}

// Build returns the metadata map.
func (b *Builder) Build() map[string]any {
	return b.meta
}

// Clone creates a copy of the builder.
func (b *Builder) Clone() *Builder {
	newMeta := make(map[string]any, len(b.meta))
	for k, v := range b.meta {
		newMeta[k] = v
	}
	return &Builder{meta: newMeta}
}

// ApplyToDocument applies the metadata to a document.
func (b *Builder) ApplyToDocument(doc *schema.Document) *schema.Document {
	if doc == nil {
		return nil
	}
	if doc.MetaData == nil {
		doc.MetaData = make(map[string]any)
	}
	for k, v := range b.meta {
		doc.MetaData[k] = v
	}
	return doc
}

// ChunkBuilder is a specialized builder for chunk metadata.
type ChunkBuilder struct {
	*Builder
	parentMeta map[string]any
	chunkIndex int
}

// NewChunkBuilder creates a new ChunkBuilder with parent metadata.
func NewChunkBuilder(parentMeta map[string]any) *ChunkBuilder {
	return &ChunkBuilder{
		Builder:    NewBuilder(),
		parentMeta: parentMeta,
		chunkIndex: 0,
	}
}

// SetChunkIndex sets the chunk index and advances it.
func (b *ChunkBuilder) SetChunkIndex(index int) *ChunkBuilder {
	b.chunkIndex = index
	b.SetInt(MetaChunkIndex, index)
	return b
}

// NextChunk advances to the next chunk index.
func (b *ChunkBuilder) NextChunk() *ChunkBuilder {
	b.chunkIndex++
	b.SetInt(MetaChunkIndex, b.chunkIndex)
	return b
}

// SetChunkRange sets the start position and size of this chunk.
func (b *ChunkBuilder) SetChunkRange(start, size int) *ChunkBuilder {
	b.SetInt(MetaChunkStart, start)
	b.SetInt(MetaChunkSize, size)
	return b
}

// SetHeaderPath sets the Markdown section path for this chunk.
func (b *ChunkBuilder) SetHeaderPath(path []string) *ChunkBuilder {
	if len(path) > 0 {
		b.meta[MetaHeaderPath.String()] = path
		b.SetInt(MetaHeaderLevel, len(path))
	}
	return b
}

// SetPage sets the PDF page number for this chunk.
func (b *ChunkBuilder) SetPage(page int) *ChunkBuilder {
	b.SetInt(MetaPage, page)
	return b
}

// InheritFromParent copies all parent metadata that isn't already set.
func (b *ChunkBuilder) InheritFromParent() *ChunkBuilder {
	if b.parentMeta != nil {
		for k, v := range b.parentMeta {
			if _, exists := b.meta[k]; !exists {
				b.meta[k] = v
			}
		}
	}
	return b
}

// BuildChunk builds the final chunk metadata.
func (b *ChunkBuilder) BuildChunk() map[string]any {
	b.InheritFromParent()
	return b.meta
}

// ApplyToChunk applies the metadata to a chunk document.
func (b *ChunkBuilder) ApplyToChunk(doc *schema.Document) *schema.Document {
	b.BuildChunk()
	b.ApplyToDocument(doc)
	return doc
}

// BuildChunkMetadata is a helper function for building chunk metadata.
func BuildChunkMetadata(parentMeta map[string]any, chunkIndex, start, size int, headerPath []string) map[string]any {
	return NewChunkBuilder(parentMeta).
		SetChunkIndex(chunkIndex).
		SetChunkRange(start, size).
		SetHeaderPath(headerPath).
		BuildChunk()
}

// CopyDocumentMetadata creates a copy of a document's metadata.
func CopyDocumentMetadata(doc *schema.Document) map[string]any {
	if doc == nil || doc.MetaData == nil {
		return make(map[string]any)
	}
	result := make(map[string]any, len(doc.MetaData))
	for k, v := range doc.MetaData {
		result[k] = v
	}
	return result
}

// GetMetadata safely retrieves a metadata value by key.
func GetMetadata(meta map[string]any, key MetadataKey, defaultValue any) any {
	if meta == nil {
		return defaultValue
	}
	if val, ok := meta[key.String()]; ok {
		return val
	}
	return defaultValue
}

// GetMetadataString retrieves a string metadata value.
func GetMetadataString(meta map[string]any, key MetadataKey, defaultValue string) string {
	val := GetMetadata(meta, key, defaultValue)
	if str, ok := val.(string); ok {
		return str
	}
	return defaultValue
}

// GetMetadataInt retrieves an int metadata value.
func GetMetadataInt(meta map[string]any, key MetadataKey, defaultValue int) int {
	val := GetMetadata(meta, key, defaultValue)
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return defaultValue
}

// GetMetadataSlice retrieves a slice metadata value.
func GetMetadataSlice(meta map[string]any, key MetadataKey) []string {
	val := GetMetadata(meta, key, nil)
	if val == nil {
		return nil
	}
	if slice, ok := val.([]string); ok {
		return slice
	}
	if slice, ok := val.([]any); ok {
		result := make([]string, 0, len(slice))
		for _, item := range slice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}
