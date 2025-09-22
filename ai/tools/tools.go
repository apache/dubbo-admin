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

type Tool struct {
	ToolName  string  `json:"tool_name"`
	Parameter any     `json:"parameter"`
	Entity    ai.Tool `json:"-"`
}

func (t Tool) String() string {
	return fmt.Sprintf("ToolInput{Input: %v}", t.Parameter)
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

func (t Tool) Call(g *genkit.Genkit, ctx context.Context) (toolOutput ToolOutput, err error) {
	tool := genkit.LookupTool(g, t.ToolName)
	if tool == nil {
		return toolOutput, fmt.Errorf("tool not found: %s", t.ToolName)
	}

	// Pass Parameter directly to the tool, not the entire ToolInput
	rawToolOutput, err := tool.RunRaw(ctx, t.Parameter)
	if err != nil {
		return toolOutput, fmt.Errorf("failed to call tool %s: %w", t.ToolName, err)
	}

	if rawToolOutput == nil {
		return toolOutput, fmt.Errorf("tool %s returned nil output", t.ToolName)
	}

	err = mapstructure.Decode(rawToolOutput, &toolOutput)
	if err != nil {
		return toolOutput, fmt.Errorf("failed to decode tool output for %s: %w", t.ToolName, err)
	}

	// Ensure ToolName is set
	if toolOutput.ToolName == "" {
		toolOutput.ToolName = t.ToolName
	}

	return toolOutput, nil
}
