package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/genkit"
)

// type ToolOut interface {
// 	FromRaw(any) ToolOutput
// 	Output() string
// }

// func (s StringOutput) Output() string {
// 	return s.value
// }

// ToolDesc 表示要执行的工具调用信息
type ToolDesc struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

func (toolDesc ToolDesc) Call(g *genkit.Genkit, ctx context.Context) (string, error) {
	tool := genkit.LookupTool(g, toolDesc.ToolName)
	if tool == nil {
		return "", fmt.Errorf("tool not found: %s", toolDesc.ToolName)
	}

	rawToolOutput, err := tool.RunRaw(ctx, toolDesc.ToolInput)
	if err != nil {
		return "", fmt.Errorf("failed to call tool %s: %w", toolDesc.ToolName, err)
	}

	jsonOutput, err := json.Marshal(rawToolOutput)
	return string(jsonOutput), err
}
