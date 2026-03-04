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
	"dubbo-admin-ai/component/tools"
	"dubbo-admin-ai/runtime"
	"fmt"
)

// AgentComponent Agent component implementation
type AgentComponent struct {
	instanceName           string
	Agent                  *ReActAgent
	agentType              string
	model                  string
	promptBasePath         string
	maxIterations          int
	stageChannelBufferSize int
	mcpHostName            string
	stages                 []StageInfo
}

func NewAgentComponent(
	agentType string,
	model string,
	promptBasePath string,
	maxIterations int,
	stageChannelBufferSize int,
	mcpHostName string,
	stages []StageInfo,
) (runtime.Component, error) {
	return &AgentComponent{
		agentType:              agentType,
		model:                  model,
		promptBasePath:         promptBasePath,
		maxIterations:          maxIterations,
		stageChannelBufferSize: stageChannelBufferSize,
		mcpHostName:            mcpHostName,
		stages:                 stages,
	}, nil
}

func (a *AgentComponent) Name() string {
	if a.instanceName != "" {
		return a.instanceName
	}
	return "agent"
}

func (a *AgentComponent) SetName(name string) {
	a.instanceName = name
}

func (a *AgentComponent) Validate() error {
	cfg := AgentSpec{
		AgentType:              a.agentType,
		Model:                  a.model,
		PromptBasePath:         a.promptBasePath,
		MaxIterations:          a.maxIterations,
		StageChannelBufferSize: a.stageChannelBufferSize,
		MCPHostName:            a.mcpHostName,
		Stages:                 a.stages,
	}
	return cfg.Validate()
}

func (a *AgentComponent) Init(rt *runtime.Runtime) error {
	toolsComp, err := rt.GetComponent("tools")
	if err != nil {
		return fmt.Errorf("tools component not found: %w", err)
	}
	tools, ok := toolsComp.(*tools.ToolsComponent)
	if !ok {
		return fmt.Errorf("invalid tools component type")
	}
	toolRefs := tools.GetToolRefs()
	reactAgent, err := NewReactAgent(rt.GetGenkitRegistry(), a.promptBasePath, a.model, a.maxIterations, a.stages, toolRefs)
	if err != nil {
		return fmt.Errorf("failed to create ReAct agent: %w", err)
	}

	a.Agent = reactAgent

	rt.GetLogger().Info("Agent component initialized",
		"agent_type", a.agentType,
		"model", a.model,
		"max_iterations", a.maxIterations,
		"stages", len(a.stages))

	return nil
}

func (a *AgentComponent) Start() error {
	return nil
}

func (a *AgentComponent) Stop() error {
	return nil
}
