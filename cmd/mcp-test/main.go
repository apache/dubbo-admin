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
	"encoding/json"
	"flag"
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
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/mcp/core"
	"github.com/apache/dubbo-admin/pkg/mcp/tools"
)

type TestOptions struct {
	configPath  string
	mesh        string
	keyword     string
	serviceName string
	side        string
	jsonMode    bool
	interactive bool
}

func main() {
	opts := parseFlags()

	if opts.interactive {
		runInteractiveMode(opts)
	} else {
		runTestMode(opts)
	}
}

func parseFlags() *TestOptions {
	opts := &TestOptions{}

	flag.StringVar(&opts.configPath, "config", "app/dubbo-admin/dubbo-admin.yaml", "配置文件路径")
	flag.StringVar(&opts.mesh, "mesh", "default", "Mesh 名称")
	flag.StringVar(&opts.keyword, "keyword", "", "搜索关键字")
	flag.StringVar(&opts.serviceName, "service", "", "服务名称")
	flag.StringVar(&opts.side, "side", "provider", "服务端 (provider/consumer)")
	flag.BoolVar(&opts.jsonMode, "json", false, "JSON 模式输出")
	flag.BoolVar(&opts.interactive, "i", false, "交互模式")
	flag.Parse()

	return opts
}

func runInteractiveMode(opts *TestOptions) {
	fmt.Println("🔧 MCP 工具交互式测试")
	fmt.Println("==========================")

	// 初始化 runtime
	ctx, cancel := signalHandler()
	defer cancel()

	consoleCtx, server, err := initializeRuntime(opts.configPath, ctx)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	fmt.Println("\n可用命令:")
	fmt.Println("  1. cluster    - 获取集群信息")
	fmt.Println("  2. search     - 全局搜索")
	fmt.Println("  3. services   - 搜索服务")
	fmt.Println("  4. detail     - 服务详情")
	fmt.Println("  5. list       - 列出所有工具")
	fmt.Println("  6. help       - 显示帮助")
	fmt.Println("  7. quit       - 退出")

	for {
		fmt.Print("\n> ")
		var cmd string
		fmt.Scanln(&cmd)

		switch cmd {
		case "1", "cluster":
			testClusterInfo(consoleCtx, server, opts.mesh)
		case "2", "search":
			fmt.Print("输入搜索关键字: ")
			var keyword string
			fmt.Scanln(&keyword)
			testGlobalSearch(consoleCtx, server, keyword, opts.mesh)
		case "3", "services":
			fmt.Print("输入服务关键字: ")
			var keyword string
			fmt.Scanln(&keyword)
			testSearchServices(consoleCtx, server, keyword, opts.mesh)
		case "4", "detail":
			fmt.Print("输入服务名: ")
			var serviceName string
			fmt.Scanln(&serviceName)
			testServiceDetail(consoleCtx, server, serviceName, opts.side, opts.mesh)
		case "5", "list":
			listTools(server)
		case "6", "help":
			showHelp()
		case "7", "quit", "exit", "q":
			fmt.Println("👋 再见!")
			return
		default:
			fmt.Printf("未知命令: %s\n", cmd)
		}
	}
}

func runTestMode(opts *TestOptions) {
	fmt.Println("🧪 MCP 工具测试")
	fmt.Println("================")

	ctx, cancel := signalHandler()
	defer cancel()

	consoleCtx, server, err := initializeRuntime(opts.configPath, ctx)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	// 1. 列出所有工具
	fmt.Println("\n📋 已注册的工具:")
	listTools(server)

	// 2. 测试集群信息
	fmt.Println("\n1️⃣ 测试: get_cluster_info")
	testClusterInfo(consoleCtx, server, opts.mesh)

	// 3. 测试全局搜索（如果有关键字）
	if opts.keyword != "" {
		fmt.Println("\n2️⃣ 测试: global_search")
		testGlobalSearch(consoleCtx, server, opts.keyword, opts.mesh)
	}

	// 4. 测试服务搜索
	if opts.keyword != "" {
		fmt.Println("\n3️⃣ 测试: search_services")
		testSearchServices(consoleCtx, server, opts.keyword, opts.mesh)
	}

	// 5. 测试服务详情（如果有服务名）
	if opts.serviceName != "" {
		fmt.Println("\n4️⃣ 测试: get_service_detail")
		testServiceDetail(consoleCtx, server, opts.serviceName, opts.side, opts.mesh)
	}

	fmt.Println("\n✅ 测试完成!")
}

func initializeRuntime(configPath string, ctx context.Context) (consolectx.Context, *core.Server, error) {
	// 加载配置
	cfg := app.DefaultAdminConfig()
	if err := config.Load(configPath, &cfg); err != nil {
		return nil, nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 初始化 runtime
	rt, err := bootstrap.Bootstrap(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("bootstrap 失败: %w", err)
	}

	// 启动组件
	if err := rt.Start(ctx.Done()); err != nil {
		return nil, nil, fmt.Errorf("启动组件失败: %w", err)
	}

	// 等待组件就绪
	time.Sleep(2 * time.Second)

	// 创建 console context
	consoleCtx := consolectx.NewConsoleContext(rt)

	// 创建 MCP 服务器
	server := core.NewServer("dubbo-admin-mcp-test", "1.0.0")
	reg := server.GetRegistry()

	// 注册所有工具
	reg.RegisterRegistrar(&tools.MetricsRegistrar{})
	reg.RegisterRegistrar(&tools.ResourceSearchRegistrar{})
	reg.RegisterRegistrar(&tools.ServiceRegistrar{})
	reg.RegisterAll()

	// 设置 console context
	server.SetConsoleContext(consoleCtx)

	logger.Init(cfg.Log)

	return consoleCtx, server, nil
}

func testClusterInfo(consoleCtx consolectx.Context, server *core.Server, mesh string) {
	args := map[string]any{"mesh": mesh}

	result, err := tools.GetClusterInfo(consoleCtx, args)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		return
	}

	printJSON("Cluster Info", result.Content[0].Text)
}

func testGlobalSearch(consoleCtx consolectx.Context, server *core.Server, keyword, mesh string) {
	args := map[string]any{
		"keyword":    keyword,
		"searchType": "serviceName",
		"mesh":       mesh,
		"pageSize":   10,
		"pageNumber": 1,
	}

	result, err := tools.GlobalSearch(consoleCtx, args)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		return
	}

	printJSON("Global Search Result", result.Content[0].Text)
}

func testSearchServices(consoleCtx consolectx.Context, server *core.Server, keywords, mesh string) {
	args := map[string]any{
		"keywords":   keywords,
		"mesh":       mesh,
		"pageSize":   10,
		"pageNumber": 1,
	}

	result, err := tools.SearchServices(consoleCtx, args)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		return
	}

	printJSON("Search Services Result", result.Content[0].Text)
}

func testServiceDetail(consoleCtx consolectx.Context, server *core.Server, serviceName, side, mesh string) {
	args := map[string]any{
		"serviceName": serviceName,
		"group":       "",
		"version":     "",
		"side":        side,
		"mesh":        mesh,
	}

	result, err := tools.GetServiceDetail(consoleCtx, args)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		return
	}

	printJSON("Service Detail", result.Content[0].Text)
}

func testToolViaServer(server *core.Server, toolName string, args map[string]any) (*core.JSONRPCResponse, error) {
	req := &core.JSONRPCRequest{
		JSONRPC: core.JSONRPCVersion,
		ID:      1,
		Method:  core.MethodToolsCall,
		Params: map[string]any{
			"name":      toolName,
			"arguments": args,
		},
	}

	resp := server.HandleRequest(req)
	if resp.Error != nil {
		return nil, fmt.Errorf("tool call failed: %s", resp.Error.Message)
	}

	return resp, nil
}

func listTools(server *core.Server) {
	reg := server.GetRegistry()
	tools := reg.List()

	for i, tool := range tools {
		fmt.Printf("  %d. %s\n", i+1, tool.Name)
		fmt.Printf("     描述: %s\n", tool.Description)

		if len(tool.InputSchema.Required) > 0 {
			fmt.Printf("     必需参数: %v\n", tool.InputSchema.Required)
		}
		fmt.Println()
	}
}

func showHelp() {
	fmt.Println("\n帮助信息:")
	fmt.Println("  cluster    - 获取集群信息（应用数、服务数、实例数）")
	fmt.Println("  search     - 全局搜索（支持按服务名、实例、应用搜索）")
	fmt.Println("  services   - 搜索 Dubbo 服务")
	fmt.Println("  detail     - 获取服务详情（服务分布、实例信息）")
	fmt.Println("  list       - 列出所有可用工具")
	fmt.Println("  quit       - 退出程序")
}

func printJSON(title, jsonStr string) {
	fmt.Printf("\n=== %s ===\n", title)

	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
		formatted, _ := json.MarshalIndent(data, "  ", "  ")
		fmt.Printf("  %s\n", string(formatted))
	} else {
		fmt.Printf("  %s\n", jsonStr)
	}

	fmt.Println("==================")
}

func signalHandler() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

func waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\n🛑 收到停止信号...")
}
