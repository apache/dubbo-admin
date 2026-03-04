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

import (
	"dubbo-admin-ai/runtime"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/openai/openai-go/option"
)

type ModelsComponent struct {
	defaultModel     string
	defaultEmbedding string
	providers        map[string]ProviderConfig
}

// NewModelsComponent creates a Models component instance
func NewModelsComponent(
	defaultModel string,
	defaultEmbedding string,
	providers map[string]ProviderConfig,
) (runtime.Component, error) {
	return &ModelsComponent{
		defaultModel:     defaultModel,
		defaultEmbedding: defaultEmbedding,
		providers:        providers,
	}, nil
}

func (m *ModelsComponent) Name() string {
	return "models"
}

func (m *ModelsComponent) Validate() error {
	if m.defaultModel == "" {
		return fmt.Errorf("default_model is required")
	}
	if m.defaultEmbedding == "" {
		return fmt.Errorf("default_embedding is required")
	}
	if len(m.providers) == 0 {
		return fmt.Errorf("at least one provider must be configured")
	}
	for name, provider := range m.providers {
		if provider.BaseURL == "" {
			return fmt.Errorf("provider %s base_url is required", name)
		}
	}
	return nil
}

func (m *ModelsComponent) Init(rt *runtime.Runtime) error {
	var plugins []api.Plugin

	for providerName, cfg := range m.providers {
		if cfg.APIKey == "" {
			continue
		}
		plugin := createProviderPlugin(providerName, cfg)

		if plugin != nil {
			plugins = append(plugins, plugin)
		}
	}

	ctx := rt.GetContext()
	genkitRegistry := genkit.Init(ctx,
		genkit.WithPlugins(plugins...),
		genkit.WithDefaultModel(m.defaultModel),
	)

	if rt.GetGenkitRegistry() == nil {
		rt.SetGenkitRegistry(genkitRegistry)
	} else {
		rt.GetLogger().Warn("Genkit registry already set, skipping initialization")
	}

	totalModels := 0
	totalEmbedders := 0

	for _, plugin := range plugins {
		if oaiCompat, ok := plugin.(*compat_oai.OpenAICompatible); ok {
			providerName := oaiCompat.Provider
			providerCfg, exists := m.providers[providerName]
			if !exists {
				continue
			}
			// Register all models
			for _, modelCfg := range providerCfg.Models {
				registerModel(oaiCompat, providerName, modelCfg)
				totalModels++
			}
			// Register all embedding models
			for _, embedderCfg := range providerCfg.Embedders {
				registerEmbedder(oaiCompat, providerName, embedderCfg)
				totalEmbedders++
			}
		}
	}

	rt.GetLogger().Info("Models component initialized",
		"default_model", m.defaultModel,
		"default_embedding", m.defaultEmbedding,
		"providers", len(m.providers),
		"total_models", totalModels,
		"total_embedders", totalEmbedders)

	return nil
}

func (m *ModelsComponent) Start() error {
	return nil
}

func (m *ModelsComponent) Stop() error {
	return nil
}

func registerModel(oaiCompat *compat_oai.OpenAICompatible, providerName string, cfg ModelInfo) {
	var supports *ai.ModelSupports
	switch cfg.Type {
	case "chat":
		supports = &compat_oai.BasicText
	case "multimodal":
		supports = &compat_oai.Multimodal
	case "code":
		supports = &compat_oai.BasicText
	default:
		supports = &compat_oai.BasicText
	}

	oaiCompat.DefineModel(providerName, cfg.Key, ai.ModelOptions{
		Label:    cfg.Name,
		Supports: supports,
		Versions: []string{cfg.Key},
	})
}

func registerEmbedder(oaiCompat *compat_oai.OpenAICompatible, providerName string, cfg EmbedderInfo) {
	var inputTypes []string
	embedderType := cfg.Type
	if embedderType == "" {
		embedderType = "text"
	}

	switch embedderType {
	case "text":
		inputTypes = []string{"text"}
	case "image":
		inputTypes = []string{"image", "text"}
	case "audio":
		inputTypes = []string{"audio"}
	case "multimodal":
		inputTypes = []string{"text", "image", "audio"}
	default:
		inputTypes = []string{"text"}
	}

	oaiCompat.DefineEmbedder(providerName, cfg.Key, &ai.EmbedderOptions{
		Label:      cfg.Name,
		Supports:   &ai.EmbedderSupports{Input: inputTypes},
		Dimensions: cfg.Dimensions,
	})
}

// createProviderPlugin creates a single provider plugin
func createProviderPlugin(providerName string, cfg ProviderConfig) api.Plugin {
	// Gemini requires special handling (not currently supported)
	if providerName == "gemini" {
		return nil
	}

	// Check API Key
	if cfg.APIKey == "" {
		return nil
	}

	// Use OpenAICompatible directly
	return &compat_oai.OpenAICompatible{
		Provider: providerName,
		Opts: []option.RequestOption{
			option.WithAPIKey(cfg.APIKey),
			option.WithBaseURL(cfg.BaseURL),
		},
	}
}
