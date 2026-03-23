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

package query

import (
	"context"
	"time"
)

// Processor defines the interface for query understanding operations.
type Processor interface {
	// Process processes the input query and returns the result.
	// If processing fails, returns the original query (fallback behavior).
	Process(ctx context.Context, query string) (*Result, error)
}

// Result represents the output of query processing.
type Result struct {
	Query      string   // Processed query (may be original if no changes)
	Queries    []string // Multiple queries (for expansion)
	Intent     string   // Detected intent (for intent recognition)
	Hypothetical string // Hypothetical document (for HyDE)
	Modified   bool     // Whether the query was modified
	Metadata   map[string]any // Additional metadata
}

// Config defines configuration for query processing.
type Config struct {
	// Processing steps to enable (executed in order)
	EnableIntent     bool
	EnableRewrite    bool
	EnableExpansion  bool
	EnableHyDE       bool

	// Model configuration
	Model           string
	Timeout         time.Duration
	Temperature     float64

	// Fallback behavior
	FallbackOnError bool // Return original query on error (default: true)
}

// DefaultConfig returns default query processor configuration.
func DefaultConfig() *Config {
	return &Config{
		EnableIntent:     false,
		EnableRewrite:    false,
		EnableExpansion:  false,
		EnableHyDE:       false,
		Model:            "dashscope/qwen-max",
		Timeout:          5 * time.Second,
		Temperature:      0.3,
		FallbackOnError:  true,
	}
}

// StepType represents a query processing step type.
type StepType string

const (
	StepIntent     StepType = "intent"     // Intent recognition
	StepRewrite    StepType = "rewrite"    // Query rewrite
	StepExpansion  StepType = "expansion"  // Query expansion
	StepHyDE       StepType = "hyde"       // Hypothetical document embeddings
)

// Step represents a single query processing step.
type Step interface {
	// Name returns the step name.
	Name() string
	// Type returns the step type.
	Type() StepType
	// Process executes the step on the query.
	Process(ctx context.Context, query string) (*Result, error)
	// Enabled returns whether the step is enabled.
	Enabled() bool
}

// Layer provides unified query understanding with multiple processing steps.
type Layer struct {
	steps []Step
	cfg   *Config
}

// NewLayer creates a new query processing layer.
func NewLayer(cfg *Config, steps ...Step) *Layer {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Layer{
		steps: steps,
		cfg:   cfg,
	}
}

// Process executes all enabled steps in sequence.
// Each step receives the output of the previous step.
// Returns the final result or original query on error.
func (l *Layer) Process(ctx context.Context, query string) (*Result, error) {
	if query == "" {
		return nil, ErrEmptyQuery
	}

	current := query
	result := &Result{
		Query:    query,
		Modified: false,
		Metadata: make(map[string]any),
	}

	// Execute steps in sequence
	for _, step := range l.steps {
		if !step.Enabled() {
			continue
		}

		// Apply timeout if configured
		stepCtx := ctx
		if l.cfg.Timeout > 0 {
			var cancel context.CancelFunc
			stepCtx, cancel = context.WithTimeout(ctx, l.cfg.Timeout)
			defer cancel()
		}

		stepResult, err := step.Process(stepCtx, current)
		if err != nil {
			if l.cfg.FallbackOnError {
				// Continue with current query on error
				result.Metadata[step.Name()+"_error"] = err.Error()
				continue
			}
			// Return original query on error
			return &Result{Query: query, Modified: false}, err
		}

		if stepResult != nil {
			// Update query for next step
			if stepResult.Query != "" {
				current = stepResult.Query
				result.Query = current
				result.Modified = true
			}

			// Merge results
			if stepResult.Queries != nil {
				result.Queries = stepResult.Queries
			}
			if stepResult.Intent != "" {
				result.Intent = stepResult.Intent
			}
			if stepResult.Hypothetical != "" {
				result.Hypothetical = stepResult.Hypothetical
			}

			// Merge metadata
			for k, v := range stepResult.Metadata {
				result.Metadata[k] = v
			}
		}
	}

	// Ensure at least original query is returned
	if result.Query == "" {
		result.Query = query
	}

	return result, nil
}

// AddStep adds a processing step to the layer.
func (l *Layer) AddStep(step Step) {
	l.steps = append(l.steps, step)
}

// Errors
var (
	ErrEmptyQuery = &QueryError{Message: "query is empty"}
)

// QueryError represents a query processing error.
type QueryError struct {
	Message string
	Step    string
	Err     error
}

func (e *QueryError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *QueryError) Unwrap() error {
	return e.Err
}
