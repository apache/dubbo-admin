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

	"github.com/cloudwego/eino/schema"
)

// ConcatConfig holds configuration for concatenation merger.
type ConcatConfig struct {
	TopK        int    // Maximum results to return (0 = no limit)
	IDField     string // Field to identify duplicates (default: "id")
	KeepOrder   bool   // Keep original path order (default: true)
	EnableScore bool   // Include merge metadata
}

// DefaultConcatConfig returns default concat configuration.
func DefaultConcatConfig() *ConcatConfig {
	return &ConcatConfig{
		TopK:        0,
		IDField:     "id",
		KeepOrder:   true,
		EnableScore: true,
	}
}

// NewConcatMerger creates a new concatenation merger.
//
// Strategy: Concatenate results from all paths, removing duplicates.
// First occurrence is kept, subsequent duplicates are skipped.
func NewConcatMerger(cfg *ConcatConfig) (*ConcatMerger, error) {
	if cfg == nil {
		cfg = DefaultConcatConfig()
	}
	if cfg.IDField == "" {
		cfg.IDField = "id"
	}
	return &ConcatMerger{cfg: cfg}, nil
}

// ConcatMerger merges results by simple concatenation with deduplication.
type ConcatMerger struct {
	cfg *ConcatConfig
}

// Merge combines multiple retrieval paths by concatenation.
func (m *ConcatMerger) Merge(ctx context.Context, results [][]*schema.Document) ([]*schema.Document, error) {
	if len(results) == 0 {
		return []*schema.Document{}, nil
	}

	seen := make(map[string]bool)
	output := make([]*schema.Document, 0)

	for pathIdx, pathResults := range results {
		for _, doc := range pathResults {
			docID := m.getDocID(doc)

			if seen[docID] {
				continue // Skip duplicate
			}
			seen[docID] = true

			cloned := &schema.Document{
				ID:       doc.ID,
				Content:  doc.Content,
				MetaData: make(map[string]any),
			}
			for k, v := range doc.MetaData {
				cloned.MetaData[k] = v
			}
			if m.cfg.EnableScore {
				cloned.MetaData["merge_path"] = pathIdx
			}

			output = append(output, cloned)

			// Apply TopK limit
			if m.cfg.TopK > 0 && len(output) >= m.cfg.TopK {
				return output, nil
			}
		}
	}

	return output, nil
}

func (m *ConcatMerger) getDocID(doc *schema.Document) string {
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
