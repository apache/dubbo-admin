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

package models

// ModelsSpec Models configuration
type ModelsSpec struct {
	DefaultModel     string                    `yaml:"default_model"`
	DefaultEmbedding string                    `yaml:"default_embedding"`
	Providers        map[string]ProviderConfig `yaml:"providers"`
}

// ProviderConfig Provider configuration
type ProviderConfig struct {
	APIKey    string         `yaml:"api_key"`
	BaseURL   string         `yaml:"base_url"`
	Models    []ModelInfo    `yaml:"models,omitempty"`
	Embedders []EmbedderInfo `yaml:"embedders,omitempty"`
	Config    map[string]any `yaml:"config,omitempty"`
}

// ModelInfo Model information
type ModelInfo struct {
	Name   string         `yaml:"name"`
	Key    string         `yaml:"key"`
	Type   string         `yaml:"type,omitempty"`
	Config map[string]any `yaml:"config,omitempty"`
}

// EmbedderInfo Embedder information
type EmbedderInfo struct {
	Name       string         `yaml:"name"`
	Key        string         `yaml:"key"`
	Type       string         `yaml:"type,omitempty"`
	Dimensions int            `yaml:"dimensions"`
	Config     map[string]any `yaml:"config,omitempty"`
}

func DefaultModelsSpec() *ModelsSpec {
	return &ModelsSpec{
		DefaultModel:     "dashscope/qwen-max",
		DefaultEmbedding: "dashscope/qwen3-embedding",
		Providers:        make(map[string]ProviderConfig),
	}
}
