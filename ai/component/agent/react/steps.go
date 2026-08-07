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
	"strings"
	"time"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/memory"
	toolEngine "dubbo-admin-ai/component/tools/engine"
	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
)

// buildSteps materializes the step closures for one interaction, binding the
// per-interaction channels so progress/streaming reaches the right consumer.
func (ra *ReActAgent) buildSteps(chans *agent.Channels) []step {
	steps := make([]step, 0, len(ra.stages))
	for _, st := range ra.stages {
		switch st.kind {
		case flowReasonAct:
			steps = append(steps, ra.reasonActStep(st.prompt, chans, st.timeout))
		case flowObserve:
			steps = append(steps, ra.observeStep(st.prompt, chans, st.timeout))
		}
	}
	return steps
}

// historyFromCtx pulls the session-scoped history out of ctx, replacing the
// pointer/value assertion churn the old flows repeated at every stage.
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

// reasonActStep merges the old think + act stages: one model call reasons about
// the request and, via native function calling, either issues tool requests
// (which it executes) or issues none (answering directly). The observe stage
// then composes the reply, so this step never terminates the loop.
func (ra *ReActAgent) reasonActStep(prompt ai.Prompt, chans *agent.Channels, timeout time.Duration) step {
	return func(ctx context.Context, s *state) (bool, error) {
		emitStageProgress(chans, flowReasonAct, true)
		defer emitStageProgress(chans, flowReasonAct, false)

		history, sessionID, err := historyFromCtx(ctx)
		if err != nil {
			return false, err
		}
		if history.IsEmpty(sessionID) {
			return false, fmt.Errorf("history is empty")
		}

		// Only the model call is bound by the stage timeout; tool execution below
		// runs on the original ctx so a slow reasoning step can't starve the tools
		// it just asked for (which would otherwise fail hard on the shared deadline).
		lctx, cancel := withTimeout(ctx, timeout)
		resp, err := prompt.Execute(lctx, ai.WithMessages(history.WindowMemory(sessionID)...))
		cancel()
		if err != nil {
			return false, fmt.Errorf("failed to execute reasonAct prompt: %w", err)
		}
		s.addUsage(resp.Usage)

		toolReqs := resp.ToolRequests()
		runtime.GetLogger().Info("tool requests:", "req", toolReqs)

		// No tools needed: the model answered directly. Record its reasoning so
		// the observe stage can build on it, and leave tool outputs empty.
		if len(toolReqs) == 0 {
			if text := resp.Text(); text != "" {
				history.AddHistory(sessionID, ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart(text)))
			}
			s.Tools = &schema.ToolOutputs{UsageInfo: &ai.GenerationUsage{}}
			return false, nil
		}

		var parts []*ai.Part
		actOuts := &schema.ToolOutputs{UsageInfo: &ai.GenerationUsage{}}
		for _, req := range toolReqs {
			// Each tool runs under its own timeout (per-tool override, else the
			// shared default), independent of the model call's budget above.
			tctx, cancel := withTimeout(ctx, ra.toolTimeouts.For(req.Name))
			output, err := toolEngine.Call(tctx, ra.registry, req.Name, req.Input)
			cancel()
			if err != nil {
				// Degrade instead of aborting: record the failure as a tool output
				// so the observe stage can still compose an answer (or explain the
				// gap) from whatever other tools returned.
				runtime.GetLogger().Warn("tool call failed, continuing with degraded context",
					"tool", req.Name, "error", err)
				output = toolEngine.ToolOutput{
					ToolName: req.Name,
					Summary:  fmt.Sprintf("tool %q failed: %v", req.Name, err),
				}
			}
			outputJson, err := json.Marshal(output)
			if err != nil {
				return false, fmt.Errorf("failed to marshal output: %w", err)
			}
			parts = append(parts, ai.NewJSONPart(string(outputJson)))
			actOuts.Add(&output)
		}
		runtime.GetLogger().Info("act out:", "out", actOuts)
		// ai.RoleTool's messages will be ignored by ai.WithMessages
		history.AddHistory(sessionID, ai.NewMessage(ai.RoleModel, nil, parts...))
		s.Tools = actOuts
		return false, nil
	}
}

func (ra *ReActAgent) observeStep(prompt ai.Prompt, chans *agent.Channels, timeout time.Duration) step {
	return func(ctx context.Context, s *state) (bool, error) {
		emitStageProgress(chans, flowObserve, true)
		defer emitStageProgress(chans, flowObserve, false)

		history, sessionID, err := historyFromCtx(ctx)
		if err != nil {
			return false, err
		}
		if history.IsEmpty(sessionID) {
			return false, fmt.Errorf("history is empty")
		}

		obsCtx, cancel := withTimeout(ctx, timeout)
		defer cancel()

		var observation *schema.Observation
		resp, err := prompt.Execute(obsCtx, ai.WithMessages(history.WindowMemory(sessionID)...))
		switch {
		case err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)):
			runtime.GetLogger().Warn("Observe stage timeout, returning fallback response", "timeout", timeout)
			fb := generateFallbackObservation(s)
			observation = &fb
		case err != nil:
			return false, fmt.Errorf("failed to execute observe prompt: %w", err)
		default:
			// The model responded and consumed tokens regardless of whether its
			// output parses, so account for usage before attempting the parse.
			s.addUsage(resp.Usage)
			observation, err = ra.fallback.ParseObservation(resp)
			if err != nil {
				runtime.GetLogger().Warn("Failed to parse observation, returning fallback", "error", err)
				fb := generateFallbackObservation(s)
				observation = &fb
			}
		}
		runtime.GetLogger().Info("Observe out:", "out", observation)

		history.AddHistory(sessionID, ra.fallback.MarshalObservation(observation))
		observation.UsageInfo = s.Usage
		s.Observe = observation

		// Stream the observation to the user, preserving the old emission order.
		emitObservation(chans, observation)

		return !observation.Heartbeat && observation.FinalAnswer != "", nil
	}
}

// emitObservation streams the observation's user-facing text and closes the
// content block. Only FinalAnswer is user-facing; Summary is an internal status
// line (see agentObserve.txt's Output Contract) and is deliberately NOT streamed
// — emitting it would prepend an internal status to every answer. Progress is
// already conveyed by the stage markers (emitStageProgress).
func emitObservation(chans *agent.Channels, obs *schema.Observation) {
	if chans == nil {
		return
	}
	if obs.FinalAnswer != "" {
		chans.Send(schema.NewStreamFeedback(obs.FinalAnswer + "\n"))
	}
	chans.Send(schema.StreamEnd())
}

// generateFallbackObservation creates a fallback observation when the observe
// stage times out or its output can't be parsed. It prefers concrete tool
// outputs, then the think stage's thought, matching the old switch behaviour.
func generateFallbackObservation(s *state) schema.Observation {
	fb := schema.Observation{
		Heartbeat:   false,
		FinalAnswer: "",
		Summary:     "Generate response based on available context",
		Evidence:    "Timeout - using available context",
	}

	switch {
	case s.Tools != nil && len(s.Tools.Outputs) > 0:
		fb.FinalAnswer = generateResponseFromToolOutputs(s.Tools.Outputs)
	default:
		fb.FinalAnswer = "I apologize, but I need more time to process your request. Based on the available context, I cannot provide a complete answer at this moment."
	}

	return fb
}

// generateResponseFromToolOutputs generates a response from tool outputs
func generateResponseFromToolOutputs(outputs []toolEngine.ToolOutput) string {
	if len(outputs) == 0 {
		return "No tool results available to answer your question."
	}

	var resultParts []string
	for _, output := range outputs {
		if output.Summary != "" {
			resultParts = append(resultParts, output.Summary)
		}
	}

	if len(resultParts) > 0 {
		return fmt.Sprintf("Tool execution results: %s", strings.Join(resultParts, "; "))
	}
	return "Tool execution completed but no detailed results available."
}

// emitStageProgress renders the react-specific progress line for a stage
// boundary and streams it via the generic agent primitive.
func emitStageProgress(chans *agent.Channels, stageName string, started bool) {
	agent.EmitProgress(chans, stageProgressText(stageName, started))
}

func stageProgressText(stageName string, started bool) string {
	if started {
		switch stageName {
		case flowReasonAct:
			return "🔍 分析问题并调用工具中...\n"
		case flowObserve:
			return "🧠 整理结论中...\n"
		default:
			return fmt.Sprintf("⏳ %s 阶段处理中...\n", stageName)
		}
	}

	switch stageName {
	case flowReasonAct:
		return "✅ 分析与工具调用完成。\n"
	case flowObserve:
		return "✅ 结论整理完成。\n"
	default:
		return fmt.Sprintf("✅ %s 阶段完成。\n", stageName)
	}
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
