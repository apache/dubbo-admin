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

import "math"

// NormalizeMethod defines score normalization methods.
type NormalizeMethod string

const (
	NormalizeNone   NormalizeMethod = "none"   // No normalization
	NormalizeMinMax NormalizeMethod = "minmax" // Min-max scaling to [0, 1]
	NormalizeZScore NormalizeMethod = "zscore" // Z-score standardization
	NormalizeRank   NormalizeMethod = "rank"   // Rank-based normalization
	NormalizeSoftmax NormalizeMethod = "softmax" // Softmax transformation
	NormalizeSigmoid NormalizeMethod = "sigmoid" // Sigmoid transformation
	NormalizeLog    NormalizeMethod = "log"    // Logarithmic scaling
)

// ScoreNormalizer normalizes scores to a common scale.
type ScoreNormalizer struct {
	method NormalizeMethod
}

// NewScoreNormalizer creates a new score normalizer.
func NewScoreNormalizer(method NormalizeMethod) *ScoreNormalizer {
	if method == "" {
		method = NormalizeMinMax
	}
	return &ScoreNormalizer{method: method}
}

// Normalize normalizes a slice of scores.
func (n *ScoreNormalizer) Normalize(scores []float64) []float64 {
	if len(scores) == 0 || n.method == NormalizeNone {
		return scores
	}

	result := make([]float64, len(scores))

	switch n.method {
	case NormalizeMinMax:
		result = n.minMax(scores)
	case NormalizeZScore:
		result = n.zScore(scores)
	case NormalizeRank:
		result = n.rank(scores)
	case NormalizeSoftmax:
		result = n.softmax(scores)
	case NormalizeSigmoid:
		result = n.sigmoid(scores)
	case NormalizeLog:
		result = n.logScale(scores)
	default:
		return scores
	}

	return result
}

// minMax scales scores to [0, 1] range.
func (n *ScoreNormalizer) minMax(scores []float64) []float64 {
	result := make([]float64, len(scores))

	min, max := scores[0], scores[0]
	for _, s := range scores {
		if s < min {
			min = s
		}
		if s > max {
			max = s
		}
	}

	rangeVal := max - min
	if rangeVal == 0 {
		for i := range result {
			result[i] = 1.0
		}
		return result
	}

	for i, s := range scores {
		result[i] = (s - min) / rangeVal
	}

	return result
}

// zScore standardizes scores to mean=0, std=1, then shifts to positive.
func (n *ScoreNormalizer) zScore(scores []float64) []float64 {
	result := make([]float64, len(scores))

	// Calculate mean
	count := float64(len(scores))
	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	mean := sum / count

	// Calculate standard deviation
	sumSq := 0.0
	for _, s := range scores {
		diff := s - mean
		sumSq += diff * diff
	}
	stdDev := math.Sqrt(sumSq / count)

	if stdDev == 0 {
		for i := range result {
			result[i] = 1.0
		}
		return result
	}

	// Standardize and shift to positive
	for i, s := range scores {
		result[i] = (s-mean)/stdDev + 3 // +3 to shift to positive range
	}

	return result
}

// rank normalizes by rank: score = 1 - (rank-1)/n.
func (n *ScoreNormalizer) rank(scores []float64) []float64 {
	result := make([]float64, len(scores))

	// Sort indices by score
	type idxScore struct {
		idx   int
		score float64
	}
	sorted := make([]idxScore, len(scores))
	for i, s := range scores {
		sorted[i] = idxScore{idx: i, score: s}
	}

	// Sort descending
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].score < sorted[j].score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Assign rank-based scores
	count := float64(len(scores))
	for rank, item := range sorted {
		result[item.idx] = 1.0 - (float64(rank) / count)
	}

	return result
}

// softmax transforms scores using exp(s) / sum(exp(s)).
func (n *ScoreNormalizer) softmax(scores []float64) []float64 {
	result := make([]float64, len(scores))

	// Find max for numerical stability
	max := scores[0]
	for _, s := range scores {
		if s > max {
			max = s
		}
	}

	// Compute exp and sum
	sum := 0.0
	exps := make([]float64, len(scores))
	for i, s := range scores {
		exps[i] = math.Exp(s - max)
		sum += exps[i]
	}

	if sum == 0 {
		for i := range result {
			result[i] = 1.0 / float64(len(scores))
		}
		return result
	}

	for i, exp := range exps {
		result[i] = exp / sum
	}

	return result
}

// sigmoid transforms scores using 1 / (1 + exp(-x)).
func (n *ScoreNormalizer) sigmoid(scores []float64) []float64 {
	result := make([]float64, len(scores))

	for i, s := range scores {
		result[i] = 1.0 / (1.0 + math.Exp(-s))
	}

	return result
}

// logScale applies logarithmic scaling.
func (n *ScoreNormalizer) logScale(scores []float64) []float64 {
	result := make([]float64, len(scores))

	// Find min for offset (assuming scores are positive or we shift them)
	min := scores[0]
	for _, s := range scores {
		if s < min {
			min = s
		}
	}

	offset := 0.0
	if min <= 0 {
		offset = -min + 1.0 // Shift to make all values positive
	}

	// Apply log
	sum := 0.0
	logs := make([]float64, len(scores))
	for i, s := range scores {
		logs[i] = math.Log(s + offset)
		sum += logs[i]
	}

	// Normalize to [0, 1]
	if sum == 0 {
		for i := range result {
			result[i] = 1.0 / float64(len(scores))
		}
		return result
	}

	for i, log := range logs {
		result[i] = log / sum
	}

	return result
}
