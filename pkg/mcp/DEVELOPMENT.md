# MCP 工具开发指南

本文档介绍如何在 dubbo-admin 中添加、修改或删除 MCP 工具，基于项目真实代码演示完整开发流程。

---

## 目录结构

```
pkg/mcp/
├── server.go           # MCP 服务器核心
├── register.go         # 工具注册中心（新增工具必须在此注册）
├── component.go        # 组件集成
├── common/             # 公共类型和工具
│   ├── constants.go    # 常量定义（DefaultPageSize, DefaultPageNumber 等）
│   ├── types.go        # 协议类型（ToolDef, InputSchema, ToolResult 等）
│   └── utils.go        # ArgsHelper, JsonResult, ErrorResult 等工具函数
└── tools/              # 工具业务实现
    ├── application.go  # 应用相关工具
    ├── cluster.go      # 集群相关工具
    ├── instance.go     # 实例相关工具
    ├── search.go       # 搜索相关工具
    └── service.go      # 服务相关工具
```

---

## 快速上手

添加新 MCP 工具只需三步：

```
1. pkg/mcp/tools/xxx.go    → 实现处理函数
2. pkg/mcp/register.go    → 注册工具定义
3. go build && curl 测试
```

**核心代码模板：**

```go
// 1. 处理函数 (pkg/mcp/tools/my_tool.go)
package tools

func MyTool(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
    helper := common.NewArgsHelper(args)
    mesh := common.GetMeshArg(ctx, args)
    // ... 业务逻辑 ...
    return common.JsonResult(result)
}

// 2. 注册 (pkg/mcp/register.go)
server.RegisterTool(&common.ToolDef{
    Name:        "my_tool",
    Description: "工具描述",
    InputSchema: common.InputSchema{...},
    Handler:     tools.MyTool,
})
```

---

## 完整案例：添加应用搜索工具

### 背景

现有 MCP 工具中有 `search_services` 和 `search_instances`，但缺少 `search_applications`。本案例演示如何添加这个缺失的工具。

### 现有服务层 API

项目中已有应用搜索服务：

```go
// pkg/console/service/application.go:339
func SearchApplications(ctx consolectx.Context, req *model.ApplicationSearchReq) (*model.SearchPaginationResult, error)
```

### 第一步：在 application.go 中添加处理函数

编辑 `pkg/mcp/tools/application.go`，添加 `SearchApplications` 工具函数：

```go
// pkg/mcp/tools/application.go
// 在文件末尾添加以下函数

// SearchApplications 搜索应用
func SearchApplications(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	keywords := helper.GetString("keywords", "")
	mesh := common.GetMeshArg(ctx, args)
	pageSize := helper.GetInt("pageSize", common.DefaultPageSize)
	pageNumber := helper.GetInt("pageNumber", common.DefaultPageNumber)

	req := &model.ApplicationSearchReq{
		Keywords: keywords,
		Mesh:     mesh,
		PageReq:  common.BuildPageReq(pageNumber, pageSize),
	}

	result, err := service.SearchApplications(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	applications, totalCount := extractApplications(result)

	return common.JsonResult(map[string]any{
		"keywords":     keywords,
		"mesh":         mesh,
		"pageNumber":   pageNumber,
		"pageSize":     pageSize,
		"applications": applications,
		"totalCount":   totalCount,
	})
}

// extractApplications 从分页结果中提取应用列表
func extractApplications(result *model.SearchPaginationResult) ([]any, int) {
	if result == nil || result.List == nil {
		return []any{}, 0
	}

	apps, ok := result.List.([]*model.ApplicationSearchResp)
	if !ok {
		return []any{}, 0
	}

	resultSlice := make([]any, 0, len(apps))
	for _, app := range apps {
		if app != nil {
			resultSlice = append(resultSlice, map[string]any{
				"appName":          app.AppName,
				"instanceCount":    app.InstanceCount,
				"deployClusters":   app.DeployClusters,
				"registryClusters": app.RegistryClusters,
			})
		}
	}
	return resultSlice, int(result.PageInfo.Total)
}
```

### 第二步：在 register.go 中注册工具

编辑 `pkg/mcp/register.go`，在 `RegisterTools` 函数中添加注册代码：

```go
// pkg/mcp/register.go

func RegisterTools(server *Server) {
	// ... 其他工具注册 ...

	// 注册应用搜索工具
	server.RegisterTool(&common.ToolDef{
		Name:        "search_applications",
		Description: "搜索应用，支持关键字搜索和分页",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"keywords": {
					Type:        "string",
					Description: "搜索关键字",
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码，默认为1",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页大小，默认为10",
				},
			},
		},
		Handler: tools.SearchApplications,
	})
}
```

### 第三步：编译测试

```bash
# 编译
go build -o dubbo-admin.exe ./app/dubbo-admin

# 启动服务
./dubbo-admin.exe run -c dubbo-admin.yaml
```

### 第四步：验证测试

```bash
# 1. 验证工具列表
curl -s -X POST http://localhost:8888/api/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | jq '.result.tools[] | select(.name == "search_applications")'

# 2. 测试工具调用（搜索所有应用）
curl -s -X POST http://localhost:8888/api/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_applications","arguments":{"pageNumber":1,"pageSize":10}}}' | jq -r '.result.content[0].text' | jq

# 3. 测试关键字搜索
curl -s -X POST http://localhost:8888/api/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_applications","arguments":{"keywords":"user"}}}' | jq -r '.result.content[0].text' | jq
```

---

## 工具定义规范

### InputSchema 格式（基于真实代码）

```go
// 来自 pkg/mcp/register.go 的真实示例

InputSchema: common.InputSchema{
    Type: "object",
    Properties: map[string]common.PropertyDef{
        "keywords": {
            Type:        "string",
            Description: "搜索关键字",
        },
        "mesh": {
            Type:        "string",
            Description: "网格名称，默认使用第一个 discovery 配置的 id",
        },
        "pageNumber": {
            Type:        "integer",
            Description: "页码，默认为1",
        },
        "pageSize": {
            Type:        "integer",
            Description: "每页大小，默认为10",
        },
        "side": {
            Type:        "string",
            Description: "服务端类型: provider, consumer",
            Enum:        []string{"provider", "consumer"},
        },
    },
    Required: []string{"serviceName"},  // 必填参数列表
}
```

### 返回结果格式

**使用 common.JsonResult（推荐）：**

```go
return common.JsonResult(map[string]any{
    "appName":    "user-service",
    "instanceCount": 2,
})
```

**错误处理：**

```go
if err != nil {
    return common.ErrorResult(err), nil
}
```

---

## 项目常用工具函数

### ArgsHelper 参数辅助器

```go
// 来自 pkg/mcp/common/utils.go
helper := common.NewArgsHelper(args)

// 获取字符串参数（带默认值）
keywords := helper.GetString("keywords", "")
mesh := helper.GetString("mesh", "")

// 获取整数参数（带默认值）
pageNumber := helper.GetInt("pageNumber", common.DefaultPageNumber)
pageSize := helper.GetInt("pageSize", common.DefaultPageSize)

// 获取布尔参数（带默认值）
enabled := helper.GetBool("enabled", false)

// 获取必需参数（无默认值）
appName, ok := helper.GetRequiredString("appName")
if !ok {
    return common.ErrorResult(fmt.Errorf("appName is required")), nil
}
```

### GetMeshArg 获取网格参数

```go
// 来自 pkg/mcp/common/utils.go
// 优先使用 args 中的 mesh，否则使用配置中的第一个 discovery id
mesh := common.GetMeshArg(ctx, args)
```

### BuildPageReq 构建分页请求

```go
// 来自 pkg/mcp/common/utils.go
pageReq := common.BuildPageReq(pageNumber, pageSize)
```

### JsonResult 创建 JSON 结果

```go
// 来自 pkg/mcp/common/utils.go
return common.JsonResult(map[string]any{
    "result": "data",
})
```

### ErrorResult 创建错误结果

```go
// 来自 pkg/mcp/common/utils.go
return common.ErrorResult(err)
```

---

## 常量定义

```go
// 来自 pkg/mcp/common/constants.go

// 分页默认值
const (
    DefaultPageSize   = 10
    DefaultPageNumber = 1
)

// 搜索类型
type SearchType string
const (
    SearchTypeIP           SearchType = "ip"
    SearchTypeInstanceName SearchType = "instanceName"
    SearchTypeAppName      SearchType = "appName"
    SearchTypeName         SearchType = "serviceName"
)

// 服务端类型
type ServiceSide string
const (
    ServiceSideProvider ServiceSide = "provider"
    ServiceSideConsumer ServiceSide = "consumer"
)
```

---

## 现有工具参考

| 工具名 | 文件 | 功能 | 必填参数 |
|--------|------|------|----------|
| get_cluster_info | cluster.go | 获取集群统计信息 | 无 |
| global_search | search.go | 全局搜索 | keyword |
| search_services | service.go | 搜索服务 | 无 |
| get_service_detail | service.go | 获取服务详情 | serviceName |
| get_service_instances | service.go | 获取服务实例列表 | serviceName |
| search_instances | instance.go | 搜索实例 | 无 |
| get_instance_detail | instance.go | 获取实例详情 | instanceName |
| get_instance_metrics | instance.go | 获取实例指标 | instanceName |
| get_application_detail | application.go | 获取应用详情 | appName |
| get_application_instances | application.go | 获取应用实例列表 | appName |
| get_application_services | application.go | 获取应用服务列表 | appName |

---

## 删除工具

### 步骤 1：从 register.go 移除注册

在 `pkg/mcp/register.go` 中删除对应的 `server.RegisterTool(...)` 调用。

### 步骤 2：删除处理函数（可选）

如果该工具的处理函数没有其他地方使用，删除 `pkg/mcp/tools/` 下的对应代码。

### 步骤 3：更新 AI Schema（可选）

从 `ai/schema/json/tools.schema.json` 中移除对应定义。

---

## 总结

添加新 MCP 工具的核心步骤：

| 步骤 | 文件 | 操作 |
|------|------|------|
| 1 | `pkg/mcp/tools/xxx.go` | 实现处理函数，调用 service 层 |
| 2 | `pkg/mcp/register.go` | 注册工具定义 |
| 3 | 命令行 | `go build` 编译 |
| 4 | 命令行 | `curl` 测试 MCP 端点 |

**关键点：**
- 使用 `common.NewArgsHelper` 解析参数
- 使用 `common.GetMeshArg` 获取网格参数
- 使用 `common.JsonResult` 返回结果
- 使用 `common.ErrorResult` 处理错误
- 使用 `common.DefaultPageSize` 和 `common.DefaultPageNumber` 常量

## 可观测性诊断工具

Dubbo Admin MCP 提供以下只读诊断工具：

| 工具 | 数据源 | 用途 |
|------|--------|------|
| `query_prometheus` | Prometheus | 执行即时 PromQL 查询 |
| `query_prometheus_range` | Prometheus | 执行最长 24 小时的区间 PromQL 查询 |
| `get_trace_by_id` | Jaeger | 按 TraceID 查询规范化调用链 |
| `get_observability_capabilities` | 本地配置 | 查询数据源和查询限制 |

Prometheus 工具复用 `observability.prometheus`。区间查询的 `step` 不得低于 15 秒，返回结果最多包含 100 个时间序列和 5000 个采样点。结果被裁剪时，响应中的 `truncated` 为 `true`。

Jaeger 查询需要配置 trace provider：

```yaml
observability:
  tracing:
    defaultProvider: jaeger-main
    providers:
      - name: jaeger-main
        type: jaeger
        endpoint: http://jaeger.monitoring.svc:16686
        bearerToken: "" # 可选
        tenant: ""      # 可选，通过 X-Scope-OrgID 发送
```

诊断工具不会记录完整 PromQL、Bearer Token 或完整 trace attributes。工具只读，不提供流量、配置或部署变更能力。
