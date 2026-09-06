package reacttest

import (
	"strings"
	"testing"

	compReact "dubbo-admin-ai/component/agent/react"
)

func validAgentSpec() *compReact.AgentSpec {
	return &compReact.AgentSpec{
		AgentType:         compReact.AgentTypeReAct,
		Model:             "qwen-max",
		PromptBasePath:    "./prompts",
		PromptFile:        "agentReasonAct.txt",
		MaxIterations:     5,
		ChannelBufferSize: 2,
		Temperature:       0.7,
		TopP:              0.9,
		MaxTokens:         1000,
		Timeout:           30,
	}
}

func TestAgentSpec_Validate(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*compReact.AgentSpec)
		errContain string
	}{
		{name: "prompt_required", mutate: func(c *compReact.AgentSpec) { c.PromptFile = "" }, errContain: "prompt_file is required"},
		{name: "temperature_out_of_range", mutate: func(c *compReact.AgentSpec) { c.Temperature = 3 }, errContain: "temperature must be in"},
		{name: "top_p_out_of_range", mutate: func(c *compReact.AgentSpec) { c.TopP = 2 }, errContain: "top_p must be in"},
		{name: "max_tokens_required", mutate: func(c *compReact.AgentSpec) { c.MaxTokens = 0 }, errContain: "max_tokens must be greater than 0"},
		{name: "timeout_required", mutate: func(c *compReact.AgentSpec) { c.Timeout = 0 }, errContain: "timeout must be greater than 0"},
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

func TestAgentSpec_Validate_AllowsZeroTemperatureAndTopP(t *testing.T) {
	cfg := validAgentSpec()
	cfg.Temperature = 0
	cfg.TopP = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("temperature=0 and top_p=0 must be valid, got %v", err)
	}
}
