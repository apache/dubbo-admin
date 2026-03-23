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
	"fmt"
	"os"
	"path"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/openai/openai-go"
	"gopkg.in/yaml.v3"
)

// LayerSpec defines the configuration for creating a query processing layer.
type LayerSpec struct {
	// Global configuration
	Model      string `yaml:"model"`      // Default model
	Timeout    string `yaml:"timeout"`    // Default timeout (e.g., "5s")
	Temperature float64 `yaml:"temperature"` // Default temperature

	// Step configurations
	Intent     *StepSpec `yaml:"intent"`
	Rewrite    *StepSpec `yaml:"rewrite"`
	Expansion  *StepSpec `yaml:"expansion"`
	HyDE       *StepSpec `yaml:"hyde"`
}

// StepSpec defines configuration for a single processing step.
type StepSpec struct {
	Enabled     bool    `yaml:"enabled"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	Prompt      string  `yaml:"prompt"` // Path to custom prompt file
}

// NewLayerFromSpec creates a query processing layer from specification.
func NewLayerFromSpec(g *genkit.Genkit, spec *LayerSpec, promptBasePath string) (*Layer, error) {
	if spec == nil {
		return &Layer{}, nil
	}

	// Apply defaults
	if spec.Model == "" {
		spec.Model = "dashscope/qwen-max"
	}

	layer := &Layer{
		cfg: &Config{
			Model:           spec.Model,
			Temperature:     spec.Temperature,
			FallbackOnError: true,
		},
	}

	// Parse timeout
	if spec.Timeout != "" {
		// Parse duration string (e.g., "5s", "100ms")
		// For simplicity, assuming proper format
	}

	// Create steps
	if spec.Intent != nil && spec.Intent.Enabled {
		cfg := &IntentConfig{
			Enabled:     true,
			Model:       spec.Intent.Model,
			Temperature: spec.Intent.Temperature,
		}
		if cfg.Model == "" {
			cfg.Model = spec.Model
		}
		step, err := NewIntentStep(g, cfg, promptBasePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create intent step: %w", err)
		}
		layer.AddStep(step)
	}

	if spec.Rewrite != nil && spec.Rewrite.Enabled {
		cfg := &RewriteConfig{
			Enabled:     true,
			Model:       spec.Rewrite.Model,
			Temperature: spec.Rewrite.Temperature,
		}
		if cfg.Model == "" {
			cfg.Model = spec.Model
		}
		step, err := NewRewriteStep(g, cfg, promptBasePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create rewrite step: %w", err)
		}
		layer.AddStep(step)
	}

	if spec.Expansion != nil && spec.Expansion.Enabled {
		cfg := &ExpansionConfig{
			Enabled:     true,
			Model:       spec.Expansion.Model,
			Temperature: spec.Expansion.Temperature,
			NumQueries:  3,
		}
		if cfg.Model == "" {
			cfg.Model = spec.Model
		}
		step, err := NewExpansionStep(g, cfg, promptBasePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create expansion step: %w", err)
		}
		layer.AddStep(step)
	}

	if spec.HyDE != nil && spec.HyDE.Enabled {
		cfg := &HyDEConfig{
			Enabled:     true,
			Model:       spec.HyDE.Model,
			Temperature: spec.HyDE.Temperature,
		}
		if cfg.Model == "" {
			cfg.Model = spec.Model
		}
		step, err := NewHyDEStep(g, cfg, promptBasePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create hyde step: %w", err)
		}
		layer.AddStep(step)
	}

	return layer, nil
}

// NewLayerFromYAML creates a query processing layer from YAML configuration.
func NewLayerFromYAML(g *genkit.Genkit, yamlContent string, promptBasePath string) (*Layer, error) {
	var spec LayerSpec
	if err := yaml.Unmarshal([]byte(yamlContent), &spec); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}
	return NewLayerFromSpec(g, &spec, promptBasePath)
}

// NewSimpleLayer creates a simple query processing layer with just rewrite enabled.
func NewSimpleLayer(g *genkit.Genkit, model string, promptBasePath string) (*Layer, error) {
	if g == nil {
		return &Layer{}, nil
	}

	if model == "" {
		model = "dashscope/qwen-max"
	}

	cfg := &RewriteConfig{
		Enabled:     true,
		Model:       model,
		Temperature: 0.3,
	}

	step, err := NewRewriteStep(g, cfg, promptBasePath)
	if err != nil {
		return nil, err
	}

	return &Layer{
		steps: []Step{step},
		cfg: &Config{
			Model:           model,
			Temperature:     0.3,
			FallbackOnError: true,
		},
	}, nil
}

// buildPrompt creates an AI prompt with the given parameters.
func buildPrompt(registry *genkit.Genkit, inType, outType any, tag, prompt string, temp float64, model string, extraPrompt string, tools ...ai.ToolRef) (ai.Prompt, error) {
	opts := []ai.PromptOption{
		ai.WithSystem(prompt),
		ai.WithConfig(&openai.ChatCompletionNewParams{
			Temperature: openai.Float(temp),
		}),
		ai.WithModelName(model),
	}
	if inType != nil {
		opts = append(opts, ai.WithInputType(inType))
	}
	if outType != nil {
		opts = append(opts, ai.WithOutputType(outType))
	}
	if extraPrompt != "" {
		opts = append(opts, ai.WithPrompt(extraPrompt))
	}
	if tools != nil {
		opts = append(opts, ai.WithTools(tools...), ai.WithReturnToolRequests(true))
	}

	return genkit.DefinePrompt(registry, tag, opts...), nil
}

// readPromptFile reads a prompt from a file.
func readPromptFile(promptPath string) (string, error) {
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GetPromptPath returns the full path to a prompt file.
func GetPromptPath(basePath, name string) string {
	return path.Join(basePath, name+".txt")
}
