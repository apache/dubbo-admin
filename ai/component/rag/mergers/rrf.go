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

// RRFConfig holds configuration for RRF merger.
type RRFConfig struct {
	K           int  // RRF constant (default: 60)
	TopK        int  // Maximum results to return (0 = no limit)
	IDField     string // Field to identify duplicates (default: "id")
	EnableScore bool   // Include merge score in metadata
}

// DefaultRRFConfig returns default RRF configuration.
func DefaultRRFConfig() *RRFConfig {
	return &RRFConfig{
		K:           60,
		TopK:        0,
		IDField:     "id",
		EnableScore: true,
	}
}

// NewRRFMerger creates a new RRF (Reciprocal Rank Fusion) merger.
//
// RRF formula: score(d) = Σ 1 / (k + rank_i(d))
// where rank_i(d) is the rank of document d in path i (1-indexed).
func NewRRFMerger(cfg *RRFConfig) (*RRFMerger, error) {
	if cfg == nil {
		cfg = DefaultRRFConfig()
	}
	if cfg.K <= 0 {
		cfg.K = 60
	}
	return &RRFMerger{cfg: cfg}, nil
}

// RRFMerger merges results using Reciprocal Rank Fusion.
type RRFMerger struct {
	cfg *RRFConfig
}

// Merge combines multiple retrieval paths into a single ranked list using RRF.
func (m *RRFMerger) Merge(ctx context.Context, results [][]*schema.Document) ([]*schema.Document, error) {
	if len(results) == 0 {
		return []*schema.Document{}, nil
	}

	// Track documents by ID for deduplication
	type scoredDoc struct {
		doc     *schema.Document
		score   float64
		sources []int
	}
	docScores := make(map[string]*scoredDoc)

	// Process each retrieval path
	for pathIdx, pathResults := range results {
		for rank, doc := range pathResults {
			docID := m.getDocID(doc)

			if existing, found := docScores[docID]; found {
				existing.score += m.rrfScore(rank + 1)
				existing.sources = append(existing.sources, pathIdx)
			} else {
				docScores[docID] = &scoredDoc{
					doc:     doc,
					score:   m.rrfScore(rank + 1),
					sources: []int{pathIdx},
				}
			}
		}
	}

	// Sort by score
	merged := make([]*scoredDoc, 0, len(docScores))
	for _, sd := range docScores {
		merged = append(merged, sd)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].score > merged[j].score
	})

	// Apply TopK limit
	topK := m.cfg.TopK
	if topK > 0 && len(merged) > topK {
		merged = merged[:topK]
	}

	// Build output with metadata
	output := make([]*schema.Document, 0, len(merged))
	for _, sd := range merged {
		doc := sd.doc
		cloned := &schema.Document{
			ID:       doc.ID,
			Content:  doc.Content,
			MetaData: make(map[string]any),
		}
		// Copy existing metadata
		for k, v := range doc.MetaData {
			cloned.MetaData[k] = v
		}
		// Add merge metadata
		if m.cfg.EnableScore {
			cloned.MetaData["rrf_score"] = sd.score
			cloned.MetaData["merge_sources"] = sd.sources
		}

		output = append(output, cloned)
	}

	return output, nil
}

func (m *RRFMerger) rrfScore(rank int) float64 {
	return 1.0 / float64(m.cfg.K+rank)
}

func (m *RRFMerger) getDocID(doc *schema.Document) string {
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
	return fmt.Sprintf("%p", doc)
}
