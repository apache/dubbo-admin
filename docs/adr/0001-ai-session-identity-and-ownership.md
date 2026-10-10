# ADR 0001: AI Session Identity and Ownership

## Status

Accepted for Issue #1539 Phase 1, PR 1.

## Context

The AI HTTP service has no existing user authentication or tenant model. Session and message operations currently use only a Session ID, so the storage layer cannot prevent one caller from reading or mutating another caller's conversation.

## Decision

- Introduce a server-owned `Subject` and explicit `AccessContext` containing tenant, user, roles, scopes, and authentication method.
- Use a pluggable HTTP `Authenticator`. The first adapter is a server-configured static identity for local development only; it ignores identity-like client headers and request fields but is not production authentication.
- Require every Router constructor to receive a non-nil `Authenticator`; there is no implicit default identity below the validated ServerComponent boundary.
- Reject system maintenance access in HTTP authentication middleware even if a custom Authenticator returns it. System access must never enter a request handler.
- Default static mode to `127.0.0.1`, no cross-origin access, tenant `default`, and user `local-user`. Reject non-loopback listeners unless `allow_insecure_static_auth` is explicitly enabled. Production multi-user deployments require a later JWT or signed-gateway adapter.
- Persist tenant and owner on every Session. Every SessionStore and MessageStore operation receives `AccessContext` explicitly, and both Memory and GORM stores enforce ownership.
- Reserve `ai:sessions:admin` for access to other owners in the same tenant. Cross-tenant administration is not supported in this PR.
- Return the same external `RESOURCE_NOT_ACCESSIBLE` error for missing and unauthorized Sessions.
- Keep legacy adoption disabled by default. When `adopt_legacy_sessions` is explicitly enabled, a system maintenance capability may assign the configured tenant and owner only to rows where both ownership fields are empty. Partially owned rows remain isolated for explicit repair. New writes require complete ownership.
- Separate privileged expiration cleanup and legacy adoption into `MaintenanceStore`; ordinary business code depends only on `SessionStore` and `MessageStore`. The ServerComponent owns startup adoption, while the Session Manager receives maintenance access only for lifecycle cleanup.
- Treat resource references from clients as untrusted input. A later resource Policy must produce authorized scopes with policy-version and decision metadata before a Store or Context Source may consume them.
- Keep authentication and baseline ownership enforcement always enabled; they are not feature-flagged.

## Consequences

- Local deployments using the default static identity remain compatible, but static mode must not be described as production user isolation.
- Store interface changes are intentionally breaking inside the AI module so callers cannot accidentally query user data with only a Session ID.
- Tenant and owner database columns remain nullable during schema migration so existing populated tables can be upgraded before legacy rows are adopted. Application validation rejects unowned new Sessions.
- Session expiration continues to use the existing 24-hour activity window. Touching a Session advances its version and expiration; configurable retention belongs to PR 3.
- PR 1 does not add a deletion status or tombstone. The deletion state machine, outbox, derived-data cleanup, and reconciliation belong to PR 3.
- `MaintenanceStore` and the private marker reduce accidental dependency and string-based forgery, but they are not a capability-security boundary against arbitrary code in the same Go process: trusted internal packages can still request the public maintenance interface and system context. A stricter boundary requires bootstrap-owned private capabilities or a separate maintenance process.
