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

package react

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/openai/openai-go"
)

// builtStage is a prompt assembled once at construction time plus the metadata
// the matching step needs at run time. Steps are (re)built per Interact from
// these so a single agent can serve concurrent interactions.
type builtStage struct {
	kind    string // "reasonAct" | "observe"
	prompt  ai.Prompt
	timeout time.Duration
}

// buildStages assembles one prompt per configured stage, in order.
func (ra *ReActAgent) buildStages(g *genkit.Genkit, stagesCfg []StageInfo, promptBasePath string, defaultModel string, toolRefs []ai.ToolRef) ([]builtStage, error) {
	var stages []builtStage

	for _, stageCfg := range stagesCfg {
		// Read prompt file.
		promptPath := path.Join(promptBasePath, stageCfg.PromptFile)
		systemPrompt, err := os.ReadFile(promptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
		}

		// Only the reasonAct stage calls tools.
		needsTools := stageCfg.FlowType == flowReasonAct
		var tools []ai.ToolRef
		if needsTools && stageCfg.EnableTools {
			tools = toolRefs
		}

		// Advertise the available tool names to the reasonAct stage.
		extraPrompt := stageCfg.ExtraPrompt
		if needsTools && extraPrompt == "" {
			toolNames := make([]string, 0, len(toolRefs))
			for _, toolRef := range toolRefs {
				toolNames = append(toolNames, toolRef.Name())
			}
			toolsJson, err := json.Marshal(toolNames)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool names: %w", err)
			}
			extraPrompt = fmt.Sprintf("available tools: %s", string(toolsJson))
			runtime.GetLogger().Debug("Tool details", "extraPrompt", extraPrompt)
		}

		// Each step knows its own in/out types. reasonAct uses native tool
		// calling (no structured output); observe returns a structured decision.
		var inType, outType any
		switch stageCfg.FlowType {
		case flowReasonAct:
			// native function calling — no structured in/out type
		case flowObserve:
			outType = schema.Observation{}
		default:
			return nil, fmt.Errorf("unknown flow type: %s", stageCfg.FlowType)
		}

		// Use default model if not specified in configuration.
		model := stageCfg.Model
		if model == "" {
			model = defaultModel
		}

		prompt := buildPrompt(g, inType, outType, stageCfg.Name, string(systemPrompt),
			stageCfg.Temperature, stageCfg.TopP, stageCfg.MaxTokens, model, extraPrompt, tools...)

		timeout := time.Duration(stageCfg.Timeout) * time.Second
		stages = append(stages, builtStage{kind: stageCfg.FlowType, prompt: prompt, timeout: timeout})
	}

	return stages, nil
}

// buildPrompt assembles a genkit prompt for one stage from its model settings
// and, when provided, its structured in/out types and tool set.
func buildPrompt(registry *genkit.Genkit, inType, outType any, tag, prompt string, temp, topP float64, maxTokens int, model string, extraPrompt string, tools ...ai.ToolRef) ai.Prompt {
	cfg := &openai.ChatCompletionNewParams{
		Temperature: openai.Float(temp),
	}
	if topP > 0 {
		cfg.TopP = openai.Float(topP)
	}
	if maxTokens > 0 {
		cfg.MaxTokens = openai.Int(int64(maxTokens))
	}

	opts := []ai.PromptOption{
		ai.WithSystem(prompt),
		ai.WithConfig(cfg),
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

	return genkit.DefinePrompt(registry, tag, opts...)
}
