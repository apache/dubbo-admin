---
name: dubbo-admin-events
description: Implements and reviews dubbo-admin EventBus dispatching. Use when the user asks about Event, Emitter, Subscriber, SubscriptionManager, EventBus component registration, Subscribe or Unsubscribe behavior, Send dispatch rules, subscriber ProcessEvent flow, resource-kind based routing, or event error handling under pkg/core/events/. Do not use for a domain subscriber unless the task changes the event bus contract or dispatch semantics.
---

# dubbo-admin Events

## Purpose

Use this skill to modify or explain the shared EventBus contract used by discovery, engine, informers, and subscribers.

## When to use

Use for EventBus interfaces, resource-kind dispatching, subscriber registration uniqueness, synchronous processing, and event dispatch error behavior.

Do not use for business logic inside a single subscriber unless EventBus semantics change.

## Inputs

Required:
- EventBus behavior, interface, or dispatch problem.
- Event producer or subscriber involved.

Optional:
- Event type.
- Old and new resource objects.
- Subscriber names for the same resource kind.

If missing, inspect `pkg/core/events/eventbus.go` and `pkg/core/events/component.go` first.

## Workflow

1. Read `pkg/core/events/eventbus.go` for Event, Emitter, Subscriber, SubscriptionManager, and EventBus interfaces.
2. Read `pkg/core/events/component.go` for the concrete EventBus implementation.
3. Trace producers that call `events.NewResourceChangedEvent` and `Emitter.Send`.
4. Trace consumers implementing `ResourceKind`, `Name`, and `ProcessEvent`.
5. Read `references/eventbus-dispatch.md` before changing dispatch semantics.

## Output format

Return event source, resource kind, subscribers called, error behavior, changed files, and validation evidence.

## Validation

- Confirm `Send` uses `NewObj()` first and `OldObj()` for delete-like events.
- Confirm subscriber names are unique per resource kind.
- Confirm subscriber errors are logged and do not stop later subscribers.
- Run tests or targeted reasoning for discovery and engine consumers when dispatch changes.

## Edge cases

- Events with both old and new nil cannot be routed.
- Dispatch is synchronous; long subscriber work blocks the caller.
- EventBus should stay domain-neutral; put business logic in subscribers.

## References

- Read `references/eventbus-dispatch.md` for concrete dispatch rules and failure behavior.
