# Dubbo Admin 业务 MCP Server MVP 设计方案

## 1. 摘要

本方案为 Dubbo Admin 增加业务 MCP 能力。管理员从 Nacos 中已经发现的 Dubbo 方法里选择需要开放的方法，为 MCP Server 和 MCP Tool 补充业务说明，然后发布配置。MCP Client 可以通过 Dubbo Admin 查看这些 Tools 并发起调用。

本功能不会自动把所有 Dubbo 方法暴露给 Agent。Provider metadata 提供结构契约，管理员提供业务语义和开放范围。

```text
Provider 向 Nacos 上报 metadata
  -> Admin 发现精确的方法签名和类型定义
  -> 用户选择方法并填写描述、参数别名
  -> Admin 校验草稿并原子发布
  -> MCP Client 获取已发布的 Tools
  -> Agent 根据 inputSchema 生成 JSON 参数
  -> Admin 校验参数并恢复 RPC 参数顺序
  -> dubbo-go 通过正常的注册中心链路执行一次泛化调用
  -> Admin 返回 Dubbo 的原始业务结果
```

一个 MCP Tool 绑定一个确定的 Dubbo 方法签名和调用协议。一个 MCP Server 可以组合多个 Dubbo Service 中的方法，但这些方法必须来自同一个 discovery。

## 2. 背景与业务场景

### 2.1 普通 Consumer 与泛化 Consumer 的区别

普通 Java Consumer 会依赖 API JAR。JAR 中的接口已经为业务代码提供了方法名、参数类型、参数顺序，以及开发者在源码中赋予它们的含义。Provider metadata 不是用来替普通 Consumer 自动填写业务参数的。

泛化 Consumer 不需要 API JAR，它通过以下通用契约调用 Dubbo：

```java
Object $invoke(String method, String[] parameterTypes, Object[] args)
```

方法名和有序参数类型仍然不可缺少。它们用于定位重载方法，并告诉 Provider 应该把通用的 Map、List 和标量值还原成哪些真实参数类型。

泛化调用解决的是编译期接口依赖问题，不会自动告诉 Agent 一个方法的业务作用，也不会自动解释每个参数的业务含义。

### 2.2 Provider metadata 是结构契约，不是语义契约

Apache Dubbo Java 的 `MethodDefinition` 包含 `name`、`parameterTypes` 和 `returnType`。其中已经废弃的 `parameters` 字段是 `TypeDefinition` 列表，不是源码参数名列表。`ServiceDefinitionBuilder` 读取的是 Java 反射参数类型，没有调用 `Parameter.getName()`。

因此，现有 Provider metadata 可以告诉 Admin 某个方法是：

```text
createOrder(java.lang.String, com.example.OrderRequest)
```

它不能可靠地告诉 Agent 第一个参数表示 `tenantId`，也不能解释创建订单会产生什么业务效果。

本方案把调用契约分为两层：

| 层次 | 来源 | 职责 |
| --- | --- | --- |
| 结构契约 | Provider metadata | Service 标识、方法名、有序参数类型、返回类型和关联类型定义 |
| 语义契约 | Admin 用户配置 | Server 描述、Tool 名称、Tool 描述、顶层参数别名和说明、调用协议，以及可选 Tool annotations |

这和手工维护另一份完整 API 定义不同。用户不需要重写 DTO 结构、RPC 序列化、Provider 发现、路由、负载均衡或泛化对象还原规则。用户只负责选择开放范围，并补齐 metadata 本身不具备的业务语义。

### 2.3 适用场景

当 MCP Client 需要调用一组经过筛选的内部 Dubbo 能力，而又不适合为每个服务嵌入 API JAR 或生成专用 Consumer 时，可以使用本功能。例如，平台团队可以把现有服务中的少量操作开放给编码 Agent、客服 Agent 或内部自动化工具。

本功能不用于替代普通应用之间的强类型 Dubbo 调用。

## 3. 目标与非目标

### 3.1 MVP 目标

MVP 完成以下闭环：

1. 读取 Admin 已经从 Nacos 发现的 Dubbo Provider service definition。
2. 允许用户创建一个或多个 MCP Server，并选择精确的 Dubbo 方法签名作为 Tools。一个 MCP Server 绑定一个 discovery。
3. 要求用户为 MCP Server 和每个 Tool 填写描述。
4. 允许用户为每个顶层方法参数设置别名和说明。
5. 根据 Provider metadata 和用户语义配置生成严格的 MCP `inputSchema`。
6. 支持草稿校验，并一次性原子发布完整的生效版本。
7. 通过无状态 Streamable HTTP 暴露每个已发布的 MCP Server。
8. 使用 Admin 管理的机器凭证鉴权。
9. 通过正常 Dubbo 注册中心、路由和负载均衡链路调用 Provider，支持 `dubbo` 和 `tri` 两种协议的泛化调用。
10. 保证一次 `tools/call` 最多发起一次 RPC。
11. 不做用户自定义的输出转换，直接返回 Dubbo 的业务结果。
12. 确认 Provider 已经下线时，把对应 Tool 从 `tools/list` 中隐藏。

### 3.2 MVP 非目标

MVP 不包含：

- 自动暴露所有 Provider 方法；
- 在 MCP 层实现人工确认或审批状态机；
- 编排多个 Tool 的执行顺序；
- Triple IDL 或流式 RPC；
- 一个 MCP Server 跨多个 discovery 组合方法；
- ZooKeeper discovery；
- 通过实例 `dubbo.metadata.revision` 精确校验接口导出状态；
- Admin 自己选择 Provider、实现本地轮询或通过直连地址故障转移；
- RPC 重试、协议回退或实例回退；
- 嵌套 DTO 字段别名、字段说明或自定义约束；
- 输出 Schema、输出别名或业务结果映射；
- 已发布版本历史或按 MCP Session 固定版本；
- Tool 级机器凭证权限；
- 限流、并发限制或 Tool 数量硬限制；
- MCP `tools/list` 分页；
- Provider metadata 变化后的自动契约迁移；
- Provider 异常信息脱敏。

这些是明确的 MVP 边界，不是未定义行为。

## 4. 源码基线与现有能力

本方案基于以下源码版本核查：

| 项目 | 核查版本 | 与本方案直接相关的结论 |
| --- | --- | --- |
| dubbo-admin | [`2df3e07`](https://github.com/apache/dubbo-admin/commit/2df3e07674c6cff6ec05fc78fd4563206efdc8e9) | 已能从 Nacos 读取 Provider definition；Console 已能解析重载方法并进行调试型泛化调用；ResourceStore 没有 CAS 接口；现有 MCP Server 是静态实现 |
| Apache Dubbo Java | [`d0bf5c36d0`](https://github.com/apache/dubbo/commit/d0bf5c36d092da8067b15a1d1d00884a8c399e8e) | `FullServiceDefinition` 包含方法和类型结构，但不包含源码参数名；泛化调用依赖方法名、参数类型名和有序参数值 |
| dubbo-go | [`35ea886421f9`](https://github.com/apache/dubbo-go/commit/35ea886421f9) | 当前固定版本支持基于注册中心的 `GenericService`、fail-fast cluster 和显式 retries；默认 cluster 是 failover，默认 retries 是 2，默认 Consumer 请求超时是 3 秒，默认负载均衡是 random；`dubbo` 和 `tri` 两种协议都支持泛化调用；cluster 层与协议无关，`fail-fast` 加 `retries=0` 的单次调用语义对两种协议一致；`NewGenericService` 对两种协议都强制 `Hessian2Serialization`；Triple 的 timeout 传递在 `triple_invoker.go` 中标注为临时方案 |
| MCP Go SDK | [`v1.4.0`](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.4.0) | 这是兼容仓库 Go 1.24 基线并完整支持 MCP 2025-11-25 的官方 SDK 版本；低层动态 Tool handler 仍需要调用方自行校验参数 |

源码核查时，官方 MCP Go SDK [`v1.7.0`](https://github.com/modelcontextprotocol/go-sdk/blob/v1.7.0/go.mod) 已要求 Go 1.25。Admin toolchain 升级和 MCP SDK 升级应作为独立依赖变更处理，本方案不隐式升级仓库的 Go 版本。

关键的外部源码入口包括 Dubbo Java 的 [`MethodDefinition`](https://github.com/apache/dubbo/blob/d0bf5c36d092da8067b15a1d1d00884a8c399e8e/dubbo-common/src/main/java/org/apache/dubbo/metadata/definition/model/MethodDefinition.java)、[`ServiceDefinitionBuilder`](https://github.com/apache/dubbo/blob/d0bf5c36d092da8067b15a1d1d00884a8c399e8e/dubbo-common/src/main/java/org/apache/dubbo/metadata/definition/ServiceDefinitionBuilder.java)、[`MetadataUtils`](https://github.com/apache/dubbo/blob/d0bf5c36d092da8067b15a1d1d00884a8c399e8e/dubbo-registry/dubbo-registry-api/src/main/java/org/apache/dubbo/registry/client/metadata/MetadataUtils.java) 和 [`AbstractMetadataReport`](https://github.com/apache/dubbo/blob/d0bf5c36d092da8067b15a1d1d00884a8c399e8e/dubbo-metadata/dubbo-metadata-api/src/main/java/org/apache/dubbo/metadata/report/support/AbstractMetadataReport.java)，以及当前固定 dubbo-go 版本的 [`client/options.go`](https://github.com/apache/dubbo-go/blob/35ea886421f9/client/options.go) 和 [`failfast/cluster_invoker.go`](https://github.com/apache/dubbo-go/blob/35ea886421f9/cluster/cluster/failfast/cluster_invoker.go)。

### 4.1 Admin 当前从 Nacos 读取的两类数据

本功能会涉及 Nacos 中两类不同的数据：

| Nacos 能力 | 数据含义 | 在本方案中的用途 |
| --- | --- | --- |
| Naming Service | 已注册的 Provider 应用或实例及其可用状态 | 提供正常 Dubbo Consumer directory |
| Config Service metadata report | 某个 interface、group、version、side 和 application 的 `FullServiceDefinition` | 提供 Tool 选择与 Schema 生成所需的方法签名和类型定义 |

Java Provider metadata 位于 Nacos Config Service 的 `dubbo` group。data ID 按以下字段组成：

```text
serviceInterface:version:group:provider:application
```

例如：

```text
group  = dubbo
dataId = com.example.OrderService:1.0.0:trade:provider:order-provider
```

其内容可以简化表示为：

```json
{
  "canonicalName": "com.example.OrderService",
  "methods": [
    {
      "name": "createOrder",
      "parameterTypes": [
        "java.lang.String",
        "com.example.OrderRequest"
      ],
      "returnType": "com.example.Order"
    }
  ],
  "types": [
    {
      "type": "com.example.OrderRequest",
      "properties": {
        "productId": "java.lang.String",
        "quantity": "java.lang.Integer"
      }
    }
  ],
  "parameters": {
    "application": "order-provider",
    "group": "trade",
    "version": "1.0.0"
  }
}
```

[`pkg/discovery/nacos2/factory.go`](../../pkg/discovery/nacos2/factory.go) 已经为 `dubbo` group 中的 `*:provider:*` 创建 Config Service watcher，并把内容转换为 `ServiceProviderMetadataResource`。

当前 Nacos factory 使用 discovery 的 `address.configCenter` 创建这条 metadata watcher，不会使用 `address.metadataReport` 另建连接。因此，MVP 要求 Provider metadata report 对 Admin 配置的同一个 Nacos Config Center 可见，并位于 `dubbo` group。支持独立 metadata report address 或自定义 group 需要另行扩展 discovery，不在本方案范围内。

Naming Service 数据负责说明哪些 Provider 可用，service definition 负责说明接口结构。两者都不包含缺失的业务语义。

#### 活跃服务索引

Config Service 中的 service definition 是持久化契约，Provider 下线后仍可能存在，不能用它判断实例是否存活。Admin 新增 `LiveServiceIndex`，把注册中心中的运行状态转换成统一的活跃服务视图。内部索引项使用以下标识：

```text
mesh + providerApplication + serviceName + group + version + protocol
```

`protocol` 是索引标识的一部分。一个接口可能同时导出 `dubbo` 和 `tri`，也可能只导出其中一种，而每个 Tool binding 绑定了确定的调用协议，因此存活判断必须按协议粒度进行。协议信息来自 Nacos 实例 metadata 中的 endpoint，现有泛化调用已经在读取该字段。

MVP 采用如下判定规则：

> 声明导出该接口的那些 Provider 应用中，是否还存在至少一个健康实例，且该实例导出了 binding 所需的协议。

Service definition 的 `parameters.application` 提供"哪些应用声明导出了这个接口"，Naming Service 提供"这些应用当前有哪些健康实例"。两者取交集即可，不需要读取 application metadata。

每个 Service identity 加协议的状态为以下三种之一：

| 状态 | 判定 |
| --- | --- |
| `ACTIVE` | 声明导出该接口的应用中，至少一个健康实例导出了所需协议 |
| `INACTIVE` | 注册中心已完成同步，确认没有这样的实例 |
| `UNKNOWN` | 初始同步尚未完成，或注册中心不可用 |

`UNKNOWN` 不能折叠为 `INACTIVE`。索引必须保存 discovery 同步状态和最近一次成功更新时间，不能用"当前结果为空"同时表达"确认没有实例"和"尚未取得数据"。这条区分在 MVP 中是**运行时安全属性**而非仅仅是诊断信息：§8.3 规定 `INACTIVE` 会让 Tool 从 `tools/list` 消失，如果 `UNKNOWN` 被误判为 `INACTIVE`，一次注册中心抖动就会让所有 Agent 的能力集合瞬间清零。

#### MVP 不做实例级 revision 校验

精确的做法是读取每个活跃实例的 `dubbo.metadata.revision`，再读取该 revision 对应的 application `MetadataInfo.services`，从而知道该实例此刻真正导出了哪些接口。MVP 不做这一层。

代价是一种确定的误判：应用升级后仍然在线，但新版本已经不再导出某个接口，而旧 definition 没有被删除。此时上述规则仍会判定 `ACTIVE`。

接受这个误判的理由是**误判方向是安全的**。缺少 revision 校验只会导致"该隐藏的 Tool 没有隐藏"，不会导致"不该隐藏的 Tool 被隐藏"——因为一旦应用的全部实例消失，上述规则同样会正确判定 `INACTIVE`。而"没有隐藏"已经有干净的兜底：Agent 调用该 Tool 会得到 §9.2 定义的 `not-executed` 加"没有可用 Provider"，这是一个分类正确、可理解的错误，不是数据损坏。

实例级 revision 校验作为后续版本的独立改进，届时需要新增按 `application + revision` 读取和缓存 `MetadataInfo` 的 resolver（Nacos 中读取标识是 `dataId=application`、`group=revision`），该数据不在现有 `*:provider:*` definition watcher 的读取范围内。

#### 现有代码需要补齐的部分

当前 Admin 已经分别创建 Nacos Config Service client 和 Naming Service client，也会根据 Naming Service 的变更增删 `RPCInstanceResource`。现有代码仍缺少两部分。

第一，[`NacosServiceEventSubscriber`](../../pkg/core/discovery/subscriber/nacos_service.go) 会直接忽略 `providers:<interface>` 接口级服务。在接口级注册模式下，注册中心里只有这类条目，没有可供关联的应用实例。如果不补齐这条路径，接口级注册模式下的每一个 binding 都会被判为 `INACTIVE`，再叠加 §8.3 的隐藏规则，结果是该模式下**所有 Tool 全部从 `tools/list` 消失**。这是 MVP 的必做项，不是优化项。

第二，Nacos SDK 返回的实例还包含 `Healthy`、`Enable` 和 `Weight`，而 [`NacosServiceListerWatcher`](../../pkg/discovery/nacos2/listerwatcher/nacos_service.go) 转换资源时只保留 IP、端口和 metadata。MVP 必须在转换前过滤 `Healthy=true`、`Enable=true`、`Weight>0` 的实例，或者扩展 `NacosInstance` resource 保留这些字段后再由索引过滤。否则一个仍在注册表中但已经被 Nacos 判为不健康或禁用的实例，也会把接口错误标记为 `ACTIVE`。

`LiveServiceIndex` 在 MVP 中被用于两处：§6.2 发布校验的 warning，以及 §8.3 的 `tools/list` 隐藏。它**不参与 RPC 实例选择**，正式调用仍由 dubbo-go 的 directory、router 和 load balancer 选择 Provider。

### 4.2 现有方法解析能力

[`pkg/console/service/service.go`](../../pkg/console/service/service.go) 已经可以按 mesh 和 service key 查询 Provider metadata，按方法名和签名去重，并通过精确签名定位重载方法。MCP resolver 应抽取并复用这些规则。

当前合并类型定义的 helper 不能直接用于证明所有 Provider definition 一致，因为它遇到同名类型时会保留第一份定义。MCP 发布前必须分别规范化每份匹配的 Provider definition，再比较规范化结果。

### 4.3 现有泛化调用能力

[`pkg/console/service/service_generic_invoke.go`](../../pkg/console/service/service_generic_invoke.go) 已经通过 dubbo-go `GenericService` 发起调用。现有 Console API 面向人工调试，调用者需要选择实例，Admin 还可能尝试多个协议或序列化目标。

业务 MCP 不能复用这段编排控制流。非幂等 RPC 可能已经执行成功，只是 Admin 在接收响应时发生了超时或连接错误。此时继续尝试另一个目标会造成重复业务效果。

可以复用或抽取精确方法查找、基础类型参数解码、结果转换为 JSON 兼容值等纯 helper。Provider 选择和重试控制必须为业务 MCP 单独实现。

### 4.4 现有 MCP endpoint

[`pkg/mcp/server.go`](../../pkg/mcp/server.go) 是面向静态运维 Tools 的 MCP Server。它维护内存 Tool map，并且只做浅层 required 校验。

它的 `InputSchema` 类型是扁平的，只有 `Type`、`Properties map[string]PropertyDef` 和 `Required` 三个字段，不支持 `$defs`、`$ref` 或任何嵌套 schema 组合；其 Tool 的 property 名都是手写常量，从不从类型名派生。因此业务 MCP 的 schema builder（含 §7.3 的 `$defs` key 生成）没有任何可复用的现成实现，是全新代码。这也意味着不存在一个已按 Java 特有字符写死的历史 sanitizer 需要改造——白名单规则从第一版就应当直接写对。

现有 `/api/mcp` endpoint 及其鉴权行为保持不变。业务 MCP 使用独立 endpoint 和官方 Go SDK，避免把现有运维 MCP 改造成动态多租户运行时。

### 4.5 现有存储边界

现有 `ResourceManager` 只允许写入治理规则资源。MCP 配置不是治理规则，不应为了复用写接口而扩大 Governor 的资源边界。

本功能为 ResourceStore 注册 MCP 资源类型，并在 ResourceStore 之上增加职责单一的 MCP repository 或 manager。它不修改 Governor 的资源范围。

## 5. 领域模型

### 5.1 术语

| 术语 | 含义 |
| --- | --- |
| Mesh | Admin discovery 的 ID，即一套注册中心加配置中心 |
| Service identity | `mesh + serviceName + group + version` |
| Operation identity | Service identity 加 `methodName + ordered parameterTypes` |
| MCP Server | 用户定义的一组 Operation 和 Server 描述，绑定单个 mesh |
| MCP Tool | 面向 Agent 的名称和描述，绑定一个 Operation identity 和一个调用协议 |
| Draft | 尚未对 MCP Client 生效的可编辑配置 |
| Published revision | 当前唯一对 MCP Client 生效的完整配置 |
| Machine credential | Admin 签发的机器凭证，授权访问一个 MCP Server 的所有 Tools |

`provider` 和 `consumer` 仍然是 Dubbo 运行时角色。MCP Server、MCP Tool、Draft 和 Credential 是 Admin 配置概念，不是新的 Dubbo 身份。

### 5.2 Binding identity

每个 Tool binding 保存精确的 Operation identity，以及 identity 之外的调用协议：

```yaml
source:
  mesh: production-nacos
  serviceName: com.example.OrderService
  group: trade
  version: 1.0.0
  methodName: createOrder
  parameterTypes:
    - java.lang.String
    - com.example.OrderRequest
  protocol: dubbo
```

有序参数类型列表是 identity 的组成部分。仅保存方法名无法区分重载方法。

`protocol` 取值为 `dubbo` 或 `tri`，在创建 binding 时确定并持久化。Admin 默认自动选择：两种协议都可用时选 `dubbo`，只有一种时选那一种；用户可以覆盖。

协议必须存进 binding，不能在调用时动态跟随 Provider 当前广播的内容。同一个 Tool 的两次调用如果走了不同协议，其超时传递方式（§8.6）和错误信息形状（§9.2）都不一样，会让线上问题无法归因。持久化之后，审计日志（§12）也能记录本次调用实际使用的协议。

`protocol` 是 binding 的属性，不进入 §6.4 的契约 fingerprint——它是传输选择，不是结构契约。但它进入 §8.5 的 reference 复用 key 和 §4.1 的存活判断。

MCP Tool 名称由用户定义，在同一个 MCP Server 内必须唯一，不要求与 Dubbo 方法名相同。

一个 MCP Server 内的所有 binding 必须使用同一个 `mesh`。跨 discovery 组合方法不在 MVP 范围内。

### 5.3 MCPServerResource

本功能新增 `MCPServerResource`。它与现有资源一样是 mesh 作用域的——`mesh` 即该 Server 绑定的 discovery ID。这样可以直接复用现有的 `ResourceKey` 构造方式和 `ByMeshIndex` 过滤，不需要向资源模型引入"集群级"这个当前并不存在的概念（`ResourceModel.Mesh` 是 `not null`，所有查询路径都以 mesh 为维度）。

建议的逻辑结构如下：

```yaml
metadata:
  name: orders-mcp
  mesh: production-nacos
  resourceVersion: "42"
spec:
  draft:
    displayName: Order Operations
    description: Create and query orders for internal support workflows.
    tools: []
  published:
    revision: 7
    publishedAt: "2026-08-26T10:00:00Z"
    displayName: Order Operations
    description: Create and query orders for internal support workflows.
    tools: []
```

每个已发布 Tool 快照保存：

- Tool 名称和描述；
- 可选的 `readOnlyHint`、`destructiveHint` 和 `idempotentHint`；
- 精确的 Operation identity，含调用协议；
- 有序的顶层参数别名和说明；
- 生成的 `inputSchema`；
- 规范化的 `inputFingerprint` 和 `outputFingerprint`（见 §6.4）；
- 可选的超时开关和超时值。

Draft 和 Published snapshot 保存在同一个资源中，因此一次 publish 可以通过一次 CAS 替换完整生效版本。MVP 不保存已发布版本历史。

### 5.4 MCPCredentialResource

Credential 与 Published revision 分开保存，防止配置发布恢复旧凭证或撤销已经完成的 revoke。

```yaml
metadata:
  name: credential-id
  mesh: production-nacos
spec:
  serverId: orders-mcp
  name: production-agent
  secretHash: sha256:...
  status: active
  expiresAt: null
  createdAt: "2026-08-26T10:00:00Z"
```

一个 MCP Server 可以有多个命名 Credential。所有 Credential 权限相同，只能访问该 Server 的全部 Tools，不能访问 Admin Console API 或其他 MCP Server。

删除 MCP Server 时，必须级联删除该 Server 的全部 Credential，避免留下无法通过任何 UI 路径查看或撤销的孤儿凭证。

Token 包含非敏感的 credential ID 和随机 secret：

```text
mcp_<credential-id>.<random-secret>
```

Admin 只在创建时返回一次明文 Token。Admin 保存高熵随机 secret 的 SHA-256 hash，并使用常量时间比较。secret 至少 256 bit，来自密码学安全随机源。过期时间可选，默认永不过期。每个 Credential 可以单独 revoke。

Credential 的权威数据源是 ResourceStore。因此业务 MCP **要求使用数据库 store**：memory store 是进程内的，多副本部署下在一个副本上完成的 revoke 对其他副本不可见，被撤销的凭证仍然可用。这是安全约束，不是性能取舍。

Agent 不需要在对话中记住或重复 Token。MCP Client 负责保存凭证，并在每次 HTTP 请求中自动添加 `Authorization` header。

## 6. 控制面

### 6.1 草稿编辑

创建或编辑 Server 时只修改 `spec.draft`。MCP Client 继续读取上一个 `spec.published` snapshot。

用户从 Admin 已发现的 Service catalog 中选择方法。Admin 保存精确签名，而不是只保存 interface 或方法名。

Service catalog 必须显示每个候选方法的 `LiveServiceIndex` 状态。Config Service 中的 definition 是持久化的，一个早已下线的服务，其 definition 仍然留在 Nacos 里。如果不显示存活状态，管理员面对的是一份混杂着大量僵尸接口的列表，无法判断哪些还可以选择。三种状态在 UI 上必须可区分，尤其不能把 `UNKNOWN`（discovery 故障）显示成 `INACTIVE`（Provider 下线）。

用户还需要为每个方法选择调用协议。Admin 根据当前导出情况给出默认值（两种都可用时默认 `dubbo`），用户可以改。只导出了一种协议时，另一种在 UI 上置灰并说明原因。

Admin 按顺序展示每个 RPC 参数类型，这些结构字段不可修改。由于 Java metadata 没有可靠的参数名，Admin 默认显示 `arg0`、`arg1` 等名称。用户可以修改参数别名，并填写参数说明。

例如：

| Position | RPC type | 默认名称 | 用户别名 | 说明 |
| --- | --- | --- | --- | --- |
| 0 | `java.lang.String` | `arg0` | `tenantId` | 订单所属租户 |
| 1 | `com.example.OrderRequest` | `arg1` | `request` | 创建订单请求 |

Position 和 RPC type 不可修改。运行时按保存的 Position 把别名映射回有序参数。

MVP 不提供嵌套 DTO 字段编辑器。嵌套对象继续使用 metadata 中的属性名和类型。如需说明特殊字段含义，可以写在 Tool description 中。

### 6.2 草稿校验

草稿校验是只读操作，不发布配置，也不会自动发送业务 RPC。

至少校验以下内容：

1. Server description 非空。
2. Tool 名称符合 MCP 命名规则，并且在 Server 内唯一。
3. 每个 Tool description 非空。
4. 每个顶层参数别名非空，并且在 Tool 内唯一。
5. 引用的 discovery 存在，并且是 MVP 支持的 Nacos discovery。
6. Provider metadata 中仍然存在精确的方法签名。
7. 同一个 Service identity 的所有匹配 definition 可以解析为唯一一致的结构契约。
8. 所有输入类型都可以转换为闭合的 JSON Schema。
9. 生成的 Schema 可以成功编译。
10. 超时开关启用时，超时值为正数。
11. 所有 binding 的 `mesh` 相同，且等于 Server 自身的 mesh。
12. 每个 binding 声明的协议（`dubbo` 或 `tri`）在 Provider metadata 与注册中心中存在对应导出。

以上均为 error，校验不通过不能发布。

以下为 warning，不阻塞发布：

- binding 对应的 `LiveServiceIndex` 状态不是 `ACTIVE`。

存活状态不作为发布门禁，原因有三。第一，运行时对这种情况已有干净的处理：§8.3 会把 `INACTIVE` 的 Tool 从 `tools/list` 隐藏，§9.2 对仍然发起的调用返回分类正确的 `not-executed`。第二，规则 6 已经拦掉了真正的误配置（方法名写错、签名不符、接口已删），存活校验在它之上多拦的只有"定义还在但此刻没人在跑"这一种。第三，初始同步尚未完成时状态就是 `UNKNOWN`，如果它是 error，Admin 每次重启后到全量同步完成之前将无法发布任何 binding。

Admin 在 warning 详情中必须保留 `INACTIVE` 和 `UNKNOWN` 的区别，避免把 discovery 故障显示成 Provider 下线。

校验结果分别返回 errors 和 warnings，不修改当前生效版本。

### 6.3 原子发布

Publish 会基于当前 metadata 重新执行完整校验，生成不可变的 Published snapshot，递增 Server revision，并通过一次 CAS 替换 `spec.published`。

如果校验或 CAS 失败，旧的 Published snapshot 继续生效。MCP Client 不会看到部分更新的 Tool 列表。

所有 Draft 更新和 Publish 请求都必须携带期望的 ResourceStore `resourceVersion`。版本落后的写入返回 conflict，调用者需要重新读取后再操作。

### 6.4 契约 fingerprint

Admin 为每个 binding 生成两个确定性的 fingerprint。

**`inputFingerprint`**，参与 fail closed：

- Service name、group 和 version；
- 方法名和有序参数类型；
- 从入参可达的所有类型定义；
- 上述闭包内的 collection item type、enum value 和 object property。

**`outputFingerprint`**，只用于告警：

- 返回类型；
- 从返回类型可达的所有类型定义。

Hash 前对 Map key 和 object property 排序。用户填写的别名、描述、annotations 和 timeout 不属于 Provider 结构契约，不进入任何 fingerprint。binding 的调用协议同样不进入——它是传输选择，不是结构契约。

Provider 存活状态也不进入 fingerprint。它变化频繁，不会改变已经发布的 Tool 契约。

#### 为什么只有入参参与 fail closed

已发布的对外契约只有 `inputSchema`。MVP 不生成 output schema（§3.2），也不做输出转换（§9.1），Provider 返回什么就原样透传什么。

因此入参结构的变化会真实地让已发布契约失效：§7.3 对每个 POJO 设置了 `additionalProperties: false`，Provider 新增一个入参字段后，Agent 在结构上就**无法**再传这个字段。继续调用的结果要么是 Provider 业务校验失败，要么更糟——字段缺失被当成默认值静默处理。这种情况必须 fail closed。

返回值结构的变化则不会让任何已发布的东西失效。Provider 给响应 DTO 增加一个字段（Dubbo/Hessian 生态中最常见的向后兼容演进方式），多出来的字段会原样透传给 Client，不会破坏任何东西。如果把它也纳入 fail closed，等于把最常见的兼容变更配置成了最高级别的故障。

返回值 fingerprint 变化时，Admin 在管理页对该 Tool 显示"Provider 返回结构已变更，建议重新发布"，不影响 `tools/list` 和 `tools/call`。

#### 校验时机与失败范围

在 `tools/list` 和 `tools/call` 时，Admin 重新读取当前 `ServiceProviderMetadataResource` 并计算 `inputFingerprint`。definition 缺失、存在冲突或 `inputFingerprint` 不一致时，**该 Tool** fail closed——它不出现在 `tools/list` 中，对它的 `tools/call` 返回错误。其余 Tool 不受影响。

这与 §8.3 对 `INACTIVE` 的处理是同一套降级模型：单个 binding 的问题只影响单个 Tool，不会让整个 Server 的能力集合归零。

Admin 不会静默更新已发布 Schema。Provider 发生入参 breaking change 时，应发布新的 group 或 version，再由管理员发布新的 binding。

### 6.5 Provider metadata 发布前置条件

Admin 可以发现 Published fingerprint 与 Nacos 当前 metadata 不一致，但无法发现新 Provider binary 正在使用仍然保持旧 fingerprint 的陈旧 metadata。

当前 Dubbo Java 源码没有把 metadata 发布成功作为 Provider 注册屏障：

1. `ServiceConfig.exportRemote` 先导出并注册服务，再调用 `MetadataUtils.publishServiceDefinition`。
2. `AbstractMetadataReport` 默认异步上报。
3. metadata 写入失败后会记录日志并进入后台重试，不会让已经导出的服务自动停止提供流量。

因此，生产使用需要额外的 Provider 发布约束：新版本的 service definition 必须成功写入并经过验证，然后新 Provider 实例才能进入流量。只配置 `sync-report=true` 仍然不够，因为源码调用顺序依旧是在 export 之后才发布 metadata。

这个约束可以通过 Provider 侧改造实现，也可以由部署系统在 metadata 验证通过前保持新实例不可用。它不属于 Admin 运行时，但它是强契约一致性的前置条件。

## 7. 从 metadata 生成 MCP inputSchema

### 7.1 顶层参数模型

Tool input 固定为 JSON object，其 properties 与方法的有序参数一一对应。

```json
{
  "type": "object",
  "properties": {
    "tenantId": {
      "type": "string",
      "description": "订单所属租户"
    },
    "request": {
      "$ref": "#/$defs/com.example.OrderRequest",
      "description": "创建订单请求"
    }
  },
  "required": ["tenantId", "request"],
  "additionalProperties": false
}
```

方法的每一个顶层参数都进入 `required`。这里的 required 表示属性必须出现，nullability 由类型规则单独决定。

Binding 单独保存参数 Position，不能依赖 JSON object 的遍历顺序决定 RPC 参数顺序。

### 7.2 类型映射

MVP 把闭合的 Java metadata 类型映射为以下 JSON 表达：

| Dubbo metadata type | JSON 表达 |
| --- | --- |
| `boolean`、`java.lang.Boolean` | boolean |
| `byte`、`short`、`int` 及其 wrapper | 带对应范围的 integer |
| `long`、`java.lang.Long` | 十进制 string，避免 JSON Client 丢失整数精度 |
| `float`、`double` 及其 wrapper | number |
| `char`、`java.lang.Character` | 长度为 1 的 string |
| `java.lang.String` | string |
| enum | 带 `enum` values 的 string |
| array 或参数化 collection | 带 item schema 的 array |
| POJO | 使用 metadata property names 的 object |
| `Map<String, T>` | 使用 `additionalProperties` value schema 的 object |

Admin proto 中的 `Type` 用同一个 `items` 字段表达 collection 和 map，两者靠元数区分：collection 的 `items` 只有一个元素（item type），map 的 `items` 有两个元素（key type 和 value type）。实现必须依据元数判别，不能只看类型名字符串。

`long` 映射为 string 的代价在于解码：参数在送入泛化调用之前必须转回 int64。这个转换是递归的，见 §7.4。

命名类型使用本地 `$defs` 和 `$ref`。只有在本地引用图可以安全编译时才接受递归类型。

Java primitive 不接受 `null`。Provider metadata 没有 nullability annotation，因此引用类型可以接受 `null`。metadata 同样没有字段 required 信息，因此 POJO 内部字段默认都是可选项。Provider 侧业务校验仍然可以拒绝缺失或为 null 的字段。

以下契约在 MVP 中直接校验失败，不生成含义不确定的 Schema：

- 没有 item definition 的 raw collection 或 raw map；
- key 不是 string 的 Map；
- 找不到定义的类型引用；
- 同名但定义不同的重复类型；
- `Object`、wildcard、未绑定 type variable 等开放类型；
- 尚未约定稳定 JSON 表达的 Date、Time 和任意精度数字等特殊类型。

#### dubbo-go Provider 的发布侧类型映射

dubbo-go Provider 也可以接入这套设计，但**必须以 Java 类型词汇发布 service definition**，不能发布 Go 原生类型名。这是对发布方的规范要求，Admin 侧不做 Go 类型识别，也不按 `release` 前缀分派不同的映射器。

这条要求不是 Admin 的偏好，而是 dubbo-go 运行时自身的既定契约。`filter/generic/service_filter.go` 中，非变参方法的 realize 只依赖 Go 反射得到的 `argsType`，`types` 参数确实未被使用；但变参方法会走 `realizeVariadicArg`，由 `types` 的最后一个元素决定是否把打包的尾参展开成 slice，而该判断的实现是：

```go
func shouldUnwrapPackedVariadicArg(variadicType string, variadicSliceType reflect.Type) bool {
	if slices.Contains(javaTypeNamesForType(variadicSliceType), variadicType) {
		return true
	}
	...
}
```

`javaTypeNamesForType` 的候选值来自 `hessian.GetJavaName()` 和 JVM array descriptor（形如 `[Ljava.lang.String;`）。因此发布 Go 词汇会让变参方法的尾参匹配失败，打包的 slice 被当成单个变参值处理——**静默错误，不产生任何报错**。

发布侧映射规则：

| Go | Java | 说明 |
| --- | --- | --- |
| `bool` | `boolean` | |
| `int8` / `int16` / `int32` / `int64` | `byte` / `short` / `int` / `long` | |
| `uint8` | `short` | 0–255 无损 |
| `uint16` | `int` | 无损 |
| `uint32` | `long` | 无损 |
| `uint` / `uint64` | —— | 无无损落点，发布时拒绝 |
| `float32` / `float64` | `float` / `double` | |
| `string` | `java.lang.String` | |
| `[]T` | `java.util.List<T>` | `items` 一个元素 |
| `map[K]V` | `java.util.Map<K,V>` | `items` 两个元素，顺序为 key、value |
| `*T` | `T` | nullability 由本节的引用类型规则表达，Java 类型名同样无法表达指针 |

`uint` 和 `uint64` 在发布时直接拒绝，与本节的拒绝清单一致。放开它们需要同时定义任意精度数字的 JSON 表达和解码规则，为一个在 DTO 中罕见的类型开这个口子不划算。

变参方法的 `parameterTypes` 尾项必须是 Java 类型名或 JVM array descriptor 之一，否则变参 realize 会出错。

采用 Java 词汇后，`int64` 自然落到 `long`，因此本节的十进制 string 规则自动覆盖 Go 侧 19 位 ID 的精度问题，无需额外处理。

#### 字段名不做映射

property 名必须保持 Go 侧的 wire name（`ID` 字段的 wire name 是 `iD`，来自 `LowerFirstRune`），不能一并"Java 化"。

类型名和字段名的性质不同：类型名除变参尾项外是描述性的，而 property 名是 wire 级的，Generalizer 的 `Realize` 依赖它做字段匹配，改动会直接破坏调用。

由此产生的后果记录在 §16：同一个接口如果同时存在 Java 和 Go Provider，两者的 property 名不同（`id` 与 `iD`），§6.2 规则 7 会判定结构契约不一致，binding 无法发布。

### 7.3 严格拒绝未知字段

未知字段必须被拒绝，不能先删除再调用：

- 顶层 arguments object 设置 `additionalProperties: false`。
- `$defs` 中的每个 POJO object 设置 `additionalProperties: false`。
- `Map<String, T>` 允许动态 key，但每个 value 都必须符合 `T`。

这条规则需要递归执行。只校验顶层 required 的 validator 不满足要求。

生成的 `inputSchema` 使用 JSON Schema Draft 2020-12。官方 SDK 的低层动态 `Server.AddTool` handler 不会校验 raw arguments。业务 MCP 应直接复用 MCP Go SDK `v1.4.0` 已经固定的 `github.com/google/jsonschema-go v0.4.2`，只编译 Admin 生成的内存 Schema，不允许加载外部网络 `$ref`。

#### `$defs` key 的构造

不能直接用类型名作为 `$defs` key。`$ref` 的取值是一个 URI 引用，而类型名中普遍存在 URI fragment 不接受的字符：Java 参数化类型形如 `java.util.List<com.example.Item>`，带 `<`、`>`、`,`；Go 类型名含 import path，带 `/`。不同 validator 对这类字符的宽容度不一致，直接拼接会带来无法预期的编译或解析失败。

转换规则必须按**白名单**定义，而不是枚举某种语言特有的字符：把类型名中所有不属于 `[A-Za-z0-9_.-]` 的字符一律转义。按白名单实现时，Java 的 `<>,` 和 Go 的 `/` 会被同一条规则覆盖，不需要为任何语言写特例；反过来，如果实现成"替换 `<`、`>`、`,` 这三个字符"的黑名单，Go import path 里的 `/` 就会漏网。

Admin 在 binding 内保存类型名到 key 的映射。转换必须是确定性的且无碰撞——同一份 metadata 每次生成的 key 必须一致，否则 §6.4 的 fingerprint 会出现伪变化。

### 7.4 恢复有序参数

参数通过 Schema 校验后，Admin 按保存的 Position 组装泛化调用参数：

```go
GenericService.Invoke(
    ctx,
    "createOrder",
    []string{
        "java.lang.String",
        "com.example.OrderRequest",
    },
    []hessian.Object{
        arguments["tenantId"],
        arguments["request"],
    },
)
```

Admin 不负责实例化 Java POJO，真实类型还原由 Provider 的 generic filter 完成。但在交给 generic filter 之前，Admin 必须完成协议边界所需的标量解码。

#### 标量解码必须递归

§7.2 的类型映射在 JSON 表达和 Java 标量之间引入了几处不等价，这些都必须在 dispatch 之前修正：

| 情况 | 必须做的处理 |
| --- | --- |
| `long` / `java.lang.Long` | 十进制 string 转 int64，转换失败或溢出则拒绝 |
| `byte` / `short` / `int` 及其 wrapper | JSON integer 的范围检查，越界则拒绝 |
| `char` / `java.lang.Character` | 校验字符串长度为 1 |
| `float` / `double` | 校验为有限值 |

关键在于这些标量**不只出现在顶层参数**。一个 `long` 可以嵌在 POJO 的属性里、嵌在 List 的元素里、嵌在 `Map<String, T>` 的 value 里，也可以嵌在上述任意组合的多层嵌套中。

因此 Admin 必须**按已发布的 `inputSchema` 递归遍历整棵参数树**完成解码，而不是只处理顶层的几个参数。这一步的工作量明显大于"若干技术转换"，实施计划中应作为独立事项排期（见 §15）。

递归遍历的路径与 §7.3 的递归校验一致，两者可以合并为一次遍历：先校验后解码，或在同一次下降中完成。

解码失败一律在 RPC dispatch 之前拒绝，返回 §9.2 的 `not-executed`。

## 8. MCP 运行时与 RPC 调用

### 8.1 Endpoint 与无状态行为

每个已发布 Server 暴露一个业务 endpoint：

```text
POST /mcp/{serverId}
```

现有运维 endpoint 保持不变：

```text
POST /api/mcp
```

业务 endpoint 刻意不与 `/api/mcp` 共享路径前缀。原因见 §8.2：现有鉴权中间件按精确路径匹配运维 endpoint，任何共前缀的方案都需要改动那段代码，而改动它的主要风险是让运维 endpoint 的静态 API Key 校验被意外绕过。

业务 endpoint 使用官方 MCP Go SDK Streamable HTTP handler：

```go
mcp.StreamableHTTPOptions{
    Stateless:    true,
    JSONResponse: true,
}
```

MVP 不使用 GET/SSE Session、服务端主动请求、Session state 或 revision pinning。

每个 HTTP 请求都读取当前 `MCPServerResource`，并使用当前 Published snapshot。

缓存需要分两层，因为一次 `tools/list` 的结果不再只由已发布配置决定：

- **按 `serverId + published revision` 缓存**：编译后的 `inputSchema` 和 SDK Server object。这些只随 publish 变化。
- **按 Provider metadata 的 `resourceVersion` 缓存**：`inputFingerprint` 的计算结果。它随 Provider 发版变化，与 published revision 无关。

不能只用 `serverId + published revision` 做 key，否则 §6.4 要求的 fingerprint 重算会被缓存掉，契约漂移将检测不到。反过来，如果每个请求都对每个 binding 重新读取并规范化 metadata，在数据库 store 下，一次 `tools/list` 会退化成 N 次索引查询加 N 次规范化。

`LiveServiceIndex` 的状态是进程内的，直接读取即可，不需要额外缓存。

选择任何缓存之前必须先读取权威资源。

如果 Agent 执行 `tools/list` 后发生了 publish，后续 `tools/call` 使用新的 Published revision。Tool 已删除或参数已经不兼容时，Admin 返回配置已更新并要求重新获取 Tool 列表的错误。

由于 MVP 是无状态的，服务端无法主动发送 `notifications/tools/list_changed`。Agent 手中的 Tool 列表可能已经过期，它只能通过下一次 `tools/list` 或一次失败的 `tools/call` 发现这一点。这是 §8.3 动态列表的已知代价。

### 8.2 鉴权

管理 API 继续使用 Admin 现有登录和授权系统。业务 MCP endpoint 使用 5.4 节定义的 Admin 机器凭证。

请求必须包含：

```http
Authorization: Bearer mcp_<credential-id>.<random-secret>
```

鉴权发生在 MCP request handler 之前。Admin 校验 Credential 处于 active 状态、没有过期，并且属于 path 中的 `{serverId}`。失败时直接返回 HTTP 401 或 403，不暴露 MCP Tool 信息。

#### 与现有中间件的关系

现有 [`authMiddleware`](../../pkg/console/component.go) 是全局注册的（`r.Use(c.authMiddleware())`），它按**精确路径相等**判断是否为 MCP 请求：

```go
isMCPRequest := requestPath == "/api/mcp" || (c.mcpPath != "" && requestPath == c.mcpPath)
```

任何不等于运维 endpoint 的路径都会落到会话认证分支，检查 session 中的 `user`。因此业务 MCP endpoint 如果沿用该中间件，机器凭证请求会在到达 MCP handler 之前就被判为未登录并返回 401。

业务 MCP 挂在**独立的 gin RouterGroup** 上，使用自己的凭证中间件，不复用也不修改 `authMiddleware`。这样 §13 要求的"现有 Console browser authentication 保持不变"是结构上保证的，而不是靠 review 保证的。

不采用"把精确匹配改成前缀匹配"的方案：`/api/mcp` 与 `/api/mcp/...` 共前缀，一旦匹配条件写宽，运维 endpoint 的静态 API Key 校验就会被绕过，而这正是最不应该出现回归的地方。

### 8.3 tools/list

`tools/list` 返回当前 Published snapshot 中**通过运行时过滤**的 Tools。MVP 不设置 Tool 数量硬限制，也不分页。

#### 逐 Tool 过滤

Admin 对每个 binding 独立判定，判定结果只影响该 Tool：

| 条件 | 该 Tool 是否出现在 `tools/list` |
| --- | --- |
| definition 缺失、存在冲突，或 `inputFingerprint` 不一致 | 否 |
| `LiveServiceIndex` 为 `INACTIVE`，且持续时间超过宽限期 | 否 |
| `LiveServiceIndex` 为 `INACTIVE`，但仍在宽限期内 | 是 |
| `LiveServiceIndex` 为 `UNKNOWN` | 是 |
| `outputFingerprint` 不一致 | 是（仅在管理页告警） |

任何单个 binding 的问题都不会导致整个请求失败。返回一份少了几个 Tool 的列表，优于返回一个错误——后者会让 Agent 的能力集合整体归零，包括那些完全健康的 Tool。

#### `UNKNOWN` 必须保持显示

这是本节的安全底线。`INACTIVE` 表示"已经同步完成，确认没有实例"；`UNKNOWN` 表示"不知道"。只有前者可以隐藏。

如果 `UNKNOWN` 也隐藏，那么一次注册中心不可用、或者 Admin 刚重启尚未完成初始同步，都会让所有 Agent 的全部能力瞬间消失。故障范围会从"Admin 看不到状态"放大成"所有依赖 MCP 的自动化全部停摆"。

#### 宽限期

只有一到两个实例的应用做滚动重启时，中间存在一段实例数为零的窗口。此时 Nacos 数据完整、同步正常，状态是**确定的 `INACTIVE`**，不是 `UNKNOWN`。

如果立即隐藏，每次例行发版都会让 Agent 的工具短暂消失又出现。正在执行多步任务的 Agent 会认为该能力不存在，转而走完全不同的路径。

因此 `INACTIVE` 必须**连续持续超过宽限期**才触发隐藏，默认 60 秒，可配置。索引需要记录每个 identity 进入 `INACTIVE` 的时刻；状态回到 `ACTIVE` 时该计时清零。

#### 返回内容

每个 Tool 包含：

- 用户定义的 Tool name；
- 用户定义的 Tool description；
- 生成的 `inputSchema`；
- 用户实际配置时才返回的 `readOnlyHint`、`destructiveHint` 和 `idempotentHint`。

MCP Server description 通过 `initialize` 响应中的 `instructions` 字段暴露给 Client。

### 8.4 tools/call

一次 `tools/call` 按以下顺序执行：

1. 校验机器凭证。
2. 读取当前 Published snapshot。
3. 按 Tool name 查找 binding。
4. 重新计算并比较 `inputFingerprint`。
5. 使用保存的 inputSchema 递归校验 arguments。
6. 递归解码标量并恢复有序 Dubbo 参数（§7.4）。
7. 计算调用 context 和可选 timeout。
8. 按 binding 记录的协议执行一次 `GenericService.Invoke`。
9. 只为 JSON 编码需要，把返回的 Go value 转换为等价的 Dubbo 业务结果。
10. 返回 MCP result 并记录 audit event。

整个链路不会重试 RPC。

第 4 步失败时返回契约已变更的错误。存活状态**不在**调用路径上判定：即使某个 Tool 因 `INACTIVE` 已经从 `tools/list` 隐藏，针对它的 `tools/call` 仍然照常进入上述流程，最终由 Dubbo directory 给出结果——没有可用 Provider 时返回 `not-executed`。

这样处理的原因是无状态模式下服务端无法推送列表变更，Agent 手中的列表必然可能过期。返回"该能力暂时不可用"比返回"该能力不存在"更有用：前者 Agent 会稍后重试或如实告知用户，后者可能让它永久放弃这条路径。同时，最终可用性本来就以本次调用使用的 Dubbo directory 为准，而不是以索引的缓存状态为准。

### 8.5 使用正常 Dubbo 路由与负载均衡

MCP runtime 是一个正常 Dubbo Consumer，固定 application name 为：

```text
dubbo-admin-mcp
```

由于一个 MCP Server 绑定单个 discovery（§5.2），长生命周期 generic client manager 为每个用到的 discovery 创建一个 dubbo-go client。Generic service reference 按以下 key 复用：

```text
discoveryId + serviceName + group + version + protocol
```

`protocol` 必须进入复用 key：同一个服务的 `dubbo` reference 和 `tri` reference 是两个不同的对象，不能互相复用。

Reference 使用：

```go
client.WithRegistry(...)
client.WithProtocolDubbo()   // 或 client.WithProtocol(constant.TriProtocol)，按 binding 记录的协议选择
client.WithGroup(group)
client.WithVersion(version)
client.WithClusterFailFast()
client.WithRetries(0)
```

以上均为 `ReferenceOption`。协议按 binding 持久化的取值动态选择，不是写死的。

创建 client 时，除 Nacos registry address 外，还需要按 discovery 配置提供 metadata report / config center 地址。应用级服务发现模式下，dubbo-go consumer 需要解析 `MetadataInfo` 才能完成订阅，只配置 registry address 不足以建立完整链路。

Reference 不使用 `client.WithURL`，也不显式配置 load balancer。

这是有意保留的边界。当前固定 dubbo-go 版本默认 cluster 是 failover，默认 retries 是 2，因此必须显式配置 fail-fast 和 zero retries。默认 load balancer 已经是 random，而且 Dubbo governance 可以改变最终生效的路由策略，因此 Admin 不应重新声明 load balancer 并覆盖正常 Dubbo 策略。

序列化不需要配置。`client.NewGenericService` 内部对两种协议都强制 `WithIDL(NONIDL)`、`WithGeneric()` 和 `WithSerialization(Hessian2Serialization)`。因此 §4.3 提到的现有调试链路"尝试多个序列化目标"的做法在业务 MCP 上不存在。

Nacos 提供 Provider directory。dubbo-go 继续执行自己的 directory、router、governance 和最终生效的 load balancer，选择一个 Provider 并调用一次。Admin 不计算调用顺序，也不实现本地 round-robin。

调用时不根据 `LiveServiceIndex` 中缓存的地址直连 Provider。即使索引刚刚显示 `ACTIVE`，实例也可能在 RPC 前下线；反过来，索引显示 `INACTIVE` 后也可能马上有新实例注册。最终可用性以本次调用使用的 Dubbo directory 为准，没有可用 Provider 时返回 `not-executed` Tool error。

fail-fast cluster 的源码流程是列出有效 invokers，获取最终生效的 load balancer，通过 `DoSelect` 选出一个 invoker，然后调用该 invoker 一次。这就是生产 MCP 调用要求的行为。

这一保证与协议无关。`failfastClusterInvoker.Invoke` 只依赖 `Directory.List`、`GetLoadBalance` 和 `DoSelect`，不涉及任何协议特定分支，cluster 层位于 protocol 层之上。因此 `dubbo` 和 `tri` 两条路径的"最多一次 RPC"语义完全一致。

Client 和 reference 必须是长生命周期对象。每次 `tools/call` 都创建新的 registry client 会反复创建订阅和连接。其生命周期跟随 Console component，并接入 dubbo-go graceful shutdown 流程。

### 8.6 Tool 可选超时

Timeout 是每个 Tool 的可选配置：

- 开关关闭：Admin 不增加 timeout override，使用最终生效的 Dubbo Consumer、method、governance 和 library 默认配置。在当前固定版本的 programmatic client 中，如果没有其他覆盖，Consumer 默认值是 3 秒。
- 开关开启：用户必须填写正数 duration。Admin 写入内部 Dubbo timeout attachment，并创建 call context deadline。如果 MCP Client request context 的 deadline 更短，以更短的 deadline 为准。

Agent 不能覆盖 timeout，也不能传入任意 Dubbo attachments。

#### 两种协议的 timeout 传递不同

- `dubbo`：invoker 读取 timeout attachment，并把它**回写为 attachment 发送给 Provider**，因此 Java Provider 端也能感知本次调用的超时预算。
- `tri`：invoker 把 timeout 放进 context value 传给底层 triple 实现，该处在 dubbo-go 源码中标注为临时方案（`Todo(finalt) Temporarily solve the problem that the timeout time is not valid`）。

因此在 `tri` 协议上，**call context deadline 是主要保障而不是兜底**。实现必须无条件创建 context deadline，不能依赖 attachment 生效。

RPC dispatch 后发生 timeout 或连接断开时，Provider 可能已经执行了操作，只是 Admin 没有收到结果。Admin 返回 `unknown-outcome`，并且不重试。

### 8.7 绑定的 discovery 缺失时的行为

一个 MCP Server 绑定单个 discovery（§5.2）。如果该 discovery 从 Admin 配置中被移除，metadata watcher 会停止，`ServiceProviderMetadataResource` 随之消失，`LiveServiceIndex` 也取不到任何数据。

这种情况必须单独定义，否则 §8.3 的两条规则会给出互相矛盾的结果：存活状态变成 `UNKNOWN`，按规则应当**保持显示**；而 definition 缺失，按 §6.4 应当**逐 Tool fail closed**。最终表现将取决于实现的判定顺序，而不是设计决定的行为。

MVP 的处理分两层：

**前置防护。** 删除 discovery 前校验它是否仍被任何 MCP Server 引用。存在引用时拒绝删除，并在错误中列出引用它的 Server。这是正常路径上唯一应该发生的结果。

**运行时兜底。** 配置仍可能绕过 Console 被直接修改，因此 endpoint 必须有确定行为：Server 绑定的 discovery 不存在时，`/mcp/{serverId}` 的所有请求——包括 `tools/list`——一律返回错误，而不是返回一份正在缩水的 Tool 列表。管理页把该 Server 标记为配置错误。

这里与 §8.3 的逐 Tool 降级采取不同策略是有意的。逐 Tool 降级针对的是单个 Provider 的运行时波动，此时 Server 本身仍然是健康且配置正确的。而 discovery 缺失是**配置错误**，影响该 Server 的全部 binding。返回一份逐渐变空的列表会让 Agent 误以为这些能力被有意收回；整体报错才能让问题立刻暴露给运维。

## 9. 返回值与错误

### 9.1 成功返回

Admin 保留 Dubbo 泛化调用产生的业务结果。MVP 没有用户自定义的 output schema、alias、wrapper、字段删除或字段重命名。

协议边界仍然需要必要的技术转换。Hessian map、list、struct 和 scalar 必须先变成 JSON 兼容的 Go value，MCP SDK 才能编码。这个过程不能改变业务结构。

如果结果是 JSON object，Admin 同时返回：

- 保存该 object 的 `structuredContent`；
- 保存同一份 JSON 文本的 text content，兼容不读取 structured content 的 Client。

如果结果是 scalar、array 或 null，Admin 返回对应 JSON text。MCP structured content 要求根节点是 object。如果额外包装成 `{"result": ...}`，会违反原始返回结构要求。

MVP 不声明 `outputSchema`。SDK 只在设置了 `outputSchema` 时才校验 `structuredContent`，因此不声明是可行的；但部分 Client 会因为没有 `outputSchema` 而忽略 `structuredContent`。同时返回 text content 的设计正是为了覆盖这种 Client，实现时应对目标 Client 实测确认。

#### `long` 的表示是不对称的

§7.2 把入参中的 `long` 映射为十进制 string，用于防止 JSON Client 丢失精度（Java `long` 上界约 9.2e18，而 JavaScript 的安全整数上界约 9e15，19 位的 ID 必然溢出）。

返回方向没有对应处理。Java 返回的 `long` 会以 JSON number 编码输出，同一个精度问题原样存在。

保留这个不对称是有意的取舍。两个方向的后果不对等：入参丢精度会**写错数据**（例如给错误的租户创建订单），返回值丢精度只影响**读取和展示**。修复返回方向需要按返回类型闭包遍历整个结果树并改写标量，这与本节不做输出转换、§3.2 不做 output schema 的边界直接冲突。

由此产生一个 MVP 无法修复的往返缺口：Agent 从某个 Tool 拿到一个 19 位 ID，再把它作为参数传给下一个 Tool 时，精度在第一步就已经丢失，第二步传出去的是错误的值。该缺口记录在 §16。

### 9.2 错误返回

鉴权失败在 MCP 处理前返回 HTTP error。格式错误的 MCP request 继续返回 protocol error。参数校验、契约、Provider 可用性、timeout 和调用失败作为 `isError: true` 的 Tool error 返回。

MVP 返回 Admin 能取得的全部错误信息。可取得的内容按协议不同：

| 字段 | `dubbo` | `tri` |
| --- | --- | --- |
| exception type / message | ✅ | ✅（status code + message） |
| cause chain | ✅ | ❌ |
| stack trace | ✅ | ❌ |
| Dubbo error code | ✅ | 部分（视 Provider 是否在 trailer 中携带） |
| selected Provider address | 能取得时返回 | 能取得时返回 |
| outcome classification | ✅ | ✅ |

`dubbo` 协议通过 hessian 反序列化 Java 异常对象，因此能拿到完整的 cause chain 和 stack trace。`tri` 协议回传的是 connect/gRPC 风格的 status 加 message，结构上不携带 Java 异常的这些细节。

不为了对齐两种协议而砍掉 `dubbo` 侧的信息。本节已经明确选择返回全部可得信息并接受泄露风险，主动降级到两者的交集只会在不减少风险的前提下削弱可诊断性。协议差异在文档中写明即可。

三种 outcome 分类在两种协议上都可判定，这一层是统一的。

Outcome 分为：

| Outcome | 含义 |
| --- | --- |
| `not-executed` | RPC dispatch 前已经拒绝，例如参数非法、标量解码失败、契约缺失或没有可用 Provider |
| `failed` | 收到了确定的 Provider 或 Dubbo error |
| `unknown-outcome` | 可能已经 dispatch，但 timeout、cancel 或连接中断导致结果不确定 |

原始异常可能泄露内部类名、地址和调用栈。MVP 接受这个风险，并在 release notes 中明确说明。

## 10. 存储一致性与多副本

### 10.1 Resource 注册

ResourceStore 会为初始化前已经注册的 Resource schema 创建 store。因此，`MCPServerResource` 和 `MCPCredentialResource` 必须在 package initialization 阶段注册 schema 和查询所需的 indexes。

建议至少提供：

- MCP Server 的 mesh + name index；
- Credential 的 server ID index；
- Credential ID index，业务 endpoint 需要按 token 中的 credential ID 直接定位凭证。

### 10.2 CAS 能力

当前 ResourceStore 没有 expected-version atomic update。Gorm store 会在 update transaction 之前读取旧 row，随后执行 unconditional update，因此 handler 先 read 再 update 不能保证多副本安全。

`ResourceModel` 当前也没有独立的版本列，只有 JSON `data`、`created_at` 和 `updated_at`。实现 CAS 时**只为 `MCPServerResource` 和 `MCPCredentialResource` 两张表增加 `version` 列**，并把它作为这两类资源 `ObjectMeta.resourceVersion` 的存储来源。新资源从版本 1 开始；每次 Add、Update、CAS 或 Delete 都必须在同一个 mutation lock 或 database transaction 中处理版本，不能只修改 JSON 内的字符串。

不给所有资源表加版本列。CAS 只有 MCP 需要，而 `RPCInstance` 这类资源在每次注册中心推送时都会发生 Add / Update / Delete，为其增加版本簿记是没有收益的热路径开销。此外 `ObjectMeta.resourceVersion` 目前在整个仓库中没有任何读取方，一旦对所有资源生效，所有资源的 JSON payload 都会改变，迁移面显著扩大。

增加范围有限的可选 store capability：

```go
type ConditionalResourceStore interface {
    CompareAndSwap(obj model.Resource, expectedVersion string) error
    CompareAndDelete(obj model.Resource, expectedVersion string) error
}
```

MCP repository 对可变资源强制要求这个 capability，其余资源的 store 行为不发生任何变化。

- Memory store 在同一个 write lock 内比较版本并更新资源。
- Gorm store 在同一个 database transaction 中读取当前 row，比较 `version`，生成 `version + 1` 的资源 JSON，再通过包含旧 `version` 条件的 update 写入 data、indexes 和新版本。条件 update 影响 0 行时返回 conflict。
- 版本不一致时返回 typed conflict error，并映射为 HTTP 409。

ResourceStore 是权威数据源。进程内 compiled schema、MCP Server object 和 Dubbo reference 都只是派生缓存。

### 10.3 多副本前提

业务 MCP **要求使用数据库 store**（MySQL 或 PostgreSQL）。

memory store 的数据完全在进程内。多副本部署下，在一个副本上完成的 credential revoke 对其他副本不可见，被撤销的凭证在其他副本上仍然可用；publish 的 CAS 也只在单个进程内互斥，无法阻止两个副本同时发布出互相覆盖的版本。前者是安全问题。

memory store 仅适用于单副本的开发和测试环境。启动时如果检测到 MCP 功能已启用而 store 为 memory 且副本数大于一，应记录明确的警告。

## 11. Console 管理 API

建议增加以下 Console endpoints：

| Method | Path | 用途 |
| --- | --- | --- |
| GET | `/api/v1/mcp/servers` | 查询已配置的 MCP Servers |
| POST | `/api/v1/mcp/servers` | 创建包含 editable draft 的 Server |
| GET | `/api/v1/mcp/servers/{serverId}` | 查询 Draft、Published snapshot、revision、warnings 和不含 secret 的 Credentials |
| PUT | `/api/v1/mcp/servers/{serverId}/draft` | 使用期望 `resourceVersion` 替换 Draft |
| POST | `/api/v1/mcp/servers/{serverId}/validate` | 校验当前 Draft，不发布 |
| POST | `/api/v1/mcp/servers/{serverId}/publish` | 校验并原子发布 Draft |
| DELETE | `/api/v1/mcp/servers/{serverId}` | 使用期望 `resourceVersion` 删除 Server，并级联删除其全部 Credential |
| POST | `/api/v1/mcp/servers/{serverId}/credentials` | 创建命名 Credential，并只返回一次明文 Token |
| GET | `/api/v1/mcp/servers/{serverId}/credentials` | 查询 Credential metadata |
| DELETE | `/api/v1/mcp/servers/{serverId}/credentials/{credentialId}` | revoke Credential |

方法选择尽量复用现有 Service 和 Method detail APIs。如果现有 response model 无法为 Draft editor 提供完整的精确签名，可以增加一个返回规范化 operation candidate 的小型 endpoint。

基础管理 UI 应提供 Server description、discovery 选择、精确方法选择（含 `LiveServiceIndex` 状态显示和调用协议选择）、Tool name 和 description、顶层参数 alias 和 description、可选 annotations、timeout switch、validation result（errors 与 warnings 分开展示）、publish action，以及 Credential 创建和 revoke。已发布 Tool 列表需要显示当前是否因 `INACTIVE` 被隐藏，以及 `outputFingerprint` 是否已漂移。MVP 不提供嵌套 DTO editor。

## 12. 审计日志

每次业务 `tools/call` 写入一条 structured audit event，包含：

- timestamp 和 request ID；
- Server ID 和 Published revision；
- Credential ID 和 Credential name；
- Tool name；
- Dubbo Service identity 和精确方法签名；
- 本次调用使用的协议；
- 能够取得时的 selected Provider address；
- latency；
- outcome 和 status；
- error class。

Audit event 不记录 request arguments、result values、plaintext token 或 token hash。

### 12.1 控制面审计

除调用审计外，以下控制面操作同样必须写入 structured audit event：

| 事件 | 记录内容 |
| --- | --- |
| publish | 操作者、Server ID、新旧 revision、本次快照中的 Tool 数量与名称列表、发布时的 warnings |
| Credential 创建 | 操作者、Server ID、Credential ID 和 name、过期时间 |
| Credential revoke | 操作者、Server ID、Credential ID 和 name |
| Server 删除 | 操作者、Server ID、级联删除的 Credential ID 列表 |

本功能的核心价值是受控地把内部能力暴露给 Agent，因此"谁在什么时候开放了什么、给谁发了凭证、什么时候撤销"比"谁调用了什么"更接近安全审计的关注点。只审计 `tools/call` 会留下无法回溯的权限变更历史。

同样不记录 plaintext token 或 token hash。

MVP 通过现有 structured application logging path 写入这些字段。专用的可查询 audit store 留到后续版本。

## 13. 模块边界

实现应保持以下职责边界：

- Nacos discovery 继续负责导入 `ServiceProviderMetadataResource`。
- `LiveServiceIndex` 负责把注册中心运行状态归约为三态存活视图，只对外提供判定结果，不做实例选择。
- Console service layer 负责 Draft validation、publish、Credential 和业务 MCP handler。
- ResourceStore 负责持久化配置和 optimistic concurrency。
- dubbo-go 负责 registry subscription、routing、load balancing、serialization 和 Provider selection。
- Provider generic filter 负责 POJO realization。
- Agent 或 MCP Client 负责人工确认策略。

不需要新增 runtime component type。`pkg/mcp` 虽然在 [`pkg/core/bootstrap/init.go`](../../pkg/core/bootstrap/init.go) 中被 blank import 从而注册了自己的 component，但 [`bootstrap.go`](../../pkg/core/bootstrap/bootstrap.go) 只按固定的具名列表启动 EventBus、ResourceStore、ResourceDiscovery、ResourceEngine、ResourceManager、Console 和 RuleGovernor，MCP component 从不会被启动——现有 `/api/mcp` 路由实际由 Console component 注册。业务 MCP managers 同样应随 Console component 创建和停止。

以下现有行为必须保持不变：

- `/api/mcp` 运维 Tools；
- `/api/v1/service/generic/invoke` 调试调用语义；
- 现有 Console browser authentication；
- 现有 discovery 和 governance resources。

## 14. 测试与验收

### 14.1 单元测试

Metadata 和 Schema 测试至少覆盖：

- 按精确有序参数类型解析重载方法；
- Java 参数名缺失时生成默认 `argN`；
- alias 到 Position 的参数恢复；
- primitive、enum、array、collection、map、POJO 和 recursive reference；
- 依据 `Type.items` 元数区分 collection 与 map；
- 含泛型参数的类型名生成合法且稳定的 `$defs` key，且不与其他类型碰撞；
- `$defs` key 的转义按白名单实现：Java 泛型名的 `<>,` 与 Go import path 的 `/` 都被同一条规则覆盖，不存在只处理其中一类的实现；
- 同一接口的 Java 与 Go definition 因 property 名不同（`id` / `iD`）被判定为契约冲突，错误信息点明命名差异；
- 拒绝 unresolved type 和 open type；
- 拒绝顶层和嵌套 POJO unknown fields；
- 允许 `Map<String, T>` dynamic keys，并校验 value；
- 顶层参数 required 和嵌套字段 optional；
- 嵌套在 POJO、List 和 Map value 中的 `long` 被正确解码为 int64，溢出被拒绝；
- 嵌套的 `byte`/`short`/`int` 范围检查和 `char` 长度检查在 dispatch 前拒绝非法值；
- `inputFingerprint` 和 `outputFingerprint` 都不受 map iteration order 影响；
- 仅返回类型变化时 `inputFingerprint` 不变、`outputFingerprint` 变化；
- 入参类型变化时 `inputFingerprint` 变化；
- metadata mismatch 和 conflicting definition 使**该 Tool** fail closed，其余 Tool 不受影响。

Storage 和 Credential 测试至少覆盖：

- Draft 修改不影响 active snapshot；
- validation 失败时保留旧 Published snapshot；
- 一次 atomic publish 更新完整 revision；
- memory 和 Gorm store 都会拒绝过期 `resourceVersion`；
- 非 MCP 资源的 store 行为与增加 CAS 之前完全一致；
- 一个 Server 存在多个 Credentials；
- plaintext token 只返回一次，hash verification、expiration 和 revoke 正常；
- Credential 不能跨 Server 使用；
- 删除 Server 会级联删除其全部 Credential。

### 14.2 Runtime 测试

Runtime 测试必须证明：

- 每个请求读取当前 Published revision；
- list 和 call 之间发生 publish 时，必要情况下返回 relist error；
- 单个 binding 的 `inputFingerprint` 漂移只让**该 Tool** 从 `tools/list` 消失，同 Server 其余 Tool 正常返回；
- `outputFingerprint` 漂移不影响 `tools/list` 和 `tools/call`，只产生管理页告警；
- `LiveServiceIndex` 能区分 `ACTIVE`、`INACTIVE` 和 `UNKNOWN`；
- 状态为 `UNKNOWN` 时（初始同步未完成或 discovery 故障）Tool **保持**在 `tools/list` 中；
- 状态为 `INACTIVE` 但未超过宽限期时 Tool 保持在 `tools/list` 中；
- 状态为 `INACTIVE` 且超过宽限期后 Tool 从 `tools/list` 消失，恢复 `ACTIVE` 后重新出现且计时被重置；
- 接口级注册模式下的 binding 能被正确判定为 `ACTIVE`；
- 被 Nacos 判为 `Healthy=false`、`Enable=false` 或 `Weight<=0` 的实例不会让接口判定为 `ACTIVE`；
- 已被隐藏的 Tool 若仍被 `tools/call` 调用，返回 `not-executed` 而非 tool-not-found；
- Server 绑定的 discovery 不存在时，`tools/list` 和 `tools/call` 整体返回错误，而不是返回一份逐渐变空的列表；
- 删除仍被 MCP Server 引用的 discovery 会被拒绝，错误中列出引用者；
- reference 没有配置 Provider direct URL；
- Dubbo directory 和 router 的结果进入最终生效的 load balancer；
- fail-fast 加 `retries=0` 只产生一次 RPC attempt，`dubbo` 和 `tri` 各验证一次；
- Admin 没有本地 round-robin 或 fallback；
- timeout 和不确定的 transport failure 返回 `unknown-outcome`，并且不重试，`dubbo` 和 `tri` 各验证一次；
- `tri` 上仅依靠 context deadline 也能正确超时；
- object、array、scalar 和 null 保持原始业务结构；
- `dubbo` 的 error response 包含 cause chain 和 stack trace，`tri` 的 error response 包含 status code 和 message；
- 三种 outcome 分类在两种协议上都能正确判定；
- audit logs 不包含 arguments、results 和 secrets；
- publish、credential 创建与 revoke 都产生控制面 audit event。

### 14.3 端到端验收

端到端环境包含：

- 同时作为 Naming Service 和 metadata report 的 Nacos；
- 一个 Apache Dubbo Java Provider，暴露重载方法和嵌套 POJO，并**同时导出 `dubbo` 和 `tri` 两种协议**；
- 一个只导出 `tri` 的 Java Provider，用于验证协议选择；
- 使用当前固定 dubbo-go Consumer、并配置数据库 store 的 Dubbo Admin；
- 使用已保存 Bearer Credential 的 MCP Client。

验收流程如下：

1. 在 Admin 中发现 Java service definition。
2. 等待 `LiveServiceIndex` 把该 Operation 标记为 `ACTIVE`，确认 UI 上能区分三种状态。
3. 创建 Draft，选择精确 Operation，填写语义描述和 aliases，确认协议默认选中 `dubbo` 且可切换到 `tri`。
4. Validate 并 Publish。
5. 创建两个 Credentials，分别调用，revoke 其中一个，验证只有被 revoke 的 Credential 失败。
6. 验证 `tools/list` 返回 Published schema。
7. 验证合法参数通过正常 Dubbo routing 调用一个 Provider，`dubbo` 和 `tri` 各验证一次。
8. 下线全部 Provider：验证宽限期内 Tool 仍在 `tools/list`；超过宽限期后 Tool 从 `tools/list` 消失；此时仍然发起 `tools/call`，验证返回 `not-executed`。重新上线后验证 Tool 回到列表。
9. 模拟 discovery 不可用，验证状态为 `UNKNOWN`，**已发布 Tool 仍在 `tools/list`**，发布新 binding 时给出 warning 但不阻塞。
10. 把应用升级为不再导出目标接口的版本，但保留其他接口以使实例继续在线，同时不删除 Config Service 中该接口的旧 definition。验证 Tool **不会**被隐藏（MVP 已知缺口，见 §16），调用返回 `not-executed`。
11. 验证未知嵌套字段在 RPC dispatch 前被拒绝。
12. 验证嵌套在 POJO 与 List 中的 19 位 `long` 能以 string 正确传入并被 Provider 收到完整精度。
13. 修改 Provider 的**入参** DTO，验证该 Tool 从 `tools/list` 消失、同 Server 其余 Tool 正常，直到重新 Publish。
14. 修改 Provider 的**返回值** DTO，验证 `tools/list` 和 `tools/call` 均不受影响，仅管理页出现告警。
15. 尝试为只导出 `tri` 的服务选择 `dubbo` 协议，验证发布被拒绝并给出明确原因。
16. 制造响应 timeout，验证没有第二个 Provider 收到调用，`dubbo` 和 `tri` 各验证一次。
17. 启动两个 Admin 副本共用同一个数据库：在副本 A 上 revoke 一个 Credential，验证该凭证在副本 B 上立即失效；两个副本用同一个过期 `resourceVersion` 并发 publish，验证只有一个成功。

## 15. 建议实施顺序

1. 增加 MCP Resource schema、indexes、CAS storage（仅 MCP 两张表）和 repository tests。
2. 抽取精确 Operation resolver，并实现 `inputFingerprint` 与 `outputFingerprint` 的规范化计算。
3. 实现 metadata-to-schema 转换和 recursive validation，含 `$defs` key 生成。
4. 实现**递归标量解码**（`long` string→int64、整数范围检查、`char` 长度检查，覆盖 POJO / collection / map value 的任意嵌套）。这一步的工作量明显大于其名称暗示的规模，应独立排期，不要并入第 3 步。
5. 增加 `LiveServiceIndex`（MVP 便宜版）：补齐接口级注册订阅、实例健康过滤、三态与宽限期计时。不含实例 revision resolver。此步与第 1–4 步无依赖，可并行。
6. 增加 Draft、Validate、Publish 和 Credential Console APIs。
7. 增加官方 MCP SDK handler、独立 RouterGroup 和 machine credential middleware。
8. 增加基于 registry 的 generic client manager，按 binding 协议动态选择 `dubbo` 或 `tri`，显式配置 fail-fast 和 zero retries。
9. 实现 `tools/list` 的逐 Tool 过滤（fingerprint 漂移与 `INACTIVE` 隐藏）和两层缓存。
10. 增加 result、error mapping（按协议分别处理）和 structured audit logs（含控制面事件）。
11. 增加基础管理 UI。
12. 执行 MCP conformance、Java Provider integration（两种协议）和 multi-replica storage tests。

## 16. MVP 已接受的风险与后续工作

第一版接受以下风险：

| 风险 | MVP 处理方式 |
| --- | --- |
| Provider binary 与陈旧 metadata 可能不一致 | 要求 Provider publication gate，并对 Admin 能观察到的入参 mismatch 逐 Tool fail closed |
| 原始异常可能泄露内部信息 | 按已确认要求返回，并明确记录风险 |
| Tool 数量过多可能消耗大量 Agent context | 暂不设置硬限制或分页 |
| 没有限流和并发限制 | 暂时依赖部署环境控制 |
| metadata 不包含嵌套 required 信息 | 嵌套字段作为 optional，由 Provider business validation 处理 |
| timeout 可能造成不确定业务结果 | 返回 `unknown-outcome`，不重试 |
| 没有 revision history 或 Session pinning | 每个请求读取当前 Published snapshot，变更后要求重新 list |
| 应用仍在线但新版本已移除某接口时，Tool 不会被隐藏 | 不做实例 revision 校验；调用返回 `not-executed`，误判方向是"少隐藏"而非"误隐藏" |
| 滚动发布期间实例数归零，Tool 可能短暂消失 | 宽限期默认 60 秒，可配置；缩短宽限期会放大该风险 |
| 无状态模式下无法推送 `tools/list_changed` | Agent 手中的列表可能过期，只能通过下一次 list 或一次失败的 call 发现 |
| `long` 往返精度丢失 | 入参用 string 保护写入方向；返回值仍为 JSON number，Agent 把返回的 19 位 ID 再传给下一个 Tool 时会传出错误的值 |
| `tri` 协议的错误信息不含 Java cause chain 和 stack trace | 按协议分别定义错误契约，文档写明差异 |
| `tri` 协议的 timeout attachment 在 dubbo-go 中是临时方案 | 无条件创建 context deadline 作为主要保障 |
| 接口级与应用级注册的存活判定走不同数据路径 | 两条路径分别实现，任一路径有缺陷都会让该注册模式下的整类 Tool 被误隐藏；§14.2 对两种模式分别设置用例 |
| memory store 下多副本的 revoke 不生效 | 业务 MCP 强制要求数据库 store，启动时对错误配置告警 |
| 绕过 Console 直接删除仍被引用的 discovery | Console 侧前置拒绝；运行时该 Server 整体返回错误并在管理页标记为配置错误（§8.7） |
| 同一接口同时存在 Java 和 Go Provider 时无法发布 binding | Go 侧 property 名是 wire name（`iD`），与 Java 的 `id` 不同，§6.2 规则 7 会判定契约不一致。fail closed 是正确的——两边 wire 契约确实不同，生成任一份 schema 都会让另一半调用失败。校验错误信息必须点明是 Java/Go property 命名差异，否则混合部署者无法自行定位 |
| dubbo-go Provider 发布了 Go 原生类型词汇 | Admin 不做 Go 类型识别，schema 生成会失败或产生错误结构；变参方法还会在 dubbo-go 自身的 realize 路径上静默出错。发布侧必须遵循 §7.2 的 Java 词汇要求 |

后续版本可以考虑实例 revision 精确校验、嵌套字段语义、output schema、result redaction、Credential scope、rate limit、Tool pagination、revision history、Test Call、跨 discovery 组合、ZooKeeper discovery、Triple IDL，以及 Java/Go 混合 Provider 的 property 名归一。

## 17. 最终决策摘要

MVP 是建立在 Dubbo 现有结构 metadata 之上的精选语义层：

- 用户选择精确方法，而不是整个 interface；一个 MCP Server 绑定一个 discovery。
- 用户填写 Server 和 Tool descriptions，并可为顶层参数配置 aliases 和 descriptions。
- 用户为每个 Tool 选择调用协议，`dubbo` 和 `tri` 都支持，协议持久化在 binding 中。
- Admin 一次性原子发布一个完整版本。
- MCP 保持 stateless，每个请求读取当前 Published revision。
- Machine credentials 由 Admin 签发和校验，MCP Client 自动携带；业务 endpoint 使用独立路由与独立中间件，不触碰现有 Console 认证。
- 契约 fingerprint 按方向拆分：入参结构漂移让该 Tool fail closed，返回值结构漂移只告警。
- `LiveServiceIndex` 根据注册中心给出 `ACTIVE`、`INACTIVE`、`UNKNOWN`。确认下线（`INACTIVE` 且超过宽限期）的 Tool 从 `tools/list` 隐藏；状态未知（`UNKNOWN`）时一律保持显示。
- 所有降级都是逐 Tool 的。单个 binding 的问题不会让整个 Server 的能力集合归零。
- 存活状态不作为发布门禁，只产生 warning；最终可用性以调用时的 Dubbo directory 为准。
- Provider routing 和 load balancing 由 Dubbo 执行，不由 Admin 实现。
- `fail-fast` 和 `retries=0` 保证一次 RPC attempt，该语义与协议无关。
- 返回值保持 Dubbo 业务结构。
- 人工确认由 Agent 负责。

这个边界充分使用 Dubbo 已经上报的数据，只增加缺失的语义配置，同时保留 Dubbo 原有的运行时职责。
