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
	"strings"
	"testing"

	"dubbo-admin-ai/component/memory"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// scriptPrompt is an ai.Prompt whose Execute dispenses queued responses/errors
// in call order, letting the loop tests drive several iterations without a live
// model. testAgent points actPrompt and answerPrompt at the same instance so
// calls are consumed in the exact order run() makes them; tests that must tell
// the two prompts apart override answerPrompt with a second instance.
type scriptPrompt struct {
	resps []*ai.ModelResponse
	errs  []error
	calls int
}

func (s *scriptPrompt) Name() string { return "script" }

func (s *scriptPrompt) Execute(ctx context.Context, opts ...ai.PromptExecuteOption) (*ai.ModelResponse, error) {
	i := s.calls
	s.calls++
	var err error
	if i < len(s.errs) {
		err = s.errs[i]
	}
	var resp *ai.ModelResponse
	if i < len(s.resps) {
		resp = s.resps[i]
	}
	return resp, err
}

func (s *scriptPrompt) Render(ctx context.Context, input any) (*ai.GenerateActionOptions, error) {
	return &ai.GenerateActionOptions{}, nil
}

func contextWithHistory(sessionID string) (context.Context, *memory.HistoryMemory) {
	ctx := memory.NewMemoryContext(memory.ChatHistoryKey)
	history, _ := memory.GetHistoryMemory(ctx, memory.ChatHistoryKey)
	history.AddHistory(sessionID, ai.NewUserMessage(ai.NewTextPart("hello")))
	ctx = context.WithValue(ctx, memory.SessionIDKey, sessionID)
	return ctx, history
}

func testAgent(g *genkit.Genkit, maxIter int, script *scriptPrompt) *ReActAgent {
	return &ReActAgent{
		registry:      g,
		memoryCtx:     memory.NewMemoryContext(memory.ChatHistoryKey),
		actPrompt:     script,
		answerPrompt:  script,
		toolTimeouts:  newToolTimeoutResolver(defaultToolTimeoutSeconds, nil),
		maxIterations: maxIter,
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

// historyText concatenates all text parts recorded in the session's window.
func historyText(h *memory.HistoryMemory, sessionID string) string {
	var b strings.Builder
	for _, m := range h.WindowMemory(sessionID) {
		for _, p := range m.Content {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func TestRun_AnswersDirectly(t *testing.T) {
	g := genkit.Init(context.Background())
	script := &scriptPrompt{resps: []*ai.ModelResponse{textResp("Dubbo is an RPC framework.")}}
	ra := testAgent(g, 3, script)
	ctx, history := contextWithHistory("s1")

	usage, err := ra.run(ctx, nil)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if usage == nil {
		t.Fatal("expected non-nil usage")
	}
	if script.calls != 1 {
		t.Fatalf("expected a single model call, got %d", script.calls)
	}
	if !strings.Contains(historyText(history, "s1"), "Dubbo is an RPC framework.") {
		t.Fatalf("final answer not recorded in history: %q", historyText(history, "s1"))
	}
}

func TestRun_CallsToolThenAnswers(t *testing.T) {
	g := genkit.Init(context.Background())
	genkit.DefineTool(g, "mock_tool", "mock tool", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
		return map[string]any{"tool_name": "mock_tool", "summary": "ok", "result": map[string]any{"echo": input["q"]}}, nil
	})
	script := &scriptPrompt{resps: []*ai.ModelResponse{
		toolReqResp("mock_tool", map[string]any{"q": "ping"}),
		textResp("Here is the answer."),
	}}
	ra := testAgent(g, 2, script)
	ctx, history := contextWithHistory("s2")

	if _, err := ra.run(ctx, nil); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if script.calls != 2 {
		t.Fatalf("expected two model calls (tool round + answer), got %d", script.calls)
	}
	text := historyText(history, "s2")
	if !strings.Contains(text, "mock_tool") {
		t.Fatalf("tool output not recorded in history: %q", text)
	}
	if !strings.Contains(text, "Here is the answer.") {
		t.Fatalf("final answer not recorded in history: %q", text)
	}
}

func TestRun_ToolErrorDegradesNotAborts(t *testing.T) {
	g := genkit.Init(context.Background())
	genkit.DefineTool(g, "broken_tool", "broken", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
		return nil, errors.New("boom")
	})
	script := &scriptPrompt{resps: []*ai.ModelResponse{
		toolReqResp("broken_tool", map[string]any{"x": 1}),
		textResp("Answered despite the failure."),
	}}
	ra := testAgent(g, 2, script)
	ctx, history := contextWithHistory("s3")

	if _, err := ra.run(ctx, nil); err != nil {
		t.Fatalf("a failing tool must not abort the interaction, got: %v", err)
	}
	text := historyText(history, "s3")
	if !strings.Contains(text, "broken_tool") || !strings.Contains(text, "failed") {
		t.Fatalf("expected degraded failure output recorded, got: %q", text)
	}
}

func TestRun_PropagatesExecuteError(t *testing.T) {
	g := genkit.Init(context.Background())
	script := &scriptPrompt{errs: []error{errors.New("execute failed")}}
	ra := testAgent(g, 3, script)
	ctx, _ := contextWithHistory("s4")

	_, err := ra.run(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to execute react prompt") {
		t.Fatalf("expected wrapped execute error, got %v", err)
	}
}

// TestRun_ForcedFinalIterationUsesAnswerPrompt pins the loop's core guarantee:
// once the tool-round budget is spent, the forced final iteration must switch to
// the tool-less answer prompt. actPrompt and answerPrompt are distinct spies so
// selecting the wrong one is observable (unlike testAgent's shared instance).
func TestRun_ForcedFinalIterationUsesAnswerPrompt(t *testing.T) {
	g := genkit.Init(context.Background())
	genkit.DefineTool(g, "loop_tool", "keeps the loop going", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
		return map[string]any{"tool_name": "loop_tool", "summary": "ok"}, nil
	})
	// actPrompt always asks for a tool, so the loop only terminates once the
	// forced final iteration switches to answerPrompt.
	act := &scriptPrompt{resps: []*ai.ModelResponse{
		toolReqResp("loop_tool", map[string]any{"q": "1"}),
		toolReqResp("loop_tool", map[string]any{"q": "2"}),
		toolReqResp("loop_tool", map[string]any{"q": "3"}),
	}}
	answer := &scriptPrompt{resps: []*ai.ModelResponse{textResp("Final synthesized answer.")}}
	ra := testAgent(g, 3, act)
	ra.answerPrompt = answer
	ctx, history := contextWithHistory("s5")

	if _, err := ra.run(ctx, nil); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if act.calls != 2 {
		t.Fatalf("expected actPrompt used for the 2 non-final iterations, got %d", act.calls)
	}
	if answer.calls != 1 {
		t.Fatalf("expected answerPrompt used exactly once on the forced final iteration, got %d", answer.calls)
	}
	if !strings.Contains(historyText(history, "s5"), "Final synthesized answer.") {
		t.Fatalf("forced final answer (from answerPrompt) not recorded: %q", historyText(history, "s5"))
	}
}

// TestRun_EmptyResponseRetries covers a tool-free empty response mid-loop: it
// carries nothing to stream, so run() must retry rather than finish on silence.
func TestRun_EmptyResponseRetries(t *testing.T) {
	g := genkit.Init(context.Background())
	script := &scriptPrompt{resps: []*ai.ModelResponse{textResp(""), textResp("Recovered answer.")}}
	ra := testAgent(g, 2, script)
	ctx, history := contextWithHistory("s6")

	if _, err := ra.run(ctx, nil); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if script.calls != 2 {
		t.Fatalf("expected empty response to trigger a retry (2 calls), got %d", script.calls)
	}
	if !strings.Contains(historyText(history, "s6"), "Recovered answer.") {
		t.Fatalf("expected recovered answer recorded, got: %q", historyText(history, "s6"))
	}
}

// TestRun_EmptyForcedAnswerFallsBack covers an empty response on the forced final
// iteration: with no iterations left to retry, run() must emit an explicit
// fallback so the interaction never ends with bare stream markers.
func TestRun_EmptyForcedAnswerFallsBack(t *testing.T) {
	g := genkit.Init(context.Background())
	script := &scriptPrompt{resps: []*ai.ModelResponse{textResp("")}}
	ra := testAgent(g, 1, script)
	ctx, history := contextWithHistory("s7")

	if _, err := ra.run(ctx, nil); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if !strings.Contains(historyText(history, "s7"), fallbackAnswer) {
		t.Fatalf("expected fallback answer recorded, got: %q", historyText(history, "s7"))
	}
}
