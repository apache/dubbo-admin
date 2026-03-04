package reacttest

import (
	"context"
	"errors"
	"strings"
	"testing"

	compReact "dubbo-admin-ai/component/agent/react"
	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type stubPrompt struct {
	resp *ai.ModelResponse
	err  error
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

func TestActFlow(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*genkit.Genkit) ai.Prompt
		input      schema.ThinkOutput
		errContain string
		assertFn   func(t *testing.T, out schema.Schema)
	}{
		{
			name:  "general_inquiry_no_tools",
			setup: func(*genkit.Genkit) ai.Prompt { return &stubPrompt{} },
			input: schema.ThinkOutput{Intent: schema.GeneralInquiry},
			assertFn: func(t *testing.T, out schema.Schema) {
				actOut, ok := out.(schema.ToolOutputs)
				if !ok {
					t.Fatalf("output type = %T, want schema.ToolOutputs", out)
				}
				if len(actOut.Outputs) != 0 {
					t.Fatalf("expected no tool outputs, got %d", len(actOut.Outputs))
				}
			},
		},
		{
			name: "with_tool_call_returns_outputs",
			setup: func(g *genkit.Genkit) ai.Prompt {
				genkit.DefineTool(g, "mock_tool", "mock tool", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
					return map[string]any{"tool_name": "mock_tool", "summary": "ok", "result": map[string]any{"echo": input["q"]}}, nil
				})
				return &stubPrompt{resp: &ai.ModelResponse{Message: ai.NewMessage(ai.RoleModel, nil,
					ai.NewToolRequestPart(&ai.ToolRequest{Name: "mock_tool", Input: map[string]any{"q": "ping"}}),
				)}}
			},
			input: schema.ThinkOutput{Intent: schema.PerformanceInvestigation, SuggestedTools: []string{"mock_tool"}},
			assertFn: func(t *testing.T, out schema.Schema) {
				actOut, ok := out.(schema.ToolOutputs)
				if !ok {
					t.Fatalf("output type = %T, want schema.ToolOutputs", out)
				}
				if len(actOut.Outputs) < 1 || actOut.Outputs[0].ToolName != "mock_tool" {
					t.Fatalf("unexpected tool outputs: %+v", actOut.Outputs)
				}
			},
		},
		{
			name: "tool_error_handling",
			setup: func(g *genkit.Genkit) ai.Prompt {
				genkit.DefineTool(g, "broken_tool", "broken", func(ctx *ai.ToolContext, input map[string]any) (map[string]any, error) {
					return nil, errors.New("boom")
				})
				return &stubPrompt{resp: &ai.ModelResponse{Message: ai.NewMessage(ai.RoleModel, nil,
					ai.NewToolRequestPart(&ai.ToolRequest{Name: "broken_tool", Input: map[string]any{"x": 1}}),
				)}}
			},
			input:      schema.ThinkOutput{Intent: schema.PerformanceInvestigation, SuggestedTools: []string{"broken_tool"}},
			errContain: "broken_tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := genkit.Init(context.Background())
			flow := compReact.ActFlow(g, nil, tt.setup(g))
			out, err := flow.Run(contextWithHistory("session"), tt.input)
			if tt.errContain != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("expected error containing %q, got %v", tt.errContain, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("act flow error: %v", err)
			}
			tt.assertFn(t, out)
		})
	}
}

func TestThinkFlow(t *testing.T) {
	g := genkit.Init(context.Background())
	flow := compReact.ThinkFlow(g, &stubPrompt{resp: nil, err: errors.New("execute failed")})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	_, err := flow.Run(contextWithHistory("s3"), schema.ThinkInput{SessionID: "s3", UserInput: &schema.UserInput{Content: "hi"}})
	if err == nil || !strings.Contains(err.Error(), "failed to execute agentThink prompt") {
		t.Fatalf("expected wrapped execute error, got %v", err)
	}
}
