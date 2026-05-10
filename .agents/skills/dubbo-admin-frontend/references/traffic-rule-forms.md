# Traffic Rule Form Design

Use this reference when changing condition routing, tag routing, dynamic config, YAML views, or traffic rule API payloads under `ui-vue3/src/views/traffic/`.

## Shared traffic layout

Traffic pages follow this structure:

```text
views/traffic/<rule>/index.vue           list page
views/traffic/<rule>/tabs/*.vue          detail, form, YAML, add, update tabs
views/traffic/<rule>/slots/*.vue         tab header slots
api/service/traffic.ts                   API wrappers
router/defaultRoutes.ts                  tab routes
```

The API wrappers map directly to backend traffic endpoints:

```text
/condition-rule/search, /condition-rule/{ruleName}
/tag-rule/search, /tag-rule/{ruleName}
/configurator/search, /configurator/{encodedName}
```

`configurator` names are encoded with `encodeURIComponent`; keep that behavior for keys containing dots, colons, or other service-name characters.

## Condition routing implementation chain

Condition routing form logic is centralized in `routingRule/composables/useRoutingRule.ts`:

```text
route condition string -> parseCondition...ToArray() -> routeList form state
routeList selected types + values -> mergeConditions() -> API conditions[]
```

Supported condition syntax:

- Single values: `host=1.1.1.1`, `application!=demo`, `method=sayHello`
- Arguments: `arguments[0]=value`
- Attachments: `attachments[key]!=value`
- Custom keys: `region=hangzhou`
- Route target separator: `match & match => target & target`

Key functions:

```ts
parseConditionMatchStringToArray(matchStr, routeItemIndex)
parseConditionToStringToArray(toStr, routeItemIndex)
mergeConditions()
mergeConditionItems(selectedTypes, conditionItems)
```

`parseConditionString()` also updates selected type arrays on `routeList`: `selectedMatchConditionTypes` for request matching and `selectedRouteDistributeMatchTypes` for target selection. If you add a condition type, update the type options, parser, merge logic, default row creation, delete behavior, description text, and i18n.

Detail and YAML tabs read from the same backend detail endpoint:

```text
formView.vue -> getConditionRuleDetailAPI(ruleName) -> parse conditions for display
YAMLView.vue -> getConditionRuleDetailAPI(ruleName) -> yaml.dump(data)
```

Add/update tabs must keep form and YAML behavior equivalent; do not add fields to one representation only.

## Dynamic config implementation chain

Dynamic config form state lives in `dynamicConfig/model/ConfigModel.ts`:

```text
backend Configurator detail -> ViewDataModel.fromApiOutput()
configs[] -> ConfigModel.parseMatches() + parseParameters()
form edits -> ViewDataModel.toApiInput(check) -> add/save configurator API
```

Important model responsibilities:

- `ConfigModel.matchesValue` stores match fields such as `address`, `providerAddress`, `service`, `application`, and free `param` matches.
- `ConfigModel.parametersValue` stores common parameter fields such as `retries`, `timeout`, `accesslog`, `weight`, and free `other` keys.
- `parseMatches()` converts backend match maps into UI rows.
- `parseParameters()` maps known parameters to typed rows and unknown keys to `other`.
- `toApiInput(check)` rebuilds the backend payload and performs validation when `check` is true.

The form tab uses `TAB_LAYOUT_STATE.dynamicConfigForm.data` to preserve edits across sibling tabs. Resetting clears that state and reloads from the API.

Payload shape produced by `toApiInput()`:

```json
{
  "ruleName": "service.configurators",
  "scope": "service",
  "key": "service",
  "enabled": true,
  "configVersion": "v3.0",
  "configs": [
    { "match": {}, "parameters": {}, "enabled": true, "side": "provider" }
  ]
}
```

When adding new match or parameter fields, update both `parse...` and `toApiInput()` so existing data round-trips.

## Tag routing implementation chain

Tag routing pages live in `traffic/tagRule/` and mirror the condition-rule tab structure. Detail view calls `getTagRuleDetailAPI(ruleName)` and displays `tags[].match[]` entries as key/relation/value labels. YAML and add/update views should preserve this backend structure:

```json
{
  "configVersion": "v3.0",
  "scope": "application",
  "key": "shop-user",
  "enabled": true,
  "runtime": true,
  "tags": [
    { "name": "gray", "match": [{ "key": "version", "value": { "exact": "v1" } }] }
  ]
}
```

## Review checklist

- Keep form view, YAML view, and API payload fields in sync.
- Verify parse/merge round trips for condition strings containing `=>`, `&`, `arguments[]`, `attachments[]`, and custom keys.
- Preserve `TAB_LAYOUT_STATE` when switching dynamic config form/YAML tabs.
- Encode configurator names in API calls.
- Update i18n route labels and tab headers when adding traffic tabs.
- Run `cd ui-vue3 && yarn lint && yarn type-check` for TypeScript or route changes; add `yarn test:unit` when logic is refactored.
