---
name: dubbo-admin-console-api
description: Implements and reviews dubbo-admin Console API changes. Use when the user asks about HTTP endpoints, Gin routes, handlers, request or response models, console services, error handling, API pagination, or the handler to service to manager to store flow under pkg/console/. Do not use for frontend-only, discovery-only, or store-internals tasks unless they affect Console API behavior.
---

# dubbo-admin Console API

## Purpose

Use this skill to change or explain dubbo-admin Web MVC behavior consistently across router, handler, model, service, manager, and store layers.

## When to use

Use for Console API endpoint additions, request binding, response models, service logic, route registration, API error handling, and backend API review.

Do not use for Vue-only changes, registry discovery internals, or memory store implementation details unless an API contract depends on them.

## Inputs

Required:
- Target endpoint, handler, model, or package path.
- Intended API behavior or observed bug.

Optional:
- Example request or response payload.
- Related frontend API caller.
- Existing route or service function to mirror.

If missing, inspect `pkg/console/router/router.go` first and ask only for behavior that cannot be inferred from code.

## Workflow

1. Locate the route in `pkg/console/router/router.go`.
2. Read the matching handler in `pkg/console/handler/` for binding and response style.
3. Read or update request and response types in `pkg/console/model/`.
4. Put business logic in `pkg/console/service/`; keep handlers focused on HTTP concerns.
5. Use `ctx.ResourceManager()` and helpers in `pkg/core/manager/` for resource access.
6. Return responses with `model.NewSuccessResp`, `util.HandleArgumentError`, or `util.HandleServiceError`.
7. If endpoint behavior is unclear, read `references/web-mvc-flow.md`.

## Output format

Return changed files, endpoint behavior, request and response contract, validation performed, and any frontend impact.

## Validation

- Confirm the route is registered under `/api/v1`.
- Confirm handler binding matches method semantics: query for reads, JSON body for mutations.
- Confirm service code returns typed models and propagates business errors.
- Run focused Go tests for changed packages; use `make test` for shared API behavior.

## Edge cases

- Some handlers include lightweight auth, cookie, or parameter logic; do not force all logic into services.
- If changing response fields, check frontend callers under `ui-vue3/src/api/`.
- If a resource query becomes slow, switch to `dubbo-admin-store`.

## References

- Read `references/web-mvc-flow.md` for the current route, handler, service, and response pattern.
