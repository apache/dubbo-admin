# Router 规则下发链路任务留痕

## 任务信息

- 日期：2026-08-13
- 分支：`feature/admin-router-rule-chain`
- Issue：`[Admin Feature] Align router in Dubbo-Go on Dubbo-Java version and Dubbo-Admin #1523`
- 需求来源：`agent.md`、`TODO.md`
- 分支策略：用户明确要求在当前分支完成，不新建分支。

## 背景与目标

当前分支已经修复 Service Argument Route upsert、Condition/Tag 规则命名及部分 Zookeeper 标准路径和配置中心地址问题。本任务继续以 `develop` 为基线补齐其余链路，使 Admin 创建、更新、删除的 Router 规则能够按 Dubbo 外部 YAML 契约写入 Zookeeper/Nacos，被 Admin watcher 无损回读，并覆盖 Affinity、Condition v3.1 和 Script Router 的最小可用管理能力。

## 实现边界

1. 公共链路：统一规则 codec；Zookeeper 标准 group、父节点、legacy 只读兼容和根节点过滤；ConfigCenter 地址；Nacos/Zookeeper 一致的外部 YAML。
2. Affinity：资源接线、`affinityAware` codec、校验、Console CRUD、ZK/Nacos watcher 与最小前端 CRUD。
3. Condition v3.1：v3.0 字符串与 v3.1 结构化 conditions 并存，无损 codec、校验、Console API 和 YAML 编辑/展示。
4. Script：proto/resource、codec、校验、Console CRUD、ZK/Nacos watcher、最小前端 CRUD。
5. 验证：单元测试、前端测试与构建、仓库格式检查、Zookeeper/Nacos 及 dubbo-go RouterChain 端到端验证；保存运行证据和截图。

本期遵循需求限定：先打通核心链路；性能、重试、可靠性增强及 legacy 主动迁移不在本次范围，Admin 不执行用户脚本。

## 关键设计

- 以 `ResourceKind` 分派统一 codec，内部 proto/resource 不直接作为外部配置中心契约。
- Condition 根据 `configVersion` 分派 v3.0/v3.1 模型；任何回读和再次发布不得跨版本隐式转换。
- 规则名、scope/key、ratio/weight、脚本类型/大小等在 ResourceManager 写入前校验。
- Zookeeper 新写入固定为 `/dubbo/config/dubbo/<rule-key>`；legacy 仅回读，不随新规则写入或删除而清理。
- 前端优先提供完整 YAML 路径；已有 v3.0 表单保持兼容，结构化 v3.1 不进入字符串表单解析。

## 风险与测试策略

- proto 变更：运行生成流程并检查生成文件一致性。
- v3.0 回归：Condition/Tag/Configurator 现有测试及新增 codec round-trip。
- watcher 删除语义：覆盖 Add/Update/Delete 与根节点过滤测试。
- API 契约：覆盖非法输入在 governor 调用前失败、合法资源 CRUD。
- 前端：组件单测、类型检查、lint、build，并验证 YAML create/update/detail。
- E2E：分别记录配置中心真实内容、Admin 回读、dubbo-go 配置事件和最终路由结果。

## 实现结果

### 公共下发与回读

- Service Argument Route 改为 Create/Update upsert，并覆盖首次创建、已有规则更新和 nil Spec。
- Condition service/application 与 Tag application 规则名按 Dubbo 外部契约生成。
- 新增统一 `EncodeRule`、`DecodeRule`、`ValidateRule`，Zookeeper 与 Nacos 不再直接暴露内部 proto YAML。
- Zookeeper 新写入固定为 `/dubbo/config/dubbo/<rule-key>`，自动创建父节点；标准节点不存在时 Update 会完成创建。
- Zookeeper Governor 与配置 watcher 使用 `Address.ConfigCenter`；保留 Registry 回退和 legacy 只读兼容，不主动迁移或删除 legacy 节点。
- Nacos 初始化失败会立即返回，避免后续客户端初始化覆盖原始错误。

### Affinity、Condition v3.1 与 Script

- Affinity 已接入资源集合、codec、校验、Console CRUD、版本历史/差异/回滚、ZK/Nacos watcher 与前端列表/YAML 编辑器。
- Condition 以 `configVersion` 分派 v3.0 `[]string` 和 v3.1 结构化 `from/to/weight`，API、codec 与 YAML 页面保持无损；显式保留 `weight: 0`。
- Script 已新增 proto/resource、codec、校验、Console CRUD、版本历史/差异/回滚、ZK/Nacos watcher 与前端脚本 YAML 编辑器；只允许 application scope、`javascript` 和非空且受大小限制的脚本。
- Affinity/Script URL 中规则名统一编码，写操作带 author/reason 等 mutation options。

## 自动化验证记录

在 `D:/environment/github/dubbo-go/dubbo-admin` 执行：

```powershell
& "D:\tools\Git\bin\bash.exe" -lc 'export GOROOT="D:/tools/go"; export PATH="/c/Users/57512/go/bin:$PATH"; cd /d/environment/github/dubbo-go/dubbo-admin && make fmt'
git diff --check
go test ./...
```

结果：全部通过。当前 Makefile 没有 `check-fmt` target，因此按仓库实际能力使用 `make fmt` 后以 `git diff --check` 检查差异。

前端在 `ui-vue3` 执行：

```powershell
yarn vitest run
yarn eslint <本次修改的 TS/Vue 文件>
yarn vite build
yarn type-check
yarn prettier-check
```

- Vitest：6 files / 11 tests passed。
- 定向 ESLint：0 error；仅有仓库既有未使用变量 warning。
- Vite 生产构建：成功；保留既有 `updateInstanceTrafficSwitch` 未导出及大 chunk warning。
- `yarn type-check`：仓库基线失败，错误集中在 home、resource detail/Grafana/instance 等既有文件；输出中没有本次新增 Affinity、Script、Condition 或 shared router-rule 文件，日志为 `e2e/router-rule-chain/type-check.log`。
- `yarn prettier-check`：仓库基线的 159 个文件不符合 Prettier；本次改动文件已通过定向格式化/检查，日志为 `e2e/router-rule-chain/prettier-check.log`。

独立复现工程在 `D:/environment/github/dubbo-go/quickstart-demo` 执行：

```powershell
go test ./e2e/admin-router-rule-chain/...
```

四个 Condition/Affinity client/server package 均编译通过。复现说明位于 `e2e/admin-router-rule-chain/README.md`。

## 真实 E2E 验证

### ZooKeeper + dubbo-go RouterChain

环境：已有 ZooKeeper `127.0.0.1:2181`、Admin `127.0.0.1:8888`、两个 dubbo-go provider `20000/20001` 和持续调用 consumer。

已完成 Condition v3.0 和 v3.1 的 create/update/delete：

- Admin 写入标准路径 `/dubbo/config/dubbo/org.apache.dubbo.quickstart.Greeter:1.0.0:demo.condition-router`。
- consumer 收到动态配置变化，v3.1 更新为单一 Hangzhou destination 后，连续调用全部命中 Hangzhou；切换 v3.0 destination 后能够命中 Shanghai；删除后恢复两 provider 的无约束选择。
- Admin 更新前后回读保持 v3.1 `from/to/weight` 结构，证据为 `e2e/router-rule-chain/condition-v31-update-read.json`。
- `weight: 0` 不表示禁用 destination：当前 dubbo-go `newCondSet` 会把 `<= 0` 替换为默认权重。这是 dubbo-go 既有语义，不是 Admin codec 丢失；Admin 已验证 0 值无损保存。

测试过程日志在被 `.gitignore` 排除的 `e2e/router-rule-chain/admin-zk.log`、`consumer.log` 与 provider 日志中。

### Nacos

临时使用 `nacos/nacos-server:v2.3.2` standalone 容器，Admin 监听 `127.0.0.1:8889`，mesh 为 `local-nacos`。

- Condition、Affinity、Script 均通过 Console API create/read/update/delete。
- Nacos 中 `Group=dubbo`，DataId 使用对应 `*.condition-router`、`*.affinity-router`、`*.script-router`，原始 YAML 与统一外部契约一致。
- Affinity 更新保留 `enabled: false`；从 Nacos 外部直接把 ratio 改为 33 后，Admin watcher/API 回读同步为 33。
- 删除后详情返回 `NotFoundError`；测试数据均已删除。
- 证据保存在被忽略的 `e2e/router-rule-chain/admin-nacos.log`。

### UI 运行证据

使用真实 Admin `8888` 和 Vite `8881` 登录并由 Cypress/Edge headless 验证页面和 API 数据，测试通过 `1 spec / 1 test`。截图保存在被忽略的：

- `e2e/router-rule-chain/affinity-rule-list.png`
- `e2e/router-rule-chain/script-rule-list.png`
- `e2e/router-rule-chain/condition-v31-yaml-detail.png`

截图用临时规则在测试结束后已通过 API 删除。

## 上游能力边界

### Script Router

当前依赖的 dubbo-go 中 Script factory 注册被注释：

```go
// TODO(finalt) Temporarily removed until fixed
// extension.SetRouterFactory(constant.ScriptRouterFactoryKey, NewScriptRouterFactory)
```

因此本次真实验证覆盖 Admin 发布、Nacos/ZK 外部契约、watcher 回读和删除，但默认 RouterChain 无法执行 Script；不将其记录为路由执行 E2E 通过。

### Affinity Router

- 当前 `dubbo-go/v3/imports` 未 blank-import affinity router，复现 consumer 必须显式 import。
- 显式注册后，配置下发和 ZK 变化已验证；但当前 service-discovery URL 合并会让用于测试的 consumer reference 参数在两个 invoker 上相同，无法可靠证明最终筛选结果。
- `quickstart-demo/e2e/admin-router-rule-chain/affinity` 保留了该诊断流程，README 明确其用途和限制，不将它作为最终 Affinity 路由命中的证明。

## 清理与交付状态

- E2E 创建的 Nacos 数据和临时 UI 规则已删除；临时 Admin、provider、consumer、Vite 进程及 Nacos 容器在收尾时精确清理。
- 不清理用户已有 legacy 配置；只删除本任务明确创建的标准 ZK 测试节点。
- 当前分支保持未提交、未推送，等待用户 review 后再执行远程推送和 PR。
