# Memory 组件集成问题修复文档

## 1. 问题概述

### 1.1 现象
- Memory 组件虽然在 runtime 中被正确初始化，但 Agent 实际运行时未使用该组件
- Agent 使用的是自己创建的独立 memory context，导致记忆状态无法在组件间共享

### 1.2 影响范围
- `ai/component/agent/react/react.go` - ReActAgent 实现
- `ai/component/agent/react/component.go` - AgentComponent 初始化
- `ai/component/agent/react/factory.go` - AgentFactory 创建

---

## 2. 根因分析

### 2.1 调用链路

```
main.go
  └─ runtime.Bootstrap()
       └─ registerFactorys() 注册 memory factory
            └─ memory.MemoryFactory 创建 MemoryComponent
                 └─ MemoryComponent.Init() 初始化独立 memoryCtx
                      └─ ❌ AgentComponent 未获取该 memoryCtx

AgentFactory
  └─ NewAgentComponent()
       └─ AgentComponent.Init()
            └─ NewReactAgent()
                 └─ ❌ 创建新的独立 memoryCtx (memory.NewMemoryContext)
```

### 2.2 问题代码位置

**文件**: `ai/component/agent/react/react.go`

```go
// 第 78 行 - NewReactAgent 函数
func NewReactAgent(g *genkit.Genkit, promptBasePath string, ...) (*ReActAgent, error) {
    memoryCtx := memory.NewMemoryContext(memory.ChatHistoryKey)  // ❌ 创建独立 context
    // ...
    return &ReActAgent{
        memoryCtx: memoryCtx,  // ❌ 使用独立 context
        // ...
    }
}
```

**文件**: `ai/component/agent/react/component.go`

```go
// 第 83 行 - AgentComponent.Init 方法
func (a *AgentComponent) Init(rt *runtime.Runtime) error {
    toolsComp, err := rt.GetComponent("tools")
    // ... 处理 tools ...

    // ❌ 没有获取 memory 组件
    reactAgent, err := NewReactAgent(
        rt.GetGenkitRegistry(),
        a.promptBasePath,
        // ...
    )
    // ...
}
```

### 2.3 问题本质

1. Runtime 中初始化的 `MemoryComponent` 与 `ReActAgent` 使用的是两个独立的 `HistoryMemory` 实例
2. 两者之间没有建立连接，导致：
   - MemoryComponent 成为"僵尸"组件，未被实际使用
   - 每个 ReActAgent 实例都有独立的内存，无法共享
   - 无法通过 runtime 访问和管理局部的记忆状态

---

## 3. 解决方案

### 3.1 设计原则

1. **最小侵入性** - 尽量少改动现有代码
2. **与现有模式一致** - 参考 tools 组件的获取方式
3. **保持接口稳定** - 避免影响外部调用

### 3.2 实施方案

在 `AgentComponent.Init()` 中从 runtime 获取 memory 组件，并将其 context 传递给 `NewReactAgent`。

### 3.3 改动清单

| 文件 | 改动内容 |
|------|----------|
| `component/agent/react/component.go` | Init 方法增加获取 memory 组件的逻辑 |
| `component/agent/react/react.go` | NewReactAgent 增加 memoryCtx 参数 |
| `component/agent/react/factory.go` | 无需改动（参数通过 Init 传递） |

---

## 4. 代码修改

### 4.1 修改 `component.go`

**文件**: `ai/component/agent/react/component.go`

```go
func (a *AgentComponent) Init(rt *runtime.Runtime) error {
    // 获取 tools 组件（现有代码）
    toolsComp, err := rt.GetComponent("tools")
    if err != nil {
        return fmt.Errorf("tools component not found: %w", err)
    }
    tools, ok := toolsComp.(*tools.ToolsComponent)
    if !ok {
        return fmt.Errorf("invalid tools component type")
    }
    toolRefs := tools.GetToolRefs()

    // === 新增：获取 memory 组件 ===
    memoryComp, err := rt.GetComponent("memory")
    if err != nil {
        return fmt.Errorf("memory component not found: %w", err)
    }
    memComponent, ok := memoryComp.(*memory.MemoryComponent)
    if !ok {
        return fmt.Errorf("invalid memory component type")
    }
    memoryCtx := memComponent.GetContext()
    // ============================

    // 创建 ReActAgent 时传入 memoryCtx
    reactAgent, err := NewReactAgent(
        rt.GetGenkitRegistry(),
        memoryCtx,  // 新增参数
        a.promptBasePath,
        a.model,
        a.maxIterations,
        a.stages,
        toolRefs,
    )
    if err != nil {
        return fmt.Errorf("failed to create ReAct agent: %w", err)
    }

    a.Agent = reactAgent

    rt.GetLogger().Info("Agent component initialized",
        "agent_type", a.agentType,
        "model", a.model,
        "max_iterations", a.maxIterations,
        "stages", len(a.stages))

    return nil
}
```

### 4.2 修改 `react.go`

**文件**: `ai/component/agent/react/react.go`

```go
// NewReactAgent 创建一个新的 ReActAgent 实例
func NewReactAgent(
    g *genkit.Genkit,
    memoryCtx context.Context,  // 新增参数：从外部传入 memory context
    promptBasePath string,
    defaultModel string,
    maxIterations int,
    stagesCfg []StageInfo,
    toolRefs []ai.ToolRef,
) (*ReActAgent, error) {
    // 删除：memoryCtx := memory.NewMemoryContext(memory.ChatHistoryKey)
    // 直接使用传入的 memoryCtx

    channels := agent.NewChannels(len(stagesCfg))
    stages, err := buildStagesFromConfig(g, stagesCfg, promptBasePath, defaultModel, toolRefs)
    if err != nil {
        return nil, err
    }

    return &ReActAgent{
        registry:       g,
        orchestrator:   agent.NewOrderOrchestrator(maxIterations, stages...),
        memoryCtx:      memoryCtx,  // 使用传入的 context
        channels:       channels,
        defaultModel:   defaultModel,
        promptBasePath: promptBasePath,
        maxIterations:  maxIterations,
    }, nil
}
```

### 4.3 无需修改的文件

**`factory.go`** - 无需改动，因为 `NewAgentComponent` 不涉及具体参数传递，参数在 `Init` 阶段注入。

---

## 5. 验证方法

### 5.1 单元测试

运行现有的单元测试，确保未破坏现有功能：

```bash
cd ai
go test ./component/agent/react/test/... -v
```

### 5.2 集成测试

验证 memory 组件与 agent 的集成：

```go
// 验证 agent 使用的 memory 与 runtime 中的 memory 是同一个实例
memoryComp, _ := rt.GetComponent("memory")
memCtx := memoryComp.(*memory.MemoryComponent).GetContext()

agentComp, _ := rt.GetComponent("agent")
agent := agentComp.(*react.AgentComponent).Agent

// 两者应该指向同一个 HistoryMemory
assert.Equal(t, memCtx, agent.GetMemory())
```

### 5.3 功能验证

1. 启动 agent 服务，发送多轮对话
2. 验证 agent 能正确记忆上下文
3. 通过 runtime 获取 memory 组件，检查历史记录是否正确保存

---

## 6. 风险评估

| 风险项 | 影响 | 缓解措施 |
|--------|------|----------|
| 接口变更影响其他调用者 | 低 | factory.go 无需改动，外部调用不受影响 |
| memory 组件未正确初始化 | 中 | 添加错误处理和日志记录 |
| 现有测试失败 | 低 | 现有测试创建独立 context，不受影响 |

---

## 7. 后续优化建议

1. **统一依赖注入模式** - 考虑让所有组件依赖都通过 runtime 注入，而不是在内部创建
2. **添加组件健康检查** - 在 runtime 启动时验证组件间的依赖关系
3. **Memory 组件抽象** - 考虑引入 Memory 接口，支持不同的 memory 实现

---

## 8. 变更历史

| 日期 | 版本 | 描述 | 作者 |
|------|------|------|------|
| 2025-04-22 | 1.0 | 初始版本 | Claude |
| 2025-04-22 | 1.1 | 实施完成：修改 component.go 和 react.go | Claude |
| 2025-04-22 | 1.2 | 优化存储策略：只存用户问题和最终结果 + 修复 WithValue 问题 | Claude |

## 9. 实施记录

### 9.1 已修改文件

| 文件 | 修改内容 |
|------|----------|
| `ai/component/agent/react/component.go` | 1. 新增 memory 包导入<br>2. Init 方法增加获取 memory 组件的逻辑<br>3. 将 memoryCtx 传递给 NewReactAgent |
| `ai/component/agent/react/react.go` | 1. NewReactAgent 签名增加 memoryCtx 参数<br>2. 删除内部创建 memoryCtx 的代码<br>3. **删除 ThinkFlow 的 AddHistory**（不存储思考过程）<br>4. **删除 ActFlow 的 AddHistory**（不存储工具调用）<br>5. **修复 WithValue 问题**（使用局部 ctx 而非覆盖 ra.memoryCtx） |

### 9.2 存储策略变更

**变更前**：存储所有消息（用户输入 + Think + Act + Observe）
- 每个 Turn 约 4 条消息
- 10 Turn = ~40 条消息全传给 LLM
- 包含大量中间推理过程

**变更后**：只存储用户输入和最终结果
- 每个 Turn 约 2 条消息（用户输入 + Observe 最终结果）
- 10 Turn = ~20 条消息
- 减少 Token 消耗约 50%

### 9.3 测试验证

- 编译通过：`go build ./...`
- Agent 测试通过：`go test ./component/agent/react/test/...`
- Memory 测试通过：`go test ./component/memory/test/...`
