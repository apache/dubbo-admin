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

// RewriteConfig configures query rewriting.
type RewriteConfig struct {
	Enabled     bool
	Model       string
	Temperature float64
	Prompt      string // Custom system prompt
}

// DefaultRewriteConfig returns default rewrite configuration.
func DefaultRewriteConfig() *RewriteConfig {
	return &RewriteConfig{
		Enabled:     false,
		Model:       "dashscope/qwen-max",
		Temperature: 0.3,
		Prompt:      defaultRewritePrompt,
	}
}

// RewriteStep implements query rewriting.
type RewriteStep struct {
	g      *genkit.Genkit
	prompt ai.Prompt
	cfg    *RewriteConfig
}

// NewRewriteStep creates a new query rewrite step.
func NewRewriteStep(g *genkit.Genkit, cfg *RewriteConfig, promptBasePath string) (*RewriteStep, error) {
	if cfg == nil {
		cfg = DefaultRewriteConfig()
	}
	if !cfg.Enabled {
		return &RewriteStep{cfg: cfg}, nil
	}

	if g == nil {
		return nil, fmt.Errorf("genkit registry is required for query rewrite")
	}

	promptText := cfg.Prompt
	if promptText == "" {
		promptText = defaultRewritePrompt
	}

	prompt, err := buildPrompt(g, RewriteInput{}, RewriteOutput{},
		"rewrite", promptText, cfg.Temperature, cfg.Model, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build rewrite prompt: %w", err)
	}

	return &RewriteStep{
		g:      g,
		prompt: prompt,
		cfg:    cfg,
	}, nil
}

// Name returns the step name.
func (s *RewriteStep) Name() string {
	return "rewrite"
}

// Type returns the step type.
func (s *RewriteStep) Type() StepType {
	return StepRewrite
}

// Enabled returns whether the step is enabled.
func (s *RewriteStep) Enabled() bool {
	return s.cfg != nil && s.cfg.Enabled
}

// Process executes query rewriting.
func (s *RewriteStep) Process(ctx context.Context, query string) (*Result, error) {
	if !s.Enabled() || s.prompt == nil {
		return nil, nil
	}

	input := RewriteInput{Query: query}

	resp, err := s.prompt.Execute(ctx, ai.WithInput(input))
	if err != nil {
		return nil, fmt.Errorf("query rewrite failed: %w", err)
	}

	var output RewriteOutput
	if err := resp.Output(&output); err != nil {
		return nil, fmt.Errorf("failed to parse rewrite output: %w", err)
	}

	if output.RewrittenQuery == "" {
		return &Result{Query: query, Modified: false}, nil
	}

	return &Result{
		Query:    output.RewrittenQuery,
		Modified: true,
		Metadata: map[string]any{
			"original_query": query,
			"rewrite_reason": output.Reason,
		},
	}, nil
}

// RewriteInput represents the input for query rewriting.
type RewriteInput struct {
	Query string `json:"query"`
}

// RewriteOutput represents the output from query rewriting.
type RewriteOutput struct {
	RewrittenQuery string `json:"rewritten_query"`
	Reason         string `json:"reason,omitempty"`
}

const defaultRewritePrompt = `You are a query optimization expert. Rewrite the user's query to improve information retrieval.

Guidelines:
- Clarify ambiguous terms
- Expand abbreviations
- Add relevant context
- Fix grammatical errors
- Preserve the original intent
- Keep the query concise

Respond with JSON:
{
  "rewritten_query": "optimized query",
  "reason": "brief explanation of changes"
}`
