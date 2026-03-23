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

// GeneralMerger is the common interface for all merger types.
type GeneralMerger interface {
	Merge(ctx context.Context, results [][]*schema.Document) ([]*schema.Document, error)
}

// Config represents a generic merger configuration.
type Config struct {
	Type string // "rrf", "weighted", "concat"

	// RRF options
	RRF *RRFConfig

	// Weighted options
	Weighted *WeightedConfig

	// Concat options
	Concat *ConcatConfig
}

// NewMerger creates a merger based on the configuration type.
func NewMerger(cfg *Config) (GeneralMerger, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	switch cfg.Type {
	case MergerTypeRRF:
		return NewRRFMerger(cfg.RRF)
	case MergerTypeWeighted:
		return NewWeightedMerger(cfg.Weighted)
	case MergerTypeConcat:
		return NewConcatMerger(cfg.Concat)
	default:
		return nil, fmt.Errorf("unknown merger type: %s", cfg.Type)
	}
}

// Merger types supported
const (
	MergerTypeRRF      = "rrf"
	MergerTypeWeighted = "weighted"
	MergerTypeConcat   = "concat"
)
