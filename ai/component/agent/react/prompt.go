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
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/openai/openai-go"
)

const actPolicy = `# Act Phase
Tools are available in this phase through their provided names, descriptions,
and input schemas.

- Call a tool only when its result could materially improve or change the answer.
- Select tools from their provided descriptions and supply only supported input.
- If no tool is needed, answer the user directly.
- When calling tools, emit only the native tool call. Independent calls may be
  made together.
- After results arrive, reassess the evidence. Call another tool only to resolve
  a material remaining gap or test a stated hypothesis; otherwise answer.
- Do not repeat an identical call unless retrying a transient failure is
  justified. If a tool fails, times out, is unavailable, or returns no useful
  data, use other available evidence and follow the shared incomplete-evidence
  rules.`

const finalAnswerPolicy = `# Final-Answer Phase
No tools are available in this phase. Produce the final user-facing answer now
using only evidence already present in the conversation.

- Do not emit or request a tool call.
- Do not claim that a check, lookup, or verification occurred unless its result
  is present in the conversation.
- If the available evidence is insufficient, state the uncertainty and the next
  verification step rather than guessing.`

// buildPrompts assembles the two genkit prompts a ReAct agent needs from a
// shared policy and separate phase policies:
//   - act reasons with tools available (native function calling); each iteration
//     either calls tools or answers directly.
//   - answer has an explicit tool-less policy and forces a final answer when the
//     iteration budget is exhausted.
//
// They share every model setting and the phase-independent policy loaded from
// the configured prompt file.
func (ra *ReActAgent) buildPrompts(g *genkit.Genkit, spec *AgentSpec, model string, toolRefs []ai.ToolRef) (act, answer ai.Prompt, err error) {
	promptPath := path.Join(spec.PromptBasePath, spec.PromptFile)
	sharedPolicy, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
	}

	actSystemPrompt := assembleSystemPrompt(string(sharedPolicy), actPolicy)
	answerSystemPrompt := assembleSystemPrompt(string(sharedPolicy), finalAnswerPolicy)
	act = buildPrompt(g, "react_act", actSystemPrompt, spec, model, toolRefs)
	answer = buildPrompt(g, "react_answer", answerSystemPrompt, spec, model, nil)
	return act, answer, nil
}

func assembleSystemPrompt(sharedPolicy, phasePolicy string) string {
	return strings.TrimSpace(sharedPolicy) + "\n\n" + strings.TrimSpace(phasePolicy)
}

// buildPrompt assembles a genkit prompt from the shared model settings, binding
// the given tool set when one is provided.
func buildPrompt(registry *genkit.Genkit, tag, systemPrompt string, spec *AgentSpec, model string, tools []ai.ToolRef) ai.Prompt {
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
		ai.WithSystem("%s", systemPrompt),
		ai.WithConfig(cfg),
		ai.WithModelName(model),
	}
	if len(tools) > 0 {
		opts = append(opts, ai.WithTools(tools...), ai.WithReturnToolRequests(true))
	}

	return genkit.DefinePrompt(registry, tag, opts...)
}
