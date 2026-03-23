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
	"math"
	"sort"

	"github.com/cloudwego/eino/schema"
)

// docScore represents a document with its associated score.
type docScore struct {
	doc   *schema.Document
	score float64
}

// WeightedConfig holds configuration for weighted score merger.
type WeightedConfig struct {
	Weights     []float64 // Per-path weights (default: equal weights)
	TopK        int       // Maximum results to return (0 = no limit)
	ScoreField  string    // Metadata field for original score (default: "score")
	IDField     string    // Field to identify duplicates (default: "id")
	EnableScore bool      // Include merge score in metadata
	Normalize   string    // Normalization method: "minmax" | "zscore" | "none" (default: "minmax")
}

// DefaultWeightedConfig returns default weighted configuration.
func DefaultWeightedConfig() *WeightedConfig {
	return &WeightedConfig{
		Weights:     nil,
		TopK:        0,
		ScoreField:  "score",
		IDField:     "id",
		EnableScore: true,
		Normalize:   "minmax",
	}
}

// NewWeightedMerger creates a new weighted score merger.
//
// Formula: score(d) = Σ w_i · norm(score_i(d))
// where w_i is the weight for path i and norm normalizes scores.
func NewWeightedMerger(cfg *WeightedConfig) (*WeightedMerger, error) {
	if cfg == nil {
		cfg = DefaultWeightedConfig()
	}
	if cfg.ScoreField == "" {
		cfg.ScoreField = "score"
	}
	if cfg.IDField == "" {
		cfg.IDField = "id"
	}
	if cfg.Normalize == "" {
		cfg.Normalize = "minmax"
	}
	return &WeightedMerger{cfg: cfg}, nil
}

// WeightedMerger merges results using weighted score fusion.
type WeightedMerger struct {
	cfg *WeightedConfig
}

// Merge combines multiple retrieval paths using weighted score fusion.
func (m *WeightedMerger) Merge(ctx context.Context, results [][]*schema.Document) ([]*schema.Document, error) {
	if len(results) == 0 {
		return []*schema.Document{}, nil
	}

	// Setup weights
	weights := m.cfg.Weights
	if weights == nil || len(weights) != len(results) {
		// Equal weights
		weights = make([]float64, len(results))
		for i := range weights {
			weights[i] = 1.0 / float64(len(results))
		}
	}

	// Extract scores per path and normalize
	normalizedPaths := make([][]docScore, len(results))
	for pathIdx, pathResults := range results {
		scores := m.extractScores(pathResults)
		normalized := m.normalize(scores, m.cfg.Normalize)
		normalizedPaths[pathIdx] = normalized
	}

	// Merge with weights
	type mergedDoc struct {
		doc     *schema.Document
		score   float64
		sources []int
	}
	docScores := make(map[string]*mergedDoc)

	for pathIdx, pathResults := range results {
		weight := weights[pathIdx]
		for i, doc := range pathResults {
			docID := m.getDocID(doc)
			normScore := normalizedPaths[pathIdx][i].score
			weightedScore := weight * normScore

			if existing, found := docScores[docID]; found {
				existing.score += weightedScore
				existing.sources = append(existing.sources, pathIdx)
			} else {
				docScores[docID] = &mergedDoc{
					doc:     doc,
					score:   weightedScore,
					sources: []int{pathIdx},
				}
			}
		}
	}

	// Sort by score
	merged := make([]*mergedDoc, 0, len(docScores))
	for _, md := range docScores {
		merged = append(merged, md)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].score > merged[j].score
	})

	// Apply TopK
	topK := m.cfg.TopK
	if topK > 0 && len(merged) > topK {
		merged = merged[:topK]
	}

	// Build output
	output := make([]*schema.Document, 0, len(merged))
	for _, md := range merged {
		doc := md.doc
		cloned := &schema.Document{
			ID:       doc.ID,
			Content:  doc.Content,
			MetaData: make(map[string]any),
		}
		for k, v := range doc.MetaData {
			cloned.MetaData[k] = v
		}
		if m.cfg.EnableScore {
			cloned.MetaData["weighted_score"] = md.score
			cloned.MetaData["merge_sources"] = md.sources
		}
		output = append(output, cloned)
	}

	return output, nil
}

func (m *WeightedMerger) extractScores(docs []*schema.Document) []docScore {
	scores := make([]docScore, len(docs))
	for i, doc := range docs {
		score := 1.0 // default
		if doc.MetaData != nil {
			if s, ok := doc.MetaData[m.cfg.ScoreField]; ok {
				switch v := s.(type) {
				case float64:
					score = v
				case float32:
					score = float64(v)
				case int:
					score = float64(v)
				}
			}
		}
		scores[i] = docScore{doc: doc, score: score}
	}
	return scores
}

func (m *WeightedMerger) normalize(scores []docScore, method string) []docScore {
	if len(scores) == 0 || method == "none" {
		return scores
	}

	result := make([]docScore, len(scores))

	switch method {
	case "minmax":
		min, max := scores[0].score, scores[0].score
		for _, s := range scores {
			if s.score < min {
				min = s.score
			}
			if s.score > max {
				max = s.score
			}
		}
		rangeVal := max - min
		if rangeVal == 0 {
			for i, s := range scores {
				result[i] = docScore{doc: s.doc, score: 1.0}
			}
		} else {
			for i, s := range scores {
				result[i] = docScore{doc: s.doc, score: (s.score - min) / rangeVal}
			}
		}

	case "zscore":
		// Calculate mean and std dev
		n := float64(len(scores))
		sum, sumSq := 0.0, 0.0
		for _, s := range scores {
			sum += s.score
			sumSq += s.score * s.score
		}
		mean := sum / n
		variance := (sumSq / n) - (mean * mean)
		stdDev := math.Sqrt(variance)

		if stdDev == 0 {
			for i, s := range scores {
				result[i] = docScore{doc: s.doc, score: 1.0}
			}
		} else {
			for i, s := range scores {
				result[i] = docScore{doc: s.doc, score: (s.score-mean)/stdDev + 3} // shift to positive
			}
		}
	default:
		return scores
	}

	return result
}

func (m *WeightedMerger) getDocID(doc *schema.Document) string {
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
