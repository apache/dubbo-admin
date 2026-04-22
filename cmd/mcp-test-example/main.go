/*
 * MCP 工具测试示例
 * 直接运行: go run cmd/mcp-test-example/main.go
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/core/bootstrap"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/tools"
	"github.com/apache/dubbo-admin/pkg/mcp/core"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
)

// ANSI 颜色代码
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Bold   = "\033[1m"
)

func main() {
	// 1. 加载配置
	configPath := "cmd/mcp-test-example/test-config.yaml"
	cfg := app.DefaultAdminConfig()
	if err := config.Load(configPath, &cfg); err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 2. 初始化 runtime
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rt, err := bootstrap.Bootstrap(ctx, cfg)
	if err != nil {
		log.Fatalf("❌ Bootstrap 失败: %v", err)
	}

	// 3. 创建可取消的 stop channel 用于测试
	stopCh := make(chan struct{})

	// 4. 在 goroutine 中启动 runtime
	go func() {
		if err := rt.Start(stopCh); err != nil {
			log.Printf("⚠️  Runtime 返回错误: %v", err)
		}
	}()

	// 5. 等待组件初始化 (增加等待时间确保就绪)
	fmt.Printf("⏳ 等待组件初始化")
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		fmt.Printf(".")
	}
	fmt.Printf(" 完成!\n\n")

	// 6. 创建 console context
	consoleCtx := consolectx.NewConsoleContext(rt)

	// 7. 运行测试
	runTests(consoleCtx)

	// 8. 通知 runtime 停止
	close(stopCh)
	time.Sleep(500 * time.Millisecond) // 给 goroutine 时间退出
}

// runTests 运行所有测试
func runTests(ctx consolectx.Context) {
	fmt.Printf("%s%s═══════════════════════════════════════════════════════════%s\n", Cyan, Bold, Reset)
	fmt.Printf("%s%s                    MCP 工具测试%s\n", Cyan, Bold, Reset)
	fmt.Printf("%s%s═══════════════════════════════════════════════════════════%s\n\n", Cyan, Bold, Reset)

	// 创建 MCP Server 并注册所有工具（用于验证工具定义）
	server := core.NewServer("test", "1.0.0")
	reg := server.GetRegistry()
	reg.RegisterRegistrar(&tools.MetricsRegistrar{})
	reg.RegisterRegistrar(&tools.ResourceSearchRegistrar{})
	reg.RegisterRegistrar(&tools.ServiceRegistrar{})
	reg.RegisterRegistrar(&tools.DetailRegistrar{})
	reg.RegisterAll()

	fmt.Printf("已注册 %d 个工具\n\n", len(reg.List()))

	// 测试 1: 获取集群信息
	runToolTest("GetClusterInfo", func() (*types.ToolResult, error) {
		return tools.GetClusterInfo(ctx, map[string]any{
			"mesh": "nacos2.5", // 使用配置中的 mesh 名称
		})
	})

	// 测试 2: 全局搜索 (空关键字获取所有服务)
	runToolTest("GlobalSearch", func() (*types.ToolResult, error) {
		return tools.GlobalSearch(ctx, map[string]any{
			"keyword":    "", // 空关键字返回所有数据
			"searchType": "serviceName",
			"mesh":       "nacos2.5",
			"pageSize":   100, // 获取更多数据
			"pageNumber": 1,
		})
	})

	// 测试 3: 搜索服务 (不传 keywords 获取所有)
	runToolTest("SearchServices", func() (*types.ToolResult, error) {
		return tools.SearchServices(ctx, map[string]any{
			// 不传 keywords 参数，会获取所有服务
			"mesh":       "nacos2.5",
			"pageSize":   100,
			"pageNumber": 1,
		})
	})

	// 测试 4: 获取应用详情 (shop-user)
	runToolTest("GetApplicationDetail", func() (*types.ToolResult, error) {
		return tools.GetApplicationDetail(ctx, map[string]any{
			"appName": "shop-user",
			"mesh":    "nacos2.5",
		})
	})

	// 测试 5: 获取应用的实例
	runToolTest("GetApplicationInstances", func() (*types.ToolResult, error) {
		return tools.GetApplicationInstances(ctx, map[string]any{
			"appName": "shop-user",
			"mesh":    "nacos2.5",
			"pageSize": 20,
		})
	})

	// 测试 6: 搜索实例
	runToolTest("SearchInstances", func() (*types.ToolResult, error) {
		return tools.SearchInstances(ctx, map[string]any{
			"appName": "shop-user",
			"mesh":    "nacos2.5",
			"pageSize": 20,
		})
	})

	fmt.Printf("\n%s%s═══════════════════════════════════════════════════════════%s\n", Green, Bold, Reset)
	fmt.Printf("%s%s                      测试完成 ✓%s\n", Green, Bold, Reset)
	fmt.Printf("%s%s═══════════════════════════════════════════════════════════%s\n\n", Green, Bold, Reset)
}

// runToolTest 执行单个工具测试并格式化输出
func runToolTest(toolName string, fn func() (*types.ToolResult, error)) {
	fmt.Printf("\n%s%s▶ %s()%s\n", Yellow, Bold, toolName, Reset)

	result, err := fn()
	if err != nil {
		fmt.Printf("  %s%s✗ 错误: %v%s\n", Red, Bold, err, Reset)
		return
	}

	if result.IsError {
		fmt.Printf("  %s%s✗ Tool 返回错误:%s\n", Red, Bold, Reset)
		fmt.Printf("  %s%s%s\n", Red, result.Content[0].Text, Reset)
		return
	}

	printToolResult(result.Content[0].Text)
}

// printToolResult 格式化打印工具结果
func printToolResult(jsonStr string) {
	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
		formatted, _ := json.MarshalIndent(data, "  ", "  ")
		lines := fmt.Sprintf("%s", string(formatted))

		// 为每一行添加颜色和缩进
		fmt.Printf("  %s%s┌─ 结果:%s\n", Purple, Bold, Reset)
		for _, line := range splitLines(lines) {
			fmt.Printf("  %s│%s  %s%s\n", Purple, Reset, Cyan, line)
		}
		fmt.Printf("  %s└─────────────────────────────────%s\n\n", Purple, Reset)
	} else {
		fmt.Printf("  %s%s[原始结果]:%s %s%s%s\n\n", Purple, Bold, Reset, Green, jsonStr, Reset)
	}
}

// splitLines 分割字符串为行
func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
