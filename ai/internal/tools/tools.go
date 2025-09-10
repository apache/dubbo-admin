package tools

import (
	"context"
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

func (to ToolOutput) String() string {
	return fmt.Sprintf("ToolOutput{ToolName: %s, Summary: %s, Result: %v}", to.ToolName, to.Summary, to.Result)
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

	rawToolOutput, err := tool.RunRaw(ctx, toolInput)

	if rawToolOutput == nil {
		return toolOutput, err
	}

	mapstructure.Decode(rawToolOutput, &toolOutput)
	if err != nil {
		return toolOutput, fmt.Errorf("failed to call tool %s: %w", toolInput.ToolName, err)
	}

	return toolOutput, nil
}
