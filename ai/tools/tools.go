package tools

import (
	"context"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/memory"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/mitchellh/mapstructure"
)

type ToolOutput struct {
	ToolName string `json:"tool_name"`
	Result   any    `json:"result,omitempty"`
	Summary  string `json:"summary"`
	// Success  bool   `json:"success" jsonschema_description:"Indicates whether the tool execution was successful"`
}

func Call(g *genkit.Genkit, mcp *MCPToolManager, toolName string, input any) (toolOutput ToolOutput, err error) {
	var (
		tool      ai.Tool
		isMCPTool bool
	)
	tool = genkit.LookupTool(g, toolName)
	if mcp != nil && tool == nil {
		tool = mcp.GetToolByName(toolName)
		isMCPTool = tool != nil
	}
	if tool == nil {
		return toolOutput, fmt.Errorf("tool not found: %s", toolName)
	}

	// Pass Parameter directly to the tool, not the entire ToolInput
	rawToolOutput, err := tool.RunRaw(context.Background(), input)
	if err != nil {
		return toolOutput, fmt.Errorf("failed to call tool %s: %w", toolName, err)
	}

	if rawToolOutput == nil {
		return toolOutput, fmt.Errorf("tool %s is unavailable", toolName)
	}
	manager.GetLogger().Info("Tool output:", "output", rawToolOutput)

	if isMCPTool {
		toolOutput = ToolOutput{
			ToolName: toolName,
			Summary:  "MCP tool executed",
			Result:   rawToolOutput,
		}
		return toolOutput, nil
	}

	err = mapstructure.Decode(rawToolOutput, &toolOutput)
	if err != nil {
		return toolOutput, fmt.Errorf("failed to decode tool output for %s: %w", toolName, err)
	}

	// Ensure tool_name is set correctly if not already provided
	if toolOutput.ToolName == "" {
		toolOutput.ToolName = toolName
	}

	return toolOutput, nil
}

type ToolRegistry struct {
	managers []ToolManager
}

func NewToolRegistry(managers ...ToolManager) *ToolRegistry {
	return &ToolRegistry{managers: managers}
}

func (tr *ToolRegistry) AllToolRefs() (toolRefs []ai.ToolRef) {
	for _, manager := range tr.managers {
		toolRefs = append(toolRefs, manager.ToolRefs()...)
	}
	return toolRefs
}

type ToolManager interface {
	ToolRefs() []ai.ToolRef
}

type InternalToolManager struct {
	registry *genkit.Genkit
	tools    []ai.Tool
}

func NewInternalToolManager(g *genkit.Genkit, history *memory.History) *InternalToolManager {
	var tools []ai.Tool
	tools = append(tools, defineMemoryTools(g, history)...)
	return &InternalToolManager{
		registry: g,
		tools:    tools,
	}
}

func (itm *InternalToolManager) ToolRefs() (toolRef []ai.ToolRef) {
	for _, tool := range itm.tools {
		toolRef = append(toolRef, tool)
	}
	return toolRef
}
