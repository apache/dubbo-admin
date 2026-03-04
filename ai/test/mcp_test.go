package test

import (
	"context"
	"fmt"
	"os"
	"testing"

	rt "dubbo-admin-ai/runtime"
	"dubbo-admin-ai/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func TestMCP(t *testing.T) {
	// 设置测试环境
	os.Setenv("AI_ENVIRONMENT", "test")

	ctx := context.Background()
	g := genkit.Init(ctx, nil)

	mcpHostName := "mcpHost" // 默认值
	promptDir := "../../prompts" // 默认值

	mcpToolManager, err := tools.NewMCPToolManager(g, mcpHostName)
	if err != nil {
		t.Fatalf("failed to create MCP tool bootstrap: %v", err)
	}

	toolRefs := mcpToolManager.ToolRefs()

	prompt, err := os.ReadFile(promptDir + "/agentTool.txt")
	if err != nil {
		t.Fatalf("failed to read prompt file: %v", err)
	}

	resp, err := genkit.Generate(ctx, g,
		ai.WithSystem(string(prompt)),
		ai.WithPrompt("What are the existing namespaces?"),
		ai.WithTools(toolRefs...),
		ai.WithOutputType(tools.ToolOutput{}),
	)

	if err != nil {
		t.Fatalf("failed to generate text: %v", err)
	}

	rt.GetLogger().Info("Generated response:", "text", resp.Text())
}

func TestMCPFlow(t *testing.T) {
	ctx := context.Background()
	g := genkit.Init(ctx, nil)

	mcpHostName := "mcpHost" // 默认值
	promptDir := "../../prompts" // 默认值

	flow := genkit.DefineFlow(g, "mcpTest",
		func(ctx context.Context, userPrompt string) (string, error) {
			mcpToolManager, err := tools.NewMCPToolManager(g, mcpHostName)
			if err != nil {
				return "", fmt.Errorf("failed to create MCP tool bootstrap: %v", err)
			}

			toolRefs := mcpToolManager.ToolRefs()

			prompt, err := os.ReadFile(promptDir + "/agentSystem.txt")
			if err != nil {
				return "", fmt.Errorf("failed to read prompt file: %v", err)
			}

			resp, err := genkit.Generate(ctx, g,
				ai.WithSystem(string(prompt)),
				ai.WithPrompt(userPrompt),
				ai.WithTools(toolRefs...),
				ai.WithReturnToolRequests(true),
			)

			if err != nil {
				return "", fmt.Errorf("failed to generate text: %v", err)
			}

			return resp.Text(), nil
		})

	resp, err := flow.Run(context.Background(), "List all namespaces in the Kubernetes cluster")
	if err != nil {
		t.Fatalf("failed to run MCP flow: %v", err)
	}
	rt.GetLogger().Info("MCP Flow response:", "response", resp)
}
