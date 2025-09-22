package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/plugins/dashscope"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/mcp"
)

type WeatherInput struct {
	Location string `json:"location" jsonschema_description:"Location to get weather for"`
}

func defineWeatherFlow(g *genkit.Genkit) *core.Flow[WeatherInput, string, struct{}] {
	getWeatherTool := genkit.DefineTool(g, "getWeather", "Gets the current weather in a given location",
		func(ctx *ai.ToolContext, input WeatherInput) (string, error) {
			// Here, we would typically make an API call or database query. For this
			// example, we just return a fixed value.
			log.Printf("Tool 'getWeather' called for location: %s", input.Location)
			return fmt.Sprintf("The current weather in %s is 63°F and sunny.", input.Location), nil
		})

	return genkit.DefineFlow(g, "getWeatherFlow",
		func(ctx context.Context, location WeatherInput) (string, error) {
			resp, err := genkit.Generate(ctx, g,
				ai.WithTools(getWeatherTool),
				ai.WithPrompt("What's the weather in %s?", location.Location),
			)
			if err != nil {
				return "", err
			}
			return resp.Text(), nil
		})
}

func TestTextGeneration(t *testing.T) {
	g := manager.Registry(dashscope.Qwen3.Key(), nil)
	_ = react.Create(g)
	ctx := context.Background()

	resp, err := genkit.GenerateText(ctx, g, ai.WithPrompt("Hello, Who are you?"))
	if err != nil {
		t.Fatalf("failed to generate text: %v", err)
	}
	t.Logf("Generated text: %s", resp)

	fmt.Printf("%s", resp)
}

func TestWeatherFlowRun(t *testing.T) {
	g := manager.Registry(dashscope.Qwen3.Key(), nil)
	_ = react.Create(g)
	ctx := context.Background()

	flow := defineWeatherFlow(g)
	flow.Run(ctx, WeatherInput{Location: "San Francisco"})
}

func TestMCP(t *testing.T) {
	ctx := context.Background()
	g := manager.Registry(dashscope.Qwen3.Key(), nil)

	homeDir, _ := os.UserHomeDir()
	chromaDataDir := filepath.Join(homeDir, "chroma-data") // Adjust this path

	// Create MCP client for Chroma
	client, err := mcp.NewGenkitMCPClient(mcp.MCPClientOptions{
		Name: "chroma-server",
		Stdio: &mcp.StdioConfig{
			Command: "uvx",
			Args: []string{
				"chroma-mcp",
				"--client-type", "persistent",
				"--data-dir", chromaDataDir,
			},
		},
	})
	if err != nil {
		log.Fatal("Failed to connect to Chroma:", err)
	}

	// Get available tools from Chroma
	tools, err := client.GetActiveTools(ctx, g)
	if err != nil {
		log.Fatal("Failed to get Chroma tools:", err)
	}

	log.Printf("Available Chroma tools: %d", len(tools))
	for _, tool := range tools {
		log.Printf("- %s: %s", tool.Name(), tool.Definition().Description)
	}

	for _, tool := range tools {
		if tool.Name() == "chroma-server_chroma_list_collections" {
			input := map[string]any{
				"limit":  5,
				"offset": 0,
			}
			output, err := tool.RunRaw(ctx, input)
			fmt.Printf("Output: %v, err: %v", output, err)
			break
		}
	}

}
