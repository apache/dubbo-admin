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
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// DedupMethod defines deduplication methods.
type DedupMethod string

const (
	DedupMethodID       DedupMethod = "id"        // Exact ID match
	DedupMethodContent  DedupMethod = "content"   // Content hash similarity
	DedupMethodJaccard  DedupMethod = "jaccard"   // Jaccard similarity on word sets
	DedupMethodCosine   DedupMethod = "cosine"    // Cosine similarity on TF-IDF vectors
	DedupMethodHybrid   DedupMethod = "hybrid"    // Combine multiple methods
)

// DedupConfig configures deduplication behavior.
type DedupConfig struct {
	Method     DedupMethod // Deduplication method (default: "id")
	Threshold  float64     // Similarity threshold (0-1, default: 0.95)
	IDField    string      // Field to use for ID-based dedup (default: "id")
	Keep       string      // Which doc to keep: "first" | "highest_score" (default: "highest_score")
	ScoreField string      // Metadata field for score comparison (default: "score")
}

// DefaultDedupConfig returns default deduplication configuration.
func DefaultDedupConfig() *DedupConfig {
	return &DedupConfig{
		Method:     DedupMethodID,
		Threshold:  0.95,
		IDField:    "id",
		Keep:       "highest_score",
		ScoreField: "score",
	}
}

// Deduplicator removes duplicate documents based on configured method.
type Deduplicator struct {
	cfg *DedupConfig
}

// NewDeduplicator creates a new deduplicator.
func NewDeduplicator(cfg *DedupConfig) *Deduplicator {
	if cfg == nil {
		cfg = DefaultDedupConfig()
	}
	return &Deduplicator{cfg: cfg}
}

// Deduplicate removes duplicates from the document list.
func (d *Deduplicator) Deduplicate(docs []*schema.Document) []*schema.Document {
	if len(docs) <= 1 {
		return docs
	}

	switch d.cfg.Method {
	case DedupMethodID:
		return d.dedupByID(docs)
	case DedupMethodContent:
		return d.dedupByContentHash(docs)
	case DedupMethodJaccard:
		return d.dedupByJaccard(docs)
	case DedupMethodCosine:
		return d.dedupByCosine(docs)
	case DedupMethodHybrid:
		return d.dedupHybrid(docs)
	default:
		return d.dedupByID(docs)
	}
}

// dedupByID removes exact ID matches.
func (d *Deduplicator) dedupByID(docs []*schema.Document) []*schema.Document {
	type group struct {
		docs  []*schema.Document
		best  *schema.Document
		bestS float64
	}

	groups := make(map[string]*group)

	for _, doc := range docs {
		id := d.extractID(doc)
		if id == "" {
			// No ID, use content hash as fallback ID
			id = fmt.Sprintf("hash_%d", hashString(doc.Content))
		}

		if g, exists := groups[id]; exists {
			g.docs = append(g.docs, doc)
			score := d.extractScore(doc)
			if score > g.bestS {
				g.best = doc
				g.bestS = score
			}
		} else {
			groups[id] = &group{
				docs:  []*schema.Document{doc},
				best:  doc,
				bestS: d.extractScore(doc),
			}
		}
	}

	// Keep best from each group
	result := make([]*schema.Document, 0, len(groups))
	for _, g := range groups {
		if d.cfg.Keep == "highest_score" {
			result = append(result, g.best)
		} else {
			result = append(result, g.docs[0])
		}
	}

	return result
}

// dedupByContentHash removes near-duplicates using content hash similarity.
func (d *Deduplicator) dedupByContentHash(docs []*schema.Document) []*schema.Document {
	// Use simple hash-based grouping
	// For production, consider MinHash or SimHash

	type docWithHash struct {
		doc  *schema.Document
		hash uint64
	}

	hashed := make([]docWithHash, len(docs))
	for i, doc := range docs {
		hashed[i] = docWithHash{
			doc:  doc,
			hash: hashString(doc.Content),
		}
	}

	// Group similar hashes (within threshold)
	kept := make([]*schema.Document, 0)
	seen := make(map[uint64]bool)

	for _, item := range hashed {
		if seen[item.hash] {
			continue
		}

		// Mark this hash and nearby hashes as seen
		seen[item.hash] = true
		kept = append(kept, item.doc)

		// Mark similar hashes as duplicates
		for _, other := range hashed {
			if other.hash != item.hash && hashSimilarity(item.hash, other.hash) >= d.cfg.Threshold {
				seen[other.hash] = true
			}
		}
	}

	return kept
}

// dedupByJaccard removes duplicates using Jaccard similarity on word sets.
func (d *Deduplicator) dedupByJaccard(docs []*schema.Document) []*schema.Document {
	kept := make([]*schema.Document, 0)
	removed := make(map[int]bool)

	for i, doc := range docs {
		if removed[i] {
			continue
		}

		kept = append(kept, doc)
		wordsI := tokenize(doc.Content)

		for j := i + 1; j < len(docs); j++ {
			if removed[j] {
				continue
			}

			wordsJ := tokenize(docs[j].Content)
			sim := jaccardSimilarity(wordsI, wordsJ)

			if sim >= d.cfg.Threshold {
				removed[j] = true
			}
		}
	}

	return kept
}

// dedupByCosine removes duplicates using cosine similarity on TF-IDF vectors.
func (d *Deduplicator) dedupByCosine(docs []*schema.Document) []*schema.Document {
	// Build vocabulary
	vocab := buildVocab(docs)

	// Compute TF-IDF vectors
	vectors := make([]map[string]float64, len(docs))
	for i, doc := range docs {
		words := tokenize(doc.Content)
		vectors[i] = computeTFIDF(words, vocab, docs)
	}

	// Find and remove duplicates
	kept := make([]*schema.Document, 0)
	removed := make(map[int]bool)

	for i := range docs {
		if removed[i] {
			continue
		}

		kept = append(kept, docs[i])

		for j := i + 1; j < len(docs); j++ {
			if removed[j] {
				continue
			}

			sim := cosineSimilarity(vectors[i], vectors[j])
			if sim >= d.cfg.Threshold {
				removed[j] = true
			}
		}
	}

	return kept
}

// dedupHybrid combines multiple deduplication methods.
func (d *Deduplicator) dedupHybrid(docs []*schema.Document) []*schema.Document {
	// First pass: exact ID dedup
	idDedup := d.dedupByID(docs)

	// Second pass: content similarity
	contentDedup := d.dedupByContentHash(idDedup)

	// Third pass: Jaccard similarity
	return d.dedupByJaccard(contentDedup)
}

// Helper functions

func (d *Deduplicator) extractID(doc *schema.Document) string {
	if doc.ID != "" {
		return doc.ID
	}
	if doc.MetaData != nil {
		if id, ok := doc.MetaData[d.cfg.IDField].(string); ok {
			return id
		}
	}
	return ""
}

func (d *Deduplicator) extractScore(doc *schema.Document) float64 {
	if doc.MetaData == nil {
		return 0
	}
	if score, ok := doc.MetaData[d.cfg.ScoreField].(float64); ok {
		return score
	}
	return 0
}

// hashString computes a simple hash of a string.
func hashString(s string) uint64 {
	// DJB2 hash algorithm
	h := uint64(5381)
	for _, c := range s {
		h = ((h << 5) + h) + uint64(c)
	}
	return h
}

// hashSimilarity computes similarity between two hashes (0-1).
func hashSimilarity(h1, h2 uint64) float64 {
	// Simple bit similarity
	xor := h1 ^ h2
	diffBits := 0
	for xor > 0 {
		diffBits++
		xor &= xor - 1
	}
	return 1.0 - float64(diffBits)/64.0
}

// tokenize splits text into words.
func tokenize(text string) map[string]bool {
	words := make(map[string]bool)
	current := make([]rune, 0)

	for _, ch := range text {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			current = append(current, ch)
		} else {
			if len(current) > 0 {
				words[strings.ToLower(string(current))] = true
				current = current[:0]
			}
		}
	}
	if len(current) > 0 {
		words[strings.ToLower(string(current))] = true
	}

	return words
}

// jaccardSimilarity computes Jaccard similarity between two word sets.
func jaccardSimilarity(set1, set2 map[string]bool) float64 {
	if len(set1) == 0 && len(set2) == 0 {
		return 1.0
	}

	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// buildVocabulary builds a vocabulary from all documents.
func buildVocab(docs []*schema.Document) map[string]int {
	vocab := make(map[string]int)
	idx := 0

	for _, doc := range docs {
		words := tokenize(doc.Content)
		for word := range words {
			if _, exists := vocab[word]; !exists {
				vocab[word] = idx
				idx++
			}
		}
	}

	return vocab
}

// computeTFIDF computes TF-IDF vector for a document.
func computeTFIDF(words map[string]bool, vocab map[string]int, docs []*schema.Document) map[string]float64 {
	// Compute term frequency
	tf := make(map[string]int)
	for word := range words {
		tf[word]++
	}

	// Compute document frequency
	df := make(map[string]int)
	for _, doc := range docs {
		docWords := tokenize(doc.Content)
		for word := range docWords {
			df[word]++
		}
	}

	// Compute TF-IDF
	vector := make(map[string]float64)
	maxTF := 1
	for _, count := range tf {
		if count > maxTF {
			maxTF = count
		}
	}

	for word, count := range tf {
		// Normalized TF
		normalizedTF := float64(count) / float64(maxTF)
		// IDF
		idf := math.Log(float64(len(docs)+1) / float64(df[word]+1))
		vector[word] = normalizedTF * idf
	}

	return vector
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(v1, v2 map[string]float64) float64 {
	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	// Compute dot product and norms
	allKeys := make(map[string]bool)
	for k := range v1 {
		allKeys[k] = true
	}
	for k := range v2 {
		allKeys[k] = true
	}

	for k := range allKeys {
		v1Val := v1[k]
		v2Val := v2[k]
		dotProduct += v1Val * v2Val
		norm1 += v1Val * v1Val
		norm2 += v2Val * v2Val
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// BatchDeduplicate deduplicates multiple document lists and returns merged unique results.
func (d *Deduplicator) BatchDeduplicate(docLists [][]*schema.Document) []*schema.Document {
	// Flatten all documents
	allDocs := make([]*schema.Document, 0)
	for _, list := range docLists {
		allDocs = append(allDocs, list...)
	}

	// Sort by score if keeping highest score
	if d.cfg.Keep == "highest_score" {
		sort.Slice(allDocs, func(i, j int) bool {
			return d.extractScore(allDocs[i]) > d.extractScore(allDocs[j])
		})
	}

	return d.Deduplicate(allDocs)
}
