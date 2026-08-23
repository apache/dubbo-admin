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

	"dubbo-admin-ai/component/agent/fallback"
	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// stubPrompt is an ai.Prompt whose Execute returns a canned response/error,
// letting the step unit tests run without a live model.
type stubPrompt struct {
	resp *ai.ModelResponse
	err  error
}

type panicPrompt struct{ stubPrompt }

func (p *panicPrompt) Execute(context.Context, ...ai.PromptExecuteOption) (*ai.ModelResponse, error) {
	panic("model panic")
}

func (s *stubPrompt) Name() string { return "stub" }
func (s *stubPrompt) Execute(ctx context.Context, opts ...ai.PromptExecuteOption) (*ai.ModelResponse, error) {
	return s.resp, s.err
}
func (s *stubPrompt) Render(ctx context.Context, input any) (*ai.GenerateActionOptions, error) {
	return &ai.GenerateActionOptions{}, nil
}

func contextWithHistory(sessionID string) context.Context {
	ctx := memory.NewMemoryContext(memory.ChatHistoryKey)
	history, _ := memory.GetHistoryMemory(ctx, memory.ChatHistoryKey)
	history.AddHistory(sessionID, ai.NewUserMessage(ai.NewTextPart("hello")))
	ctx = context.WithValue(ctx, memory.SessionIDKey, sessionID)
	return ctx
}

func testAgent(g *genkit.Genkit) *ReActAgent {
	return &ReActAgent{
		registry:  g,
		memoryCtx: memory.NewMemoryContext(memory.ChatHistoryKey),
		fallback:  fallback.NewHandler(),
	}
}

func textResp(text string) *ai.ModelResponse {
	return &ai.ModelResponse{Message: ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart(text))}
}

func toolReqResp(name string, input map[string]any) *ai.ModelResponse {
	return &ai.ModelResponse{Message: ai.NewMessage(ai.RoleModel, nil,
		ai.NewToolRequestPart(&ai.ToolRequest{Name: name, Input: input}),
	)}
}

func toolReqsResp(requests ...*ai.ToolRequest) *ai.ModelResponse {
	parts := make([]*ai.Part, 0, len(requests))
	for _, request := range requests {
		parts = append(parts, ai.NewToolRequestPart(request))
	}
	return &ai.ModelResponse{Message: ai.NewMessage(ai.RoleModel, nil, parts...)}
}

func TestReasonActStep(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*genkit.Genkit) ai.Prompt
		errContain string
		assertFn   func(t *testing.T, s *state)
	}{
		{
			name:  "no_tool_call_answers_directly",
			setup: func(*genkit.Genkit) ai.Prompt { return &stubPrompt{resp: textResp("Dubbo is an RPC framework.")} },
			assertFn: func(t *testing.T, s *state) {
				if s.Tools == nil || len(s.Tools.Outputs) != 0 {
					t.Fatalf("expected empty tool outputs, got %+v", s.Tools)
				}
			},
		},
		{
			name: "with_tool_call_returns_outputs",
			setup: func(g *genkit.Genkit) ai.Prompt {
				genkit.DefineTool(g, "mock_tool", "mock tool", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
					return map[string]any{"tool_name": "mock_tool", "summary": "ok", "result": map[string]any{"echo": input["q"]}}, nil
				})
				return &stubPrompt{resp: toolReqResp("mock_tool", map[string]any{"q": "ping"})}
			},
			assertFn: func(t *testing.T, s *state) {
				if s.Tools == nil || len(s.Tools.Outputs) < 1 || s.Tools.Outputs[0].ToolName != "mock_tool" {
					t.Fatalf("unexpected tool outputs: %+v", s.Tools)
				}
			},
		},
		{
			// A failing tool must not abort the interaction: the step records the
			// failure as a tool output (so observe can degrade) and returns no error.
			name: "tool_error_degrades_not_aborts",
			setup: func(g *genkit.Genkit) ai.Prompt {
				genkit.DefineTool(g, "broken_tool", "broken", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
					return nil, errors.New("boom")
				})
				return &stubPrompt{resp: toolReqResp("broken_tool", map[string]any{"x": 1})}
			},
			assertFn: func(t *testing.T, s *state) {
				if s.Tools == nil || len(s.Tools.Outputs) != 1 {
					t.Fatalf("expected one recorded tool output, got %+v", s.Tools)
				}
				out := s.Tools.Outputs[0]
				if out.ToolName != "broken_tool" || !strings.Contains(out.Summary, "failed") {
					t.Fatalf("expected degraded failure output for broken_tool, got %+v", out)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := genkit.Init(context.Background())
			ra := testAgent(g)
			s := &state{Input: &schema.UserInput{Content: "hi"}, Session: "session", Usage: &ai.GenerationUsage{}}

			done, err := ra.reasonActStep(tt.setup(g), nil, 0)(contextWithHistory("session"), s)
			if tt.errContain != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("expected error containing %q, got %v", tt.errContain, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("reasonAct step error: %v", err)
			}
			if done {
				t.Fatalf("reasonAct step should never terminate the loop")
			}
			tt.assertFn(t, s)
		})
	}
}

func TestReasonActStepExecuteError(t *testing.T) {
	g := genkit.Init(context.Background())
	ra := testAgent(g)
	prompt := &stubPrompt{resp: nil, err: errors.New("execute failed")}
	s := &state{Input: &schema.UserInput{Content: "hi"}, Session: "s3", Usage: &ai.GenerationUsage{}}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	_, err := ra.reasonActStep(prompt, nil, 0)(contextWithHistory("s3"), s)
	if err == nil || !strings.Contains(err.Error(), "failed to execute reasonAct prompt") {
		t.Fatalf("expected wrapped execute error, got %v", err)
	}
}

func TestObserveTimeoutReturnsFallback(t *testing.T) {
	ra := testAgent(nil)
	prompt := &cancelAwarePrompt{started: make(chan struct{})}
	s := &state{Input: &schema.UserInput{Content: "hi"}, Session: "session", Usage: &ai.GenerationUsage{}}

	done, err := ra.observeStep(prompt, nil, time.Millisecond)(contextWithHistory("session"), s)
	if err != nil {
		t.Fatalf("observeStep() error = %v", err)
	}
	if !done || s.Observe == nil || s.Observe.FinalAnswer == "" {
		t.Fatalf("observe fallback = %+v, done = %v", s.Observe, done)
	}
}

func TestObserveCancellationPropagatesWithoutFallback(t *testing.T) {
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var modelEnd hooks.State
	if err := manager.Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventModelCallEnd},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			modelEnd = state
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}
	s := &state{
		Input: &schema.UserInput{Content: "hi"}, Session: "session", Usage: &ai.GenerationUsage{},
		hookManager: manager,
	}
	ctx, cancel := context.WithCancel(contextWithHistory("session"))
	cancel()

	_, err := testAgent(nil).observeStep(&cancelAwarePrompt{started: make(chan struct{})}, nil, time.Minute)(ctx, s)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("observeStep() error = %v, want context.Canceled", err)
	}
	if s.FallbackUsed || s.Observe != nil {
		t.Fatalf("cancelled observe produced fallback: state=%+v observation=%+v", s, s.Observe)
	}
	if modelEnd.FallbackUsed || modelEnd.FallbackReason != "" {
		t.Fatalf("cancelled model span was marked as fallback: %+v", modelEnd)
	}
}

func TestObserveFallbackIsVisibleOnModelAndStageEnd(t *testing.T) {
	tests := []struct {
		name         string
		prompt       ai.Prompt
		timeout      time.Duration
		wantReason   string
		wantEvidence string
	}{
		{
			name:         "timeout",
			prompt:       &cancelAwarePrompt{started: make(chan struct{})},
			timeout:      time.Millisecond,
			wantReason:   hooks.FallbackReasonTimeout,
			wantEvidence: "Timeout - using available context",
		},
		{
			name:         "parse error",
			prompt:       &stubPrompt{resp: textResp("not structured output")},
			wantReason:   hooks.FallbackReasonParseError,
			wantEvidence: "Parse error - using available context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
			var modelEnd hooks.State
			var stageEnd hooks.State
			if err := manager.Register(hooks.Registration{
				Events: []hooks.Event{hooks.EventModelCallEnd, hooks.EventStageEnd},
				Hook: func(ctx context.Context, state hooks.State) context.Context {
					switch state.Event {
					case hooks.EventModelCallEnd:
						modelEnd = state
					case hooks.EventStageEnd:
						stageEnd = state
					}
					return ctx
				},
			}); err != nil {
				t.Fatal(err)
			}
			ra := testAgent(nil)
			s := &state{
				Input:         &schema.UserInput{Content: "hi"},
				Session:       "session",
				InteractionID: "interaction",
				Iteration:     1,
				Usage:         &ai.GenerationUsage{},
				hookManager:   manager,
			}
			stage := builtStage{name: flowObserve, model: "test/model"}
			_, err := withStageHooks(stage, ra.observeStep(tt.prompt, nil, tt.timeout))(contextWithHistory("session"), s)
			if err != nil {
				t.Fatalf("observe stage error: %v", err)
			}
			if !stageEnd.FallbackUsed || stageEnd.FallbackReason != tt.wantReason {
				t.Fatalf("stage fallback = (%v, %q), want (true, %q)", stageEnd.FallbackUsed, stageEnd.FallbackReason, tt.wantReason)
			}
			if !modelEnd.FallbackUsed || modelEnd.FallbackReason != tt.wantReason {
				t.Fatalf("model fallback = (%v, %q), want (true, %q)", modelEnd.FallbackUsed, modelEnd.FallbackReason, tt.wantReason)
			}
			gotEvidence := ""
			if s.Observe != nil {
				gotEvidence = s.Observe.Evidence
			}
			if gotEvidence != tt.wantEvidence {
				t.Fatalf("fallback evidence = %q, want %q", gotEvidence, tt.wantEvidence)
			}
		})
	}
}

func TestReasonActStepEmitsMultipleToolsInRequestOrder(t *testing.T) {
	g := genkit.Init(context.Background())
	for _, name := range []string{"first_tool", "second_tool"} {
		toolName := name
		genkit.DefineTool(g, toolName, toolName, func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
			return map[string]any{"tool_name": toolName, "summary": toolName}, nil
		})
	}
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var events []string
	if err := manager.Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventModelCallStart, hooks.EventModelCallEnd, hooks.EventToolCallStart, hooks.EventToolCallEnd},
		Tools:  []string{hooks.AllTools},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			events = append(events, string(state.Event)+":"+state.ToolName+":"+state.ToolCallID)
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}
	s := &state{
		Input: &schema.UserInput{Content: "hi"}, Session: "session", InteractionID: "interaction",
		Iteration: 1, Stage: flowReasonAct, Model: "test/model", Usage: &ai.GenerationUsage{}, hookManager: manager,
	}
	prompt := &stubPrompt{resp: toolReqsResp(
		&ai.ToolRequest{Ref: "call-first", Name: "first_tool", Input: map[string]any{"n": 1}},
		&ai.ToolRequest{Ref: "call-second", Name: "second_tool", Input: map[string]any{"n": 2}},
	)}
	if _, err := testAgent(g).reasonActStep(prompt, nil, 0)(contextWithHistory("session"), s); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"model_call.start::", "model_call.end::",
		"tool_call.start:first_tool:call-first", "tool_call.end:first_tool:call-first",
		"tool_call.start:second_tool:call-second", "tool_call.end:second_tool:call-second",
	}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("events = %v, want %v", events, want)
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
	s := &state{Session: "session", InteractionID: "interaction", Iteration: 1, Stage: flowReasonAct, hookManager: manager}
	_, err := testAgent(g).executeToolCall(context.Background(), s, &ai.ToolRequest{Name: "broken_tool", Input: map[string]any{"x": 1}})
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

func TestReasonActStepEmitsModelAndSelectedToolEvents(t *testing.T) {
	g := genkit.Init(context.Background())
	genkit.DefineTool(g, "selected_tool", "selected", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
		return map[string]any{"tool_name": "selected_tool", "summary": "ok", "result": input}, nil
	})

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
	if err := manager.Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventToolCallStart, hooks.EventToolCallEnd},
		Tools:  []string{"selected_tool"},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			events = append(events, state)
			return ctx
		},
	}); err != nil {
		t.Fatalf("register tool hook: %v", err)
	}

	ra := testAgent(g)
	s := &state{
		Input:         &schema.UserInput{Content: "hi"},
		Session:       "session",
		InteractionID: "interaction",
		Iteration:     1,
		Stage:         "reasonAct",
		Model:         "test/model",
		Usage:         &ai.GenerationUsage{},
		hookManager:   manager,
	}
	prompt := &stubPrompt{resp: toolReqResp("selected_tool", map[string]any{"q": "ping"})}
	if _, err := ra.reasonActStep(prompt, nil, 0)(contextWithHistory("session"), s); err != nil {
		t.Fatalf("reasonActStep() error = %v", err)
	}

	want := []hooks.Event{
		hooks.EventModelCallStart,
		hooks.EventModelCallEnd,
		hooks.EventToolCallStart,
		hooks.EventToolCallEnd,
	}
	if len(events) != len(want) {
		t.Fatalf("events = %+v, want %v", events, want)
	}
	for i := range want {
		if events[i].Event != want[i] {
			t.Fatalf("events[%d] = %s, want %s", i, events[i].Event, want[i])
		}
	}
	if events[2].ToolName != "selected_tool" || events[3].ToolName != "selected_tool" {
		t.Fatalf("tool events lost tool name: %+v", events[2:])
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
	s := &state{Session: "session", InteractionID: "interaction", hookManager: manager}

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
