package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dubbo-admin-ai/component/agent/react"
	"dubbo-admin-ai/component/logger"
	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/component/models"
	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/component/server"
	"dubbo-admin-ai/component/tools"
	"dubbo-admin-ai/runtime"
)

// registerFactorys explicitly registers all component factories
// Registration order determines component initialization order
func registerFactorys(rt *runtime.Runtime) {
	// Core components (no dependencies)
	rt.RegisterFactory("logger", logger.LoggerFactory)
	rt.RegisterFactory("memory", memory.MemoryFactory)

	// Model components (depend on logger)
	rt.RegisterFactory("models", models.ModelsFactory)

	// Tools components (depend on models, memory)
	rt.RegisterFactory("tools", tools.ToolsFactory)

	// RAG components (depend on models)
	rt.RegisterFactory("rag", compRag.RAGFactory)

	// Server components (depend on all other components)
	rt.RegisterFactory("server", server.ServerFactory)

	// Agent components (depend on tools, rag)
	rt.RegisterFactory("agent", react.AgentFactory)
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to the AI configuration file")
	flag.Parse()

	rt, err := runtime.Bootstrap(*configPath, registerFactorys)
	if err != nil {
		log.Fatalf("Failed to initialize runtime: %v", err)
	}
	rt.GetLogger().Info("🤖 Dubbo Admin AI Agent Server initialized successfully")

	// Wait for interrupt signal to gracefully shutdown server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("🛑 Shutting down server...")

	// Stop all components
	if err := stopComponents(rt); err != nil {
		log.Printf("Warning: Error stopping components: %v", err)
	}

	fmt.Println("✅ Server exited")
}

func stopComponents(rt *runtime.Runtime) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error)

	go func() {
		rt.Components.Range(func(key, value interface{}) bool {
			comp := value.(runtime.Component)

			if err := comp.Stop(); err != nil {
				rt.GetLogger().Error("Failed to stop component",
					"name", comp.Name(),
					"error", err)
			}

			return true
		})
		done <- nil
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout")
	case <-done:
		return nil
	}
}
