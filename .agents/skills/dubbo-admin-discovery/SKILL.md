---
name: dubbo-admin-discovery
description: Implements and reviews dubbo-admin registry discovery changes. Use when the user asks about Zookeeper or Nacos discovery, ListAndWatch, informers, discovery factories, discovery subscribers, ServiceProviderMetadata or ServiceConsumerMetadata processing, or Application, Service, RPCInstance, and Instance derivation under pkg/core/discovery/ and pkg/core/controller/. Do not use for engine-sourced RuntimeInstance behavior.
---

# dubbo-admin Discovery

## Purpose

Use this skill to trace registry-sourced resources from ListWatcher input through informer events, EventBus dispatch, subscribers, store writes, and derived mesh resources.

## When to use

Use for Zookeeper or Nacos registry flow, discovery ListWatchers, discovery subscribers, metadata-derived services, application instance counts, and registry event bugs.

Do not use for Kubernetes runtime infrastructure engine flow; use `dubbo-admin-engine` for RuntimeInstance behavior.

## Inputs

Required:
- Registry type, resource kind, subscriber, or discovery package path.
- The event or resource transition being changed or debugged.

Optional:
- Example registry metadata.
- Store query or index involved.
- Console API result affected by discovery data.

If missing, start at `pkg/core/discovery/component.go` and identify the subscriber for the resource kind.

## Workflow

1. Read `pkg/core/discovery/component.go` for dependencies, informer creation, subscriber registration, and start behavior.
2. Inspect `pkg/core/controller/listwatcher.go` and `pkg/core/controller/informer.go`.
3. Follow registry-specific ListWatcher creation through `pkg/core/discovery/factory.go`.
4. Read the relevant subscriber under `pkg/core/discovery/subscriber/`.
5. Verify resource writes against store and index behavior.
6. Read `references/discovery-flow.md` for subscriber registration and event flow details.

## Output format

Return the resource flow, affected subscriber, store/index interactions, changed files, and validation commands or residual test gaps.

## Validation

- Confirm informer events emit the expected `cache.DeltaType`.
- Confirm subscriber `ResourceKind()` matches the event resource kind.
- Confirm derived resources use stable resource keys.
- Run targeted subscriber or controller tests; use `make test` for broad discovery changes.

## Edge cases

- Nacos and Zookeeper register extra subscribers conditionally.
- Delete events may need old objects and must not assume `NewObj()` exists.
- If dispatch semantics are the problem, switch to `dubbo-admin-events`.

## References

- Read `references/discovery-flow.md` for ListWatcher, informer, and subscriber details.
