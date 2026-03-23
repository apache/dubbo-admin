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

// IntentType represents the type of user intent.
type IntentType string

const (
	IntentSearch      IntentType = "search"      // Information search
	IntentQA          IntentType = "qa"          // Question answering
	IntentSummary     IntentType = "summary"     // Document summary
	IntentComparison  IntentType = "comparison"  // Comparison between items
	IntentHowTo       IntentType = "howto"       // How-to instructions
	IntentDefinition  IntentType = "definition"  // Definition of a term
	IntentUnknown     IntentType = "unknown"     // Unknown intent
)

// IntentResult represents the result of intent recognition.
type IntentResult struct {
	Intent      IntentType `json:"intent"`
	Confidence  float64    `json:"confidence"`
	Keywords    []string   `json:"keywords"`
	EntityType  string     `json:"entity_type"`
}

// IntentConfig configures intent recognition.
type IntentConfig struct {
	Enabled     bool
	Model       string
	Temperature float64
	Prompt      string // Custom system prompt
}

// DefaultIntentConfig returns default intent configuration.
func DefaultIntentConfig() *IntentConfig {
	return &IntentConfig{
		Enabled:     false,
		Model:       "dashscope/qwen-max",
		Temperature: 0.1, // Low temperature for consistent classification
		Prompt:      defaultIntentPrompt,
	}
}

// IntentStep implements intent recognition.
type IntentStep struct {
	g      *genkit.Genkit
	prompt ai.Prompt
	cfg    *IntentConfig
}

// NewIntentStep creates a new intent recognition step.
func NewIntentStep(g *genkit.Genkit, cfg *IntentConfig, promptBasePath string) (*IntentStep, error) {
	if cfg == nil {
		cfg = DefaultIntentConfig()
	}
	if !cfg.Enabled {
		return &IntentStep{cfg: cfg}, nil
	}

	if g == nil {
		return nil, fmt.Errorf("genkit registry is required for intent recognition")
	}

	// Build prompt
	promptText := cfg.Prompt
	if promptText == "" {
		promptText = defaultIntentPrompt
	}

	// Read prompt file if specified
	if promptBasePath != "" && promptText == defaultIntentPrompt {
		// Try to read from file
		// For now, use default
	}

	prompt, err := buildPrompt(g, IntentInput{}, IntentResult{},
		"intent", promptText, cfg.Temperature, cfg.Model, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build intent prompt: %w", err)
	}

	return &IntentStep{
		g:      g,
		prompt: prompt,
		cfg:    cfg,
	}, nil
}

// Name returns the step name.
func (s *IntentStep) Name() string {
	return "intent"
}

// Type returns the step type.
func (s *IntentStep) Type() StepType {
	return StepIntent
}

// Enabled returns whether the step is enabled.
func (s *IntentStep) Enabled() bool {
	return s.cfg != nil && s.cfg.Enabled
}

// Process executes intent recognition.
func (s *IntentStep) Process(ctx context.Context, query string) (*Result, error) {
	if !s.Enabled() || s.prompt == nil {
		return nil, nil
	}

	input := IntentInput{Query: query}

	resp, err := s.prompt.Execute(ctx, ai.WithInput(input))
	if err != nil {
		return nil, fmt.Errorf("intent recognition failed: %w", err)
	}

	var output IntentResult
	if err := resp.Output(&output); err != nil {
		return nil, fmt.Errorf("failed to parse intent output: %w", err)
	}

	return &Result{
		Query:  query, // Intent doesn't modify query
		Intent: string(output.Intent),
		Metadata: map[string]any{
			"intent_confidence": output.Confidence,
			"intent_keywords":   output.Keywords,
			"entity_type":       output.EntityType,
		},
	}, nil
}

// IntentInput represents the input for intent recognition.
type IntentInput struct {
	Query string `json:"query"`
}

const defaultIntentPrompt = `You are an intent classification system. Analyze the user query and classify its intent.

Possible intents:
- search: General information search
- qa: Specific question expecting a factual answer
- summary: Request for summarization
- comparison: Comparison between multiple items
- howto: Step-by-step instructions
- definition: Definition of a term or concept
- unknown: Unable to determine

Respond with JSON:
{
  "intent": "intent_type",
  "confidence": 0.0-1.0,
  "keywords": ["key", "words"],
  "entity_type": "type if applicable"
}`
