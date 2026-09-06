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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// ReActAgent is a ReAct-strategy agent: each interaction runs a single
// reason-and-act loop, bounded by maxIterations, until the model answers without
// requesting tools (or the budget is exhausted and the tool-less answer prompt
// forces a reply). A single agent is safe for concurrent interactions — all
// per-interaction state lives in the Channels/history reached through Interact,
// not on the agent.
type ReActAgent struct {
	registry  *genkit.Genkit
	memoryCtx context.Context

	actPrompt    ai.Prompt // reasons with tools available (native function calling)
	answerPrompt ai.Prompt // tool-less; forces a final answer when the budget is exhausted
	toolTimeouts toolTimeoutResolver
	hookManager  *hooks.Manager
	model        string

	maxIterations int
	callTimeout   time.Duration
	bufferSize    int
	lifecycleMu   sync.Mutex
	stopping      bool
	active        map[string]context.CancelFunc
	activeWG      sync.WaitGroup
}

var interactionSequence atomic.Uint64

var ErrAgentStopping = errors.New("agent is stopping")

// NewReActAgent builds a ReActAgent, assembling its prompts up front so the
// per-interaction hot path only executes them. It returns an error if the
// configured prompt file is missing.
func NewReActAgent(g *genkit.Genkit, spec *AgentSpec, toolTimeouts toolTimeoutResolver, hookManager *hooks.Manager, toolRefs []ai.ToolRef) (*ReActAgent, error) {
	memoryCtx := memory.NewMemoryContext(memory.ChatHistoryKey)

	ra := &ReActAgent{
		registry:      g,
		memoryCtx:     memoryCtx,
		toolTimeouts:  toolTimeouts,
		hookManager:   hookManager,
		model:         spec.Model,
		maxIterations: spec.MaxIterations,
		callTimeout:   time.Duration(spec.Timeout) * time.Second,
		bufferSize:    max(spec.ChannelBufferSize, 1),
	}

	act, answer, err := ra.buildPrompts(g, spec, spec.Model, toolRefs)
	if err != nil {
		return nil, err
	}
	ra.actPrompt = act
	ra.answerPrompt = answer
	return ra, nil
}

// Interact runs one interaction asynchronously and returns immediately with the
// Channels the caller streams from. The loop, final answer emission, and channel
// close all happen on a background goroutine; the caller owns draining Channels.
func (ra *ReActAgent) Interact(parent context.Context, input *schema.UserInput, sessionID string) *agent.Channels {
	chans := agent.NewChannels(max(ra.bufferSize, 1))
	interactionID := fmt.Sprintf("interaction-%d", interactionSequence.Add(1))
	interactionCtx, ok := ra.beginInteraction(parent, interactionID)
	if !ok {
		chans.ErrorChan <- ErrAgentStopping
		chans.Close()
		return chans
	}
	interactionStartedAt := time.Now()
	interactionCtx = emitHook(ra.hookManager, interactionCtx, hooks.State{
		Event:         hooks.EventInteractionStart,
		InteractionID: interactionID,
		SessionID:     sessionID,
		Iteration:     0,
		StartedAt:     interactionStartedAt,
	})
	chans.SetTraceID(hooks.TraceIDFromContext(interactionCtx))
	go func() {
		defer ra.finishInteraction(interactionID)
		var (
			history          *memory.HistoryMemory
			interactionState *interactionTrace
			interactionErr   error
		)
		defer func() {
			endedAt := time.Now()
			if recovered := recover(); recovered != nil {
				interactionErr = fmt.Errorf("agent interaction panicked: %v", recovered)
				chans.ErrorChan <- interactionErr
			}

			// Emit specialized events before the final interaction.end
			if interactionErr != nil {
				errorEvent := hooks.EventInteractionError
				if errors.Is(interactionErr, context.Canceled) {
					errorEvent = hooks.EventInteractionCancel
				}
				errorState := hooks.State{
					Event:         errorEvent,
					InteractionID: interactionID,
					SessionID:     sessionID,
					Error:         hooks.ErrorMessage(interactionErr),
					ErrorType:     hooks.ErrorType(interactionErr),
					StartedAt:     interactionStartedAt,
					EndedAt:       endedAt,
				}
				if interactionState != nil && interactionState.Usage != nil {
					errorState.InputTokens = interactionState.Usage.InputTokens
					errorState.OutputTokens = interactionState.Usage.OutputTokens
					errorState.TotalTokens = interactionState.Usage.TotalTokens
				}
				emitHook(ra.hookManager, interactionCtx, errorState)
			} else if interactionState != nil && (interactionState.FallbackUsed || interactionState.Degraded) {
				// Emit degrade event if fallback was used or any tool degraded
				degradeState := hooks.State{
					Event:          hooks.EventInteractionDegrade,
					InteractionID:  interactionID,
					SessionID:      sessionID,
					FallbackUsed:   interactionState.FallbackUsed,
					FallbackReason: interactionState.FallbackReason,
					Degraded:       interactionState.Degraded,
					StartedAt:      interactionStartedAt,
					EndedAt:        endedAt,
				}
				if interactionState.Usage != nil {
					degradeState.InputTokens = interactionState.Usage.InputTokens
					degradeState.OutputTokens = interactionState.Usage.OutputTokens
					degradeState.TotalTokens = interactionState.Usage.TotalTokens
				}
				emitHook(ra.hookManager, interactionCtx, degradeState)
			}

			endState := hooks.State{
				Event:         hooks.EventInteractionEnd,
				InteractionID: interactionID,
				SessionID:     sessionID,
				Error:         hooks.ErrorMessage(interactionErr),
				ErrorType:     hooks.ErrorType(interactionErr),
				StartedAt:     interactionStartedAt,
				EndedAt:       endedAt,
			}
			if interactionState != nil && interactionState.Usage != nil {
				endState.InputTokens = interactionState.Usage.InputTokens
				endState.OutputTokens = interactionState.Usage.OutputTokens
				endState.TotalTokens = interactionState.Usage.TotalTokens
			}
			emitHook(ra.hookManager, interactionCtx, endState)
			chans.Close()
			if history != nil {
				history.NextTurn(sessionID)
			}
		}()

		ctx, interactionHistory, err := ra.newInteraction(interactionCtx, input, sessionID)
		history = interactionHistory
		if err != nil {
			interactionErr = err
			chans.ErrorChan <- err
			return
		}
		interactionState = &interactionTrace{
			InteractionID: interactionID, Session: sessionID,
			Model: ra.model, hookManager: ra.hookManager, Usage: &ai.GenerationUsage{},
		}
		usage, err := ra.run(ctx, chans, interactionState)
		if err != nil {
			interactionErr = err
			chans.ErrorChan <- err
		}
		chans.Send(schema.StreamFinal(&schema.Observation{UsageInfo: usage}))

	}()
	return chans
}

func (ra *ReActAgent) beginInteraction(parent context.Context, interactionID string) (context.Context, bool) {
	if parent == nil {
		parent = context.Background()
	}
	ra.lifecycleMu.Lock()
	defer ra.lifecycleMu.Unlock()
	if ra.stopping {
		return nil, false
	}
	ctx, cancel := context.WithCancel(parent)
	if ra.active == nil {
		ra.active = make(map[string]context.CancelFunc)
	}
	ra.active[interactionID] = cancel
	ra.activeWG.Add(1)
	return ctx, true
}

func (ra *ReActAgent) finishInteraction(interactionID string) {
	ra.lifecycleMu.Lock()
	cancel := ra.active[interactionID]
	delete(ra.active, interactionID)
	ra.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	ra.activeWG.Done()
}

// Stop prevents new interactions, cancels active work, and waits until all
// interaction end hooks have run. Runtime shutdown calls this before stopping
// the hooks component, so the tracer provider can flush every tail span.
func (ra *ReActAgent) Stop() {
	ra.lifecycleMu.Lock()
	ra.stopping = true
	cancels := make([]context.CancelFunc, 0, len(ra.active))
	for _, cancel := range ra.active {
		cancels = append(cancels, cancel)
	}
	ra.lifecycleMu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	ra.activeWG.Wait()
}

// newInteraction records the user input into history and returns a session-scoped
// context plus the history store.
func (ra *ReActAgent) newInteraction(parent context.Context, input *schema.UserInput, sessionID string) (context.Context, *memory.HistoryMemory, error) {
	if input == nil {
		return nil, nil, errors.New("nil input")
	}
	if parent == nil {
		parent = context.Background()
	}
	history, err := memory.GetHistoryMemory(ra.memoryCtx, memory.ChatHistoryKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get history from context: %w", err)
	}

	// Record the user's message as plain text. The session id travels via
	// context (memory.SessionIDKey), so there is no need to wrap the input in a
	// JSON envelope the model would otherwise have to read through.
	history.AddHistory(sessionID, ai.NewUserMessage(ai.NewTextPart(input.Content)))

	ctx := context.WithValue(parent, memory.ChatHistoryKey, history)
	ctx = context.WithValue(ctx, memory.SessionIDKey, sessionID)
	ctx = withCurrentPageContext(ctx, input.Context)
	return ctx, history, nil
}

// GetMemory returns the agent's chat history store, or nil if it cannot be
// resolved from the agent's memory context.
func (ra *ReActAgent) GetMemory() *memory.HistoryMemory {
	h, err := memory.GetHistoryMemory(ra.memoryCtx, memory.ChatHistoryKey)
	if err != nil {
		return nil
	}
	return h
}
