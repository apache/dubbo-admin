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

	"dubbo-admin-ai/component/agent"
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

	maxIterations int
	callTimeout   time.Duration
	bufferSize    int
}

// NewReActAgent builds a ReActAgent, assembling its prompts up front so the
// per-interaction hot path only executes them. It returns an error if the
// configured prompt file is missing.
func NewReActAgent(g *genkit.Genkit, spec *AgentSpec, toolTimeouts toolTimeoutResolver, toolRefs []ai.ToolRef) (*ReActAgent, error) {
	memoryCtx := memory.NewMemoryContext(memory.ChatHistoryKey)

	ra := &ReActAgent{
		registry:      g,
		memoryCtx:     memoryCtx,
		toolTimeouts:  toolTimeouts,
		maxIterations: spec.MaxIterations,
		callTimeout:   time.Duration(spec.Timeout) * time.Second,
		bufferSize:    spec.ChannelBufferSize,
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
func (ra *ReActAgent) Interact(input *schema.UserInput, sessionID string) *agent.Channels {
	chans := agent.NewChannels(ra.bufferSize)
	go func() {
		ctx, history, err := ra.newInteraction(input, sessionID)
		if err != nil {
			chans.ErrorChan <- err
			chans.Close()
			return
		}

		usage, err := ra.run(ctx, chans)
		if err != nil {
			chans.ErrorChan <- err
		}
		if usage == nil {
			usage = &ai.GenerationUsage{}
		}

		// Emit the final marker for the SSE layer; it carries the accumulated
		// usage the MessageDelta needs. The answer text itself was already
		// streamed by run.
		chans.Send(schema.StreamFinal(&schema.Observation{UsageInfo: usage}))

		chans.Close()
		history.NextTurn(sessionID)
	}()
	return chans
}

// newInteraction records the user input into history and returns a session-scoped
// context plus the history store.
func (ra *ReActAgent) newInteraction(input *schema.UserInput, sessionID string) (context.Context, *memory.HistoryMemory, error) {
	history, err := memory.GetHistoryMemory(ra.memoryCtx, memory.ChatHistoryKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get history from context: %w", err)
	}

	// Record the user's message as plain text. The session id travels via
	// context (memory.SessionIDKey), so there is no need to wrap the input in a
	// JSON envelope the model would otherwise have to read through.
	history.AddHistory(sessionID, ai.NewUserMessage(ai.NewTextPart(input.Content)))

	ctx := context.WithValue(ra.memoryCtx, memory.SessionIDKey, sessionID)
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
