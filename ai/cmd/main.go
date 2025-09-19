package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/server"

	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/plugins/dashscope"
)

func main() {
	port := 8080
	reActAgent := react.Create(manager.Registry(dashscope.Qwen3.Key(), manager.PrettyLogger()))
	apiRouter := server.NewRouter(reActAgent)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: apiRouter.GetEngine(),
	}

	// 启动服务器
	go func() {
		fmt.Printf("🤖 Dubbo Admin AI Agent Server starting on port %d...\n", port)
		fmt.Printf("📖 API Documentation: http://localhost:%d/docs\n", port)
		fmt.Printf("🔍 Health Check: http://localhost:%d/api/v1/ai/health\n", port)

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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("✅ Server exited")
}
