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
	"dubbo-admin-ai/component/hooks"
	toolEngine "dubbo-admin-ai/component/tools/engine"
	"dubbo-admin-ai/runtime"
	"errors"
	"fmt"
	"github.com/firebase/genkit/go/ai"
	"time"
)

// interactionTrace holds per-interaction hook metadata and token accounting.
// It does not drive the loop or retain model/tool payloads.
type interactionTrace struct {
	Session        string
	InteractionID  string
	Iteration      int
	Stage          string
	Model          string
	hookManager    *hooks.Manager
	Usage          *ai.GenerationUsage
	FallbackUsed   bool
	FallbackReason string
	Degraded       bool
}

func emitStageEnd(s *interactionTrace, ctx context.Context, startedAt time.Time, err error) {
	now := time.Now()
	// Emit error event if the stage failed
	if err != nil {
		emitHook(s.hookManager, ctx, hooks.State{
			Event:          hooks.EventStageError,
			InteractionID:  s.InteractionID,
			SessionID:      s.Session,
			Iteration:      s.Iteration,
			Stage:          s.Stage,
			Model:          s.Model,
			Error:          hooks.ErrorMessage(err),
			ErrorType:      hooks.ErrorType(err),
			FallbackUsed:   s.FallbackUsed,
			FallbackReason: s.FallbackReason,
			StartedAt:      startedAt,
			EndedAt:        now,
		})
	}
	emitHook(s.hookManager, ctx, hooks.State{
		Event:          hooks.EventStageEnd,
		InteractionID:  s.InteractionID,
		SessionID:      s.Session,
		Iteration:      s.Iteration,
		Stage:          s.Stage,
		Model:          s.Model,
		Error:          hooks.ErrorMessage(err),
		ErrorType:      hooks.ErrorType(err),
		FallbackUsed:   s.FallbackUsed,
		FallbackReason: s.FallbackReason,
		StartedAt:      startedAt,
		EndedAt:        now,
	})
}

type modelCallAfterExecute func(*ai.ModelResponse, error)

func executeModelCall(
	ctx context.Context,
	s *interactionTrace,
	prompt ai.Prompt,
	input any,
	afterExecute modelCallAfterExecute,
	opts ...ai.PromptExecuteOption,
) (resp *ai.ModelResponse, err error) {
	startedAt := time.Now()
	startState := hooks.State{
		Event:         hooks.EventModelCallStart,
		InteractionID: s.InteractionID,
		SessionID:     s.Session,
		Iteration:     s.Iteration,
		Stage:         s.Stage,
		Model:         s.Model,
		StartedAt:     startedAt,
	}
	startState = withHookInput(s.hookManager, hooks.EventModelCallStart, "", startState, func() any {
		return genAIInputMessages(input)
	})
	ctx = emitHook(s.hookManager, ctx, startState)
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("model call panicked: %v", recovered)
			emitModelCallEnd(ctx, s, startedAt, resp, err)
			panic(recovered)
		}
		emitModelCallEnd(ctx, s, startedAt, resp, err)
	}()
	resp, err = prompt.Execute(ctx, opts...)
	if err == nil && resp == nil {
		err = errors.New("model returned a nil response")
	}
	if afterExecute != nil {
		afterExecute(resp, err)
	}
	return resp, err
}

func emitModelCallEnd(ctx context.Context, s *interactionTrace, startedAt time.Time, output any, err error) {
	now := time.Now()
	// Emit error event if the call failed
	if err != nil {
		errorState := hooks.State{
			Event:          hooks.EventModelCallError,
			InteractionID:  s.InteractionID,
			SessionID:      s.Session,
			Iteration:      s.Iteration,
			Stage:          s.Stage,
			Model:          s.Model,
			Error:          hooks.ErrorMessage(err),
			ErrorType:      hooks.ErrorType(err),
			FallbackUsed:   s.FallbackUsed,
			FallbackReason: s.FallbackReason,
			StartedAt:      startedAt,
			EndedAt:        now,
		}
		emitHook(s.hookManager, ctx, errorState)
	}

	endState := hooks.State{
		Event:          hooks.EventModelCallEnd,
		InteractionID:  s.InteractionID,
		SessionID:      s.Session,
		Iteration:      s.Iteration,
		Stage:          s.Stage,
		Model:          s.Model,
		Error:          hooks.ErrorMessage(err),
		ErrorType:      hooks.ErrorType(err),
		FallbackUsed:   s.FallbackUsed,
		FallbackReason: s.FallbackReason,
		StartedAt:      startedAt,
		EndedAt:        now,
	}
	endState = withHookOutput(s.hookManager, hooks.EventModelCallEnd, "", endState, func() any {
		return genAIOutputMessages(output)
	})
	if response, ok := output.(*ai.ModelResponse); ok && response != nil && response.Usage != nil {
		endState.InputTokens = response.Usage.InputTokens
		endState.OutputTokens = response.Usage.OutputTokens
		endState.TotalTokens = response.Usage.TotalTokens
	}
	emitHook(s.hookManager, ctx, endState)
}

func (ra *ReActAgent) executeToolCall(ctx context.Context, s *interactionTrace, req *ai.ToolRequest) (output toolEngine.ToolOutput, callErr error) {
	startedAt := time.Now()
	startState := hooks.State{
		Event:         hooks.EventToolCallStart,
		InteractionID: s.InteractionID,
		SessionID:     s.Session,
		Iteration:     s.Iteration,
		Stage:         s.Stage,
		Model:         s.Model,
		ToolName:      req.Name,
		ToolCallID:    req.Ref,
		StartedAt:     startedAt,
	}
	startState = withHookInput(s.hookManager, hooks.EventToolCallStart, req.Name, startState, func() any {
		return req.Input
	})
	ctx = emitHook(s.hookManager, ctx, startState)
	defer func() {
		if recovered := recover(); recovered != nil {
			callErr = fmt.Errorf("tool call panicked: %v", recovered)
			emitToolCallEnd(ctx, s, req, startedAt, output, callErr)
			panic(recovered)
		}
		emitToolCallEnd(ctx, s, req, startedAt, output, callErr)
	}()

	output, callErr = toolEngine.Call(ctx, ra.registry, req.Name, req.Input)
	if callErr != nil {
		// Degrade instead of aborting: record the failure as a tool output so the
		// next iteration can still compose an answer from the remaining context.
		runtime.GetLogger().Warn("tool call failed, continuing with degraded context",
			"tool", req.Name, "error", callErr)
		output = toolEngine.ToolOutput{
			ToolName: req.Name,
			Summary:  fmt.Sprintf("tool %q failed: %v", req.Name, callErr),
		}
	}
	return output, callErr
}

func emitToolCallEnd(ctx context.Context, s *interactionTrace, req *ai.ToolRequest, startedAt time.Time, output toolEngine.ToolOutput, err error) {
	now := time.Now()
	// Emit error event if the call failed (degraded)
	if err != nil {
		errorState := hooks.State{
			Event:         hooks.EventToolCallError,
			InteractionID: s.InteractionID,
			SessionID:     s.Session,
			Iteration:     s.Iteration,
			Stage:         s.Stage,
			Model:         s.Model,
			ToolName:      req.Name,
			ToolCallID:    req.Ref,
			Error:         hooks.ErrorMessage(err),
			ErrorType:     hooks.ErrorType(err),
			Degraded:      true,
			StartedAt:     startedAt,
			EndedAt:       now,
		}
		emitHook(s.hookManager, ctx, errorState)
	}

	endState := hooks.State{
		Event:         hooks.EventToolCallEnd,
		InteractionID: s.InteractionID,
		SessionID:     s.Session,
		Iteration:     s.Iteration,
		Stage:         s.Stage,
		Model:         s.Model,
		ToolName:      req.Name,
		ToolCallID:    req.Ref,
		Error:         hooks.ErrorMessage(err),
		ErrorType:     hooks.ErrorType(err),
		Degraded:      err != nil,
		StartedAt:     startedAt,
		EndedAt:       now,
	}
	if err == nil {
		endState = withHookOutput(s.hookManager, hooks.EventToolCallEnd, req.Name, endState, func() any {
			return output
		})
	}
	emitHook(s.hookManager, ctx, endState)
}

func emitIterationEnd(s *interactionTrace, ctx context.Context, startedAt time.Time, err error) {
	emitHook(s.hookManager, ctx, hooks.State{
		Event:         hooks.EventIterationEnd,
		InteractionID: s.InteractionID,
		SessionID:     s.Session,
		Iteration:     s.Iteration,
		Error:         hooks.ErrorMessage(err),
		ErrorType:     hooks.ErrorType(err),
		StartedAt:     startedAt,
		EndedAt:       time.Now(),
	})
}

func emitHook(manager *hooks.Manager, ctx context.Context, state hooks.State) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if manager == nil {
		return ctx
	}
	return manager.Emit(ctx, state)
}

func withHookInput(manager *hooks.Manager, event hooks.Event, toolName string, state hooks.State, provider func() any) hooks.State {
	if manager == nil || !manager.NeedsContent(event, toolName) {
		return state
	}
	return state.WithInputContent(provider)
}

func withHookOutput(manager *hooks.Manager, event hooks.Event, toolName string, state hooks.State, provider func() any) hooks.State {
	if manager == nil || !manager.NeedsContent(event, toolName) {
		return state
	}
	return state.WithOutputContent(provider)
}
