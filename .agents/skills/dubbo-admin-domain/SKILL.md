---
name: dubbo-admin-domain
description: Routes dubbo-admin architecture questions to the right domain skill. Use for cross-module questions, end-to-end flows, subsystem boundaries, or deciding whether runtime, discovery, engine, events, store, Console API, or frontend guidance applies. Prefer a narrower dubbo-admin-* skill when the request clearly targets one subsystem.
---

# dubbo-admin Domain Guide

## Purpose

Use this skill to choose the right dubbo-admin domain workflow and avoid mixing registry discovery, runtime engine, EventBus, store, API, and frontend responsibilities.

## When to use

Use for cross-module architecture, end-to-end data flow, skill selection, or issue triage that spans multiple subsystems.

Do not use when a request clearly targets one subsystem; load the narrower skill directly.

## Inputs

Required:
- User goal, issue, file path, or subsystem name.

Optional:
- Stack trace, API path, frontend route, resource kind, or event type.

If missing, inspect top-level paths and identify the first concrete subsystem before loading a narrower skill.

## Workflow

1. Classify the request by primary subsystem.
2. Select one narrow skill when possible.
3. If the request spans multiple domains, order them by data flow: runtime/discovery or engine, events, store, manager, Console API, frontend.
4. Read `references/architecture-map.md` when the boundary is unclear.
5. Avoid loading unrelated domain skills.

## Output format

Return selected skill or skills, why they apply, which paths to inspect first, and any boundary assumptions.

## Validation

- Confirm selected skill owns the code path being changed.
- Confirm cross-domain changes name both producer and consumer of data.
- If two skills overlap, state which one controls the contract.

## Edge cases

- Registry discovery and runtime engine both use informers but source different data.
- EventBus dispatch is shared infrastructure; use `dubbo-admin-events` when dispatch behavior changes.
- Store indexes are shared; use `dubbo-admin-store` when query behavior changes.

## References

- Read `references/architecture-map.md` for subsystem ownership and end-to-end flow.
