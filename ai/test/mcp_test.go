package test

import (
	"context"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/utils"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/mcp"
)

func TestMain(t *testing.T) {
	ctx := context.Background()
	g := manager.Registry(dashscope.Qwen3.Key(), nil)

	// Option 1: Single Chroma Server Connection
	setupSingleChromaServer(ctx, g)

	// Option 2: Multiple Servers with Chroma
	// setupMultipleServers(ctx, g)
}

// Option 1: Connect to a single Chroma MCP server
func setupSingleChromaServer(ctx context.Context, g *genkit.Genkit) {
	// Get the full path to your data directory
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

	// Get available resources from Chroma
	resources, err := client.GetActiveResources(ctx)
	if err != nil {
		log.Fatal("Failed to get Chroma resources:", err)
	}

	log.Printf("Available Chroma resources: %d", len(resources))
	for _, resource := range resources {
		log.Printf("- %s", resource.Name())
	}
}

// Option 2: Use MCPHost to manage multiple servers including Chroma
func setupMultipleServers(ctx context.Context, g *genkit.Genkit) {
	homeDir, _ := os.UserHomeDir()
	chromaDataDir := filepath.Join(homeDir, "chroma-data")

	// Create host with multiple servers
	host, err := mcp.NewMCPHost(g, mcp.MCPHostOptions{
		Name: "my-app",
		MCPServers: []mcp.MCPServerConfig{
			{
				Name: "chroma",
				Config: mcp.MCPClientOptions{
					Name: "chroma-server",
					Stdio: &mcp.StdioConfig{
						Command: "uvx",
						Args: []string{
							"chroma-mcp",
							"--client-type", "persistent",
							"--data-dir", chromaDataDir,
						},
					},
				},
			},
			// Add other MCP servers here if needed
			{
				Name: "filesystem",
				Config: mcp.MCPClientOptions{
					Name: "fs-server",
					Stdio: &mcp.StdioConfig{
						Command: "npx",
						Args:    []string{"@modelcontextprotocol/server-filesystem", "/tmp"},
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatal("Failed to create MCP host:", err)
	}

	// Get all tools from all connected servers
	allTools, err := host.GetActiveTools(ctx, g)
	if err != nil {
		log.Fatal("Failed to get tools:", err)
	}

	log.Printf("Total tools available: %d", len(allTools))
	for _, tool := range allTools {
		log.Printf("- %s: %s", tool.Name(), tool.Definition().Description)
	}

	// Get all resources from all servers
	allResources, err := host.GetActiveResources(ctx)
	if err != nil {
		log.Fatal("Failed to get resources:", err)
	}

	log.Printf("Total resources available: %d", len(allResources))
}

// Example function showing how to use Chroma tools in AI generation
func useChromaInAIGeneration(ctx context.Context, g *genkit.Genkit, chromaTools []ai.Tool, chromaResources []ai.Resource) {
	// Define a model (replace with your actual model)
	// genkit.DefineModel(g, "your-model", ...)

	// Example: Generate response using Chroma tools
	resp, err := genkit.Generate(ctx, g,
		ai.WithModelName("your-model"),
		ai.WithMessages(ai.NewUserMessage(
			ai.NewTextPart("Search for documents about machine learning"),
		)),
		ai.WithTools(utils.Tools2ToolRef(chromaTools)...), // Pass Chroma tools to the model
		ai.WithResources(chromaResources...),              // Pass Chroma resources if any
	)

	if err != nil {
		log.Printf("Generation failed: %v", err)
		return
	}

	log.Printf("AI Response: %s", resp.Text())
}

// Example helper to work with specific Chroma operations
func exampleChromaOperations(ctx context.Context, client *mcp.GenkitMCPClient, g *genkit.Genkit) {
	// Get specific tools by name (adjust names based on actual Chroma MCP tools)
	_, err := client.GetActiveTools(ctx, g)
	if err != nil {
		log.Printf("Error getting tools: %v", err)
		return
	}

	// Example usage (uncomment and adjust based on actual tool schemas):

	// if createCollectionTool != nil {
	//     result, err := createCollectionTool.Run(ctx, map[string]interface{}{
	//         "name": "documents",
	//         "metadata": map[string]interface{}{
	//             "description": "Document collection",
	//         },
	//     })
	//     log.Printf("Create collection result: %v, error: %v", result, err)
	// }

	// if addDocumentsTool != nil {
	//     result, err := addDocumentsTool.Run(ctx, map[string]interface{}{
	//         "collection_name": "documents",
	//         "documents": []string{"This is a test document"},
	//         "ids": []string{"doc1"},
	//     })
	//     log.Printf("Add documents result: %v, error: %v", result, err)
	// }

	// if queryCollectionTool != nil {
	//     result, err := queryCollectionTool.Run(ctx, map[string]interface{}{
	//         "collection_name": "documents",
	//         "query_texts": []string{"test"},
	//         "n_results": 5,
	//     })
	//     log.Printf("Query result: %v, error: %v", result, err)
	// }
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr
}
