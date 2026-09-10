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
	"fmt"

	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/component/tools"
	"dubbo-admin-ai/runtime"
	conversationstore "dubbo-admin-ai/store"
)

// AgentComponent is the runtime.Component wrapper around a ReActAgent: it holds
// the agent's YAML-sourced configuration and, on Init, resolves dependencies
// (tools) and constructs the underlying ReActAgent.
type AgentComponent struct {
	instanceName string
	Agent        *ReActAgent
	spec         AgentSpec
}

// NewAgentComponent builds an unstarted AgentComponent from a decoded spec. The
// ReActAgent itself is created later, in Init.
func NewAgentComponent(spec AgentSpec) (runtime.Component, error) {
	return &AgentComponent{spec: spec}, nil
}

// Name returns the component's instance name, or "agent" if none was set.
func (a *AgentComponent) Name() string {
	if a.instanceName != "" {
		return a.instanceName
	}
	return "agent"
}

// SetName sets the component's instance name.
func (a *AgentComponent) SetName(name string) {
	a.instanceName = name
}

// Validate checks the component's configuration without side effects.
func (a *AgentComponent) Validate() error {
	return a.spec.Validate()
}

// Init resolves the tools dependency from the runtime and constructs the
// underlying ReActAgent, wiring tool references and per-tool timeouts.
func (a *AgentComponent) Init(rt *runtime.Runtime) error {
	memoryComp, err := rt.GetComponent("memory")
	if err != nil {
		return fmt.Errorf("memory component not found: %w", err)
	}
	memoryComponent, ok := memoryComp.(*memory.MemoryComponent)
	if !ok {
		return fmt.Errorf("invalid memory component type")
	}
	conversationStore, err := memoryComponent.GetStore()
	if err != nil {
		return fmt.Errorf("failed to get conversation store: %w", err)
	}
	messageStore, ok := conversationStore.(conversationstore.MessageStore)
	if !ok {
		return fmt.Errorf("conversation store does not implement MessageStore")
	}

	toolsComp, err := rt.GetComponent("tools")
	if err != nil {
		return fmt.Errorf("tools component not found: %w", err)
	}
	tools, ok := toolsComp.(*tools.ToolsComponent)
	if !ok {
		return fmt.Errorf("invalid tools component type")
	}
	toolRefs := tools.GetToolRefs()
	toolTimeouts := newToolTimeoutResolver(defaultToolTimeoutSeconds, a.spec.ToolTimeouts)
	reactAgent, err := NewReActAgentWithStore(rt.GetGenkitRegistry(), messageStore, &a.spec, toolTimeouts, toolRefs)
	if err != nil {
		return fmt.Errorf("failed to create ReAct agent: %w", err)
	}

	a.Agent = reactAgent

	rt.GetLogger().Info("Agent component initialized",
		"agent_type", a.spec.AgentType,
		"model", a.spec.Model,
		"max_iterations", a.spec.MaxIterations)

	return nil
}

// Start is a no-op: the agent has no background work to launch.
func (a *AgentComponent) Start() error {
	return nil
}

// Stop is a no-op: the agent holds no resources that need releasing.
func (a *AgentComponent) Stop() error {
	return nil
}
