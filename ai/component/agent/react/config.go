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

import "fmt"

const AgentTypeReAct = "react"

type AgentSpec struct {
	AgentType              string      `yaml:"agent_type"`
	DefaultModel           string      `yaml:"default_model"`
	PromptBasePath         string      `yaml:"prompt_base_path"`
	MaxIterations          int         `yaml:"max_iterations"`
	StageChannelBufferSize int         `yaml:"stage_channel_buffer_size"`
	MCPHostName            string      `yaml:"mcp_host_name"`
	Stages                 []StageInfo `yaml:"stages"`
}

type StageInfo struct {
	Name        string  `yaml:"name"`
	FlowType    string  `yaml:"flow_type"`
	Model       string  `yaml:"model,omitempty"`
	PromptFile  string  `yaml:"prompt_file"`
	Temperature float64 `yaml:"temperature"`
	TopP        float64 `yaml:"top_p,omitempty"`
	MaxTokens   int     `yaml:"max_tokens"`
	Timeout     int     `yaml:"timeout"`
	EnableTools bool    `yaml:"enable_tools"`
	ExtraPrompt string  `yaml:"extra_prompt,omitempty"`
}

// ReActDefaultSpec returns default ReAct Agent configuration
func ReActDefaultSpec() *AgentSpec {
	return &AgentSpec{
		AgentType:              "react",
		DefaultModel:           "qwen-max",
		PromptBasePath:         "./prompts",
		MaxIterations:          10,
		StageChannelBufferSize: 5,
		MCPHostName:            "mcp_host",
		Stages: []StageInfo{
			{
				Name:        "agentThinking",
				FlowType:    "think",
				PromptFile:  "agentThink.txt",
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   2000,
				Timeout:     60,
				EnableTools: true,
			},
			{
				Name:        "agentTool",
				FlowType:    "act",
				PromptFile:  "agentAct.txt",
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   3000,
				Timeout:     90,
				EnableTools: true,
			},
			{
				Name:        "agentFeedback",
				FlowType:    "feedback",
				PromptFile:  "agentFeedback.txt",
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   1000,
				Timeout:     30,
				EnableTools: false,
			},
			{
				Name:        "observe",
				FlowType:    "observe",
				PromptFile:  "agentObserve.txt",
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   2000,
				Timeout:     60,
				EnableTools: false,
			},
		},
	}
}

// Validate validates the configuration
func (c *AgentSpec) Validate() error {
	if c.AgentType == "" {
		return fmt.Errorf("agent_type is required")
	}
	if c.DefaultModel == "" {
		return fmt.Errorf("default_model is required")
	}
	if c.PromptBasePath == "" {
		return fmt.Errorf("prompt_base_path is required")
	}
	if c.MaxIterations <= 0 {
		return fmt.Errorf("max_iterations must be greater than 0")
	}
	if c.StageChannelBufferSize <= 0 {
		return fmt.Errorf("stage_channel_buffer_size must be greater than 0")
	}
	if len(c.Stages) == 0 {
		return fmt.Errorf("stages is required")
	}

	for i, stage := range c.Stages {
		if err := stage.Validate(i); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the stage configuration
func (s *StageInfo) Validate(index int) error {
	// Validate name
	if s.Name == "" {
		return fmt.Errorf("stage[%d]: name is required", index)
	}

	// Validate flow type
	validFlowTypes := map[string]bool{
		"think":    true,
		"act":      true,
		"observe":  true,
		"feedback": true,
	}
	if !validFlowTypes[s.FlowType] {
		return fmt.Errorf("stage[%d]: invalid flow_type '%s', must be one of: think, act, observe, feedback", index, s.FlowType)
	}

	if s.PromptFile == "" {
		return fmt.Errorf("stage[%d]: prompt_file is required", index)
	}

	if s.Temperature <= 0 || s.Temperature > 2.0 {
		return fmt.Errorf("stage[%d]: temperature must be in (0, 2.0]", index)
	}

	if s.TopP <= 0 || s.TopP > 1.0 {
		return fmt.Errorf("stage[%d]: top_p must be in (0, 1.0]", index)
	}

	if s.MaxTokens <= 0 {
		return fmt.Errorf("stage[%d]: max_tokens must be greater than 0", index)
	}

	if s.Timeout <= 0 {
		return fmt.Errorf("stage[%d]: timeout must be greater than 0", index)
	}

	return nil
}
