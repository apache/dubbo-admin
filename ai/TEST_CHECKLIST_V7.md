# AI 模块单元测试 TODO LIST

> 版本：v7（三层校验重构统一版：Parser / Loader+Schema / Runtime / Component）

## 快速开始

```bash
go test -race -v ./config/... ./runtime/... ./component/... ./test/... ./cmd/...
```

---

## Tier 1：Parser（语法层，4个）

### Config Parser

**文件**: `config/loader_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 1 | `TestLoader_MainConfig_ParseError` | 无 | 主配置 YAML 语法错误 | 非法 `config.yaml`（如缺失冒号） | 返回 `parse error`，不进入结构校验 |
| 2 | `TestLoader_Component_ParseError` | 无 | 组件配置 YAML 语法错误 | 非法组件 YAML | 返回 `parse error`，不进入结构校验 |
| 3 | `TestLoader_MainConfig_ParseError_Priority` | 无 | 语法错误优先级 | 同时包含结构问题与语法错误的主配置 | 优先返回 `parse error` |
| 4 | `TestLoader_Component_ParseError_Priority` | 无 | 语法错误优先级 | 同时包含结构问题与语法错误的组件配置 | 优先返回 `parse error` |

---

## Tier 2：Loader + Schema（结构层，12个）

### Main Config Structural

**文件**: `config/loader_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 5 | `TestLoader_MainConfig_UnknownField` | 无 | 主配置 unknown field 拒绝 | `config.yaml` 含未定义字段 | 返回 `structural error` |
| 6 | `TestLoader_MainConfig_ComponentsTypeInvalid` | 无 | components 项类型约束 | `components.x` 为对象/数字 | 返回 `structural error`（仅允许 string/[]string） |
| 7 | `TestLoader_MainConfig_ComponentsArrayItemInvalid` | 无 | components 数组项类型约束 | `components.x` 为 `["a.yaml", 1]` | 返回 `structural error` |
| 8 | `TestLoader_MainConfig_DefaultSchemaDir` | 无 | SCHEMA_DIR 默认路径 | 环境变量未设置 | 默认使用 `schema/json` 成功加载 |

### Component Structural

**文件**: `config/loader_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 9 | `TestLoader_Component_MissingType` | 无 | 组件 type 必填 | 组件 YAML 缺失 `type` | 返回 `structural error` |
| 10 | `TestLoader_Component_MissingSpec` | 无 | 组件 spec 必填 | 组件 YAML 缺失 `spec` | 返回 `structural error` |
| 11 | `TestLoader_Component_UnknownTopField` | 无 | top-level unknown field 拒绝 | 组件 YAML 含未定义字段 | 返回 `structural error` |
| 12 | `TestLoader_Component_DefaultInjection_Server` | 无 | server 默认值注入 | 仅给最小 server 配置 | decode 后包含 port/host/timeout 默认值 |
| 13 | `TestLoader_Component_DefaultInjection_Agent` | 无 | agent 阶段默认值注入 | stage 省略 temperature/top_p 等 | decode 后包含默认值 |

### Conditional Required

**文件**: `config/loader_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 14 | `TestLoader_Tools_MCPEnabled_RequireHost` | 无 | 条件必填（tools） | `enable_mcp_tools=true` 且缺 `mcp_host_name` | 返回 `structural error` |
| 15 | `TestLoader_RAG_RerankerEnabled_RequireAPIKey` | 无 | 条件必填（rag） | `reranker.enabled=true` 且缺 `api_key` | 返回 `structural error` |
| 16 | `TestLoader_RAG_Splitter_OneOfBranchValidation` | 无 | oneOf 分支结构约束 | splitter spec 与 type 不匹配 | 返回 `structural error` |

---

## Tier 3：Runtime 调度层（6个）

### Runtime Orchestration

**文件**: `runtime/runtime_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 17 | `TestRuntime_RegisterFactory_Duplicate` | 无 | 重复注册覆盖 | 同类型注册两次 | 第二次覆盖生效；包含重复注册提示 |
| 18 | `TestRuntime_RegisterFactory_Concurrent` | 无 | 并发注册安全 | 100 goroutine 并发注册 | 无 data race；数量正确 |
| 19 | `TestRuntime_GetFactoryFn_NotFound` | 无 | 未注册工厂 | 类型名 `test` | 返回 error，包含 `not registered` |
| 20 | `TestRuntime_GetComponent_NotFound` | 无 | 未注册组件 | 名称 `agent` | 返回 error，包含 `component not found` |
| 21 | `TestRuntime_ComponentInitOrder` | Stub Component | 初始化顺序 | 注册多个工厂 | 按 `factoryOrder` 顺序 `Validate -> Init` |
| 22 | `TestBootstrap_ValidateFailStopsInit` | Stub Component | 语义失败中断 | 某组件 `Validate()` 返回 error | `Bootstrap` 返回 `failed to validate <name>`，不执行 Init |

---

## Tier 4：Component 语义层（8个）

### Component Validate

**文件**: `component/*/test/*_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 23 | `TestServerComponent_Validate_PortRange` | 无 | server 端口语义 | `port=70000` | `Validate()` 返回 error |
| 24 | `TestServerComponent_Validate_TimeoutPositive` | 无 | server 超时语义 | `read_timeout<=0` 或 `write_timeout<=0` | `Validate()` 返回 error |
| 25 | `TestMemoryComponent_Validate_MaxTurns` | 无 | memory 轮次语义 | `max_turns<=0` | `Validate()` 返回 error |
| 26 | `TestToolsComponent_Validate_MCPConfig` | 无 | tools MCP 语义 | MCP enabled 且 host 为空 | `Validate()` 返回 error |
| 27 | `TestModelsComponent_Validate_Providers` | 无 | models 语义一致性 | providers 为空或 base_url 为空 | `Validate()` 返回 error |
| 28 | `TestRAGComponent_Validate_SplitterSemantic` | 无 | rag 分块语义 | `overlap_size >= chunk_size` | `Validate()` 返回 error |
| 29 | `TestAgentComponent_Validate_StageFlowType` | 无 | agent 阶段语义 | 非法 `flow_type` | `Validate()` 返回 error |
| 30 | `TestAgentComponent_Validate_StagePromptRequired` | 无 | agent 阶段语义 | 缺 `prompt_file` | `Validate()` 返回 error |

---

## Tier 5：Business Workflows（业务流程保留项，8个）

### RAG Workflow

**文件**: `component/rag/test/workflow_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 31 | `TestRAGWorkflow_Index_Retrieve` | Mock Retriever/Indexer | 索引后可检索 | 文档 `Dubbo is RPC`，查询 `RPC` | 检索结果包含 `Dubbo` |
| 32 | `TestRAGWorkflow_Split_Index` | Mock Splitter/Indexer | 分块后索引 | 长文档 | `len(chunks)>1` 且全部进入索引 |
| 33 | `TestRAGWorkflow_Namespace` | Mock Retriever | 命名空间隔离 | ns1/ns2 各索引文档 | ns1 查询不返回 ns2 内容 |
| 34 | `TestRAG_Retrieve_EmptyQuery` | Mock Retriever | 空查询处理 | `queries=nil` | 返回空 map（非nil），无 error |

### Agent Workflow

**文件**: `component/agent/react/test/flow_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 35 | `TestActFlow_GeneralInquiry_NoTools` | Mock Prompt | 一般询问不调工具 | `Intent=GeneralInquiry` | 返回 `ToolOutputs`，`len(Outputs)=0` |
| 36 | `TestActFlow_WithToolCall_ReturnsToolOutputs` | Mock Prompt+Tool | 工具调用主流程 | `Intent=PerformanceInvestigation` | 返回 `ToolOutputs`，至少1条结果 |
| 37 | `TestActFlow_ToolErrorHandling` | Mock Prompt/工具 | 工具错误处理 | 工具返回 error | 返回 error，包含工具名 |
| 38 | `TestThinkFlow_ExecuteError_NoNilDeref` | Mock Prompt | think 异常路径健壮性 | `Execute` 返回 error | 不应因 `resp.Text()` 引发 nil deref |

---

## Tier 6：并发与边界（保留项，8个）

### Memory Concurrency & State

**文件**: `component/memory/test/history_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 39 | `TestHistoryMemory_AddHistory_UserMessage` | 无 | 添加用户消息 | session-1 + user message | 进入当前 turn 的 `UserMessages` |
| 40 | `TestHistoryMemory_NextTurn_ArchivesCurrentTurn` | 无 | 推进会话归档当前 turn | 已有1个turn | 旧 turn 进入 history，window 前移 |
| 41 | `TestHistoryMemory_NextTurn_WhenSessionFull` | 无 | 会话窗口满时行为 | 将窗口填满后 `NextTurn` | 返回 error，包含 `context is full` |
| 42 | `TestHistoryMemory_ConcurrentAddHistory` | 无 | 并发写历史安全 | 100 goroutine 写入 | 无 panic，无 race |
| 43 | `TestHistoryMemory_ConcurrentReadWrite` | 无 | 并发读写安全 | 10写+10读 goroutine | 不 panic，无 data race |
| 44 | `TestHistoryMemory_NextTurn_EmptyWindowSafety` | 无 | 空窗口推进安全 | session 被 pop 空后重复 `NextTurn` | 固定当前 panic 风险或修复后断言 error |

### Runtime/Bootstrap Boundary

**文件**: `runtime/runtime_test.go`, `component/*/test/*_test.go`

| # | 测试名称 | Mock | 说明 | 输入 | 预期输出 |
|---|---------|------|------|------|---------|
| 45 | `TestBootstrap_MissingFactoryForConfiguredType` | 无 | 配置类型无工厂 | 配置 type 未注册 | error 包含 `no factory for` |
| 46 | `TestRuntime_GetRuntime_NotInitialized` | 无 | 全局Runtime未初始化 | 直接调用 `GetRuntime()` | 触发 panic `Runtime not initialized` |

---

## 按层统计

| 层/域 | 数量 | 说明 |
|------|------|------|
| Parser | 4 | 只验证语法失败归属 |
| Loader+Schema | 12 | 只验证结构校验、条件必填、默认注入 |
| Runtime | 6 | 只验证调度顺序与生命周期 |
| Component Semantic | 8 | 只验证语义规则 |
| Business Workflows | 8 | 保留原有高价值业务流程测试 |
| Robustness & Concurrency | 8 | 保留原有并发与边界保护测试 |
| **总计** | **46** | 三层重构 + 原有单测统一清单 |

---

## 编写规范

- 错误断言按层级关键字匹配：
  - 语法层：`parse error`
  - 结构层：`structural error`
  - 语义层：`failed to validate` 或组件语义错误信息
- 业务流程测试不承担结构层职责断言。
- 并发测试建议配合 `-race` 在 CI 中执行。

---

## Mock 说明

| Mock对象 | 用途 | 实现方式 |
|---------|------|---------|
| Mock Genkit Registry | 模拟模型注册与组件初始化 | 使用 `testutils.CreateMockGenkitRegistry()` 或等价 stub |
| Mock Prompt | 模拟 LLM 返回（含 ToolRequests） | 自定义 `MockPrompt` / stub prompt，覆盖成功与失败分支 |
| Mock Tool | 模拟工具调用成功/失败/超时 | 本地 mock struct + 可配置返回值 |
| Mock Retriever/Indexer/Splitter | 控制 RAG 行为并断言参数 | 本地 mock struct + 调用计数/入参记录 |
| Stub Component | 验证 Runtime 初始化顺序/错误传播 | 自定义实现 `runtime.Component`，可注入 `Validate/Init/Start/Stop` 行为 |
| 临时配置文件夹（Fixture Dir） | 隔离配置输入与路径解析 | `t.TempDir()` 下写入最小 YAML/Schema 夹具 |

---

## Go 表格驱动单测编写规范

### 1) 基本模板

```go
func TestXXX(t *testing.T) {
    tests := []struct {
        name    string
        input   any
        wantErr bool
        errLike string
    }{
        {
            name:    "valid case",
            input:   ...,
            wantErr: false,
        },
        {
            name:    "invalid case",
            input:   ...,
            wantErr: true,
            errLike: "structural error",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // arrange
            // act
            // assert
        })
    }
}
```

### 2) 断言规范

- 错误断言使用关键子串匹配，避免脆弱的全量字符串匹配。  
- 每条测试只验证一个主行为（单一断言目标）。  
- 分层测试中禁止跨层断言：  
  - Parser 测试不验证语义  
  - Loader 测试不验证组件语义  
  - Component 测试假设结构已通过

### 3) 并发与稳定性

- 并发测试统一使用 `go test -race` 验证。  
- 避免真实网络/端口依赖，全部使用本地 mock。  
- 使用 `t.Helper()` 封装重复构建逻辑，提高可读性。

### 4) 夹具与命名

- 测试名采用 `Test<模块>_<行为>_<预期>` 风格。  
- 配置夹具优先最小化：只保留触发当前断言所需字段。  
- 使用 `t.TempDir()` 管理临时文件，避免污染仓库。

### 5) 回归要求

- 每个缺陷修复至少补 1 条失败重现用例。  
- 先写失败断言，再修复实现（红-绿流程）。  
