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

// AgentTypeReAct is the agent_type discriminator for the ReAct strategy.
const AgentTypeReAct = "react"

// defaultToolTimeoutSeconds is the tool execution timeout used for any tool
// without an explicit tool_timeouts override. It is intentionally not
// configurable — override individual tools via tool_timeouts instead.
const defaultToolTimeoutSeconds = 30

// AgentSpec is the YAML-decoded configuration for a ReAct agent component.
//
// A ReAct agent runs a single reason-and-act loop: one model call per iteration
// reasons about the request and either calls tools (whose results feed the next
// iteration) or answers directly. The model settings below configure that one
// call, so the config is flat — there is no multi-stage pipeline to describe.
type AgentSpec struct {
	AgentType      string `yaml:"agent_type"`
	Model          string `yaml:"model"`
	PromptBasePath string `yaml:"prompt_base_path"`
	PromptFile     string `yaml:"prompt_file"`
	// MaxIterations bounds the reason-act loop. The last iteration always forces a
	// tool-less answer, so the effective number of tool rounds is
	// MaxIterations - 1; a value of 1 means "answer directly, never call tools".
	MaxIterations     int            `yaml:"max_iterations"`
	ChannelBufferSize int            `yaml:"channel_buffer_size"`
	ToolTimeouts      map[string]int `yaml:"tool_timeouts,omitempty"` // per-tool timeout overrides (seconds), keyed by tool name; defaults to defaultToolTimeoutSeconds
	Temperature       float64        `yaml:"temperature"`
	TopP              float64        `yaml:"top_p,omitempty"` // 0 means "unset" — the provider default is used
	MaxTokens         int            `yaml:"max_tokens"`
	Timeout           int            `yaml:"timeout"` // per model-call timeout (seconds)
}

// Validate validates the configuration.
func (c *AgentSpec) Validate() error {
	if c.AgentType == "" {
		return fmt.Errorf("agent_type is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.PromptBasePath == "" {
		return fmt.Errorf("prompt_base_path is required")
	}
	if c.PromptFile == "" {
		return fmt.Errorf("prompt_file is required")
	}
	if c.MaxIterations <= 0 {
		return fmt.Errorf("max_iterations must be greater than 0")
	}
	if c.ChannelBufferSize <= 0 {
		return fmt.Errorf("channel_buffer_size must be greater than 0")
	}
	for name, t := range c.ToolTimeouts {
		if t <= 0 {
			return fmt.Errorf("tool_timeouts[%q] must be greater than 0", name)
		}
	}
	// temperature 0 is a valid (deterministic) setting, so the lower bound is
	// inclusive; only reject negatives and values above the provider ceiling.
	if c.Temperature < 0 || c.Temperature > 2.0 {
		return fmt.Errorf("temperature must be in [0, 2.0]")
	}
	// top_p is optional: 0 means "unset" (provider default). Reject only
	// negatives and values above 1.0.
	if c.TopP < 0 || c.TopP > 1.0 {
		return fmt.Errorf("top_p must be in [0, 1.0]")
	}
	if c.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be greater than 0")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	return nil
}
