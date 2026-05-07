package engine

import (
	"context"
	"dubbo-admin-ai/runtime"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/mcp"
)

// TODO: Add Refresh/Reconnect lifecycle support for unstable networks.
// Suggested behavior:
// 1) Reconnect MCP host/client when transport is broken.
// 2) Re-fetch active tools after reconnect.
// 3) Re-register any newly discovered tools into the main Genkit registry.
// 4) Keep registration idempotent and handle tool-name conflicts explicitly.
type MCPToolManager struct {
	registry       *genkit.Genkit
	mcpHost        *mcp.MCPHost
	availableTools map[string]ai.Tool
}

func DefineMCPHost(g *genkit.Genkit, hostName string, mcpNameCmdMap map[string][]string) (*mcp.MCPHost, error) {
	servers := make([]mcp.MCPServerConfig, 0, len(mcpNameCmdMap))
	for key, value := range mcpNameCmdMap {
		server := mcp.MCPServerConfig{
			Name: key,
		}

		// 检查连接类型
		if len(value) > 0 && value[0] == "http" {
			// HTTP连接
			url := value[1]
			server.Config = mcp.MCPClientOptions{
				Name: key,
				HTTP: &mcp.HTTPConfig{
					URL: url,
				},
			}
		} else {
			// Stdio连接（默认）
			server.Config = mcp.MCPClientOptions{
				Name: key,
				Stdio: &mcp.StdioConfig{
					Command: value[0],
					Args:    value[1:],
				},
			}
		}
		servers = append(servers, server)
	}
	host, err := mcp.NewMCPHost(g, mcp.MCPHostOptions{Name: hostName, MCPServers: servers})
	if err != nil {
		return nil, err
	}
	return host, nil
}

func NewMCPToolManager(rt *runtime.Runtime, hostName string) (*MCPToolManager, error) {
	if rt == nil {
		return nil, fmt.Errorf("runtime is nil")
	}
	g := rt.GetGenkitRegistry()
	if g == nil {
		return nil, fmt.Errorf("genkit registry is nil")
	}

	mcps := map[string][]string{
		// 本地stdio连接（用于开发）
		"dubbo-admin-local": {
			"go",
			"run",
			"cmd/mcp-server/main.go",
		},
		// HTTP连接（用于生产环境）
		"dubbo-admin": {
			"http",
			"http://localhost:8080/mcp",
		},
		"kubernetes": {
			"npx",
			"-y",
			"kubernetes-mcp-server@latest",
		},
		// "prometheus": {
		// 	"http",
		// 	"http://localhost:9090/mcp",
		// },
	}

	host, err := DefineMCPHost(g, hostName, mcps)
	if err != nil {
		return nil, err
	}

	// TODO: If network instability causes MCP disconnect, implement
	// a recovery path (Reconnect + Refresh tools) and retry this bootstrap flow.
	activeTools, err := host.GetActiveTools(context.Background(), g)
	if err != nil {
		return nil, err
	}

	var availableTools = make(map[string]ai.Tool, len(activeTools))
	for _, mcpTool := range activeTools {
		toolName := mcpTool.Name()
		if genkit.LookupTool(g, toolName) != nil {
			return nil, fmt.Errorf("duplicate tool name detected when registering MCP tool: %s", toolName)
		}

		def := mcpTool.Definition()
		desc := ""
		if def != nil {
			desc = def.Description
		}
		handler := func(ctx *ai.ToolContext, input any) (any, error) {
			return mcpTool.RunRaw(ctx.Context, input)
		}

		var registered ai.Tool
		if def != nil && len(def.InputSchema) > 0 {
			registered = genkit.DefineToolWithInputSchema(g, toolName, desc, def.InputSchema, handler)
		} else {
			registered = genkit.DefineTool(g, toolName, desc, handler)
		}
		availableTools[toolName] = registered
	}

	return &MCPToolManager{
		registry:       g,
		mcpHost:        host,
		availableTools: availableTools,
	}, nil
}

func (mtm *MCPToolManager) AllTools() []ai.Tool {
	var tools []ai.Tool
	for _, tool := range mtm.availableTools {
		tools = append(tools, tool)
	}
	return tools
}

func (mtm *MCPToolManager) ToolRefs() (toolRef []ai.ToolRef) {
	for _, tool := range mtm.availableTools {
		toolRef = append(toolRef, tool)
	}
	return toolRef
}

func (mtm *MCPToolManager) GetToolByName(name string) ai.Tool {
	return mtm.availableTools[name]
}
