# Issue #1539 MVP PR 4：记忆用户入口与验收

实现基线为 PR 3 分支 `feat/ai-1539-explicit-memory` 的 `56939dc`，功能分支为
`feat/ai-1539-memory-user-controls`。本记录覆盖核心 MVP 第四个 PR，不代表完整
G2/G3、Issue 评论中的全部要求或生产发布门已通过。

## 用户入口

在现有 AI 聊天抽屉中打开“我的记忆”，可分页查看、新增、编辑、删除、清空和用户级
启停记忆。新增/编辑点击“确认并保存”即显式确认；删除、清空、启停有独立确认。
禁用保留记录并禁止保存；仍可查看、删除、清空。列表展示来源会话（若存在）、适用
项目/任务、有效期和过期状态。过期记录仍占配额，需显式删除或清空。

表单使用业务类别，隐藏 canonical key、scope ID 和内部版本；scope 固定为 user。

| 业务类别 | Memory 类型及事实 | 适用条件 |
| --- | --- | --- |
| 回答语言/风格 | preference / response.language、response.style | 通用，无项目/任务 |
| 项目背景 | task_context / project.background | 项目必选，任务可选 |
| 诊断优先事项 | diagnostic_constraint / diagnostic.priority | diagnosis，项目可选 |
| 只读诊断 | diagnostic_constraint / diagnostic.read_only | diagnosis，值固定为 true，项目可选 |

只读诊断是上下文偏好，不能代替服务端工具授权。表单保留原记录来源会话，不修改其
不可变来源关系。普通面板新建记录没有来源会话，不会因当前聊天会话删除而被删除。

每次写入使用列表返回的 generation/state_version，编辑和删除还携带记录 version。
冲突不自动更新版本重试：保留草稿，刷新列表，由用户关闭编辑器并检查最新记录后
再次确认。不同事实身份冲突也不会静默覆盖。

聊天项目和任务只用于下一条新消息；发送、新建会话、恢复会话或关闭抽屉后清空。
失败重试保留原匹配条件的独立快照和原 Idempotency-Key，不读取之后的新选择。
条件只决定 Memory 适用性，不构成项目资源/工具授权。

删除/清空/禁用阻断后续有效记忆及可追溯依赖注入，不清除独立用户陈述、已发送给模型
的副本或过去的幂等响应。规范聊天历史仍需通过删除会话清理。

## API 与部署

新增 `GET /api/v1/ai/memory-options`，使用现有 AI 认证；从 `memory_projects`
返回当前 tenant/user 可用项目的稳定 ID 和允许的任务类型。空目录返回空项目列表，
不会回退到任意输入。未知项目仍由保存和聊天接口再次拒绝。

自定义 MatchContextValidator 可同时实现 MemoryOptionsProvider；未实现时返回
`MEMORY_OPTIONS_UNAVAILABLE`，不暴露私有目录。响应不含租户、成员名单。

聊天适配器改发顶层 `matchContext`。原页面快照 `context` 不符合严格 ChatRequest
契约，因此本 MVP 聊天入口不再显示页面携带控件，也不将快照发送给该接口。
页面采集模块仍保留供后续页面 Source 接入，当前不宣称已完成页面上下文能力。
SSE 协议保持不变。

AI 请求统一经 `aiTransport` 注入 Admin 签发的 Bearer token，并在 `WWW-Authenticate`
Bearer challenge 时刷新一次；`accessTokenEnabled` 关闭时保持匿名兼容。聊天与 Memory
共用该传输层，部署仍须通过既有可信认证代理或网关为 AI 服务提供其配置的凭证。
本 PR 不将普通浏览器 cookie、用户名或项目选择转换为可信身份，也不新增生产登录方案。

注入配置沿用 PR 3 的 `explicit_memory.enabled` 和 PR 2 的
`conversation_summary.enabled`，均默认关闭。管理 API 在注入开关关闭时仍可用。
模型窗口、输出预留、摘要恢复上限、Store MaxTurns 和 Memory 配额沿用对应文档；
存储 Turn 配额应高于长会话验收轮数。Memory/GORM 存储语义与迁移均未改变。

## 可重复验证

安装 `ui-vue3/yarn.lock` 中的依赖及 Cypress 浏览器。浏览器脚本自动启动 Go 测试服务、
Vite 和 Cypress，并在退出时关闭服务，需要本机端口 8880、8881 可用：

```sh
cd ui-vue3
yarn install --frozen-lockfile
yarn cypress install
yarn test:e2e:mvp
```

Linux 无显示服务时安装 Cypress 所需的 Xvfb；已有桌面显示服务可显式设置 DISPLAY。
桌面编辑器导出的 ELECTRON_RUN_AS_NODE 由测试脚本对子进程清除。

浏览器使用独立的 `e2e/mvp.html` 开发入口，实际挂载 AgentDrawer、Ant Design 和 i18n，
不经过无关 Console 页面或生产路由。Go fixture 仅在测试二进制中、显式设置
AI_MVP_BROWSER=1 时监听 loopback；登录、诊断、控制及停止接口不会进入生产构建。
认证使用实际 JWT 校验器；MVP 入口与 `main.ts` 一样先读取 `/auth/providers` 启用
`accessTokenManager`，fixture 再按 Admin 同形接口由 `/auth/token` 签发 Bearer token，
浏览器内的聊天与 Memory 请求不再依赖测试 cookie。这验证 token 发现、签发与校验链路，
不代替生产 OIDC/代理部署的 G1 验收。

浏览器不替换 Memory、聊天或 Provider 响应；cy.intercept 仅观察请求。使用真实 HTTP
路由、Store、ReAct、Compiler、SDK 转换及本地可控 Provider，检查其收到的最终输入。

| 验收 | 证据 |
| --- | --- |
| 模板变量及消息角色 | 服务端变量改变实际 system；用户模板语法保留为 user 数据；必填变量缺失时 Provider 调用数不增加 |
| 摘要 | 长会话覆盖游标推进、早期目标保留且不重复注入；连续非法输出不推进；恢复后追上目标；关闭摘要仍保留最近历史 |
| 预算回退 | 生成合法大摘要后降低主调用预算，确认摘要被排除而最近完整 Turn 仍在最终 Provider 请求中 |
| 跨会话 Memory | 通过 UI 保存/编辑偏好，实际请求使用新值；删除后新会话及恢复的原会话不注入旧值，规范历史保留 |
| 项目/任务 | 项目 A/B 不互相覆盖，诊断排除巡检专属背景，巡检显式选择后召回；新消息不继承项目 |
| 用户控制与隔离 | 用户级启停与清空确认；禁用保留列表；另一用户/租户列表为空且不能使用原项目 |
| 错误与交互 | DOM 表单提交、版本冲突保留草稿、分页、来源/有效期、内容转义、取消确认、选项失败清空旧选择有 Vitest 覆盖 |

其他验证命令：

```sh
cd ai
go test -count=1 ./...
go test -count=1 -race ./component/server/engine ./component/agent/react

cd ../ui-vue3
yarn test --maxWorkers 2 --minWorkers 1
yarn type-check:mvp
yarn eslint src/components/ai-memory src/api/service/aiMemory.ts src/api/service/ai.ts src/components/AgentDrawer.vue src/components/ai-chat
yarn vite build
```

直接执行 vite build 避免现有 yarn build 对整个 src 执行 Prettier 的额外格式化。
专项 tsconfig 只覆盖该功能及其依赖，不替代全项目类型检查。当前全项目类型检查仍有
未改动的 ConfigPage、GrafanaPage、home 和资源页面等既有错误。

2026-10-26 本地验证结果：

- Go 全量测试通过；engine/react 的 race 检查通过。
- Vitest 14 个文件、64 个测试通过；Cypress 5 个真实 HTTP/Provider 场景全部通过。
- PR4 专项类型检查、相关组件/API ESLint、相关新增文件 Prettier 和 diff 空白检查通过。
- Vite 生产构建通过，仍提示已有的 updateInstanceTrafficSwitch 导出缺失及大 chunk 警告。
- 全项目类型检查未通过，原因是上述既有页面错误；专项检查不将其隐藏为全项目通过。
- 交互持久化写入使用独立的短时上下文，不会继承模型/工具执行阶段的 30 秒期限；Memory/GORM 共享契约验证了记忆自身过期记录仍可在管理列表中查看，而来源会话失效的记录和召回均继续阻断读取。

## 里程碑边界与后续

PR 1 的模板/Compiler/工具协议、PR 2 的摘要、PR 3 的 Memory 后端和本 PR 的用户入口
共同形成核心三项能力的可重复 fixture 闭环。两种 Store 的底层契约、重启、CAS、
清理及依赖传播继续由 Go 契约/集成测试覆盖，浏览器 fixture 使用内存后端。

仍需独立完成生产身份代理/G1 发布证据、MySQL/PostgreSQL 实际部署并发验证、真实模型
质量与 tokenizer 误差评估。未设置 AI_TEST_MYSQL_DSN 时 MySQL 专项测试会跳过。
本 PR 不新增自动记忆提取、共享/资源 scope、语义召回、动态模板平台或页面 Source。
完整 G2/G3 和 Issue 全量需求继续按总方案跟踪，不以本 PR 默认关闭整个 Issue。
