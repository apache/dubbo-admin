# AI Agent 可观测性与 Langfuse / OTel GenAI 采集端配合

> 目标：**不翻 console 日志，就能看清 Agent 是怎么跑的**——它怎么发起调用、调了哪些 tools、思考过程是什么、输入输出长什么样。
>
> 面向两类读者：① 排查 Agent 行为的开发者；② 接入统一可观测平台的运维。

---

## 1. 为什么需要它

Dubbo Admin 的 AI 助手目前是一个黑盒。它跑了一次会话，中间调了哪些工具、模型返回了什么、有没有走降级、思考过程是什么，**只能去看 Admin 的 console 日志**。而 console 日志有两个根本缺陷：

- **没有会话维度**：一次会话的第 3 轮和第 40 轮混在同一条时间线里，无法还原「这一轮 Agent 到底拿到了什么上下文」；
- **不记录工具调用的返回值**：Agent 拿到一段格式错误的 JSON 却继续往下走，日志里看不出任何异常；
- **不是因果结构**：日志是时间序的文本流，看不出「哪个工具调用是哪个模型调用发起的」。

OpenTelemetry 的 generative AI 语义约定（semantic conventions for generative AI systems）给了这个问题一个标准答案：**把 Agent 的执行过程表达成符合规范的 trace**，导出到 OTLP，任何符合 OTel 的后端（Langfuse、OpenLIT、Phoenix、SigNoz，或企业自建的统一收集平台）都能直接消费，**不需要写任何一行适配代码**。

本文件说明这套东西怎么开、开了之后能看到什么、以及各后端字段是怎么对应起来的。

---

## 2. 采集链路

```text
  Dubbo Admin AI 助手
        │
        │  ai/component/hooks —— 观察 Agent 生命周期事件
        │  interaction / iteration / stage / model_call / tool_call
        ▼
  OTel SDK（TracerProvider + BatchSpanProcessor）
        │  OTLP/gRPC (4317) 或 OTLP/HTTP (4318)
        ▼
  企业统一收集平台（OpenTelemetry Collector）
        │
        ├──────────────▶ Langfuse   —— 会话 / 模型调用 / 工具调用 / token / 成本
        ├──────────────▶ OpenLIT   —— 链路树 / 对话还原 / Agent Loop / 治理视图
        └──────────────▶ Greptime / Tempo / 其它后端（透传原始 span）
```

Agent 侧的接入点在 `ai/component/hooks`：它订阅 Agent 的生命周期事件，把每个事件表达成一个 span，并按 OTel GenAI 约定打属性。

---

## 3. 怎么开

### 3.1 配置

`ai/component/hooks/hooks.yaml`：

```yaml
hooks:
  - name: otel-tracing
    type: tracing
    enabled: true
    config:
      protocol: grpc                       # grpc | http/protobuf
      service_name: dubbo-admin-ai        # 后端里显示的服务名
      sample_ratio: 1.0                   # 开发/测试保持 1.0
      capture_content: none                # none | truncated | full
      capture_reasoning: none             # none | truncated | full（新增）
```

导出目标由标准的 OpenTelemetry 环境变量决定，无需改配置：

```shell
export OTEL_EXPORTER_OTLP_ENDPOINT=http://collector:4317
# 或
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://langfuse:3000/api/public/otel
export OTEL_EXPORTER_OTLP_HEADERS="Authorization=Basic <base64(publicKey:secretKey)>"
```

> Langfuse 的 OTLP 入口是 `http://<langfuse>/api/public/otel`，鉴权用 `Authorization: Basic base64(publicKey:secretKey)`。

### 3.2 两个采集开关是**正交**的

| `capture_content` | `capture_reasoning` | 效果 |
| --- | --- | --- |
| `none` | 任意 | **完全不发**任何内容属性（提示词、工具入参/返回、思考过程都没有） |
| `truncated` / `full` | `none` | 发消息内容，但**不含思考过程** |
| `truncated` / `full` | `truncated` | 发消息内容，思考过程**截断**保留 |
| `truncated` / `full` | `full` | 发消息内容，思考过程**原样**保留 |

**为什么拆成两个开关**：它们是两类不同的风险。提示词与工具入参/返回值是**业务数据**（可能含用户标识、订单信息）；模型思考过程是另一类内容，敏感度与用途都不同。合成一个开关，就没法做到「开发期要盯思考过程、生产期要把它藏起来」的同时不误开业务数据。

**建议**：

| 环境 | `capture_content` | `capture_reasoning` |
| --- | --- | --- |
| 本地开发 / 调试 | `truncated` 或 `full` | `truncated` 或 `full` |
| 测试环境 | `truncated` | `none` |
| 生产环境 | `none`（默认） | `none`（默认） |

默认两个开关都是 `none`——**默认情况下链路里不存在任何明文提示词**，这是刻意的隐私保护。

### 3.3 思考过程的内容从哪来

思考过程不是新造的属性。规范里它就装在消息的 part 里：

```json
{"role": "assistant",
 "parts": [{"type": "reasoning", "content": "先按服务名和时间窗检索候选链路……"},
           {"type": "text", "content": "检索到 3 条含错误 span 的链路……"}]}
```

`gen_ai.input.messages` 与 `gen_ai.output.messages` 的 schema 都定义了 `ReasoningPart`（`type: "reasoning"`）。此前 Agent 在把模型响应转换成消息时**只处理 text / tool_call / tool_call_response 三种 part，thinking 内容被直接丢弃**——这也是「看不到思考过程」的直接原因。本次补上了 reasoning part 的转换，并由 `capture_reasoning` 策略决定是否保留、是否截断。

---

## 4. 属性与后端字段的对应关系

这是「**不需要写适配代码**」的关键——我们发的是规范属性，后端按规范自己认。

| GenAI 属性 | 含义 | Langfuse | OpenLIT |
| --- | --- | --- | --- |
| `gen_ai.operation.name` | 操作类型（`chat` / `execute_tool` / `invoke_agent`） | 观察类型 | span 分类 |
| `gen_ai.provider.name` | 提供方（`openai` / `gcp.gemini` / `deepseek` …） | Provider | Provider 维度 |
| `gen_ai.request.model` | 请求的模型 | Model | 模型维度 |
| `gen_ai.response.model` | 实际响应的模型 | 实际模型 | 「模型被替换」检测 |
| `gen_ai.conversation.id` | 会话标识 | **Session ID** | 会话维度 |
| `gen_ai.agent.name` | Agent 身份 | Agent | **Coding Agent** |
| `gen_ai.tool.name` / `gen_ai.tool.call.id` | 工具与调用标识 | Tool call | 工具调用 |
| `gen_ai.tool.call.arguments` / `.result` | 工具入参与返回 | 工具入参/返回 | 工具详情 |
| `gen_ai.input.messages` / `gen_ai.output.messages` | 消息内容 | Input / Output | 对话还原 |
| `gen_ai.usage.input_tokens` / `.output_tokens` | token 用量 | Usage / 成本 | Token 统计 |
| `error.type` | 低基数错误分类 | 错误标签 | 异常聚合 |
| `gen_ai.client.operation.exception`（事件） | GenAI 异常事件 | 异常记录 | Exceptions 视图 |

> **异常事件名跟着规范版本走。** 本实现钉在 semconv-genai 的 `6fd0d76`（2026-10-09），该版本里 GenAI 异常只有 `gen_ai.client.operation.exception` 一个事件名，**推理异常用的也是它**。规范主干在 2026-11-05 之后把异常事件拆成了 `gen_ai.client.inference.exception` / `gen_ai.execute_tool.exception` 等，并把 `gen_ai.client.operation.exception` 收窄为「推理、嵌入、agent 调用、拉取响应之外」的场景。那次拆分**尚未发布**，推理侧还被标为 breaking，本实现暂不跟进（见第 8 节）；升级规范版本时需重跑对齐检查。
>
> 异常事件**不记 `exception.stacktrace`**：hook 只看得到错误消息，在 hook 边界采到的栈对每一次失败都一样，说不出失败发生在哪里。规范里该属性是 Recommended，缺失不违规；分类信息由 `exception.type` 与 span 状态承担。

**span 命名**也遵循规范，所以后端能直接把它读成 Agent 结构：

| span 名 | 规范依据 |
| --- | --- |
| `invoke_agent {gen_ai.agent.name}` | `gen_ai.invoke_agent.internal` |
| `chat {gen_ai.request.model}` | `gen_ai.client.inference` |
| `execute_tool {gen_ai.tool.name}` | `gen_ai.execute_tool.internal` |

`gen_ai.agent.name` 取稳定常量 `dubbo-admin-sre-agent`，而不是每次运行的值——否则 span 在链路里就是匿名的。

`gen_ai.input.messages` / `gen_ai.output.messages` 不只出现在模型 span 上，**`invoke_agent` 这条 agent span 也带**——它装的是这次交互的完整对话和最终回答。多数后端（Langfuse 就是）把 trace 的 Input / Output 从 trace 根 span 上取，所以这条属性决定了链路上能不能看到这一轮聊了什么；只在模型 span 上写消息的话，trace 会渲染成「no input or output」。

`gen_ai.provider.name` 会映射到规范的 well-known 值：`gemini` → `gcp.gemini`（依据是该 provider 的 base_url 为 `generativelanguage.googleapis.com`，而规范对 `gcp.gemini` 的描述正是 "Also known as the AI Studio API"）。规范允许自定义值，所以 `dashscope`、`siliconflow` 这类值**原样保留**，不做过滤。

### 4.1 对接已有看板的迁移说明

这一版改了两处对外可见的契约，方向都是向规范靠拢，但**按旧属性或旧 span 名建的看板会失效**：

| 变化 | 影响 | 该怎么改 |
| --- | --- | --- |
| span 名从 `invoke_agent` 改成 `invoke_agent dubbo-admin-sre-agent` | 任何按 operation name 过滤的查询失效：Jaeger UI 的 operation 下拉、Grafana 的 `operation_name=` 过滤 | 把过滤条件改成 `invoke_agent dubbo-admin-sre-agent`，或改成前缀匹配 |
| 删除 `agent.usage.total_tokens` | 按这个属性算成本的看板会归零 | 改用 `gen_ai.usage.input_tokens` + `gen_ai.usage.output_tokens` 相加（规范只定义这两个，总量是两者之和） |

`gen_ai.agent.name` 目前是硬编码常量，多 agent 部署时所有 span 会报同一个名字。规范要求该属性低基数，所以硬编码在单 agent 下是对的；**将来加第二个 agent 时这里要跟着改**。

---

## 5. 调试速查

| 我想知道… | 去哪看 |
| --- | --- |
| 某次会话的**全过程** | 用 `gen_ai.conversation.id`（Langfuse 里就是 Session ID）过滤 |
| **这一轮聊了什么、答了什么** | `invoke_agent` span 上的 `gen_ai.input.messages` / `gen_ai.output.messages`（需开 `capture_content`）——Langfuse 的 trace Input / Output 就是取自这里 |
| Agent **怎么发起调用** | span 层级树：interaction → iteration → stage → model_call / tool_call |
| **用了哪些 tools** | `gen_ai.tool.name` + span 名 `execute_tool <tool>` |
| 工具的**入参与返回值** | `gen_ai.tool.call.arguments` / `gen_ai.tool.call.result`（需开 `capture_content`） |
| **思考过程** | `gen_ai.output.messages` 里的 `type: "reasoning"` part（需开 `capture_reasoning`） |
| **输入输出** | `gen_ai.input.messages` / `gen_ai.output.messages`（需开 `capture_content`） |
| **哪里失败了、为什么** | span 状态 + `error.type` + `gen_ai.client.operation.exception` 事件的 `exception.type` / `exception.message` |
| **模型与 token / 成本** | `gen_ai.request.model` + `gen_ai.usage.*`（成本由后端按价格表算） |
| 有没有**走降级 / 兜底** | `agent.degraded` / `agent.fallback.used` / `agent.fallback.reason` |

> `agent.*` 是 Dubbo Admin 私有的补充属性（不在规范 registry 内），用于表达规范没覆盖的 Agent 内部状态。它们与 `gen_ai.*` 并存，不冲突。

---

## 6. 已验证结果

在本地统一收集点（OpenTelemetry Collector → Langfuse / OpenLIT / Greptime）上连续跑 3 次完整 Agent 交互（`invoke_agent` → `iteration` → `stage` → 一次模型调用 + 两次工具调用，其中一次失败），**3/3 全部入库，18/18 span 无缺失**：

- span 层级与父子关系完整；模型 span 的 SpanKind 为 `Client`（规范要求），失败的工具 span 状态为 `Error`；
- `gen_ai.provider.name = gcp.gemini`（映射生效）、`gen_ai.agent.name = dubbo-admin-sre-agent`；
- 每次运行**恰好一条** `gen_ai.client.operation.exception` 事件（异常事件按 span 去重）；
- `gen_ai.output.messages` 中思考过程的形状为 `{"type":"reasoning"}`，符合规范；
- **OpenLIT** 将 `invoke_agent dubbo-admin-sre-agent` 识别为一条 Coding Agent 链路并重建了完整 span 树，Chat 视图还原出「用户提问 → 两次工具调用（含错误）→ 助手回答」的完整对话；
- **Langfuse** 把它识别为一条 trace，`sessionId` 直接来自 `gen_ai.conversation.id`，模型用量面板显示 `gemini-pro / 192 tokens`。

两个与本改动无关、但排查时值得知道的事实：

1. **采集端的 tail sampling 会丢链路**。若收集平台配置了 `tail_sampling`（常见策略：错误必留、慢链路必留、其余按百分比），**没有错误且不够慢的链路会被按概率丢弃**。此时「某段时间没有失败链路」这个结论并不可靠。
2. **span 必须有配对的结束事件才会被导出**。未闭合的 span 不会出现在链路里。

---

## 7. 已知限制

| 限制 | 说明 |
| --- | --- |
| **工具内部的下游调用不在链路里** | Agent 侧的 MCP client 会注入 trace context，但 Admin 的 MCP Server **没有**接收它、也没有建 server 端 span。所以工具耗时能看到，但「工具内部又调了谁」看不到。这是下一步要补的（见第 8 节）。 |
| **span 事件无法表达 severity** | GenAI 异常事件规范要求 severity 为 WARN(13)，但 OTel Go SDK 的 span 事件结构里没有 severity 字段，且 `trace.EventOption` 无法在包外实现。当前用 span status = Error 兜底。 |
| **总量口径只有一种** | 规范只定义输入与输出两个 token 属性，总量是两者之和。provider 自报的非规范 `gen_ai.usage.total_tokens` 只保留在单次调用明细里，不参与汇总。 |
| **成本不在 Admin 侧计算** | Admin 只给出「模型名 + token 数」，成本由企业自己的 API 网关或后端平台核算。 |
| **内容默认不采集** | 默认两个开关都是 `none`。**看不到**提示词与思考过程是默认行为，不是故障。 |
| **思考内容目前采不到** | `capture_reasoning` 的策略与转换都在，但**实际模型适配器不产出 reasoning part**：genkit v1.2.0 的 `compat_oai` 插件没有把 OpenAI 兼容响应里的 `reasoning_content` 映射成 `ai.NewReasoningPart`（整个 genkit 只有 `googlegenai` 插件会，而 Admin 这边对 gemini 尚未接通）。所以 `capture_reasoning: full` 目前在生产上没有可采的内容，要等接上这层适配（见第 8 节）。 |
| **Genkit 自身的 span 不受采集开关约束** | `otel.SetTracerProvider` 让 genkit 与本实现共用同一个导出 provider，而 genkit v1.2.0 的 `core/tracing` 会直接在 span 上写 `genkit:input` / `genkit:output`，不经过 Manager 的采集策略。因此 `capture_content: none` 时，**完整 trace**（含 genkit 的 prompt/model span）里仍可能带用户输入与提示词；本实现自己导出的 span 是干净的。这是 base 分支既有的接线，不是本改动引入的。 |

---

## 8. 后续

- **把 Admin 自身的调用接回 Agent 链路**：MCP Server 侧接收 trace context 并创建 `mcp.server.tools/call` span，让 Agent 的链路覆盖到注册中心访问、配置中心读取等下游操作。
- **工具返回的敏感内容过滤**：当前工具返回按设计保留上游原文（用户可以直接追问「上一次调用的模型与返回结果」），其中可能携带的业务敏感数据尚无过滤机制，需要按部署场景确定口径。
- **采样信息透出**：让检索侧能感知当前采样率，避免把「没采到」误判成「没有」。
- **接上 reasoning part 的来源**：把 provider 响应里的思考字段（如 OpenAI 兼容响应的 `reasoning_content`）适配成 `ai.NewReasoningPart`，`capture_reasoning` 才有内容可采（第 7 节「思考内容目前采不到」）。
- **统一导出边界**：让 Genkit 自己导出的 span 也受采集开关约束，或隔离它的 telemetry（第 7 节「Genkit 自身的 span 不受采集开关约束」）。
- **异常事件名的版本跟进**：规范主干已把异常事件拆分，本实现仍用 `6fd0d76` 的那一个；等拆分发布后再升版并重跑对齐检查。
- **规范版本跟进**：GenAI 规范全部文档处于 Development 状态，本实现钉在 `6fd0d76`；升级规范时需重跑对齐检查。

---

## 9. 相关文件

| 文件 | 作用 |
| --- | --- |
| `ai/component/hooks/tracing.go` | span 创建、属性、异常事件 |
| `ai/component/hooks/reasoning.go` | `capture_reasoning` 策略与 reasoning part 的过滤/截断 |
| `ai/component/hooks/component.go`、`config.go` | 配置项与默认值 |
| `ai/component/agent/react/hook_content.go` | 模型响应 → GenAI 消息（含 reasoning part）的转换 |
| `ai/schema/json/hooks.schema.json` | 配置 schema |
| `ai/component/hooks/README.md` | hooks 组件的配置说明 |

规范来源：<https://github.com/open-telemetry/semantic-conventions-genai>（实现时钉在 commit `6fd0d76`）
