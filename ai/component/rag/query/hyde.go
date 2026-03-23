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

// HyDEConfig configures Hypothetical Document Embeddings.
type HyDEConfig struct {
	Enabled     bool
	Model       string
	Temperature float64
	Prompt      string // Custom system prompt
}

// DefaultHyDEConfig returns default HyDE configuration.
func DefaultHyDEConfig() *HyDEConfig {
	return &HyDEConfig{
		Enabled:     false,
		Model:       "dashscope/qwen-max",
		Temperature: 0.7, // Higher temperature for more creative hypothetical docs
		Prompt:      defaultHyDEPrompt,
	}
}

// HyDEStep implements Hypothetical Document Embeddings (HyDE).
//
// HyDE generates a hypothetical document that would answer the user's query.
// This hypothetical document is then embedded and used for retrieval,
// which can improve performance for semantic search.
type HyDEStep struct {
	g      *genkit.Genkit
	prompt ai.Prompt
	cfg    *HyDEConfig
}

// NewHyDEStep creates a new HyDE step.
func NewHyDEStep(g *genkit.Genkit, cfg *HyDEConfig, promptBasePath string) (*HyDEStep, error) {
	if cfg == nil {
		cfg = DefaultHyDEConfig()
	}
	if !cfg.Enabled {
		return &HyDEStep{cfg: cfg}, nil
	}

	if g == nil {
		return nil, fmt.Errorf("genkit registry is required for HyDE")
	}

	promptText := cfg.Prompt
	if promptText == "" {
		promptText = defaultHyDEPrompt
	}

	prompt, err := buildPrompt(g, HyDEInput{}, HyDEOutput{},
		"hyde", promptText, cfg.Temperature, cfg.Model, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build hyde prompt: %w", err)
	}

	return &HyDEStep{
		g:      g,
		prompt: prompt,
		cfg:    cfg,
	}, nil
}

// Name returns the step name.
func (s *HyDEStep) Name() string {
	return "hyde"
}

// Type returns the step type.
func (s *HyDEStep) Type() StepType {
	return StepHyDE
}

// Enabled returns whether the step is enabled.
func (s *HyDEStep) Enabled() bool {
	return s.cfg != nil && s.cfg.Enabled
}

// Process executes HyDE to generate a hypothetical document.
func (s *HyDEStep) Process(ctx context.Context, query string) (*Result, error) {
	if !s.Enabled() || s.prompt == nil {
		return nil, nil
	}

	input := HyDEInput{Query: query}

	resp, err := s.prompt.Execute(ctx, ai.WithInput(input))
	if err != nil {
		return nil, fmt.Errorf("hyde generation failed: %w", err)
	}

	var output HyDEOutput
	if err := resp.Output(&output); err != nil {
		return nil, fmt.Errorf("failed to parse hyde output: %w", err)
	}

	if output.HypotheticalDocument == "" {
		return &Result{Query: query, Modified: false}, nil
	}

	return &Result{
		Query:        query, // Keep original query
		Hypothetical: output.HypotheticalDocument,
		Modified:     true,
		Metadata: map[string]any{
			"hyde_generated": true,
			"hyde_length":    len(output.HypotheticalDocument),
		},
	}, nil
}

// HyDEInput represents the input for HyDE.
type HyDEInput struct {
	Query string `json:"query"`
}

// HyDEOutput represents the output from HyDE.
type HyDEOutput struct {
	HypotheticalDocument string `json:"hypothetical_document"`
}

const defaultHyDEPrompt = `You are an expert at generating hypothetical documents that would answer user queries.

Given a user's query, write a concise, factual document that directly answers the question.
The document should be written as if it exists in a knowledge base.

Guidelines:
- Be specific and factual
- Use clear, professional language
- Include relevant details
- Keep it concise (100-200 words)
- Write in the same language as the query
- Do not include explanations that this is hypothetical

Respond with JSON:
{
  "hypothetical_document": "a document that would answer the user's query"
}`
