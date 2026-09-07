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
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func recordLoopHooks(t *testing.T, ra *ReActAgent) *[]hooks.State {
	t.Helper()
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	states := new([]hooks.State)
	if err := manager.Register(hooks.Registration{
		Events: hooks.AllEvents(), Tools: []string{hooks.AllTools}, CaptureContent: true,
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			*states = append(*states, state)
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}
	ra.hookManager = manager
	ra.model = "test/model"
	ra.bufferSize = 32
	return states
}

func TestSingleLoopTerminalHooks(t *testing.T) {
	tests := []struct {
		name     string
		prompt   ai.Prompt
		special  hooks.Event
		fallback bool
	}{
		{name: "answer", prompt: &stubPrompt{resp: textResp("answer")}},
		{name: "model error", prompt: &stubPrompt{err: errors.New("failed")}, special: hooks.EventInteractionError},
		{name: "model timeout", prompt: &stubPrompt{err: context.DeadlineExceeded}, special: hooks.EventInteractionError},
		{name: "cancel", prompt: &stubPrompt{err: context.Canceled}, special: hooks.EventInteractionCancel},
		{name: "panic", prompt: &panicPrompt{}, special: hooks.EventInteractionError},
		{name: "nil response", prompt: &stubPrompt{}, special: hooks.EventInteractionError},
		{name: "empty answer", prompt: &stubPrompt{resp: textResp("")}, special: hooks.EventInteractionDegrade, fallback: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ra := testAgent(nil, 1, nil)
			ra.actPrompt, ra.answerPrompt = tt.prompt, tt.prompt
			states := recordLoopHooks(t, ra)
			channels := ra.Interact(context.Background(), &schema.UserInput{Content: "hello"}, "session")
			select {
			case <-channels.Done():
			case <-time.After(2 * time.Second):
				t.Fatal("interaction failed to close")
			}
			want := []hooks.Event{hooks.EventInteractionStart, hooks.EventIterationStart, hooks.EventStageStart, hooks.EventModelCallStart}
			failed := tt.special == hooks.EventInteractionError || tt.special == hooks.EventInteractionCancel
			if failed {
				want = append(want, hooks.EventModelCallError)
			}
			want = append(want, hooks.EventModelCallEnd)
			if failed {
				want = append(want, hooks.EventStageError)
			}
			want = append(want, hooks.EventStageEnd, hooks.EventIterationEnd)
			if tt.special != "" {
				want = append(want, tt.special)
			}
			want = append(want, hooks.EventInteractionEnd)
			if len(*states) != len(want) {
				t.Fatalf("states = %+v, want %v", *states, want)
			}
			for i, event := range want {
				state := (*states)[i]
				if state.Event != event {
					t.Fatalf("event[%d] = %s, want %s", i, state.Event, event)
				}
				if state.InteractionID == "" || state.SessionID != "session" {
					t.Fatalf("missing identity: %+v", state)
				}
				if state.Event == hooks.EventModelCallEnd || state.Event == hooks.EventStageEnd {
					if state.Stage != "answer" || state.Model != "test/model" {
						t.Fatalf("wrong stage/model: %+v", state)
					}
					if state.FallbackUsed != tt.fallback {
						t.Fatalf("wrong fallback flag: %+v", state)
					}
					if tt.fallback && state.FallbackReason != hooks.FallbackReasonEmptyResponse {
						t.Fatalf("wrong fallback reason: %+v", state)
					}
				}
			}
		})
	}
}

func TestSingleLoopToolOrderUsageAndPageContext(t *testing.T) {
	g := genkit.Init(context.Background())
	for _, name := range []string{"first", "second"} {
		name := name
		genkit.DefineTool(g, name, name, func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
			if name == "second" {
				return nil, errors.New("tool failed")
			}
			return map[string]any{"tool_name": name, "summary": "ok", "result": input}, nil
		})
	}
	first := &ai.ModelResponse{
		Message: ai.NewModelMessage(
			ai.NewToolRequestPart(&ai.ToolRequest{Name: "first", Ref: "call-1", Input: map[string]any{"q": "hello"}}),
			ai.NewToolRequestPart(&ai.ToolRequest{Name: "second", Ref: "call-2", Input: map[string]any{}}),
		),
		Usage: &ai.GenerationUsage{InputTokens: 2, OutputTokens: 3, TotalTokens: 5},
	}
	final := textResp("answer despite failure")
	final.Usage = &ai.GenerationUsage{InputTokens: 7, OutputTokens: 11, TotalTokens: 18}
	ra := testAgent(g, 2, &scriptPrompt{resps: []*ai.ModelResponse{first, final}})
	states := recordLoopHooks(t, ra)
	channels := ra.Interact(context.Background(), &schema.UserInput{Content: "question", Context: &schema.AIContextSnapshot{
		Version: schema.AIContextVersion, Page: schema.AIContextPage{Path: "/home"},
	}}, "session")
	select {
	case <-channels.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("interaction failed to close")
	}
	var tools []string
	modelCalls := 0
	degraded := false
	for _, state := range *states {
		switch state.Event {
		case hooks.EventModelCallStart:
			modelCalls++
			if !strings.Contains(state.Input, "page_context") {
				t.Fatalf("context missing from model input: %s", state.Input)
			}
		case hooks.EventToolCallStart, hooks.EventToolCallEnd, hooks.EventToolCallError:
			tools = append(tools, string(state.Event)+":"+state.ToolCallID)
			if state.ToolName == "second" && state.Event != hooks.EventToolCallStart && (!state.Degraded || state.Error == "" || state.Output != "") {
				t.Fatalf("failed tool metadata = %+v", state)
			}
		case hooks.EventInteractionDegrade:
			degraded = state.Degraded
		case hooks.EventInteractionEnd:
			if state.InputTokens != 9 || state.OutputTokens != 14 || state.TotalTokens != 23 {
				t.Fatalf("incorrect aggregate usage: %+v", state)
			}
		}
	}
	wantTools := "tool_call.start:call-1,tool_call.end:call-1,tool_call.start:call-2,tool_call.error:call-2,tool_call.end:call-2"
	if strings.Join(tools, ",") != wantTools {
		t.Fatalf("tool order = %v", tools)
	}
	if modelCalls != 2 || !degraded {
		t.Fatalf("model calls=%d degraded=%v", modelCalls, degraded)
	}
	if strings.Contains(historyText(ra.GetMemory(), "session"), "page_context") {
		t.Fatal("page context leaked into persisted history")
	}
}

func TestSingleLoopRejectsNilInputAndPreservesParentCancellation(t *testing.T) {
	ra := testAgent(nil, 1, &scriptPrompt{resps: []*ai.ModelResponse{textResp("unused")}})
	channels := ra.Interact(context.Background(), nil, "session")
	select {
	case <-channels.Done():
	case <-time.After(time.Second):
		t.Fatal("nil input did not terminate")
	}
	if err := <-channels.ErrorChan; err == nil || err.Error() != "nil input" {
		t.Fatalf("error = %v", err)
	}
	parent, cancel := context.WithCancel(context.Background())
	ctx, _, err := ra.newInteraction(parent, &schema.UserInput{Content: "hello"}, "session")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("parent cancellation lost: %v", ctx.Err())
	}
}

func TestToolErrorEndEventIsDegraded(t *testing.T) {
	g := genkit.Init(context.Background())
	genkit.DefineTool(g, "broken_tool", "broken", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
		return nil, errors.New("boom")
	})
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var end hooks.State
	if err := manager.Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventToolCallEnd}, Tools: []string{"broken_tool"}, CaptureContent: true,
		Hook: func(ctx context.Context, state hooks.State) context.Context { end = state; return ctx },
	}); err != nil {
		t.Fatal(err)
	}
	s := &interactionTrace{Session: "session", InteractionID: "interaction", Iteration: 1, Stage: "reasonAct", hookManager: manager}
	_, err := testAgent(g, 1, nil).executeToolCall(context.Background(), s, &ai.ToolRequest{Name: "broken_tool", Input: map[string]any{"x": 1}})
	if err == nil {
		t.Fatal("broken tool unexpectedly succeeded")
	}
	if !end.Degraded || end.Error == "" || end.ErrorType == "" {
		t.Fatalf("tool end state = %+v, want degraded error metadata", end)
	}
	if end.Output != "" {
		t.Fatalf("failed tool exported synthetic output: %s", end.Output)
	}
}

func TestModelCallPanicStillEmitsEnd(t *testing.T) {
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var events []hooks.State
	if err := manager.Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventModelCallStart, hooks.EventModelCallEnd},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			events = append(events, state)
			return ctx
		},
	}); err != nil {
		t.Fatalf("register model hook: %v", err)
	}
	s := &interactionTrace{Session: "session", InteractionID: "interaction", hookManager: manager}

	func() {
		defer func() { _ = recover() }()
		_, _ = executeModelCall(context.Background(), s, &panicPrompt{}, nil, nil)
	}()

	if len(events) != 2 || events[0].Event != hooks.EventModelCallStart || events[1].Event != hooks.EventModelCallEnd {
		t.Fatalf("events = %+v, want model start/end", events)
	}
	if events[1].Error == "" {
		t.Fatal("model end event did not record panic error")
	}
}
