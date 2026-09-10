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
	"time"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/memory"
	rt "dubbo-admin-ai/runtime"
	"dubbo-admin-ai/schema"
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
	registry     *genkit.Genkit
	memoryCtx    context.Context
	messageStore conversationstore.MessageStore

	actPrompt    ai.Prompt // reasons with tools available (native function calling)
	answerPrompt ai.Prompt // tool-less; forces a final answer when the budget is exhausted
	toolTimeouts toolTimeoutResolver

	maxIterations int
	callTimeout   time.Duration
	bufferSize    int
}

const persistenceTimeout = 30 * time.Second

type interactionState struct {
	turnID        uint64
	persistCtx    context.Context
	persistCancel context.CancelFunc
}

func (s *interactionState) cancelPersistence() {
	if s != nil && s.persistCancel != nil {
		s.persistCancel()
	}
}

// NewReActAgent builds a ReActAgent, assembling its prompts up front so the
// per-interaction hot path only executes them. It returns an error if the
// configured prompt file is missing.
// NewReActAgent preserves the upstream constructor for callers that still use
// the compatibility HistoryMemory path. Runtime components should use
// NewReActAgentWithStore so all conversation data shares one Store instance.
func NewReActAgent(g *genkit.Genkit, spec *AgentSpec, toolTimeouts toolTimeoutResolver, toolRefs []ai.ToolRef) (*ReActAgent, error) {
	return NewReActAgentWithStore(g, nil, spec, toolTimeouts, toolRefs)
}

// NewReActAgentWithStore constructs an agent backed by the shared conversation
// store used by the memory, session, and tool components.
func NewReActAgentWithStore(g *genkit.Genkit, messageStore conversationstore.MessageStore, spec *AgentSpec, toolTimeouts toolTimeoutResolver, toolRefs []ai.ToolRef) (*ReActAgent, error) {
	ra := &ReActAgent{
		registry:      g,
		messageStore:  messageStore,
		toolTimeouts:  toolTimeouts,
		maxIterations: spec.MaxIterations,
		callTimeout:   time.Duration(spec.Timeout) * time.Second,
		bufferSize:    spec.ChannelBufferSize,
	}
	if messageStore == nil {
		ra.memoryCtx = memory.NewMemoryContext(memory.ChatHistoryKey)
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
	chans := agent.NewChannels(ra.bufferSize)
	go func() {
		if parent == nil {
			parent = context.Background()
		}
		ctx, state, err := ra.newInteraction(parent, input, sessionID)
		if err != nil {
			chans.ErrorChan <- err
			chans.Close()
			return
		}
		completed := false
		defer func() {
			if ra.messageStore != nil && !completed {
				cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), persistenceTimeout)
				if abortErr := ra.messageStore.AbortTurnForTurn(cleanupCtx, sessionID, state.turnID); abortErr != nil && !errors.Is(abortErr, conversationstore.ErrTurnNotFound) {
					rt.GetLogger().Error("Failed to abort AI interaction turn", "session_id", sessionID, "turn_id", state.turnID, "error", abortErr)
				}
				cancel()
			}
			state.cancelPersistence()
		}()

		usage, err := ra.run(ctx, chans)
		if err != nil {
			chans.ErrorChan <- err
			chans.Close()
			return
		}
		if ra.messageStore != nil {
			if err := ra.messageStore.NextTurnForTurn(state.persistCtx, sessionID, state.turnID); err != nil {
				chans.ErrorChan <- fmt.Errorf("failed to complete turn: %w", err)
				chans.Close()
				return
			}
			completed = true
		}
		if usage == nil {
			usage = &ai.GenerationUsage{}
		}

		// Emit the final marker for the SSE layer; it carries the accumulated
		// usage the MessageDelta needs. The answer text itself was already
		// streamed by run.
		chans.Send(schema.StreamFinal(&schema.Observation{UsageInfo: usage}))

		chans.Close()
		if ra.messageStore == nil {
			if history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.HistoryMemory); ok {
				history.NextTurn(sessionID)
			}
		}
	}()
	return chans
}

// newInteraction records the user input and returns a session-scoped context
// plus the interaction's turn/persistence state.
func (ra *ReActAgent) newInteraction(parent context.Context, input *schema.UserInput, sessionID string) (context.Context, *interactionState, error) {
	if input == nil {
		return nil, nil, fmt.Errorf("user input is nil")
	}
	if ra.messageStore != nil {
		turnID, err := ra.messageStore.BeginTurn(parent, sessionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to begin turn: %w", err)
		}
		persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(parent), persistenceTimeout)
		if err := ra.messageStore.AddHistoryToTurn(parent, sessionID, turnID, ai.NewUserMessage(ai.NewTextPart(input.Content))); err != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), persistenceTimeout)
			abortErr := ra.messageStore.AbortTurnForTurn(cleanupCtx, sessionID, turnID)
			cancel()
			persistCancel()
			if abortErr != nil && !errors.Is(abortErr, conversationstore.ErrTurnNotFound) {
				return nil, nil, fmt.Errorf("failed to record user message: %w (also failed to abort turn: %v)", err, abortErr)
			}
			return nil, nil, fmt.Errorf("failed to record user message: %w", err)
		}
		ctx := context.WithValue(parent, sessionIDContextKey, sessionID)
		ctx = context.WithValue(ctx, turnIDContextKey, turnID)
		ctx = context.WithValue(ctx, persistenceContextKey, persistCtx)
		return ctx, &interactionState{turnID: turnID, persistCtx: persistCtx, persistCancel: persistCancel}, nil
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

	ctx := context.WithValue(ra.memoryCtx, memory.SessionIDKey, sessionID)
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
