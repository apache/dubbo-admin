package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/server"

	"github.com/gin-gonic/gin"
)

func main() {
	port := flag.Int("port", 8880, "Port for the AI agent server")
	mode := flag.String("mode", "release", "Server mode: dev or prod")
	envPath := flag.String("env", "./.env", "Path to the .env file")
	flag.Parse()

	var logger *slog.Logger
	switch *mode {
	case "release":
		logger = manager.ProductionLogger()
		gin.SetMode(gin.ReleaseMode)
	case "dev":
		logger = manager.DevLogger()
		gin.SetMode(gin.DebugMode)
	}

	reActAgent, err := react.Create(manager.Registry(dashscope.Qwen3_coder.Key(), *envPath, logger))
	if err != nil {
		logger.Error("Failed to create ReAct agent", "error", err)
		return
	}

	apiRouter := server.NewRouter(reActAgent)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: apiRouter.GetEngine(),
	}

	// 启动服务器
	go func() {
		fmt.Printf("🤖 Dubbo Admin AI Agent Server starting on port %d...\n", *port)
		fmt.Printf("📖 API Documentation: http://localhost:%d/docs\n", *port)
		fmt.Printf("🔍 Health Check: http://localhost:%d/api/v1/ai/health\n", *port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("🛑 Shutting down server...")

	// 5秒超时的优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("✅ Server exited")
}
