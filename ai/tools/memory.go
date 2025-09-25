package tools

import (
	"dubbo-admin-ai/memory"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

const (
	GetAllMemoryTool string = "memory_all_by_session_id"
)

type MemoryToolInput struct {
	SessionID string `json:"session_id"`
}

func defineMemoryTools(g *genkit.Genkit, history *memory.History) []ai.Tool {
	tools := []ai.Tool{
		getAllMemoryBySession(g, history),
	}
	return tools
}

func getAllMemoryBySession(g *genkit.Genkit, history *memory.History) ai.Tool {
	return genkit.DefineTool(
		g, GetAllMemoryTool, "Get all history memory messages of a session by input `session_id`",
		func(ctx *ai.ToolContext, input MemoryToolInput) (ToolOutput, error) {
			if input.SessionID == "" {
				return ToolOutput{}, fmt.Errorf("sessionID is required")
			}

			if history.IsEmpty(input.SessionID) {
				return ToolOutput{
					ToolName: GetAllMemoryTool,
					Summary:  "No memory available",
				}, nil
			}

			return ToolOutput{
				ToolName: GetAllMemoryTool,
				Result:   history.AllMemory(input.SessionID),
				Summary:  "",
			}, nil
		},
	)
}
