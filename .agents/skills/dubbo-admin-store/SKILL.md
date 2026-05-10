---
name: dubbo-admin-store
description: Implements and reviews dubbo-admin resource storage and indexing changes. Use when the user asks about ResourceStore APIs, store routing, memory store behavior, index registration, IndexCondition, ListByIndexes, PageListByIndexes, ByMeshIndex, radix-tree prefix lookup, or pkg/core/store/, pkg/core/store/index/, and pkg/store/. Do not use for frontend, Console API, discovery, or engine tasks unless they require store or index changes.
---

# dubbo-admin Store

## Purpose

Use this skill to change or explain resource storage, indexes, query semantics, pagination, and prefix matching without breaking callers.

## When to use

Use for `ResourceStore`, store routing, memory store internals, index definitions, equality or prefix query behavior, pagination, and query performance.

Do not use for a domain feature unless the feature requires store or index changes.

## Inputs

Required:
- Resource kind, index name, query path, or store package path.
- Desired query behavior or bug.

Optional:
- Caller in Console API, discovery, or engine.
- Example resource keys and index values.

If missing, inspect the caller first, then follow into `pkg/core/store/` or `pkg/store/memory/`.

## Workflow

1. Read `pkg/core/store/store.go` for the ResourceStore contract.
2. Check store routing and lifecycle in `pkg/core/store/component.go`.
3. Inspect memory behavior in `pkg/store/memory/store.go`.
4. Check index registration in `pkg/core/store/index/`.
5. Prefer existing `IndexCondition` operators before adding new APIs.
6. Read `references/indexing-and-query.md` before changing query semantics.

## Output format

Return resource kind, index conditions, expected keys or resources, caller impact, changed files, and validation commands.

## Validation

- Confirm `ByMeshIndex` is included when mesh scoping is required.
- Confirm multiple index conditions intersect key sets.
- Confirm prefix queries update radix trees on add, update, delete, and dynamic index additions.
- Run `go test ./pkg/store/memory ./pkg/core/store/...` or `make test`.

## Edge cases

- Empty index conditions return no keys in memory store.
- Missing indexes should return a clear error rather than silently scanning.
- Store contract changes can affect Console API, discovery, and engine callers.

## References

- Read `references/indexing-and-query.md` for index registration and memory query behavior.
