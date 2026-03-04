# TODO 清单（全项目）

1. `component/memory/component.go:27`
   - 简介：`MemoryComponent` 目前直接依赖 `HistoryMemory`，计划注入统一 Memory 接口，支持不同内存实现。

2. `component/memory/component.go:85`
   - 简介：`GetMemory` 目前返回具体 `HistoryMemory`，计划抽象为统一接口以兼容 `VectorMemory` 等类型。

3. `component/rag/reranker.go:103`
   - 简介：Reranker 的 API Key 读取仍在运行路径中，计划迁移到组件初始化阶段，并支持多 reranker 类型使用不同密钥。

4. `component/models/component.go:88`
   - 简介：由于 `genkit.Init` 只能调用一次，当前通过集中追加插件处理；后续需重构为更灵活的插件装配机制。

5. `component/rag/indexer.go:98`
   - 简介：`Store` 中的 `namespace` 尚未做多租户相关校验，后续需补充命名空间验证策略。

6. `schema/react.go:171`
   - 简介：流式反馈 `index` 为包级全局计数器，存在并发安全和会话串扰风险；需改为会话/流级状态并加同步保护。

7. `component/tools/engine/mcp_tools.go:13`
   - 简介：`MCPToolManager` 缺少网络不稳定场景下的刷新/重连生命周期能力。

8. `component/tools/engine/mcp_tools.go:78`
   - 简介：`GetActiveTools` 启动流程缺少断连恢复路径（重连 + 刷新工具 + 重试引导）。

