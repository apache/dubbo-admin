package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/mitchellh/mapstructure"
)

type ToolInput struct {
	ToolName  string `json:"tool_name"`
	Parameter any    `json:"parameter"`
}

func (ti ToolInput) String() string {
	return fmt.Sprintf("ToolInput{Input: %v}", ti.Parameter)
}

type ToolOutput struct {
	ToolName string `json:"tool_name"`
	Summary  string `json:"summary"`
	Result   any    `json:"result"`
}

func (to ToolOutput) Map() map[string]any {
	result := make(map[string]any)
	result["tool_name"] = to.ToolName
	result["summary"] = to.Summary
	result["result"] = to.Result
	return result
}

func (to ToolOutput) String() string {
	result := "ToolOutput{\n"
	result += fmt.Sprintf("  ToolName: %s\n", to.ToolName)
	result += fmt.Sprintf("  Summary: %s\n", to.Summary)

	// Format Result based on its type
	if to.Result != nil {
		if resultJSON, err := json.MarshalIndent(to.Result, "  ", "  "); err == nil {
			result += fmt.Sprintf("  Result: %s\n", string(resultJSON))
		} else {
			result += fmt.Sprintf("  Result: %v\n", to.Result)
		}
	} else {
		result += "  Result: <nil>\n"
	}

	result += "}"
	return result
}

type ToolManager interface {
	AllToolRefs()
	AllToolNames()
	Register(...ai.Tool)
}

type ToolMetadata struct {
	ID          string       `json:"id"`
	Description string       `json:"description"`
	InputType   reflect.Type `json:"inputType"`
	OutputType  reflect.Type `json:"outputType"`
}

func (toolInput ToolInput) Call(g *genkit.Genkit, ctx context.Context) (toolOutput ToolOutput, err error) {
	tool := genkit.LookupTool(g, toolInput.ToolName)
	if tool == nil {
		return toolOutput, fmt.Errorf("tool not found: %s", toolInput.ToolName)
	}

	// Pass Parameter directly to the tool, not the entire ToolInput
	rawToolOutput, err := tool.RunRaw(ctx, toolInput.Parameter)
	if err != nil {
		return toolOutput, fmt.Errorf("failed to call tool %s: %w", toolInput.ToolName, err)
	}

	if rawToolOutput == nil {
		return toolOutput, fmt.Errorf("tool %s returned nil output", toolInput.ToolName)
	}

	err = mapstructure.Decode(rawToolOutput, &toolOutput)
	if err != nil {
		return toolOutput, fmt.Errorf("failed to decode tool output for %s: %w", toolInput.ToolName, err)
	}

	// Ensure ToolName is set
	if toolOutput.ToolName == "" {
		toolOutput.ToolName = toolInput.ToolName
	}

	return toolOutput, nil
}
