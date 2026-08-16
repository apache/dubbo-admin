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

	"dubbo-admin-ai/runtime"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/openai/openai-go"
)

// answerDirective nudges the model to commit to a final answer from whatever it
// has already gathered. It is only attached to the tool-less answer prompt,
// which the loop uses once the iteration budget is exhausted.
const answerDirective = "Provide your final answer now, using the information already gathered. Do not request any more tools."

// buildPrompts assembles the two genkit prompts a ReAct agent needs, both from
// the single configured system prompt:
//   - act reasons with tools available (native function calling); each iteration
//     either calls tools or answers directly.
//   - answer is the same system prompt without tools, used to force a final
//     answer when the iteration budget is exhausted.
//
// They share every model setting; only the tool binding (and answer's synthesis
// directive) differ.
func (ra *ReActAgent) buildPrompts(g *genkit.Genkit, spec *AgentSpec, model string, toolRefs []ai.ToolRef) (act, answer ai.Prompt, err error) {
	promptPath := path.Join(spec.PromptBasePath, spec.PromptFile)
	systemPrompt, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
	}

	// Advertise the available tool names to the model.
	toolNames := make([]string, 0, len(toolRefs))
	for _, toolRef := range toolRefs {
		toolNames = append(toolNames, toolRef.Name())
	}
	toolsJSON, err := json.Marshal(toolNames)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal tool names: %w", err)
	}
	extraPrompt := fmt.Sprintf("available tools: %s", string(toolsJSON))
	runtime.GetLogger().Debug("Tool details", "extraPrompt", extraPrompt)

	act = buildPrompt(g, "react_act", string(systemPrompt), spec, model, extraPrompt, toolRefs)
	answer = buildPrompt(g, "react_answer", string(systemPrompt), spec, model, answerDirective, nil)
	return act, answer, nil
}

// buildPrompt assembles a genkit prompt from the shared model settings, binding
// the given tool set when one is provided.
func buildPrompt(registry *genkit.Genkit, tag, systemPrompt string, spec *AgentSpec, model, extraPrompt string, tools []ai.ToolRef) ai.Prompt {
	cfg := &openai.ChatCompletionNewParams{
		Temperature: openai.Float(spec.Temperature),
	}
	if spec.TopP > 0 {
		cfg.TopP = openai.Float(spec.TopP)
	}
	if spec.MaxTokens > 0 {
		cfg.MaxTokens = openai.Int(int64(spec.MaxTokens))
	}

	opts := []ai.PromptOption{
		ai.WithSystem(systemPrompt),
		ai.WithConfig(cfg),
		ai.WithModelName(model),
	}
	if extraPrompt != "" {
		opts = append(opts, ai.WithPrompt(extraPrompt))
	}
	if len(tools) > 0 {
		opts = append(opts, ai.WithTools(tools...), ai.WithReturnToolRequests(true))
	}

	return genkit.DefinePrompt(registry, tag, opts...)
}
