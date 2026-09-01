# Admin Router 规则与 Dubbo-Go / Dubbo-Java 对齐契约

本文定义 Dubbo Admin 下发 Router 规则时必须遵守的外部配置契约。目标是让同一份规则能够被 Dubbo-Go 和 Dubbo-Java 的动态配置中心正确订阅、解析和删除；Admin 内部的 `Resource`、`mesh` 和 HTTP API 不是外部协议的一部分。

> 本文约束的是配置中心中最终可见的 **rule key、group 和 YAML 内容**。它同时是跨语言共享规则的基础，但某个 Router 的最终路由语义仍取决于各语言 SDK 实际支持的版本和能力。
>
> Dubbo-Java 现状以本地源码 `D:\environment\github\dubbo-go\_research-dubbo-java` 为核对基线。

## 1. 下发链路与责任边界

```text
Admin Console API
  -> ConditionRoute / TagRoute / AffinityRoute / ScriptRoute Resource
  -> EncodeRule：内部 proto 转公共 YAML
  -> RuleGovernor
  -> ZooKeeper 或 Nacos 配置中心
  -> Dubbo-Go / Dubbo-Java DynamicConfiguration 监听 rule key
  -> RouterChain 更新路由状态
```

- `mesh` 仅用于 Admin 区分资源所属的配置中心实例；它不拼入 rule key，也不会出现在 YAML 中。
- `ruleName` 必须等于 `key + router suffix`。Admin 在写入前执行这一校验，不能依赖 consumer 容错。
- `configVersion` 目前仅允许 `v3.0` 或 `v3.1`。
- Admin 只写公共 YAML，禁止把内部 protobuf JSON 直接写入配置中心。

## 2. 配置中心定位规则

所有 Router 使用相同的外部定位方式：

| 配置中心 | 规则位置 |
| --- | --- |
| ZooKeeper | `/dubbo/config/dubbo/<ruleName>` |
| Nacos | `DataId=<ruleName>`，`Group=dubbo` |

例如服务级 Condition Rule：

```text
ruleName = org.apache.demo.DemoService:1.0.0:demo.condition-router

ZooKeeper: /dubbo/config/dubbo/org.apache.demo.DemoService:1.0.0:demo.condition-router
Nacos:     DataId=org.apache.demo.DemoService:1.0.0:demo.condition-router, Group=dubbo
```

Admin 创建 ZooKeeper 规则时必须先确保 `/dubbo/config/dubbo` 存在；更新时若节点不存在，应按创建处理。历史上的 `/dubbo/config/<ruleName>` 可回读兼容，但新规则不得再写入该旧路径。

## 3. Rule key 与 Dubbo-Go / Dubbo-Java 订阅 key 对齐

| Router | `scope` | `key` 与 `ruleName` | 运行时订阅方式 |
| --- | --- | --- | --- |
| Condition | `service` | `key=<interface>:<version>:<group>`；`ruleName=<key>.condition-router` | `url.ColonSeparatedKey() + ".condition-router"` |
| Condition | `application` | `key=<provider-application>`；`ruleName=<key>.condition-router` | provider application + `.condition-router` |
| Tag | `application` | `key=<provider-application>`；`ruleName=<key>.tag-router` | provider application + `.tag-router` |
| Affinity | `service` | `key=<interface>:<version>:<group>`；`ruleName=<key>.affinity-router` | `url.ColonSeparatedKey() + ".affinity-router"` |
| Affinity | `application` | `key=<provider-application>`；`ruleName=<key>.affinity-router` | provider application + `.affinity-router` |
| Script | `application` | `key=<provider-application>`；`ruleName=<key>.script-router` | provider application + `.script-router` |

服务 key 必须使用 Dubbo-Go `URL.ColonSeparatedKey()` 与 Dubbo-Java service unique name 的等价结果，保留 interface、version、group 的分隔符；即使 version 或 group 为空，也不能在 Admin 中自行删减分隔符或改写格式。

Tag 和 Script 不支持 service scope。Script Router 只在 consumer 侧运行。

## 4. 三方契约对齐状态

标记说明：

- `✅ 三方已对齐`：Admin 下发契约、Dubbo-Go 消费契约、Dubbo-Java 当前源码消费契约一致。
- `⚠️ 部分对齐`：rule key 或字段结构一致，但版本约束、脚本语义或边界行为尚未完全冻结。
- `⏳ 未验证`：代码/协议可推断一致，但尚未做真实配置中心或 Java/Go 双 consumer 共同验收。

| 契约项 | dubbo-admin 现状 | Dubbo-Go 现状 | Dubbo-Java 现状 | 三方状态 | 依据与边界 |
| --- | --- | --- | --- | --- | --- |
| 配置中心定位 | ZooKeeper 写 `/dubbo/config/dubbo/<ruleName>`；Nacos 使用 `DataId=<ruleName>`、`Group=dubbo` | 默认规则 group 为 `dubbo`，按 rule key 监听配置中心 | `DynamicConfiguration.DEFAULT_GROUP=dubbo`，ZK/Nacos 配置中心测试也按默认 group 监听 rule key | ✅ 三方已对齐 | 配置位置、DataId/key 和默认 group 是公共前提。 |
| Condition rule key | service：`<interface>:<version>:<group>.condition-router`；application：`<provider-application>.condition-router` | service 使用 `URL.ColonSeparatedKey()+.condition-router`；application 使用 provider application + `.condition-router` | `ListenableStateRouter` / `ProviderAppStateRouter` 使用同一 `.condition-router` suffix 和默认 group | ✅ 三方已对齐 | key、scope 与 suffix 规则一致。 |
| Condition v3.0 / v3.1 YAML | v3.0 使用字符串数组 `conditions`；v3.1 使用结构化 `from/to/weight` | 同时具备 v3.0 字符串条件和 v3.1 结构化条件解析路径 | `ConditionRuleParser` 支持当前 v3.0/v3.1 规则模型 | ✅ 三方已对齐 | 对当前 `v3.0` / `v3.1` 成立；新增版本前需三方同步适配。 |
| Condition CRUD 与实际路由 | Admin 已有创建、更新、删除、回读和 YAML codec | Dubbo-Go 有监听、解析和 RouterChain 执行路径 | Dubbo-Java 有同 key 的 StateRouter 消费路径 | ⏳ 未做三方验证 | Admin→Dubbo-Go 主路径可验；还缺 Java 和 Go 同时消费同一条 rule 的双端 E2E。 |
| Tag rule key 与基础 YAML | application scope，写 `<provider-application>.tag-router`，管理 `force/runtime/enabled/priority/key/tags` | 应用级 Tag Router 使用 application + `.tag-router`，有 tag cache 和匹配逻辑 | `TagStateRouter` 使用 `.tag-router`，`TagRuleParser` 解析同类 TagRule 模型 | ✅ 三方已对齐 | 定位和基础字段结构一致；Admin YAML 不携带 `scope` 不阻断消费。 |
| Tag 复杂匹配语义 | Admin 负责 YAML 编解码与下发，不执行运行时匹配 | Dubbo-Go 有 tag、地址/实例选择、force 等运行时逻辑 | Dubbo-Java 有 tag、addresses、match 和优先级相关逻辑 | ⚠️ 部分对齐 | 字段结构一致；多实例、标签缺失、地址通配、优先级等边界需要 Java/Go 共用规则 E2E。 |
| Affinity rule key 与 YAML | service/application + `.affinity-router`，字段为 `affinityAware.key/ratio` | 使用同名 key，解析并执行 `affinityAware`，校验 ratio 范围 | `AffinityListenableStateRouter` 使用 `.affinity-router`，Java 注释和模型指向 `configVersion: v3.1` | ⚠️ 部分对齐 | key 和 v3.1 字段结构一致；Admin 当前全局允许 `v3.0`/`v3.1`，但 Java Affinity 只明确接收 `v3.1`，建议 Admin 对 Affinity 固定 `v3.1`。 |
| Script rule key 与字段 | application + `.script-router`，仅允许 application scope、`javascript`、非空脚本且不超过 64 KiB | 使用 provider application + `.script-router`，由 consumer 侧 JavaScript Router 执行 | `AppScriptStateRouter` 使用 `.script-router`，解析 `key/type/script` 后由 Java 运行时执行 | ⚠️ 部分对齐 | rule key 和基础字段一致；Java 脚本可访问 Java 对象，Go 使用自身 JS 引擎和 invoker，对象 API 不承诺一致。 |
| Nacos 下发、回读、监听 | 已有 Nacos Governor/watcher，按 `DataId=<ruleName>`、`Group=dubbo` 操作 | 有 Nacos 动态配置监听实现 | 有 Nacos 动态配置消费实现，默认 group 为 `dubbo` | ⏳ 未验证 | 还需要真实 Nacos create/update/delete 和 Java/Go 双端消费 E2E。 |
| 实例级禁流 | 可通过路由规则表达地址/条件选择，但没有独立冻结的“实例开关”公共契约 | 需由 Condition/Tag/Affinity 等具体 Router 语义体现 | 需由 Condition/Tag/Affinity 等具体 Router 语义体现 | ⚠️ 部分对齐 | 不能把删除规则后回退当成完整实例禁流；需要先定义公共规则和验收口径。 |

### 源码依据

以下依据只证明 key、group、字段模型和当前解析路径，不证明所有运行时边界语义已经跨语言完全一致。

- `dubbo-admin`：`pkg/common/constants/rule.go` 定义 `RuleConfigGroup=dubbo` 以及 `.condition-router`、`.tag-router`、`.affinity-router`、`.script-router` suffix；`pkg/governor/zk/governor.go` 写入 `/dubbo/config/dubbo/<ruleName>`；`pkg/governor/nacos2/governor.go` 使用 `DataId=r.ResourceMeta().Name`、`Group=constants.NacosConfigGroup`；`pkg/core/resource/apis/mesh/v1alpha1/rule_codec.go` 负责 Condition/Tag/Affinity/Script 的外部 YAML 编解码和校验。
- `Dubbo-Go`：`common/constant/key.go` 定义同名 Router suffix 和 RouterFactory key；`cluster/router/condition/dynamic_router.go` 使用 `ColonSeparatedKey()+.condition-router` 和 provider application + `.condition-router`；`cluster/router/tag/router.go`、`cluster/router/tag/cache.go` 使用 application + `.tag-router`；`cluster/router/affinity/router.go` 使用 service/application + `.affinity-router` 和 `affinityAware`；`cluster/router/script/router.go` 使用 provider application + `.script-router`；`global/router_config.go` 定义 Condition v3.1 与 Affinity `affinityAware` 结构。
- `Dubbo-Java`：`dubbo-common/src/main/java/org/apache/dubbo/common/config/configcenter/DynamicConfiguration.java` 定义 `DEFAULT_GROUP=dubbo`；`dubbo-cluster/src/main/java/org/apache/dubbo/rpc/cluster/router/condition/config/ListenableStateRouter.java`、`ProviderAppStateRouter.java` 使用 `.condition-router`；`TagStateRouter.java` 使用 `.tag-router`；`AffinityListenableStateRouter.java` 使用 `.affinity-router` 且注释/模型指向 `configVersion: v3.1`、`affinityAware`；`AppScriptStateRouter.java` 使用 `.script-router`；`dubbo-configcenter-zookeeper`、`dubbo-configcenter-nacos` 测试覆盖默认 group 下 rule key 的监听。

## 5. 公共 YAML 契约

### 5.1 Condition Router v3.0

`v3.0` 的 `conditions` 是字符串数组。

```yaml
configVersion: v3.0
priority: 0
enabled: true
force: false
runtime: true
scope: service
key: org.apache.demo.DemoService:1.0.0:demo
conditions:
  - method=SayHello => application=demo-provider
```

`configVersion` 小于 `v3.1` 时，Dubbo-Go / Dubbo-Java 均按字符串条件路由解析。此时 Admin 不得同时写入结构化的 `from` / `to` 条件。

### 5.2 Condition Router v3.1

`v3.1` 的 `conditions` 是结构化数组；`weight` 的取值范围为 `[0, 100]`。

```yaml
configVersion: v3.1
priority: 0
enabled: true
force: false
runtime: true
scope: service
key: org.apache.demo.DemoService:1.0.0:demo
conditions:
  - from:
      match: method=SayHello
    to:
      - match: application=demo-provider-hangzhou
        weight: 100
      - match: application=demo-provider-shanghai
        weight: 0
```

每条 v3.1 condition 必须包含非空 `from.match` 和至少一个 `to`；每个 destination 必须包含非空 `match` 与合法 `weight`。v3.0 与 v3.1 的 `conditions` 形状不能混用，同一 rule key 在同一时刻只能保存一个版本的配置。

### 5.3 Tag Router

```yaml
configVersion: v3.0
priority: 0
enabled: true
force: false
runtime: true
key: demo-provider
tags:
  - name: hangzhou
    addresses:
      - 10.0.0.10:20880
```

Tag 的 rule key 按 provider application 生成。Dubbo-Go / Dubbo-Java 在接收到配置变更后，以该 key 缓存对应 Tag Router 配置。

### 5.4 Affinity Router

```yaml
configVersion: v3.1
scope: application
key: demo-provider
runtime: true
enabled: true
affinityAware:
  key: region
  ratio: 80
```

外部字段必须叫 `affinityAware`，不能使用 Admin 内部 proto 字段名 `affinity`。`affinityAware.key` 必填，`ratio` 必须在 `[0, 100]` 内。

### 5.5 Script Router

```yaml
configVersion: v3.0
scope: application
key: demo-provider
enabled: true
type: javascript
script: |
  (function route(invokers, invocation, context) {
    return invokers;
  })()
```

当前 Admin 只接受 `application` scope、`javascript` 类型、非空且不超过 64 KiB 的脚本。Admin 不执行脚本；脚本的编译与执行由 consumer 侧 Script Router 完成。由于 Dubbo-Go 与 Dubbo-Java 暴露给脚本的运行时对象不同，本文只冻结 rule key 与基础字段，不承诺同一脚本文本跨语言直接运行。

## 6. Admin 写入与回读要求

1. **创建/更新前校验**：校验版本、scope、非空 key、`ruleName == key + suffix`，以及各 Router 的特定约束。
2. **统一编码**：通过 `EncodeRule` 生成本文件所列 YAML；Condition 必须按 `configVersion` 选择 v3.0 或 v3.1 的 `conditions` 结构。
3. **先配置中心，后本地状态**：以 ZK/Nacos 写入成功为准；Admin 本地 Store 由写入后的同步或 watcher 更新。
4. **无损回读**：通过 `DecodeRule` 将 YAML 还原为对应资源；v3.1 Condition 的 `from`、`to`、`weight` 和 Affinity 的 `affinityAware` 不得丢失或降级。
5. **删除语义**：删除配置中心中的同一 rule key，使 consumer 收到 `Del` 事件并移除/禁用对应动态 Router 状态。

## 7. Dubbo-Go / Dubbo-Java 消费前提与验收

要让规则真正生效，Dubbo-Go 或 Dubbo-Java consumer 必须满足：

1. 已启用相同的 ZooKeeper 或 Nacos 配置中心，并连接到 Admin 所属 mesh 对应的配置中心地址。
2. 配置中心 group 为 `dubbo`，rule key 与本文件第 3 节完全相同。
3. consumer 已启用相应 Router，并已从 provider invoker URL 取得 service key 或 provider application。
4. YAML 与 Router 所支持的 `configVersion`、字段结构一致。

每类 Router 至少执行一次以下 E2E 验收：

```text
Admin create -> 配置中心出现正确 key 与 YAML -> consumer 收到 Add/Update
Admin update -> consumer RouterChain 路由结果改变
Admin delete -> consumer 收到 Del -> 路由恢复为无该动态规则时的行为
```

Condition 必须分别覆盖 v3.0 和 v3.1；v3.1 还需覆盖多个 destination 和 `weight`。Tag、Affinity、Script 分别验证其对应的 application/service scope 限制与路由结果。标为 `✅ 三方已对齐` 的项目至少要求 key、group 和字段结构一致；标为 `⚠️` 或 `⏳` 的项目不得在验收报告中宣称 Java/Go 运行时语义完全一致。

## 8. 实现位置

| 职责 | 位置 |
| --- | --- |
| 后端 REST 路由 | `pkg/console/router/router.go` |
| 请求/响应模型与版本化 Condition 转换 | `pkg/console/model/router_rule.go` |
| 外部 YAML 编解码和写前校验 | `pkg/core/resource/apis/mesh/v1alpha1/rule_codec.go` |
| ZooKeeper 下发路径 | `pkg/governor/zk/governor.go` |
| Nacos DataId / Group 下发 | `pkg/governor/nacos2/governor.go` |
| Dubbo-Go Router 消费实现 | Dubbo-Go 仓库的 `cluster/router/{condition,tag,affinity,script}` |
| Dubbo-Java Router 消费实现 | `D:\environment\github\dubbo-go\_research-dubbo-java\dubbo-cluster\src\main\java\org\apache\dubbo\rpc\cluster\router\{condition,tag,affinity,script}` |

对某条规则做功能变更时，应先更新此契约及对应 codec 测试，再验证真实配置中心内容以及 Dubbo-Go / Dubbo-Java 的 RouterChain 结果。
