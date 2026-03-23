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

package mergers

import (
	"context"
	"fmt"
	"sort"

	"github.com/cloudwego/eino/schema"
)

// SourceLabel defines source labels for different retrieval paths.
type SourceLabel string

const (
	SourceDense   SourceLabel = "dense"
	SourceSparse  SourceLabel = "sparse"
	SourceHybrid  SourceLabel = "hybrid"
	SourceKeyword SourceLabel = "keyword"
	SourceLocal   SourceLabel = "local"
)

// MergeConfig configures the unified merge layer.
type MergeConfig struct {
	// Deduplication
	EnableDedup   bool         // Enable deduplication (default: true)
	DedupMethod   DedupMethod  // Dedup method (default: "id")
	DedupThreshold float64     // Similarity threshold (default: 0.95)
	IDField       string       // Field to identify duplicates (default: "id")

	// Score normalization
	NormalizeMethod string // "minmax" | "zscore" | "rank" | "softmax" | "sigmoid" | "log" | "none" (default: "minmax")
	ScoreField      string // Metadata field for original score (default: "score")

	// Source tracking
	EnableSourceTracking bool   // Track which sources contributed (default: true)
	SourceField          string // Metadata field for sources (default: "merge_sources")

	// Output
	TopK int // Maximum results to return (0 = no limit)

	// Strategy
	Strategy string // "rrf" | "weighted" | "concat" (default: "rrf")
	RRFK     int    // RRF constant k (default: 60)
}

// DefaultMergeConfig returns default merge configuration.
func DefaultMergeConfig() *MergeConfig {
	return &MergeConfig{
		EnableDedup:          true,
		DedupMethod:          DedupMethodID,
		DedupThreshold:       0.95,
		IDField:              "id",
		NormalizeMethod:      "minmax",
		ScoreField:           "score",
		EnableSourceTracking: true,
		SourceField:          "merge_sources",
		TopK:                 0,
		Strategy:             "rrf",
		RRFK:                 60,
	}
}

// MultiPathResult represents results from a single retrieval path.
type MultiPathResult struct {
	Label   SourceLabel         // Source label
	Results []*schema.Document  // Retrieved documents
	Weight  float64             // Weight for weighted fusion (default: 1.0)
}

// MergeLayer provides unified multi-path result merging with deduplication,
// score normalization, and source tracking.
type MergeLayer struct {
	cfg *MergeConfig
}

// NewMergeLayer creates a new merge layer.
func NewMergeLayer(cfg *MergeConfig) *MergeLayer {
	if cfg == nil {
		cfg = DefaultMergeConfig()
	}
	return &MergeLayer{cfg: cfg}
}

// Merge combines multiple retrieval paths into a single ranked list.
func (m *MergeLayer) Merge(ctx context.Context, paths []*MultiPathResult) ([]*schema.Document, error) {
	if len(paths) == 0 {
		return []*schema.Document{}, nil
	}

	// Step 1: Normalize scores per path
	normalizedPaths := m.normalizeScores(paths)

	// Step 2: Merge and deduplicate
	merged := m.mergeAndDeduplicate(normalizedPaths, paths)

	// Step 3: Sort and apply TopK
	sorted := m.sortAndTruncate(merged)

	// Step 4: Build output with source tracking
	output := m.buildOutput(sorted)

	return output, nil
}

// normalizedPath represents a path with normalized scores.
type normalizedPath struct {
	label  SourceLabel
	docID  string
	doc    *schema.Document
	score  float64
	rank   int
	weight float64
}

func (m *MergeLayer) normalizeScores(paths []*MultiPathResult) [][]normalizedPath {
	normalizer := NewScoreNormalizer(NormalizeMethod(m.cfg.NormalizeMethod))
	result := make([][]normalizedPath, len(paths))

	for pathIdx, path := range paths {
		scores := make([]float64, 0, len(path.Results))
		for _, doc := range path.Results {
			scores = append(scores, m.extractScore(doc))
		}

		// Normalize scores
		normScores := normalizer.Normalize(scores)

		// Build normalized path
		result[pathIdx] = make([]normalizedPath, len(path.Results))
		for docIdx, doc := range path.Results {
			result[pathIdx][docIdx] = normalizedPath{
				label:  path.Label,
				docID:  m.getDocID(doc),
				doc:    doc,
				score:  normScores[docIdx],
				rank:   docIdx + 1,
				weight: path.Weight,
			}
		}
	}

	return result
}

func (m *MergeLayer) extractScore(doc *schema.Document) float64 {
	if doc.MetaData == nil {
		return 1.0
	}
	if s, ok := doc.MetaData[m.cfg.ScoreField]; ok {
		switch v := s.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		}
	}
	return 1.0
}

func (m *MergeLayer) getDocID(doc *schema.Document) string {
	if doc.ID != "" {
		return doc.ID
	}
	if doc.MetaData != nil {
		if idVal, ok := doc.MetaData[m.cfg.IDField]; ok {
			if idStr, ok := idVal.(string); ok && idStr != "" {
				return idStr
			}
		}
	}
	// Fallback to content hash
	return fmt.Sprintf("%p", doc)
}

type mergedDoc struct {
	doc     *schema.Document
	docID   string
	score   float64
	sources map[SourceLabel]struct{}
}

func (m *MergeLayer) mergeAndDeduplicate(normalizedPaths [][]normalizedPath, paths []*MultiPathResult) map[string]*mergedDoc {
	docMap := m.mergeByStrategy(normalizedPaths)

	// Apply deduplication if enabled
	if m.cfg.EnableDedup {
		docMap = m.applyDeduplication(docMap)
	}

	return docMap
}

func (m *MergeLayer) mergeByStrategy(normalizedPaths [][]normalizedPath) map[string]*mergedDoc {
	switch m.cfg.Strategy {
	case "rrf":
		return m.mergeRRF(normalizedPaths)
	case "weighted":
		return m.mergeWeighted(normalizedPaths)
	case "concat":
		return m.mergeConcat(normalizedPaths)
	default:
		return m.mergeRRF(normalizedPaths)
	}
}

func (m *MergeLayer) mergeRRF(normalizedPaths [][]normalizedPath) map[string]*mergedDoc {
	docMap := make(map[string]*mergedDoc)

	for _, path := range normalizedPaths {
		for _, item := range path {
			rrfScore := 1.0 / float64(m.cfg.RRFK+item.rank)

			if existing, found := docMap[item.docID]; found {
				existing.score += rrfScore
				existing.sources[item.label] = struct{}{}
			} else {
				docMap[item.docID] = &mergedDoc{
					doc:     item.doc,
					docID:   item.docID,
					score:   rrfScore,
					sources: map[SourceLabel]struct{}{item.label: {}},
				}
			}
		}
	}

	return docMap
}

func (m *MergeLayer) mergeWeighted(normalizedPaths [][]normalizedPath) map[string]*mergedDoc {
	docMap := make(map[string]*mergedDoc)

	for _, path := range normalizedPaths {
		for _, item := range path {
			weightedScore := item.score * item.weight

			if existing, found := docMap[item.docID]; found {
				existing.score += weightedScore
				existing.sources[item.label] = struct{}{}
			} else {
				docMap[item.docID] = &mergedDoc{
					doc:     item.doc,
					docID:   item.docID,
					score:   weightedScore,
					sources: map[SourceLabel]struct{}{item.label: {}},
				}
			}
		}
	}

	return docMap
}

func (m *MergeLayer) mergeConcat(normalizedPaths [][]normalizedPath) map[string]*mergedDoc {
	docMap := make(map[string]*mergedDoc)

	for _, path := range normalizedPaths {
		for _, item := range path {
			if _, exists := docMap[item.docID]; exists {
				continue // Skip duplicate
			}
			docMap[item.docID] = &mergedDoc{
				doc:     item.doc,
				docID:   item.docID,
				score:   item.score,
				sources: map[SourceLabel]struct{}{item.label: {}},
			}
		}
	}

	return docMap
}

func (m *MergeLayer) applyDeduplication(docMap map[string]*mergedDoc) map[string]*mergedDoc {
	// Convert to document list for deduplication
	docs := make([]*schema.Document, 0, len(docMap))
	idToDocID := make(map[*schema.Document]string)

	for docID, md := range docMap {
		docs = append(docs, md.doc)
		idToDocID[md.doc] = docID
	}

	// Apply configured deduplication
	dedupCfg := &DedupConfig{
		Method:    m.cfg.DedupMethod,
		Threshold: m.cfg.DedupThreshold,
		IDField:   m.cfg.IDField,
		Keep:      "highest_score",
	}
	dedup := NewDeduplicator(dedupCfg)
	uniqueDocs := dedup.Deduplicate(docs)

	// Rebuild map with only unique documents
	result := make(map[string]*mergedDoc)
	for _, doc := range uniqueDocs {
		docID := idToDocID[doc]
		result[docID] = docMap[docID]
	}

	return result
}

func (m *MergeLayer) sortAndTruncate(docMap map[string]*mergedDoc) []*mergedDoc {
	// Convert to slice
	docs := make([]*mergedDoc, 0, len(docMap))
	for _, md := range docMap {
		docs = append(docs, md)
	}

	// Sort by score (descending)
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].score > docs[j].score
	})

	// Apply TopK
	if m.cfg.TopK > 0 && len(docs) > m.cfg.TopK {
		docs = docs[:m.cfg.TopK]
	}

	return docs
}

func (m *MergeLayer) buildOutput(docs []*mergedDoc) []*schema.Document {
	output := make([]*schema.Document, 0, len(docs))

	for _, md := range docs {
		cloned := &schema.Document{
			ID:       md.doc.ID,
			Content:  md.doc.Content,
			MetaData: make(map[string]any),
		}

		// Copy existing metadata
		if md.doc.MetaData != nil {
			for k, v := range md.doc.MetaData {
				cloned.MetaData[k] = v
			}
		}

		// Add merge metadata
		cloned.MetaData["merge_score"] = md.score

		if m.cfg.EnableSourceTracking {
			sources := make([]string, 0, len(md.sources))
			for src := range md.sources {
				sources = append(sources, string(src))
			}
			cloned.MetaData[m.cfg.SourceField] = sources
		}

		output = append(output, cloned)
	}

	return output
}
