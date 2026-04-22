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

package tools

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/core"
)

// LiveTestOptions 真实测试选项
type LiveTestOptions struct {
	ConfigPath  string
	Mesh        string
	Keyword     string
	ServiceName string
}

// LiveTester 真实数据测试器
type LiveTester struct {
	ctx        context.Context
	consoleCtx consolectx.Context
	server     *core.Server
	options    *LiveTestOptions
}

// NewLiveTester 创建真实数据测试器
func NewLiveTester(configPath string) (*LiveTester, error) {
	// 这里需要导入 bootstrap 包来初始化真实的 runtime
	// 但为了避免循环导入，我们通过依赖注入的方式接收 context

	return &LiveTester{
		ctx:     context.Background(),
		options: &LiveTestOptions{ConfigPath: configPath},
	}, nil
}

// SetConsoleContext 设置 console context（由外部初始化后传入）
func (t *LiveTester) SetConsoleContext(ctx consolectx.Context) {
	t.consoleCtx = ctx
}

// InitServer 初始化 MCP 服务器
func (t *LiveTester) InitServer() {
	t.server = core.NewServer("dubbo-admin-live-test", "1.0.0")
	reg := t.server.GetRegistry()

	// 注册所有工具
	reg.RegisterRegistrar(&MetricsRegistrar{})
	reg.RegisterRegistrar(&ResourceSearchRegistrar{})
	reg.RegisterRegistrar(&ServiceRegistrar{})
	reg.RegisterAll()

	// 设置 console context
	t.server.SetConsoleContext(t.consoleCtx)
}

// TestClusterInfo 测试获取集群信息
func (t *LiveTester) TestClusterInfo(mesh string) (string, error) {
	t.options.Mesh = mesh

	args := map[string]any{
		"mesh": mesh,
	}

	result, err := GetClusterInfo(t.consoleCtx, args)
	if err != nil {
		return "", fmt.Errorf("GetClusterInfo failed: %w", err)
	}

	return result.Content[0].Text, nil
}

// TestGlobalSearch 测试全局搜索
func (t *LiveTester) TestGlobalSearch(keyword, searchType, mesh string) (string, error) {
	args := map[string]any{
		"keyword":    keyword,
		"searchType": searchType,
		"mesh":       mesh,
		"pageSize":   10,
		"pageNumber": 1,
	}

	result, err := GlobalSearch(t.consoleCtx, args)
	if err != nil {
		return "", fmt.Errorf("GlobalSearch failed: %w", err)
	}

	return result.Content[0].Text, nil
}

// TestSearchServices 测试搜索服务
func (t *LiveTester) TestSearchServices(keywords, mesh string) (string, error) {
	args := map[string]any{
		"keywords":   keywords,
		"mesh":       mesh,
		"pageSize":   10,
		"pageNumber": 1,
	}

	result, err := SearchServices(t.consoleCtx, args)
	if err != nil {
		return "", fmt.Errorf("SearchServices failed: %w", err)
	}

	return result.Content[0].Text, nil
}

// TestGetServiceDetail 测试获取服务详情
func (t *LiveTester) TestGetServiceDetail(serviceName, group, version, side, mesh string) (string, error) {
	args := map[string]any{
		"serviceName": serviceName,
		"group":       group,
		"version":     version,
		"side":        side,
		"mesh":        mesh,
	}

	result, err := GetServiceDetail(t.consoleCtx, args)
	if err != nil {
		return "", fmt.Errorf("GetServiceDetail failed: %w", err)
	}

	return result.Content[0].Text, nil
}

// TestToolViaServer 通过服务器测试工具调用
func (t *LiveTester) TestToolViaServer(toolName string, args map[string]any) (*core.JSONRPCResponse, error) {
	req := &core.JSONRPCRequest{
		JSONRPC: core.JSONRPCVersion,
		ID:      1,
		Method:  core.MethodToolsCall,
		Params: map[string]any{
			"name":      toolName,
			"arguments": args,
		},
	}

	resp := t.server.HandleRequest(req)
	if resp.Error != nil {
		return nil, fmt.Errorf("tool call failed: %s", resp.Error.Message)
	}

	return resp, nil
}

// PrintResult 打印结果
func (t *LiveTester) PrintResult(name string, result string) {
	fmt.Printf("\n=== %s ===\n", name)
	fmt.Printf("Result:\n%s\n", result)
	fmt.Printf("==================\n\n")
}

// RunAllTests 运行所有测试
func (t *LiveTester) RunAllTests(mesh, keyword, serviceName string) error {
	fmt.Println("🧪 Starting live MCP tools tests...")

	// 1. 测试集群信息
	fmt.Println("1️⃣ Testing GetClusterInfo...")
	clusterInfo, err := t.TestClusterInfo(mesh)
	if err != nil {
		log.Printf("❌ GetClusterInfo failed: %v", err)
	} else {
		t.PrintResult("Cluster Info", clusterInfo)
	}

	// 2. 测试全局搜索
	fmt.Println("2️⃣ Testing GlobalSearch...")
	searchResult, err := t.TestGlobalSearch(keyword, "serviceName", mesh)
	if err != nil {
		log.Printf("❌ GlobalSearch failed: %v", err)
	} else {
		t.PrintResult("Global Search", searchResult)
	}

	// 3. 测试服务搜索
	fmt.Println("3️⃣ Testing SearchServices...")
	servicesResult, err := t.TestSearchServices(keyword, mesh)
	if err != nil {
		log.Printf("❌ SearchServices failed: %v", err)
	} else {
		t.PrintResult("Search Services", servicesResult)
	}

	// 4. 测试服务详情（如果有服务名）
	if serviceName != "" {
		fmt.Println("4️⃣ Testing GetServiceDetail...")
		detailResult, err := t.TestGetServiceDetail(serviceName, "", "", "provider", mesh)
		if err != nil {
			log.Printf("❌ GetServiceDetail failed: %v", err)
		} else {
			t.PrintResult("Service Detail", detailResult)
		}
	}

	fmt.Println("✅ All tests completed!")
	return nil
}

// RunViaJSONRPC 通过 JSON-RPC 接口测试
func (t *LiveTester) RunViaJSONRPC(mesh, keyword, serviceName string) error {
	fmt.Println("🧪 Testing via JSON-RPC interface...")

	// 测试 tools/list
	fmt.Println("\n1️⃣ Testing tools/list...")
	listReq := &core.JSONRPCRequest{
		JSONRPC: core.JSONRPCVersion,
		ID:      1,
		Method:  core.MethodToolsList,
	}
	listResp := t.server.HandleRequest(listReq)
	listData, _ := json.MarshalIndent(listResp, "", "  ")
	fmt.Printf("Tools list:\n%s\n", string(listData))

	// 测试 get_cluster_info
	fmt.Println("\n2️⃣ Testing get_cluster_info via JSON-RPC...")
	clusterResp, err := t.TestToolViaServer("get_cluster_info", map[string]any{"mesh": mesh})
	if err != nil {
		return err
	}
	clusterData, _ := json.MarshalIndent(clusterResp, "", "  ")
	fmt.Printf("Cluster info response:\n%s\n", string(clusterData))

	// 测试 global_search
	fmt.Println("\n3️⃣ Testing global_search via JSON-RPC...")
	searchResp, err := t.TestToolViaServer("global_search", map[string]any{
		"keyword": keyword,
		"mesh":    mesh,
	})
	if err != nil {
		return err
	}
	searchData, _ := json.MarshalIndent(searchResp, "", "  ")
	fmt.Printf("Search response:\n%s\n", string(searchData))

	return nil
}

// exampleLiveTest 示例：如何在其他地方使用 LiveTester
func exampleLiveTest() {
	// 注意：这只是一个示例，实际使用时需要从外部获取 consoleCtx
	/*
		// 1. 初始化 runtime 和 console context
		cfg := app.DefaultAdminConfig()
		config.Load("dubbo-admin.yaml", &cfg)
		rt, _ := bootstrap.Bootstrap(context.Background(), cfg)
		consoleCtx := context.NewConsoleContext(rt)

		// 2. 创建测试器
		tester, _ := NewLiveTester("dubbo-admin.yaml")
		tester.SetConsoleContext(consoleCtx)
		tester.InitServer()

		// 3. 运行测试
		tester.RunAllTests("default", "demo", "com.example.Service")
	*/
}

// main 函数可以作为独立的测试工具运行
func main() {
	configPath := flag.String("config", "./dubbo-admin.yaml", "配置文件路径")
	mesh := flag.String("mesh", "default", "Mesh 名称")
	keyword := flag.String("keyword", "", "搜索关键字")
	serviceName := flag.String("service", "", "服务名称")
	flag.Parse()

	if *configPath == "" {
		fmt.Println("请提供配置文件路径: -config <path>")
		os.Exit(1)
	}

	fmt.Printf("📋 配置: %s\n", *configPath)
	fmt.Printf("🔍 Mesh: %s\n", *mesh)
	if *keyword != "" {
		fmt.Printf("🔑 关键字: %s\n", *keyword)
	}
	if *serviceName != "" {
		fmt.Printf("🛠️  服务: %s\n", *serviceName)
	}

	// 注意：这里需要真实的 console context
	// 实际使用时需要从外部注入
	fmt.Println("\n⚠️  注意：此测试工具需要真实的 console context")
	fmt.Println("⚠️  请通过代码方式调用，在初始化 runtime 后传入 context")
	fmt.Println("\n示例代码:")
	fmt.Println(`
		rt, _ := bootstrap.Bootstrap(context.Background(), cfg)
		consoleCtx := context.NewConsoleContext(rt)

		tester := &LiveTester{}
		tester.SetConsoleContext(consoleCtx)
		tester.InitServer()
		tester.RunAllTests("default", "demo", "com.example.Service")
	`)

	// 如果有环境变量指定了真实模式，则尝试运行
	if os.Getenv("MCP_LIVE_TEST") == "true" {
		fmt.Println("\n🚀 Running in live test mode...")
		// 这里需要实际的初始化代码
		// 由于循环导入问题，需要在实际使用的地方实现
	}
}

// LiveTestHelper 辅助函数，用于在已有 context 的地方进行测试
func LiveTestHelper(consoleCtx consolectx.Context, mesh, keyword, serviceName string) {
	tester := &LiveTester{
		ctx:        context.Background(),
		consoleCtx: consoleCtx,
		options:    &LiveTestOptions{},
	}
	tester.InitServer()

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 运行测试
	done := make(chan error)
	go func() {
		done <- tester.RunAllTests(mesh, keyword, serviceName)
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("Test failed: %v", err)
		}
	case <-ctx.Done():
		log.Println("Test timeout!")
	}
}

// LiveTestViaJSONRPCHelper 通过 JSON-RPC 测试的辅助函数
func LiveTestViaJSONRPCHelper(consoleCtx consolectx.Context, mesh, keyword string) error {
	tester := &LiveTester{
		ctx:        context.Background(),
		consoleCtx: consoleCtx,
		options:    &LiveTestOptions{},
	}
	tester.InitServer()

	return tester.RunViaJSONRPC(mesh, keyword, "")
}
