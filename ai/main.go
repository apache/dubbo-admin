package main

import (
	"fmt"

	"dubbo-admin-ai/internal/agent"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/internal/tools"
)

func main() {
	err := manager.LoadEnvVars()
	if err != nil {
		fmt.Printf("Failed to load environment variables: %v", err)
		return
	}

	logger := manager.GetLogger()
	g, err := manager.GetGlobalGenkit()
	if err != nil {
		logger.Error("Failed to get global Genkit", "error", err)
		return
	}

	tools.RegisterAllMockTools(g)
	agent.InitFlows(g)

	// select {}
}
