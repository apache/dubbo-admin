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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	comp "dubbo-admin-ai/component/context"
	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/openai/openai-go"
)

// TemplateReference lets the Session API pin policy in its atomic Create.
func (ra *ReActAgent) TemplateReference() string { return ra.templateRef }

func (ra *ReActAgent) initCompiler(spec *AgentSpec, refs []ai.ToolRef) error {
	shared, err := os.ReadFile(filepath.Join(spec.PromptBasePath, spec.PromptFile))
	if err != nil {
		return err
	}
	variables := map[string]comp.Variable{}
	for key := range spec.TemplateVariables {
		variables[key] = comp.Variable{Required: true, MaxBytes: 4096}
	}
	t := comp.Template{ID: spec.TemplateID, Version: spec.TemplateVersion, Shared: string(shared), Act: actPolicy, Answer: finalAnswerPolicy, Variables: variables}
	for _, ref := range refs {
		allowed := len(spec.AllowedTools) == 0
		for _, name := range spec.AllowedTools {
			allowed = allowed || name == ref.Name()
		}
		if allowed {
			t.AllowedTools = append(t.AllowedTools, ref.Name())
		}
	}
	sort.Strings(t.AllowedTools)
	// Validate syntax and declared values at initialization as well as per call.
	if _, err := t.Render("act", spec.TemplateVariables); err != nil {
		return err
	}
	templates := []comp.Template{t}
	for _, file := range spec.TemplateArchiveFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		var old comp.Template
		decoderErr := json.Unmarshal(data, &old)
		if decoderErr != nil {
			return decoderErr
		}
		templates = append(templates, old)
	}
	c, err := comp.New(templates...)
	if err != nil {
		return err
	}
	if err = comp.RegisterCoreSources(c); err != nil {
		return err
	}
	ra.compiler = c
	ra.templateRef = t.Ref()
	if err := json.Unmarshal(comp.JSON(spec), &ra.compilerSpec); err != nil {
		return err
	}
	for _, ref := range refs {
		if len(spec.AllowedTools) > 0 {
			allowed := false
			for _, name := range spec.AllowedTools {
				allowed = allowed || name == ref.Name()
			}
			if !allowed {
				continue
			}
		}
		tool := genkit.LookupTool(ra.registry, ref.Name())
		if tool == nil {
			return fmt.Errorf("tool definition unavailable: %s", ref.Name())
		}
		d := tool.Definition()
		var schema map[string]any
		if err := json.Unmarshal(comp.JSON(d.InputSchema), &schema); err != nil {
			return err
		}
		ra.compiledTools = append(ra.compiledTools, comp.Tool{Name: d.Name, Description: d.Description, InputSchema: schema})
	}
	return nil
}

func (ra *ReActAgent) compile(ctx context.Context, state *interactionState, s *interactionTrace, answer bool) (*comp.CompiledRequest, error) {
	ctx, cancel := withTimeout(ctx, ra.callTimeout)
	defer cancel()
	phase := "act"
	if answer {
		phase = "answer"
	}
	var tools []comp.Tool
	for _, t := range ra.compiledTools {
		scope := ra.compilerSpec.ToolRequiredScopes[t.Name]
		if scope == "" || state.access.Subject.HasScope(scope) {
			tools = append(tools, t)
		}
	}
	request := comp.CompileRequest{
		CollectRequest: comp.CollectRequest{Summary: state.summaryResult.Summary, HistorySnapshot: state.summaryResult.Snapshot, SummaryReason: state.summaryResult.Reason, Access: state.access, SessionID: s.Session, InteractionID: s.InteractionID, TurnID: state.turnID, Turns: state.turns, Recovery: state.recovery, MaxItems: 1000, MaxBytes: ra.compilerSpec.MaxContextBytes, MaxCollectBytes: ra.compilerSpec.MaxCollectBytes, NotAfter: time.Now()},
		Model:          ra.model, Phase: phase, TemplateRef: state.templateRef, Variables: ra.compilerSpec.TemplateVariables, Tools: tools,
		ContextTokens: ra.compilerSpec.ContextTokens, OutputTokens: ra.compilerSpec.MaxTokens, SafetyMargin: ra.compilerSpec.SafetyMargin, Temperature: ra.compilerSpec.Temperature, TopP: ra.compilerSpec.TopP,
	}
	request.SummaryCalls = state.summaryResult.Calls
	request.SummaryBacklog = state.summaryResult.Backlog
	// Refresh state before collection; a stale/removed summary cannot suppress history.
	if request.Summary != nil && ra.summaryStore != nil {
		current, readErr := ra.summaryStore.GetSummary(ctx, state.access, s.Session)
		if summaryAccessFailure(readErr) {
			return nil, readErr
		}
		if readErr != nil || current == nil || current.Version != request.Summary.Version {
			request.Summary = nil
			request.SummaryReason = "SUMMARY_VERSION_CHANGED"
		}
	}
	result, err := ra.compiler.Compile(ctx, request)
	if err != nil {
		return nil, err
	}
	// At most one fallback selection after final provenance revalidation.
	if request.Summary != nil && ra.summaryStore != nil {
		current, readErr := ra.summaryStore.GetSummary(ctx, state.access, s.Session)
		if summaryAccessFailure(readErr) {
			return nil, readErr
		}
		if readErr != nil || current == nil || current.Version != request.Summary.Version {
			request.Summary = nil
			request.SummaryReason = "SUMMARY_VERSION_CHANGED"
			result, err = ra.compiler.Compile(ctx, request)
			if err != nil {
				return nil, err
			}
		}
	}
	state.allowedTools = map[string]bool{}
	for _, t := range result.Tools {
		state.allowedTools[t.Name] = true
	}
	return result, nil
}

func summaryAccessFailure(err error) bool {
	return errors.Is(err, store.ErrSessionNotFound) || errors.Is(err, security.ErrResourceNotAccessible) || errors.Is(err, security.ErrAuthenticationRequired)
}

// compiledPrompt has no hidden prompt bindings. It adapts exactly one compiled
// request, and verifies the tool definitions again before SDK dispatch.
type compiledPrompt struct {
	registry *genkit.Genkit
	request  *comp.CompiledRequest
}

func (p compiledPrompt) Name() string { return "compiled_react" }
func (p compiledPrompt) Render(ctx context.Context, input any) (*ai.GenerateActionOptions, error) {
	r := p.request
	opts := &ai.GenerateActionOptions{Model: r.Model, ReturnToolRequests: true, Config: &openai.ChatCompletionNewParams{Temperature: openai.Float(r.Temperature), MaxTokens: openai.Int(int64(r.OutputTokens))}}
	if r.TopP > 0 {
		opts.Config.(*openai.ChatCompletionNewParams).TopP = openai.Float(r.TopP)
	}
	for _, t := range r.Tools {
		actual := genkit.LookupTool(p.registry, t.Name)
		if actual == nil {
			return nil, fmt.Errorf("compiled tool unavailable")
		}
		d := actual.Definition()
		if d.Name != t.Name || d.Description != t.Description || !reflect.DeepEqual(comp.JSON(d.InputSchema), comp.JSON(t.InputSchema)) {
			return nil, fmt.Errorf("compiled tool definition changed")
		}
		opts.Tools = append(opts.Tools, t.Name)
	}
	for _, m := range r.Messages {
		var parts []*ai.Part
		if m.Text != "" {
			parts = append(parts, ai.NewTextPart(m.Text))
		}
		for _, call := range m.Requests {
			var input any
			if err := json.Unmarshal(call.Input, &input); err != nil {
				return nil, err
			}
			parts = append(parts, ai.NewToolRequestPart(&ai.ToolRequest{Name: call.Name, Ref: call.ID, Input: input}))
		}
		for _, call := range m.Responses {
			var output any
			if err := json.Unmarshal(call.Output, &output); err != nil {
				return nil, err
			}
			parts = append(parts, ai.NewToolResponsePart(&ai.ToolResponse{Name: call.Name, Ref: call.ID, Output: output}))
		}
		opts.Messages = append(opts.Messages, ai.NewMessage(ai.Role(m.Role), nil, parts...))
	}
	return opts, nil
}
func (p compiledPrompt) Execute(ctx context.Context, opts ...ai.PromptExecuteOption) (*ai.ModelResponse, error) {
	if len(opts) > 0 {
		return nil, fmt.Errorf("compiled requests cannot accept extra prompt options")
	}
	r, err := p.Render(ctx, nil)
	if err != nil {
		return nil, err
	}
	return genkit.GenerateWithRequest(ctx, p.registry, r, nil, nil)
}
