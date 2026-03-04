package engine

import (
	"context"
	"dubbo-admin-ai/component/memory"
	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/runtime"
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

func Call(g *genkit.Genkit, toolName string, input any) (toolOutput ToolOutput, err error) {
	tool := genkit.LookupTool(g, toolName)
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
	runtime.GetLogger().Info("Tool output:", "output", rawToolOutput)

	rawMap, ok := rawToolOutput.(map[string]any)
	if !ok {
		toolOutput = ToolOutput{
			ToolName: toolName,
			Summary:  "Tool executed",
			Result:   rawToolOutput,
		}
		return toolOutput, nil
	}

	if _, hasToolName := rawMap["tool_name"]; !hasToolName {
		if _, hasSummary := rawMap["summary"]; !hasSummary {
			if _, hasResult := rawMap["result"]; !hasResult {
				toolOutput = ToolOutput{
					ToolName: toolName,
					Summary:  "Tool executed",
					Result:   rawToolOutput,
				}
				return toolOutput, nil
			}
		}
	}

	err = mapstructure.Decode(rawMap, &toolOutput)
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

type ToolDefineFunc func(*runtime.Runtime) []ai.Tool

type InternalToolManager struct {
	registry *genkit.Genkit
	tools    []ai.Tool
}

func NewInternalToolManager(rt *runtime.Runtime) (*InternalToolManager, error) {
	if rt == nil {
		return nil, fmt.Errorf("runtime is nil")
	}

	g := rt.GetGenkitRegistry()
	if g == nil {
		return nil, fmt.Errorf("genkit registry is nil")
	}

	memoryComp, err := rt.GetComponent("memory")
	if err != nil {
		return nil, fmt.Errorf("memory component not found: %w", err)
	}

	memComp, ok := memoryComp.(*memory.MemoryComponent)
	if !ok {
		return nil, fmt.Errorf("invalid memory component type")
	}
	if _, err := memComp.GetMemory(); err != nil {
		return nil, fmt.Errorf("failed to get history from memory component: %w", err)
	}

	ragCompRaw, err := rt.GetComponent("rag")
	if err != nil {
		return nil, fmt.Errorf("rag component not found: %w", err)
	}
	ragComp, ok := ragCompRaw.(*compRag.RAGComponent)
	if !ok {
		return nil, fmt.Errorf("invalid rag component type")
	}
	ragSys := ragComp.Rag
	if ragSys == nil {
		return nil, fmt.Errorf("rag system is not initialized")
	}

	toolDefiners := []ToolDefineFunc{
		defineMemoryTools,
	}
	var tools []ai.Tool
	for _, defineTools := range toolDefiners {
		tools = append(tools, defineTools(rt)...)
	}
	return &InternalToolManager{
		registry: g,
		tools:    tools,
	}, nil
}

func (itm *InternalToolManager) ToolRefs() (toolRef []ai.ToolRef) {
	for _, tool := range itm.tools {
		toolRef = append(toolRef, tool)
	}
	return toolRef
}
