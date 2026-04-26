---
name: dubbo-admin-engine
description: Implements and reviews dubbo-admin resource engine changes. Use when the user asks about runtime infrastructure discovery, engine factories, Kubernetes or mock engine ListWatchers, RuntimeInstance informers, RuntimeInstanceEventSubscriber, engine leader election, or merging RuntimeInstance data into Instance resources under pkg/core/engine/ and pkg/engine/. Do not use for registry discovery unless the task concerns engine-sourced RuntimeInstance resources.
---

# dubbo-admin Engine

## Purpose

Use this skill to trace runtime infrastructure data from engine ListWatchers through RuntimeInstance events into Instance resource lifecycle changes.

## When to use

Use for engine component lifecycle, engine factory support, Kubernetes runtime instance watching, leader election for engine logic, and RuntimeInstance to Instance merge or delete behavior.

Do not use for registry metadata discovery; use `dubbo-admin-discovery`.

## Inputs

Required:
- Engine type, RuntimeInstance behavior, or engine package path.
- Intended runtime to instance transition.

Optional:
- RuntimeInstance sample.
- Existing Instance resource state.
- Store indexes used to find related instances.

If missing, inspect `pkg/core/engine/component.go` and then the relevant factory under `pkg/engine/`.

## Workflow

1. Read `pkg/core/engine/component.go` for dependencies, config, informers, subscribers, and start behavior.
2. Check `pkg/core/engine/factory.go` before adding or changing engine types.
3. Inspect implementation factories under `pkg/engine/`.
4. Follow RuntimeInstance ListWatcher output into `pkg/core/controller/informer.go`.
5. Read `pkg/core/engine/subscriber/runtime_instance.go` before changing Instance merge or delete behavior.
6. Read `references/runtime-engine-flow.md` for leader election and merge details.

## Output format

Return the engine flow, affected RuntimeInstance transition, store/index interactions, changed files, and validation plan.

## Validation

- Confirm engine depends on EventBus and ResourceStore.
- Confirm non-memory stores only run business logic under leader election.
- Confirm RuntimeInstance upsert and delete preserve RPC-sourced Instance state correctly.
- Use `make test` for changes touching informer events or Instance lifecycle.

## Edge cases

- Runtime-only instances require enough identity fields.
- Runtime delete should update an Instance when RPC source remains, but delete it when no source remains.
- If the EventBus dispatch contract changes, switch to `dubbo-admin-events`.

## References

- Read `references/runtime-engine-flow.md` for engine startup and RuntimeInstance handling.
