# Frontend Structure and Routing Flow

Use this reference when changing `ui-vue3/` routes, views, API clients, stores, tab pages, or resource topology UI.

## Source layout

- `ui-vue3/src/api/`: typed-by-convention API client wrappers. Most files import `@/base/http/request`.
- `ui-vue3/src/base/http/request.ts`: Axios instance, base URL, response normalization, auth redirect, and mesh injection.
- `ui-vue3/src/router/`: static route tree and `RouterMeta` contract.
- `ui-vue3/src/layout/`: shell and tab layout components.
- `ui-vue3/src/views/`: feature pages, grouped by resource or traffic domain.
- `ui-vue3/src/stores/`: Pinia stores. `stores/mesh.ts` persists the selected mesh.
- `ui-vue3/src/components/`: reusable UI widgets such as `MonacoEditor`.

## Request chain

Frontend API calls follow this chain:

```text
view/component -> src/api/... wrapper -> base/http/request.ts -> /api/v1 backend route
```

Key implementation points in `base/http/request.ts`:

```ts
const service = axios.create({ baseURL: '/api/v1', timeout: 30 * 1000 })

request.use((config) => {
  config.headers['Content-Type'] ||= 'application/json'
  if (!config.params) config.params = {}
  const { mesh } = useMeshStore()
  config.params['mesh'] = mesh
  return config
})
```

Responses are resolved only when `response.status === 200` and `response.data.code === HTTP_STATUS.SUCCESS`. Failed API responses reject with the backend response body and usually show an Ant Design Vue message. `401` clears auth state and redirects to `/login?redirect=...`.

Do not manually append `mesh` in API wrappers unless the endpoint intentionally needs a different value; the request layer already injects it.

## Route construction

Routes are declared in `src/router/defaultRoutes.ts`. `handleRoutes()` mutates each route at startup:

```text
parent path + child path -> normalized absolute path
route.redirect -> normalized relative target
route.meta._router_key -> unique id
route.meta.parent -> parent route object
route.meta.skip -> explicit true or false
```

`RouterMeta` supports:

```ts
icon, hidden, skip, tab_parent, tab, _router_key, parent, slots, headerParamKey
```

Add a new page by extending `defaultRoutes.ts`, adding its view under `src/views/...`, and adding i18n labels for route names used in tabs or menus.

## Tab page flow

Tabbed resource and traffic pages use `LayoutTab`:

```text
route with meta.tab_parent -> component: LayoutTab -> child routes with meta.tab -> a-tabs
```

`layout/tab/layout_tab.vue` computes visible tabs from `tabRoute.meta.parent.children.filter(x => x.meta.tab)` and pushes by route `name` while preserving current `params` and `query`. Back navigation uses `tabRoute.meta.back || '../'`. Header slots come from `meta.slots.header`, for example application, service, instance, and traffic rule header slots.

When adding a new tab, set:

- `meta.tab: true`
- a stable route `name` used by i18n and tab switching
- `meta.icon` for the tab label
- `meta.back` when the tab needs a deterministic return route
- matching params across sibling tabs, otherwise tab switching loses context

## Table search flow

`utils/SearchUtil.ts` defines `SearchDomain` for list pages:

```text
query config + searchApi + columns -> reactive queryForm -> onSearch() -> API -> result/pageInfo
```

`onSearch()` adds `pageSize` and `pageOffset` unless `noPaged` is set, then expects backend data shaped as:

```json
{ "data": { "list": [], "pageInfo": { "total": 0 } } }
```

If changing backend pagination or response fields, update both the API model and the `SearchDomain` consumer.

## Resource topology pattern

Topology tabs live at:

- `views/resources/applications/tabs/topology.vue`
- `views/resources/services/tabs/topology.vue`

They are wired through child tab routes under `/applications` and `/services`. Keep backend graph fields aligned with frontend expectations: `nodes[]`, `edges[]`, node `id`, `label`, `type`, `rule`, and edge `source`, `target`, `data.type`.

## Review checklist

- Confirm the API wrapper path matches the backend route under `/api/v1`.
- Confirm `mesh` is not duplicated in query params.
- For tab pages, check `meta.parent`, `meta.tab`, `meta.back`, route params, header slot, and i18n key.
- For list pages, verify `SearchDomain` still receives `data.list` and `data.pageInfo.total`.
- For topology or charts, verify empty data, loading, and detail-drawer behavior.
