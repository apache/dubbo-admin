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
	"fmt"
	"time"

	comp "dubbo-admin-ai/component/context"
	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/component/memory/summary"
	"dubbo-admin-ai/providerusage"
	"dubbo-admin-ai/store"

	"github.com/firebase/genkit/go/ai"
	"github.com/google/uuid"
)

func (ra *ReActAgent) initSummary(spec *AgentSpec) error {
	// Register even when disabled: the compiler sees no candidate without a
	// validated snapshot. Toggling the flag never revives an old assembly path.
	if err := ra.compiler.Register(comp.SummarySource{}, comp.SourcePolicy{Order: 5, HistoryCoverage: true, OptionalFailure: true, Kinds: []string{"evidence"}}); err != nil {
		return err
	}
	if !spec.ConversationSummary.Enabled {
		return nil
	}
	backend, ok := ra.messageStore.(store.SummaryStore)
	if !ok {
		return fmt.Errorf("conversation store does not support summaries")
	}
	ra.summaryStore = backend
	cfg := spec.ConversationSummary
	cfg.Defaults()
	if cfg.Model == "" {
		cfg.Model = ra.model
	}
	t := comp.Template{ID: "conversation-summary", Version: summary.TemplateVersion, Shared: summary.SystemPrompt}
	c, err := comp.New(t)
	if err != nil {
		return err
	}
	if err := c.Register(comp.SnapshotSource{Kind: "input"}, comp.SourcePolicy{Required: true, Kinds: []string{"user"}}); err != nil {
		return err
	}
	ra.summaryCompiler = c
	ra.summaryService = &summary.Service{Store: backend, Config: cfg, Generator: summary.GenerateFunc(ra.generateSummary)}
	return nil
}

func (ra *ReActAgent) generateSummary(ctx context.Context, r summary.GenerateRequest) (text string, err error) {
	started := time.Now()
	hook := hooks.State{Event: hooks.EventModelCallStart, CallID: uuid.NewString(), CallType: "summary", Auxiliary: true, SessionID: r.SessionID, InteractionID: r.InteractionID, Stage: "summary", Model: r.Model, StartedAt: started}
	ctx = emitHook(ra.hookManager, ctx, hook)
	defer func() {
		hook.Event = hooks.EventModelCallEnd
		hook.EndedAt = time.Now()
		if err != nil {
			hook.Error = hooks.ErrorMessage(err)
			hook.ErrorType = hooks.ErrorType(err)
		}
		emitHook(ra.hookManager, ctx, hook)
	}()
	t := comp.Template{ID: "conversation-summary", Version: summary.TemplateVersion, Shared: summary.SystemPrompt}
	input, err := summary.Input(r)
	if err != nil {
		return "", fmt.Errorf("SUMMARY_INPUT_INVALID: %w", err)
	}
	request := comp.CompileRequest{CollectRequest: comp.CollectRequest{Access: r.Access, SessionID: r.SessionID, InteractionID: r.InteractionID, TurnID: 1, Turns: []store.TurnContext{{ID: 1, TenantID: r.Access.Subject.TenantID, OwnerID: r.Access.Subject.UserID, Messages: []*ai.Message{ai.NewUserMessage(ai.NewTextPart(input))}}}, MaxItems: 1, MaxBytes: ra.compilerSpec.MaxContextBytes, MaxCollectBytes: ra.compilerSpec.MaxCollectBytes, NotAfter: time.Now()}, Model: r.Model, Phase: "answer", TemplateRef: t.Ref(), ContextTokens: ra.compilerSpec.ContextTokens, OutputTokens: r.OutputTokens, SafetyMargin: ra.compilerSpec.SafetyMargin}
	if r.ContextTokens > 0 {
		request.ContextTokens = r.ContextTokens
	}
	request.TurnID = r.ActiveTurn
	request.Turns[0].ID = r.ActiveTurn
	compiled, err := ra.summaryCompiler.Compile(ctx, request)
	if err != nil {
		return "", err
	}
	hook.TemplateRef = compiled.TemplateRef
	hook.CompilerVersion = compiled.Report.CompilerVersion
	hook.EstimatorVersion = compiled.Report.EstimatorVersion
	hook.EstimatedInputTokens = compiled.Report.InputTokens
	// Observe the provider boundary so an allocated all-zero Usage (the pinned
	// adapter's representation of an omitted usage object) is not persisted as
	// provider-reported zero usage.
	ctx = providerusage.WithObservation(ctx)
	response, err := (compiledPrompt{registry: ra.registry, request: compiled}).Execute(ctx)
	if err != nil {
		return "", err
	}
	if response == nil {
		return "", fmt.Errorf("empty summary response")
	}
	if response.Usage != nil {
		hook.UsagePresent = true
		hook.InputTokens = response.Usage.InputTokens
		hook.OutputTokens = response.Usage.OutputTokens
		hook.TotalTokens = response.Usage.TotalTokens
	}
	if tokens, observed := providerusage.FromContext(ctx); observed {
		hook.UsagePresent = tokens != nil
		hook.InputTokens, hook.OutputTokens, hook.TotalTokens = 0, 0, 0
		if tokens != nil {
			hook.InputTokens, hook.OutputTokens, hook.TotalTokens = tokens.Input, tokens.Output, tokens.Total
		}
	}
	return response.Text(), nil
}
