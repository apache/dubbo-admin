# dubbo-admin Architecture Map

Use this reference when a task crosses subsystem boundaries or when choosing which narrower `dubbo-admin-*` skill to use.

## End-to-end runtime flow

Dubbo Admin is built from runtime components. Startup flows through `pkg/core/bootstrap` and `pkg/core/runtime`:

```text
main/app -> bootstrap -> runtime builder -> component graph -> Init() -> Start()
```

Most backend subsystems register themselves during package init and are then ordered by runtime dependencies. Component-level questions belong to `dubbo-admin-runtime`.

## Data production paths

There are two main resource producers:

```text
Registry metadata
  -> discovery component
  -> registry ListWatcher (Zookeeper/Nacos)
  -> controller Informer
  -> ResourceStore write
  -> EventBus resource-change event
  -> discovery subscribers
  -> derived Application/Service/RPCInstance/Instance resources

Runtime infrastructure
  -> engine component
  -> engine ListWatcher (Kubernetes/mock)
  -> controller Informer
  -> ResourceStore write
  -> EventBus resource-change event
  -> RuntimeInstanceEventSubscriber
  -> Instance runtime-source merge
```

Use discovery for provider/consumer metadata, registry applications, registry instances, and registry-derived services. Use engine for Kubernetes or other runtime infrastructure, `RuntimeInstance`, leader election in engine sources, and runtime-source merging into `Instance`.

## Event and storage path

All informer-produced resource changes are dispatched through `pkg/core/events`:

```text
Informer.HandleDeltas()
  -> ResourceStore.Add/Update/Delete
  -> NewResourceChangedEvent(oldObj, newObj)
  -> EventBus.Send(event)
  -> subscribers for ResourceKind
```

Storage is handled through `pkg/core/store` and concrete stores under `pkg/store/`:

```text
ResourceStore component -> store router -> ResourceKindRoute(kind) -> memory/indexed store
```

Use `dubbo-admin-events` for dispatch contract changes. Use `dubbo-admin-store` for `ResourceStore`, index registration, `ListByIndexes`, `PageListByIndexes`, prefix lookup, or persistence behavior.

## Console API and frontend path

Console HTTP behavior is layered:

```text
Gin route -> handler -> console service/manager -> ResourceManager/store -> CommonResp JSON
```

Frontend behavior is layered:

```text
Vue view -> api/service wrapper -> base/http/request.ts -> /api/v1 -> Console API
```

Use `dubbo-admin-console-api` when changing routes, handlers, request/response models, service managers, pagination contracts, or error responses. Use `dubbo-admin-frontend` when changing Vue routes, tab metadata, API wrappers, Pinia state, topology rendering, or traffic rule form/YAML behavior.

## Skill selection table

| Task | Use skill |
| --- | --- |
| Component lifecycle, dependencies, startup/shutdown | `dubbo-admin-runtime` |
| Zookeeper/Nacos discovery, provider/consumer metadata, derived resources | `dubbo-admin-discovery` |
| Kubernetes/mock runtime discovery, `RuntimeInstance`, runtime-source instance merge | `dubbo-admin-engine` |
| `EventBus`, `Emitter`, `Subscriber`, `Subscribe`, `Send`, dispatch ordering | `dubbo-admin-events` |
| Store APIs, memory store, indexes, pagination, radix prefix lookup | `dubbo-admin-store` |
| Gin routes, handlers, console services, response JSON | `dubbo-admin-console-api` |
| Vue routes, components, API clients, traffic forms, topology tabs | `dubbo-admin-frontend` |

## Common boundary cases

- A new backend list endpoint usually needs `console-api` plus `store` if it requires indexes.
- A new graph or topology feature usually needs `console-api`, `store`, and `frontend`; add `discovery` only if new derived resources are needed.
- A registry event subscriber belongs to `discovery` unless it changes EventBus contracts.
- A Kubernetes Pod-to-resource transform belongs to `engine`, not `discovery`.
- If a change introduces a component dependency or startup ordering issue, include `runtime` even when the feature domain is elsewhere.

## Review sequence for cross-module changes

1. Identify the source resource kind and producer: discovery, engine, or manual Console API write.
2. Check whether storage needs a new kind, route, or index.
3. Check whether events or subscribers derive additional resources.
4. Verify Console API payloads match frontend expectations.
5. Verify frontend request wrappers, tab routes, and state handling.
6. Run the smallest tests for each touched layer, then broader Go/frontend checks.
