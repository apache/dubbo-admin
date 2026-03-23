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
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// ExpansionConfig configures query expansion.
type ExpansionConfig struct {
	Enabled     bool
	Model       string
	Temperature float64
	NumQueries  int  // Number of expanded queries to generate (default: 3)
	Prompt      string // Custom system prompt
}

// DefaultExpansionConfig returns default expansion configuration.
func DefaultExpansionConfig() *ExpansionConfig {
	return &ExpansionConfig{
		Enabled:     false,
		Model:       "dashscope/qwen-max",
		Temperature: 0.5,
		NumQueries:  3,
		Prompt:      defaultExpansionPrompt,
	}
}

// ExpansionStep implements query expansion.
type ExpansionStep struct {
	g      *genkit.Genkit
	prompt ai.Prompt
	cfg    *ExpansionConfig
}

// NewExpansionStep creates a new query expansion step.
func NewExpansionStep(g *genkit.Genkit, cfg *ExpansionConfig, promptBasePath string) (*ExpansionStep, error) {
	if cfg == nil {
		cfg = DefaultExpansionConfig()
	}
	if !cfg.Enabled {
		return &ExpansionStep{cfg: cfg}, nil
	}

	if g == nil {
		return nil, fmt.Errorf("genkit registry is required for query expansion")
	}

	if cfg.NumQueries <= 0 {
		cfg.NumQueries = 3
	}

	promptText := cfg.Prompt
	if promptText == "" {
		promptText = defaultExpansionPrompt
	}

	prompt, err := buildPrompt(g, ExpansionInput{}, ExpansionOutput{},
		"expansion", promptText, cfg.Temperature, cfg.Model, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build expansion prompt: %w", err)
	}

	return &ExpansionStep{
		g:      g,
		prompt: prompt,
		cfg:    cfg,
	}, nil
}

// Name returns the step name.
func (s *ExpansionStep) Name() string {
	return "expansion"
}

// Type returns the step type.
func (s *ExpansionStep) Type() StepType {
	return StepExpansion
}

// Enabled returns whether the step is enabled.
func (s *ExpansionStep) Enabled() bool {
	return s.cfg != nil && s.cfg.Enabled
}

// Process executes query expansion.
func (s *ExpansionStep) Process(ctx context.Context, query string) (*Result, error) {
	if !s.Enabled() || s.prompt == nil {
		return nil, nil
	}

	input := ExpansionInput{
		Query:     query,
		NumQueries: s.cfg.NumQueries,
	}

	resp, err := s.prompt.Execute(ctx, ai.WithInput(input))
	if err != nil {
		return nil, fmt.Errorf("query expansion failed: %w", err)
	}

	var output ExpansionOutput
	if err := resp.Output(&output); err != nil {
		return nil, fmt.Errorf("failed to parse expansion output: %w", err)
	}

	// Include original query in the list
	queries := append([]string{query}, output.Queries...)

	return &Result{
		Query:    query, // Keep original as primary
		Queries:  queries,
		Modified: true,
		Metadata: map[string]any{
			"expansion_count": len(output.Queries),
			"expanded_queries": output.Queries,
		},
	}, nil
}

// ExpansionInput represents the input for query expansion.
type ExpansionInput struct {
	Query     string `json:"query"`
	NumQueries int    `json:"num_queries"`
}

// ExpansionOutput represents the output from query expansion.
type ExpansionOutput struct {
	Queries []string `json:"queries"`
}

const defaultExpansionPrompt = `You are a query expansion expert. Generate multiple search queries that cover different aspects of the user's request.

Guidelines:
- Generate diverse queries with different phrasings
- Include synonyms and related terms
- Vary specificity (broader and narrower terms)
- Each query should be standalone and meaningful
- Preserve the core intent

Respond with JSON:
{
  "queries": ["query1", "query2", "query3"]
}`
