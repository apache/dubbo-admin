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

// Package query provides query understanding capabilities for RAG systems.
//
// This package replaces the old "processors" package and provides:
// - Intent recognition
// - Query rewriting
// - Query expansion
// - HyDE (Hypothetical Document Embeddings)
//
// The layer ensures backward compatibility with the old QueryProcessor interface.
package query

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/genkit"
)

// Legacy Processor types for backward compatibility.

// QueryProcessor is the legacy interface for backward compatibility.
// Use Layer instead for new code.
type QueryProcessor interface {
	Process(ctx context.Context, query string) (string, error)
}

// QueryProcessorConfig is the legacy config for backward compatibility.
type QueryProcessorConfig struct {
	Model           string
	Timeout         string
	Temperature     float64
	FallbackOnError bool
	Enabled         bool
}

// LegacyProcessor wraps the new Layer to implement the old QueryProcessor interface.
type LegacyProcessor struct {
	layer *Layer
}

// NewLegacyProcessor creates a legacy-compatible processor from the new Layer.
func NewLegacyProcessor(layer *Layer) *LegacyProcessor {
	return &LegacyProcessor{layer: layer}
}

// Process implements the old QueryProcessor interface.
func (p *LegacyProcessor) Process(ctx context.Context, query string) (string, error) {
	if p.layer == nil {
		return query, nil
	}

	result, err := p.layer.Process(ctx, query)
	if err != nil {
		return query, err
	}

	if result.Query == "" {
		return query, nil
	}

	return result.Query, nil
}

// GetLayer returns the underlying Layer for direct access.
func (p *LegacyProcessor) GetLayer() *Layer {
	return p.layer
}
// NewQueryProcessor creates a QueryProcessor using the legacy configuration.
// This function maintains backward compatibility with the old factory pattern.
func NewQueryProcessor(g interface{}, cfg *QueryProcessorConfig, promptBasePath string) (QueryProcessor, error) {
	if cfg == nil || !cfg.Enabled {
		return &noopProcessor{}, nil
	}

	// Convert legacy config to new LayerSpec
	spec := &LayerSpec{
		Model:       cfg.Model,
		Temperature: cfg.Temperature,
		Rewrite: &StepSpec{
			Enabled:     true,
			Model:       cfg.Model,
			Temperature: cfg.Temperature,
		},
	}

	// Create layer - pass g if it's a genkit.Genkit
	var genkitInstance *genkit.Genkit
	if g != nil {
		// Type assertion for *genkit.Genkit
		if gg, ok := g.(*genkit.Genkit); ok {
			genkitInstance = gg
		} else {
			// Type assertion failed - this shouldn't happen if rt.GetRegistry() is used
			// Return noop processor to maintain backward compatibility
			return &noopProcessor{}, nil
		}
	}

	layer, err := NewLayerFromSpec(genkitInstance, spec, promptBasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create query processor: %w", err)
	}

	return NewLegacyProcessor(layer), nil
}

// noopProcessor is a no-op processor that returns the original query.
type noopProcessor struct{}

func (p *noopProcessor) Process(ctx context.Context, query string) (string, error) {
	return query, nil
}

// Processor types (for backward compatibility)
const (
	ProcessorTypeRewrite = "rewrite"
)
