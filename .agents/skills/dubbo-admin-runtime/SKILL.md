---
name: dubbo-admin-runtime
description: Implements and reviews dubbo-admin runtime component changes. Use when the user asks about component abstraction, component composition, runtime bootstrap, dependency ordering, registration, lifecycle, startup, shutdown, runtime builder behavior, or pkg/core/runtime/ and pkg/core/bootstrap/. Do not use for discovery, engine, store, Console API, or frontend tasks unless component lifecycle or dependencies are affected.
---

# dubbo-admin Runtime

## Purpose

Use this skill to change or explain component abstraction, dependency composition, bootstrap order, activation, and runtime startup behavior.

## When to use

Use for component interfaces, `RequiredDependencies`, `Order`, `RegisterComponent`, `Builder`, dependency graph, bootstrap, start, stop, and core component failure behavior.

Do not use for domain behavior inside a component unless lifecycle or dependency contracts change.

## Inputs

Required:
- Component type, lifecycle method, or runtime package path.
- Dependency or startup behavior being changed.

Optional:
- Error message from bootstrap or start.
- Existing component to mirror.

If missing, inspect `pkg/core/runtime/component.go` and `pkg/core/bootstrap/bootstrap.go`.

## Workflow

1. Classify the change: component contract, registration, bootstrap, dependency graph, builder activation, or runtime start.
2. Read the matching reference listed below before editing or explaining implementation behavior.
3. Inspect the source files named by that reference and confirm the current code still matches the documented flow.
4. Make the smallest change at the owning layer: component interfaces, registry, bootstrap, dependency graph, builder, or runtime start.
5. Validate the exact lifecycle invariant affected by the change.

## Output format

Return component contract impact, dependency order, changed files, startup behavior, and validation performed.

## Validation

- Confirm dependencies exist and are initialized before dependents.
- Confirm `Order()` is only used for startup ordering and not dependency correctness.
- Confirm core component startup failures still panic while non-core failures log.
- Run `go test ./pkg/core/runtime ./pkg/core/bootstrap` or `make test`.

## Edge cases

- `Order()` is deprecated for dependency management but still used during start.
- Duplicate component registration panics.
- Cycles should fail bootstrap with a clear dependency error.

## References

- Read `references/component-contracts.md` when adding or modifying component interfaces, component types, or `RequiredDependencies()`.
- Read `references/bootstrap-flow.md` when changing initialization order, component gathering, `Init`, or activation.
- Read `references/dependency-graph.md` when debugging missing dependencies, ordering, deterministic sort, or cycles.
- Read `references/runtime-start.md` when changing `Start`, `Order()`, goroutine behavior, or startup error handling.
