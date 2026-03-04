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

package tools

import (
	"dubbo-admin-ai/component/tools/engine"
	"dubbo-admin-ai/runtime"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type ToolsComponent struct {
	instanceName string
	registry     *genkit.Genkit
	toolManagers []engine.ToolManager
	toolRefs     []ai.ToolRef
	config       ToolConfig
}

func NewToolsComponent(config ToolConfig) (runtime.Component, error) {
	return &ToolsComponent{
		config: config,
	}, nil
}

func (t *ToolsComponent) Name() string {
	if t.instanceName != "" {
		return t.instanceName
	}
	return "tools"
}

func (t *ToolsComponent) SetName(name string) {
	t.instanceName = name
}

func (t *ToolsComponent) Validate() error {
	if t.config.EnableMCPTools && t.config.MCPHostName == "" {
		return fmt.Errorf("mcp_host_name is required when mcp tools are enabled")
	}
	if t.config.MCPTimeout <= 0 {
		return fmt.Errorf("mcp_timeout must be greater than 0")
	}
	if t.config.MCPMaxRetries < 0 {
		return fmt.Errorf("mcp_max_retries must be >= 0")
	}
	return nil
}

func (t *ToolsComponent) Init(rt *runtime.Runtime) error {
	t.registry = rt.GetGenkitRegistry()

	// Initialize Tool Managers
	if t.config.EnableMockTools {
		t.toolManagers = append(t.toolManagers,
			engine.NewMockToolManager(rt))
	}

	if t.config.EnableInternalTools {
		internalMgr, err := engine.NewInternalToolManager(rt)
		if err != nil {
			return fmt.Errorf("failed to create internal tool manager: %w", err)
		}
		t.toolManagers = append(t.toolManagers, internalMgr)
	}

	if t.config.EnableMCPTools {
		mcpToolManager, err := engine.NewMCPToolManager(rt, t.config.MCPHostName)
		if err != nil {
			return fmt.Errorf("failed to create MCP tool manager: %w", err)
		}
		t.toolManagers = append(t.toolManagers, mcpToolManager)
	}

	t.toolRefs = engine.NewToolRegistry(t.toolManagers...).AllToolRefs()

	rt.GetLogger().Info("Tools component initialized",
		"total_tools", len(t.toolRefs),
		"total_managers", len(t.toolManagers),
		"mcp_host", t.config.MCPHostName)

	return nil
}

func (t *ToolsComponent) Start() error {
	return nil
}

func (t *ToolsComponent) Stop() error {
	return nil
}

func (t *ToolsComponent) GetToolRefs() []ai.ToolRef {
	return t.toolRefs
}

func (t *ToolsComponent) SetConfig(config ToolConfig) {
	t.config = config
}
