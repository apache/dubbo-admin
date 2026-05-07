/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/core/bootstrap"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/core"
	"github.com/apache/dubbo-admin/pkg/mcp/tools"
	httppkg "github.com/apache/dubbo-admin/pkg/mcp/transport/http"
)

const (
	// ServerName MCP服务器名称
	ServerName = "dubbo-admin-mcp-http"
	// ServerVersion MCP服务器版本
	ServerVersion = "1.0.0"
	// DefaultPort 默认HTTP端口
	DefaultPort = 8080
)

func main() {
	// 从环境变量获取配置文件路径
	configPath := os.Getenv("DUBBO_ADMIN_CONFIG")
	if configPath == "" {
		// 尝试几个默认位置
		for _, path := range []string{
			"dubbo-admin.yaml",
			"config/dubbo-admin.yaml",
			"app/dubbo-admin/dubbo-admin.yaml",
		} {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}
	}

	if configPath == "" {
		log.Fatal("配置文件未找到，请设置 DUBBO_ADMIN_CONFIG 环境变量")
	}

	// 加载配置
	cfg := app.DefaultAdminConfig()
	if err := config.Load(configPath, &cfg); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// HTTP模式下可以选择禁用Console HTTP服务器
	// cfg.Console = nil

	// 初始化runtime
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rt, err := bootstrap.Bootstrap(ctx, cfg)
	if err != nil {
		log.Fatalf("Bootstrap 失败: %v", err)
	}

	// 启动runtime
	stopCh := make(chan struct{})
	go func() {
		if err := rt.Start(stopCh); err != nil {
			log.Printf("Runtime 返回: %v", err)
		}
	}()

	// 等待runtime完成启动
	fmt.Fprintf(os.Stderr, "等待 discovery 组件同步数据...\n")
	time.Sleep(8 * time.Second)

	// 创建console context
	consoleCtx := consolectx.NewConsoleContext(rt)

	// 创建MCP Server
	server := core.NewServer(ServerName, ServerVersion)

	// 注册所有工具
	reg := server.GetRegistry()
	reg.RegisterRegistrar(&tools.MetricsRegistrar{})
	reg.RegisterRegistrar(&tools.ResourceSearchRegistrar{})
	reg.RegisterRegistrar(&tools.ServiceRegistrar{})
	reg.RegisterRegistrar(&tools.DetailRegistrar{})
	reg.RegisterAll()

	// 设置console context
	server.SetConsoleContext(consoleCtx)

	// 获取HTTP端口配置
	port := getPortConfig()

	// 创建HTTP传输层
	httpCfg := &httppkg.Config{
		Host:            "0.0.0.0",
		Port:            port,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
	transport := httppkg.NewTransportWithConfig(server, httpCfg)

	// 打印启动信息
	fmt.Fprintf(os.Stderr, "\n%s HTTP MCP Server v%s\n", ServerName, ServerVersion)
	fmt.Fprintf(os.Stderr, "已注册 %d 个工具\n", len(reg.List()))
	fmt.Fprintf(os.Stderr, "HTTP服务器启动在: http://0.0.0.0:%d\n", port)
	fmt.Fprintf(os.Stderr, "MCP端点: http://0.0.0.0:%d/mcp\n", port)
	fmt.Fprintf(os.Stderr, "\n使用curl测试:\n")
	fmt.Fprintf(os.Stderr, "  curl -X POST http://localhost:%d/mcp -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}'\n", port)
	fmt.Fprintf(os.Stderr, "\n按 Ctrl+C 停止服务器\n\n")

	// 处理信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 启动HTTP服务器
	errCh := make(chan error, 1)
	go func() {
		errCh <- transport.Start(ctx)
	}()

	// 等待信号或错误
	select {
	case <-sigCh:
		fmt.Fprintf(os.Stderr, "\n收到停止信号，正在关闭...\n")
		cancel()
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			log.Printf("服务器错误: %v", err)
		}
	}

	// 清理
	close(stopCh)
	if err := transport.Shutdown(); err != nil {
		log.Printf("关闭服务器时出错: %v", err)
	}
	fmt.Fprintf(os.Stderr, "MCP Server 已关闭\n")
}

// getPortConfig 获取端口配置
func getPortConfig() int {
	if port := os.Getenv("MCP_HTTP_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			return p
		}
	}
	return DefaultPort
}
