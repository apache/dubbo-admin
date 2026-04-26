---
name: dubbo-admin-frontend
description: Implements and reviews dubbo-admin Vue 3 frontend changes. Use when the user asks about ui-vue3, Vue views or components, frontend routing, route metadata, API clients, Pinia stores, SearchDomain, i18n, layout tabs, traffic rule form design, frontend tests, or Vite, ESLint, and Prettier configuration. Do not use for Go backend internals unless coordinating an API contract with the frontend.
---

# dubbo-admin Frontend

## Purpose

Use this skill to change or explain the Vue frontend while preserving routing, API, state, layout, i18n, and traffic-rule form patterns.

## When to use

Use for `ui-vue3/` changes, route definitions, views, components, API clients, Pinia stores, tabs, SearchDomain pages, traffic rule forms, and frontend validation.

Do not use for backend-only logic unless frontend API contracts are affected.

## Inputs

Required:
- Frontend route, view, component, or user workflow.
- Intended UI behavior or API contract.

Optional:
- Backend endpoint or payload.
- Screenshot or user interaction path.
- Existing page to mirror.

If missing, start from `ui-vue3/src/router/defaultRoutes.ts` and identify the view.

## Workflow

1. Locate the route in `ui-vue3/src/router/defaultRoutes.ts`.
2. Read the view under `ui-vue3/src/views/` and nearby components or composables.
3. Check API calls in `ui-vue3/src/api/` and request behavior in `src/base/http/request.ts`.
4. Use shared components and layout patterns before adding new abstractions.
5. Update Pinia state, i18n keys, and tests when user-visible behavior changes.
6. Read `references/traffic-rule-forms.md` for routing, dynamic config, or tag rule forms.

## Output format

Return changed UI flow, route or component paths, API contract impact, validation commands, and any screenshot or browser-test gap.

## Validation

- Run the smallest relevant command: `yarn lint`, `yarn type-check`, `yarn test:unit`, or `yarn build`.
- Confirm request payloads match backend models.
- Confirm new user-facing text has i18n keys.
- For traffic forms, confirm form and YAML paths round-trip the same data.

## Edge cases

- Mesh is injected by the Axios request layer; avoid duplicate mesh handling unless required.
- Tabbed detail pages rely on route metadata and layout state.
- If backend response fields change, check both API client types and view usage.

## References

- Read `references/frontend-structure.md` for directory and route patterns.
- Read `references/traffic-rule-forms.md` for traffic form design and round-trip checks.
