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
	compcontext "dubbo-admin-ai/component/context"
	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/component/memory"
	rt "dubbo-admin-ai/runtime"
	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"

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
	compiler         *compcontext.Compiler
	compilationStore conversationstore.CompilationStore
	compilerSpec     AgentSpec
	templateRef      string
	compiledTools    []compcontext.Tool
	registry         *genkit.Genkit
	memoryCtx        context.Context
	messageStore     conversationstore.MessageStore

	actPrompt    ai.Prompt // reasons with tools available (native function calling)
	answerPrompt ai.Prompt // tool-less; forces a final answer when the budget is exhausted
	toolTimeouts toolTimeoutResolver
	hookManager  *hooks.Manager
	model        string

	maxIterations      int
	contextWindowTurns int
	callTimeout        time.Duration
	bufferSize         int
	lifecycleMu        sync.Mutex
	stopping           bool
	active             map[string]context.CancelFunc
	activeWG           sync.WaitGroup
}

var interactionSequence atomic.Uint64

var ErrAgentStopping = errors.New("agent is stopping")

const persistenceTimeout = 30 * time.Second

type interactionState struct {
	turns         []conversationstore.TurnContext
	recovery      conversationstore.ToolRecovery
	templateRef   string
	allowedTools  map[string]bool
	access        security.AccessContext
	turnID        uint64
	interactionID string
	prepared      bool
	messages      []*ai.Message
	persistCtx    context.Context
	persistCancel context.CancelFunc
}

func (s *interactionState) cancelPersistence() {
	if s != nil && s.persistCancel != nil {
		s.persistCancel()
	}
}

// NewReActAgent retains the old constructor signature for callers migrating to
// explicit conversation storage. Model calls now require NewReActAgentWithStore.
func NewReActAgent(g *genkit.Genkit, spec *AgentSpec, toolTimeouts toolTimeoutResolver, hookManager *hooks.Manager, toolRefs []ai.ToolRef) (*ReActAgent, error) {
	return NewReActAgentWithStore(g, nil, spec, toolTimeouts, hookManager, toolRefs)
}

// NewReActAgentWithStore shares conversation storage with memory and tools.
func NewReActAgentWithStore(g *genkit.Genkit, messageStore conversationstore.MessageStore, spec *AgentSpec, toolTimeouts toolTimeoutResolver, hookManager *hooks.Manager, toolRefs []ai.ToolRef) (*ReActAgent, error) {
	if messageStore == nil {
		return nil, fmt.Errorf("conversation store is required; use NewReActAgentWithStore")
	}
	if spec == nil {
		return nil, fmt.Errorf("agent specification required")
	}
	copySpec := *spec
	copySpec.ApplyDefaults()
	spec = &copySpec
	contextWindowTurns := spec.ContextWindowTurns
	if contextWindowTurns <= 0 {
		contextWindowTurns = defaultContextWindowTurns
	}
	ra := &ReActAgent{
		registry:           g,
		messageStore:       messageStore,
		toolTimeouts:       toolTimeouts,
		hookManager:        hookManager,
		model:              spec.Model,
		maxIterations:      spec.MaxIterations,
		contextWindowTurns: contextWindowTurns,
		callTimeout:        time.Duration(spec.Timeout) * time.Second,
		bufferSize:         max(spec.ChannelBufferSize, 1),
	}

	var ok bool
	ra.compilationStore, ok = messageStore.(conversationstore.CompilationStore)
	if !ok {
		return nil, fmt.Errorf("conversation store does not support context compilation")
	}
	if err := ra.initCompiler(spec, toolRefs); err != nil {
		return nil, err
	}
	return ra, nil
}

// Interact runs one interaction asynchronously and returns immediately with the
// Channels the caller streams from. The loop, final answer emission, and channel
// close all happen on a background goroutine; the caller owns draining Channels.
func (ra *ReActAgent) Interact(parent context.Context, input *schema.UserInput, sessionID string) *agent.Channels {
	chans := agent.NewChannels(max(ra.bufferSize, 1))
	interactionID := fmt.Sprintf("interaction-%d", interactionSequence.Add(1))
	if prepared, ok := conversationstore.PreparedInteractionFromContext(parent); ok {
		interactionID = prepared.InteractionID
	}
	interactionCtx, ok := ra.beginInteraction(parent, interactionID)
	if !ok {
		ra.failPreparedInteraction(parent, sessionID)
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
		_, hasPreparedInteraction := conversationstore.PreparedInteractionFromContext(interactionCtx)
		preparedNeedsCleanup := hasPreparedInteraction
		defer func() {
			if preparedNeedsCleanup {
				ra.failPreparedInteraction(interactionCtx, sessionID)
			}
		}()
		var (
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
		}()

		ctx, state, err := ra.newInteraction(interactionCtx, input, sessionID)
		if err != nil {
			interactionErr = err
			chans.ErrorChan <- err
			return
		}
		completed := false
		defer func() {
			if ra.messageStore != nil && !completed {
				cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(interactionCtx), persistenceTimeout)
				var abortErr error
				if state.prepared {
					abortErr = ra.messageStore.FailInteraction(cleanupCtx, state.access, sessionID, state.interactionID, state.turnID)
				} else {
					abortErr = ra.messageStore.AbortTurnForTurn(cleanupCtx, state.access, sessionID, state.turnID)
				}
				if abortErr != nil && !errors.Is(abortErr, conversationstore.ErrTurnNotFound) {
					rt.GetLogger().Error("Failed to abort AI interaction turn", "session_id", sessionID, "turn_id", state.turnID, "error_type", hooks.ErrorType(abortErr))
				}
				cancel()
			}
			state.cancelPersistence()
		}()
		preparedNeedsCleanup = false

		interactionState = &interactionTrace{
			InteractionID: interactionID, Session: sessionID,
			Model: ra.model, hookManager: ra.hookManager, Usage: &ai.GenerationUsage{},
		}
		usage, err := ra.run(ctx, chans, state, interactionState)
		if err != nil {
			interactionErr = err
			chans.ErrorChan <- err
			return
		}
		if ra.messageStore != nil {
			var completeErr error
			if state.prepared {
				completeErr = ra.messageStore.CompleteInteraction(state.persistCtx, state.access, sessionID, state.interactionID, state.turnID, interactionState.FinalAnswer)
			} else {
				completeErr = ra.messageStore.NextTurnForTurn(state.persistCtx, state.access, sessionID, state.turnID)
			}
			if completeErr != nil {
				interactionErr = fmt.Errorf("failed to complete turn: %w", completeErr)
				chans.ErrorChan <- interactionErr
				return
			}
			completed = true
		} else if history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.HistoryMemory); ok {
			history.NextTurn(sessionID)
		}
		if usage == nil {
			usage = &ai.GenerationUsage{}
		}
		chans.Send(schema.StreamFinal(&schema.Observation{UsageInfo: usage}))

	}()
	return chans
}

func (ra *ReActAgent) failPreparedInteraction(parent context.Context, sessionID string) {
	if ra.messageStore == nil {
		return
	}
	prepared, ok := conversationstore.PreparedInteractionFromContext(parent)
	if !ok || prepared.Replayed {
		return
	}
	access, err := security.AccessContextFromContext(parent)
	if err != nil {
		rt.GetLogger().Error("Failed to resolve access for prepared interaction cleanup", "session_id", sessionID, "interaction_id", prepared.InteractionID, "error_type", hooks.ErrorType(err))
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), persistenceTimeout)
	defer cancel()
	if err := ra.messageStore.FailInteraction(cleanupCtx, access, sessionID, prepared.InteractionID, prepared.TurnID); err != nil && !errors.Is(err, conversationstore.ErrTurnNotFound) {
		rt.GetLogger().Error("Failed to release rejected prepared interaction", "session_id", sessionID, "interaction_id", prepared.InteractionID, "error_type", hooks.ErrorType(err))
	}
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

// newInteraction records the user input and returns a session-scoped context
// plus the interaction's turn/persistence state.
func (ra *ReActAgent) newInteraction(parent context.Context, input *schema.UserInput, sessionID string) (context.Context, *interactionState, error) {
	if parent == nil {
		parent = context.Background()
	}
	if input == nil {
		return nil, nil, errors.New("nil input")
	}
	if ra.messageStore != nil {
		access, err := security.AccessContextFromContext(parent)
		if err != nil {
			return nil, nil, err
		}
		prepared, isPrepared := conversationstore.PreparedInteractionFromContext(parent)
		var turnID uint64
		var interactionID string
		if isPrepared {
			turnID = prepared.TurnID
			interactionID = prepared.InteractionID
		} else {
			turnID, err = ra.messageStore.BeginTurn(parent, access, sessionID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to begin turn: %w", err)
			}
		}
		persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(parent), persistenceTimeout)
		if err := ra.messageStore.AddHistoryToTurn(parent, access, sessionID, turnID, ai.NewUserMessage(ai.NewTextPart(input.Content))); err != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), persistenceTimeout)
			var abortErr error
			if isPrepared {
				abortErr = ra.messageStore.FailInteraction(cleanupCtx, access, sessionID, interactionID, turnID)
			} else {
				abortErr = ra.messageStore.AbortTurnForTurn(cleanupCtx, access, sessionID, turnID)
			}
			cancel()
			persistCancel()
			if abortErr != nil && !errors.Is(abortErr, conversationstore.ErrTurnNotFound) {
				return nil, nil, fmt.Errorf("failed to record user message: %w (also failed to abort turn: %v)", err, abortErr)
			}
			return nil, nil, fmt.Errorf("failed to record user message: %w", err)
		}
		var messages []*ai.Message
		if ra.compiler == nil {
			messages, err = ra.messageStore.ContextWindowForTurn(parent, access, sessionID, turnID, ra.contextWindowTurns)
		}
		if err != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), persistenceTimeout)
			var abortErr error
			if isPrepared {
				abortErr = ra.messageStore.FailInteraction(cleanupCtx, access, sessionID, interactionID, turnID)
			} else {
				abortErr = ra.messageStore.AbortTurnForTurn(cleanupCtx, access, sessionID, turnID)
			}
			cancel()
			persistCancel()
			if abortErr != nil && !errors.Is(abortErr, conversationstore.ErrTurnNotFound) {
				return nil, nil, fmt.Errorf("failed to load conversation context: %w (also failed to abort turn: %v)", err, abortErr)
			}
			return nil, nil, fmt.Errorf("failed to load conversation context: %w", err)
		}
		state := &interactionState{access: access, turnID: turnID, interactionID: interactionID, prepared: isPrepared, messages: messages, persistCtx: persistCtx, persistCancel: persistCancel}
		if ra.compiler != nil {
			state.templateRef, err = ra.compilationStore.PinTemplate(parent, access, sessionID, ra.templateRef)
			if err == nil {
				limit := ra.contextWindowTurns
				if ra.compilerSpec.ContextCompilerMode == "minimal" {
					limit = 0
				}
				state.turns, err = ra.compilationStore.ContextTurns(parent, access, sessionID, turnID, limit, ra.compilerSpec.MaxCollectBytes)
			}
			if err == nil {
				state.recovery, err = ra.compilationStore.RecoveryToolBatches(parent, access, sessionID, turnID, 16, min(conversationstore.MaxToolBatchBytes, ra.compilerSpec.MaxCollectBytes))
			}
			if err != nil {
				if errors.Is(err, conversationstore.ErrResourceExhausted) {
					err = fmt.Errorf("%w: bounded context snapshot", compcontext.ErrMandatoryBudget)
				}
				if isPrepared {
					_ = ra.messageStore.FailInteraction(persistCtx, access, sessionID, interactionID, turnID)
				} else {
					_ = ra.messageStore.AbortTurnForTurn(persistCtx, access, sessionID, turnID)
				}
				persistCancel()
				return nil, nil, err
			}
		}
		ctx := context.WithValue(parent, sessionIDContextKey, sessionID)
		ctx = context.WithValue(ctx, turnIDContextKey, turnID)
		ctx = context.WithValue(ctx, persistenceContextKey, persistCtx)
		return ctx, state, nil
	}

	// Compatibility path for callers that construct ReActAgent directly in tests.
	if ra.memoryCtx == nil {
		ra.memoryCtx = memory.NewMemoryContext(memory.ChatHistoryKey)
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
	return ctx, &interactionState{}, nil
}

type contextKey string

const (
	sessionIDContextKey   contextKey = "session"
	turnIDContextKey      contextKey = "turn"
	persistenceContextKey contextKey = "persistence"
)

// GetMemory returns the agent's chat history store, or nil if it cannot be
// resolved from the agent's memory context.
func (ra *ReActAgent) GetMemory() *memory.HistoryMemory {
	if ra.memoryCtx == nil {
		return nil
	}
	h, err := memory.GetHistoryMemory(ra.memoryCtx, memory.ChatHistoryKey)
	if err != nil {
		return nil
	}
	return h
}
