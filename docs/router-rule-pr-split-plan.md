# Router Rule PR 拆分与描述草案

本文用于把当前 `feature/admin-router-rule-chain` 聚合改动拆成可 review 的小 PR。行数口径建议为：

- 后端核心逻辑 PR：250-350 行非注释实现代码，最多不超过 500 行；测试、生成文件、license、文档不计入。
- 前端 PR：700-1000 行实现代码以内；测试不计入，但需要说明覆盖点。
- 生成文件：单独 commit 标记 generated，不纳入核心 review 行数。

## PR 1：对齐配置中心 rule key 与 group

建议标题：

```text
fix: align router rule config center keys
```

范围：

- 统一 ZooKeeper 写入路径为 `/dubbo/config/dubbo/<ruleName>`。
- 保留旧 `/dubbo/config/<ruleName>` 的回读兼容。
- Nacos 使用 `DataId=<ruleName>`、`Group=dubbo`。
- ZK watcher 过滤根节点，只处理标准 rule key。

建议包含文件：

- `pkg/common/constants/rule.go`
- `pkg/governor/zk/governor.go`
- `pkg/governor/zk/governor_test.go`
- `pkg/governor/nacos2/governor.go`
- `pkg/discovery/zk/factory.go`
- `pkg/discovery/zk/factory_test.go`
- `pkg/discovery/zk/listerwatcher/listerwatcher.go`
- `pkg/discovery/nacos2/factory.go`

PR 描述：

```markdown
## Summary

- align router rule config keys with Dubbo dynamic configuration conventions
- write new ZK rules under `/dubbo/config/dubbo/<ruleName>`
- keep legacy `/dubbo/config/<ruleName>` readable but stop writing new rules there
- use `dubbo` as the Nacos config group for router rules

## Test

- go test ./pkg/governor/zk ./pkg/discovery/zk

## Notes

This PR only fixes the config-center addressing contract. It does not change rule YAML shape or UI behavior.
```

截图要求：不需要截图。

## PR 2：Condition v3.0/v3.1 公共 YAML round-trip

建议标题：

```text
fix: preserve condition router yaml versions
```

范围：

- 保留 v3.0 字符串 `conditions`。
- 支持 v3.1 结构化 `from/to/weight`。
- Console API 按 `configVersion` 绑定不同的 `conditions` 形状。
- `EncodeRule` / `DecodeRule` 不降级、不丢字段。

建议包含文件：

- `api/mesh/v1alpha1/condition_route.proto`
- `api/mesh/v1alpha1/condition_route.pb.go`
- `pkg/console/model/router_rule.go`
- `pkg/console/model/router_rule_test.go`
- `pkg/core/resource/apis/mesh/v1alpha1/rule_codec.go`
- `pkg/core/resource/apis/mesh/v1alpha1/rule_codec_test.go`
- `pkg/core/resource/apis/mesh/v1alpha1/conditionroute_helper.go`

PR 描述：

```markdown
## Summary

- split Condition Router input/output handling by `configVersion`
- keep v3.0 conditions as string expressions
- keep v3.1 conditions as structured `from/to/weight`
- add YAML encode/decode coverage to prevent round-trip data loss

## Test

- go test ./pkg/console/model ./pkg/core/resource/apis/mesh/v1alpha1

## Notes

Generated protobuf output is included in a separate commit. The main review target is the version-aware model and codec logic.
```

截图要求：不需要截图。

## PR 3：Affinity / Script 后端资源与 Console API

建议标题：

```text
feat: add affinity and script router rule APIs
```

范围：

- 增加 ScriptRoute proto/resource。
- 增加 Affinity/Script 的 Console API handler、router 和 service。
- 通过公共 rule mutation path 创建、更新、删除规则。
- 校验 `.affinity-router` / `.script-router` ruleName suffix。
- Script 只接受 application scope、`javascript`、非空且不超过 64 KiB。

建议包含文件：

- `api/mesh/v1alpha1/script_route.proto`
- `api/mesh/v1alpha1/script_route.pb.go`
- `pkg/core/resource/apis/mesh/v1alpha1/scriptroute_types.go`
- `pkg/core/resource/apis/mesh/v1alpha1/rule_codec.go`
- `pkg/core/resource/apis/mesh/v1alpha1/rule_codec_test.go`
- `pkg/console/handler/router_rule.go`
- `pkg/console/router/router.go`
- `pkg/console/service/affinity_rule.go`
- `pkg/console/service/script_rule.go`
- `pkg/core/discovery/subscriber/zk_config.go`
- `pkg/core/governor/governor.go`

PR 描述：

```markdown
## Summary

- add backend management APIs for Affinity Router and Script Router rules
- validate ruleName suffixes before publishing config-center keys
- encode public YAML using `affinityAware` and script rule fields consumed by Dubbo runtimes
- route create/update/delete through the existing governor-aware mutation path

## Test

- go test ./pkg/console/model ./pkg/core/resource/apis/mesh/v1alpha1

## Notes

Admin stores and publishes script rules but does not execute scripts. Runtime script semantics remain owned by Dubbo consumers.
```

截图要求：不需要截图。

## PR 4：Service argument route upsert 稳定性

建议标题：

```text
fix: preserve service condition rule fields when updating argument routes
```

范围：

- 修复 argument route upsert 时 `Spec` 为 nil 的处理。
- 只替换 method/argument 条件，不覆盖非 argument condition。
- 统一 service condition rule 的 key 为 `<interface>:<version>:<group>.condition-router`。

建议包含文件：

- `pkg/console/service/service.go`
- `pkg/console/service/service_argument_route_test.go`

PR 描述：

```markdown
## Summary

- make service argument route upsert tolerate missing condition-rule specs
- preserve non-argument condition expressions when rewriting argument routes
- keep generated service condition rule names aligned with Dubbo runtime subscription keys

## Test

- go test ./pkg/console/service

## Notes

This PR is intentionally limited to the service argument route helper path. It does not add new router kinds.
```

截图要求：不需要截图。

## PR 5：Condition v3.1 前端表单与 YAML 回显

建议标题：

```text
feat: support condition router v3.1 editing
```

范围：

- Condition 表单支持 v3.1 结构化 `from/to/weight`。
- 编辑页根据完整详情恢复 v3.0 或 v3.1 表单状态。
- YAML 页面保持 v3.1 字段回显，不降级为字符串条件。
- 菜单高亮支持隐藏的 traffic edit/detail 页面。

建议包含文件：

- `ui-vue3/src/views/traffic/routingRule/model/ConditionRuleModel.ts`
- `ui-vue3/src/views/traffic/routingRule/model/ConditionRuleModel.spec.ts`
- `ui-vue3/src/views/traffic/routingRule/components/StructuredConditionRuleList.vue`
- `ui-vue3/src/views/traffic/routingRule/tabs/addByFormView.vue`
- `ui-vue3/src/views/traffic/routingRule/tabs/updateByFormView.vue`
- `ui-vue3/src/views/traffic/routingRule/tabs/addByYAMLView.vue`
- `ui-vue3/src/views/traffic/routingRule/tabs/updateByYAMLView.vue`
- `ui-vue3/src/views/traffic/routingRule/tabs/updateByFormView.spec.ts`
- `ui-vue3/src/views/traffic/routingRule/tabs/updateByYAMLView.spec.ts`
- `ui-vue3/src/layout/menu/menuState.ts`
- `ui-vue3/src/layout/menu/menuState.spec.ts`

PR 描述：

```markdown
## Summary

- add a structured editor for Condition Router `v3.1` rules
- keep `v3.0` string conditions and `v3.1` structured conditions on separate UI paths
- preserve condition YAML during edit round-trips
- keep the traffic menu highlighted for hidden edit routes

## Test

- cd ui-vue3
- yarn test:unit ConditionRuleModel menuState updateByFormView updateByYAMLView

## Screenshots

- Condition v3.1 form editor
- Condition v3.1 YAML detail/edit page
```

截图要求：需要补 Condition v3.1 表单页和 YAML 编辑/详情页截图。

## PR 6：Tag rule key 前端对齐

建议标题：

```text
fix: align tag router rule names in UI
```

范围：

- Tag 表单和 YAML 创建时使用 `<provider-application>.tag-router`。
- 不再生成 service 粒度或错误 suffix 的 tag ruleName。
- 补 Tag 表单/YAML 单测。

建议包含文件：

- `ui-vue3/src/views/traffic/tagRule/tabs/addByFormView.vue`
- `ui-vue3/src/views/traffic/tagRule/tabs/addByFormView.spec.ts`
- `ui-vue3/src/views/traffic/tagRule/tabs/addByYAMLView.vue`
- `ui-vue3/src/views/traffic/tagRule/tabs/addByYAMLView.spec.ts`

PR 描述：

```markdown
## Summary

- generate Tag Router rule names as `<provider-application>.tag-router`
- keep Tag rules application-scoped to match Dubbo runtime subscription
- cover form and YAML creation paths with unit tests

## Test

- cd ui-vue3
- yarn test:unit addByFormView addByYAMLView

## Notes

This PR only changes ruleName generation and tests for Tag Router UI.
```

截图要求：建议补一张 Tag 创建表单截图；不是强制。

## PR 7：Affinity / Script 前端管理页

建议标题：

```text
feat: add affinity and script router rule pages
```

范围：

- 增加 Affinity/Script 列表页。
- 增加共享 YAML 编辑器。
- 增加前端 API client、路由、i18n、菜单入口。
- Affinity 默认使用 `v3.1` 和 `affinityAware`。
- Script 默认使用 application scope 和 `javascript`。

建议包含文件：

- `ui-vue3/src/api/service/traffic.ts`
- `ui-vue3/src/base/i18n/en.ts`
- `ui-vue3/src/base/i18n/zh.ts`
- `ui-vue3/src/router/defaultRoutes.ts`
- `ui-vue3/src/router/RouterMeta.ts`
- `ui-vue3/src/views/traffic/_shared/RouterRuleList.vue`
- `ui-vue3/src/views/traffic/_shared/RouterRuleYamlEditor.vue`
- `ui-vue3/src/views/traffic/affinityRule/index.vue`
- `ui-vue3/src/views/traffic/affinityRule/editor.vue`
- `ui-vue3/src/views/traffic/scriptRule/index.vue`
- `ui-vue3/src/views/traffic/scriptRule/editor.vue`

PR 描述：

```markdown
## Summary

- add UI routes for Affinity Router and Script Router management
- reuse one list and YAML editor for both rule kinds
- derive `ruleName` from `key + suffix` on create
- expose defaults that match the backend and Dubbo runtime contract

## Test

- cd ui-vue3
- yarn test:unit

## Screenshots

- Affinity rule list
- Affinity rule YAML editor
- Script rule list
- Script rule YAML editor
```

截图要求：需要补 Affinity 和 Script 的列表页、编辑页截图。

## 提交顺序建议

1. PR 1 先合，冻结配置中心 key/group。
2. PR 2 合入 Condition YAML round-trip。
3. PR 4 可独立跟进，降低 service argument route 风险。
4. PR 3 在 PR 1/2 后合，加入 Affinity/Script 后端能力。
5. PR 5、PR 6、PR 7 作为前端 PR，分别对应 Condition、Tag、Affinity/Script。

如果 reviewer 对生成文件敏感，把 `*.pb.go` 和 `*_types.go` 放在同一 PR 的独立 commit，并在 PR 描述里标记为 generated。
