package toolstest

import (
	"strings"
	"testing"

	compTools "dubbo-admin-ai/component/tools"
)

func TestToolsComponent_Validate(t *testing.T) {
	comp, err := compTools.NewToolsComponent(compTools.ToolConfig{
		EnableMCPTools: true,
		MCP:            compTools.MCPConfig{Host: ""},
		MCPTimeout:     30,
		MCPMaxRetries:  1,
	})
	if err != nil {
		t.Fatalf("NewToolsComponent() error: %v", err)
	}
	if err := comp.Validate(); err == nil || !strings.Contains(err.Error(), "mcp.host") {
		t.Fatalf("expected mcp.host validation error, got %v", err)
	}
}
