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
func (ra *ReActAgent) run(ctx context.Context, chans *agent.Channels) (*ai.GenerationUsage, error) {
	sessionID, err := sessionIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	turnID, _ := ctx.Value(turnIDContextKey).(uint64)
	var history *memory.HistoryMemory
	if ra.messageStore == nil {
		history, _, err = historyFromCtx(ctx)
		if err != nil {
			return nil, err
		}
		if history.IsEmpty(sessionID) {
			return nil, fmt.Errorf("history is empty")
		}
	} else {
		empty, err := ra.messageStore.IsTurnEmpty(ctx, sessionID, turnID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect history: %w", err)
		}
		if empty {
			return nil, fmt.Errorf("history is empty")
		}
	}

	usage := &ai.GenerationUsage{}
	for i := 0; i < ra.maxIterations; i++ {
		// The final iteration must answer: drop the tools so the model can only
		// synthesize from what it has already gathered.
		forceAnswer := i == ra.maxIterations-1
		prompt := ra.actPrompt
		if forceAnswer {
			prompt = ra.answerPrompt
		}

		// Only the model call is bound by the per-call timeout; tool execution
		// below runs on the original ctx so a slow reasoning step can't starve the
		// tools it just asked for on a shared deadline.
		lctx, cancel := withTimeout(ctx, ra.callTimeout)
		var messages []*ai.Message
		if ra.messageStore != nil {
			messages, err = ra.messageStore.WindowMemoryForTurn(ctx, sessionID, turnID)
		} else {
			messages = history.WindowMemory(sessionID)
		}
		if err != nil {
			cancel()
			return usage, fmt.Errorf("failed to load conversation history: %w", err)
		}
		resp, err := prompt.Execute(lctx, ai.WithMessages(messages...))
		cancel()
		if err != nil {
			return usage, fmt.Errorf("failed to execute react prompt: %w", err)
		}
		schema.AccumulateUsage(usage, resp.Usage)

		if !forceAnswer {
			if reqs := resp.ToolRequests(); len(reqs) > 0 {
				runtime.GetLogger().Debug("react: model requested tools", "count", len(reqs))
				agent.EmitProgress(chans, "🔍 分析问题并调用工具中...\n")
				if err := ra.execTools(ctx, history, sessionID, turnID, reqs); err != nil {
					return usage, err
				}
				continue
			}
		}

		// A tool-free response IS the final answer — but an empty response has
		// nothing to stream. While iterations remain, retry rather than finish on
		// silence; on the forced last iteration substitute an explicit fallback so
		// the loop always terminates with a real reply.
		answer := resp.Text()
		if answer == "" {
			if !forceAnswer {
				runtime.GetLogger().Warn("react: empty model response, retrying", "iteration", i)
				continue
			}
			runtime.GetLogger().Warn("react: empty forced answer, using fallback")
			answer = fallbackAnswer
		}

		if err := ra.finish(ctx, chans, history, sessionID, turnID, answer); err != nil {
			return usage, err
		}
		return usage, nil
	}

	// Unreachable: the final iteration always answers and returns above.
	return usage, nil
}

// execTools runs every requested tool under its own timeout and records the
// results into history as a model message so the next iteration can read them.
// A failed tool degrades (its error is recorded as the tool's output) rather
// than aborting the interaction, so the model can still answer from whatever
// other tools returned.
func (ra *ReActAgent) execTools(ctx context.Context, history *memory.HistoryMemory, sessionID string, turnID uint64, reqs []*ai.ToolRequest) error {
	var parts []*ai.Part
	for _, req := range reqs {
		tctx, cancel := withTimeout(ctx, ra.toolTimeouts.For(req.Name))
		output, err := toolEngine.Call(tctx, ra.registry, req.Name, req.Input)
		cancel()
		if err != nil {
			runtime.GetLogger().Warn("tool call failed, continuing with degraded context",
				"tool", req.Name, "error", err)
			output = toolEngine.ToolOutput{
				ToolName: req.Name,
				Summary:  fmt.Sprintf("tool %q failed: %v", req.Name, err),
			}
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
	message := ai.NewMessage(ai.RoleModel, nil, parts...)
	if ra.messageStore != nil {
		if err := ra.messageStore.AddHistoryToTurn(persistenceContext(ctx), sessionID, turnID, message); err != nil {
			return fmt.Errorf("failed to record tool output: %w", err)
		}
	} else {
		history.AddHistory(sessionID, message)
	}
	return nil
}

// finish records the answer into history and streams it to the user, closing the
// content block exactly once.

func (ra *ReActAgent) finish(ctx context.Context, chans *agent.Channels, history *memory.HistoryMemory, sessionID string, turnID uint64, answer string) error {
	if answer != "" {
		message := ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart(answer))
		if ra.messageStore != nil {
			if err := ra.messageStore.AddHistoryToTurn(persistenceContext(ctx), sessionID, turnID, message); err != nil {
				return fmt.Errorf("failed to persist final answer: %w", err)
			}
		} else {
			history.AddHistory(sessionID, message)
		}
	}
	if chans == nil {
		return nil
	}
	if answer != "" {
		chans.Send(schema.NewStreamFeedback(answer + "\n"))
	}
	chans.Send(schema.StreamEnd())
	return nil
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

func sessionIDFromCtx(ctx context.Context) (string, error) {
	if sessionID, ok := ctx.Value(sessionIDContextKey).(string); ok && sessionID != "" {
		return sessionID, nil
	}
	if sessionID, ok := ctx.Value(memory.SessionIDKey).(string); ok && sessionID != "" {
		return sessionID, nil
	}
	return "", fmt.Errorf("session id not found in context")
}

func persistenceContext(ctx context.Context) context.Context {
	if state, ok := ctx.Value(persistenceContextKey).(context.Context); ok && state != nil {
		return state
	}
	return ctx
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
