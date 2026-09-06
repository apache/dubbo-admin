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
	"fmt"
	"time"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/component/memory"
	toolEngine "dubbo-admin-ai/component/tools/engine"
	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
)

// fallbackAnswer is streamed when the model returns no text on the forced final
// iteration, so an interaction always ends with a user-visible reply instead of
// bare stream markers.
const fallbackAnswer = "抱歉，我暂时无法生成回答，请稍后再试。"

// run drives the reason-and-act loop for one interaction. Each iteration is a
// single model call: with native function calling the model either requests
// tools (whose results are fed back as context for the next iteration) or
// answers directly — a tool-free response IS the final answer, so no separate
// "observe" reasoning step is needed to decide when to stop. The last allowed
// iteration uses the tool-less answer prompt so the loop always terminates with
// a real answer rather than an exhausted-budget silence.
//
// run streams the answer itself and returns the interaction's accumulated token
// usage; the caller emits the final usage marker and closes the channels.
func (ra *ReActAgent) run(ctx context.Context, chans *agent.Channels, s *interactionTrace) (*ai.GenerationUsage, error) {
	history, sessionID, err := historyFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if history.IsEmpty(sessionID) {
		return nil, fmt.Errorf("history is empty")
	}

	if s.Usage == nil {
		s.Usage = &ai.GenerationUsage{}
	}
	s.Session = sessionID
	s.Model = ra.model
	s.hookManager = ra.hookManager
	for i := 0; i < ra.maxIterations; i++ {
		s.Iteration = i + 1
		done, err := ra.runIteration(ctx, chans, history, s, i == ra.maxIterations-1)
		if err != nil || done {
			return s.Usage, err
		}
	}
	return s.Usage, nil
}

// runIteration makes one model call and executes its tools or streams its answer.
func (ra *ReActAgent) runIteration(ctx context.Context, chans *agent.Channels, history *memory.HistoryMemory, s *interactionTrace, forceAnswer bool) (done bool, err error) {
	iterationStartedAt := time.Now()
	iterationCtx := emitHook(s.hookManager, ctx, hooks.State{
		Event: hooks.EventIterationStart, InteractionID: s.InteractionID,
		SessionID: s.Session, Iteration: s.Iteration, StartedAt: iterationStartedAt,
	})
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("iteration %d panicked: %v", s.Iteration, recovered)
			emitIterationEnd(s, iterationCtx, iterationStartedAt, err)
			panic(recovered)
		}
		emitIterationEnd(s, iterationCtx, iterationStartedAt, err)
	}()

	prompt := ra.actPrompt
	s.Stage = "reasonAct"
	if forceAnswer {
		prompt = ra.answerPrompt
		s.Stage = "answer"
	}
	stageStartedAt := time.Now()
	ctx = emitHook(s.hookManager, iterationCtx, hooks.State{
		Event: hooks.EventStageStart, InteractionID: s.InteractionID,
		SessionID: s.Session, Iteration: s.Iteration, Stage: s.Stage,
		Model: s.Model, StartedAt: stageStartedAt,
	})
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("stage %s panicked: %v", s.Stage, recovered)
			emitStageEnd(s, ctx, stageStartedAt, err)
			panic(recovered)
		}
		emitStageEnd(s, ctx, stageStartedAt, err)
	}()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	messages, err := injectCurrentPageContext(ctx, history.WindowMemory(s.Session))
	if err != nil {
		return false, err
	}
	// Model and tool deadlines are independent; cancellation still propagates.
	resp, err := func() (*ai.ModelResponse, error) {
		lctx, cancel := withTimeout(ctx, ra.callTimeout)
		defer cancel()
		return executeModelCall(lctx, s, prompt, messages, func(resp *ai.ModelResponse, err error) {
			if err == nil && forceAnswer && resp.Text() == "" {
				s.FallbackUsed = true
				s.FallbackReason = hooks.FallbackReasonEmptyResponse
			}
		}, ai.WithMessages(messages...))
	}()
	if err != nil {
		return false, fmt.Errorf("failed to execute react prompt: %w", err)
	}
	schema.AccumulateUsage(s.Usage, resp.Usage)
	if !forceAnswer {
		if reqs := resp.ToolRequests(); len(reqs) > 0 {
			runtime.GetLogger().Debug("react: model requested tools", "count", len(reqs))
			agent.EmitProgress(chans, "🔍 分析问题并调用工具中...\n")
			return false, ra.execTools(ctx, history, s, reqs)
		}
	}
	answer := resp.Text()
	if answer == "" {
		if !forceAnswer {
			runtime.GetLogger().Warn("react: empty model response, retrying", "iteration", s.Iteration)
			return false, nil
		}
		runtime.GetLogger().Warn("react: empty forced answer, using fallback")
		answer = fallbackAnswer
	}
	ra.finish(chans, history, s.Session, answer)
	return true, nil
}

// execTools runs every requested tool under its own timeout and records the
// results into history as a model message so the next iteration can read them.
// A failed tool degrades (its error is recorded as the tool's output) rather
// than aborting the interaction, so the model can still answer from whatever
// other tools returned.
func (ra *ReActAgent) execTools(ctx context.Context, history *memory.HistoryMemory, s *interactionTrace, reqs []*ai.ToolRequest) error {
	var parts []*ai.Part
	for _, req := range reqs {
		if err := ctx.Err(); err != nil {
			return err
		}
		output, err := func() (toolOutput toolEngine.ToolOutput, callErr error) {
			tctx, cancel := withTimeout(ctx, ra.toolTimeouts.For(req.Name))
			defer cancel()
			return ra.executeToolCall(tctx, s, req)
		}()
		if err != nil {
			s.Degraded = true
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		outputJSON, err := json.Marshal(output)
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}
		parts = append(parts, ai.NewJSONPart(string(outputJSON)))
	}
	runtime.GetLogger().Debug("react: recorded tool results", "count", len(parts))
	// ai.RoleTool messages are ignored by ai.WithMessages, so tool results are
	// recorded as a model message.
	history.AddHistory(s.Session, ai.NewMessage(ai.RoleModel, nil, parts...))
	return nil
}

// finish records the answer into history and streams it to the user, closing the
// content block exactly once.
func (ra *ReActAgent) finish(chans *agent.Channels, history *memory.HistoryMemory, sessionID, answer string) {
	if answer != "" {
		history.AddHistory(sessionID, ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart(answer)))
	}
	if chans == nil {
		return
	}
	if answer != "" {
		chans.Send(schema.NewStreamFeedback(answer + "\n"))
	}
	chans.Send(schema.StreamEnd())
}

// historyFromCtx pulls the session-scoped history out of ctx.
func historyFromCtx(ctx context.Context) (*memory.HistoryMemory, string, error) {
	history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.HistoryMemory)
	if !ok {
		return nil, "", fmt.Errorf("failed to get history from context")
	}
	sessionID, ok := ctx.Value(memory.SessionIDKey).(string)
	if !ok || sessionID == "" {
		return nil, "", fmt.Errorf("session id not found in context")
	}
	return history, sessionID, nil
}

// withTimeout wraps ctx with a deadline when timeout > 0; otherwise it returns
// ctx unchanged with a no-op cancel so callers can defer unconditionally.
func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// toolTimeoutResolver resolves a tool's execution timeout by name, falling back
// to the shared default when a tool has no explicit per-tool override.
type toolTimeoutResolver struct {
	def     time.Duration
	perTool map[string]time.Duration
}

func newToolTimeoutResolver(defSeconds int, overrideSeconds map[string]int) toolTimeoutResolver {
	perTool := make(map[string]time.Duration, len(overrideSeconds))
	for name, s := range overrideSeconds {
		perTool[name] = time.Duration(s) * time.Second
	}
	return toolTimeoutResolver{
		def:     time.Duration(defSeconds) * time.Second,
		perTool: perTool,
	}
}

func (r toolTimeoutResolver) For(name string) time.Duration {
	if d, ok := r.perTool[name]; ok {
		return d
	}
	return r.def
}
