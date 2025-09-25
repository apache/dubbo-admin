package tools

import (
	"context"
	"dubbo-admin-ai/manager"
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

type ToolManager interface {
	AllToolRefs() []ai.ToolRef
	AllTools() []ai.Tool
	// AllToolNames()
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
