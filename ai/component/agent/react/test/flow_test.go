package reacttest

import (
	"strings"
	"testing"

	compReact "dubbo-admin-ai/component/agent/react"
)

func validAgentSpec() *compReact.AgentSpec {
	return &compReact.AgentSpec{
		AgentType:              compReact.AgentTypeReAct,
		DefaultModel:           "qwen-max",
		PromptBasePath:         "./prompts",
		MaxIterations:          5,
		StageChannelBufferSize: 2,
		MCPHostName:            "mcp_host",
		Stages: []compReact.StageInfo{{
			Name:        "thinking",
			FlowType:    "think",
			PromptFile:  "agentThink.txt",
			Temperature: 0.7,
			TopP:        0.9,
			MaxTokens:   1000,
			Timeout:     30,
		}},
	}
}

func TestAgentComponent_ValidateStage(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*compReact.AgentSpec)
		errContain string
	}{
		{name: "invalid_flow_type", mutate: func(c *compReact.AgentSpec) { c.Stages[0].FlowType = "invalid" }, errContain: "invalid flow_type"},
		{name: "prompt_required", mutate: func(c *compReact.AgentSpec) { c.Stages[0].PromptFile = "" }, errContain: "prompt_file is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validAgentSpec()
			tt.mutate(cfg)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), tt.errContain) {
				t.Fatalf("expected error containing %q, got %v", tt.errContain, err)
			}
		})
	}
}
