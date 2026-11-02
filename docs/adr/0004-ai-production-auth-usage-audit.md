# ADR 0004: AI Production Authentication and Privacy-Safe Accounting

## Status

Accepted for Phase 1 PR4.

## Decisions

1. Production HTTP traffic uses PR #1600's Admin-issued RS256 bearer tokens and
   JWKS verifier. PR #1600 is a prerequisite of this PR; its Admin endpoints,
   key ring and frontend token lifecycle are reused. The verifier validates
   signature, issuer, audience, expiry, issued-at and subject. A signed
   `tenant_id` selects the tenant; tokens without it use the trusted deployment
   `auth.tenant_id` (default `default`, for the single-tenant Admin issuer).
   There is no `tenant` alias or request-header/body identity fallback. Roles
   do not imply scopes or internal system capabilities.
2. Static authentication remains loopback-only development support. Enabling
   the `production_auth` flag requires `auth.enabled: true`. One `auth` block
   configures both modes; there is no parallel `authentication` block.
3. Usage and audit records contain identifiers, lifecycle status, reason codes,
   model/provider metadata, versions, and token counts. They never contain
   complete prompts, model responses, tool arguments, or tool results.
   Durable audit events are limited to interaction start/end/error/cancel/degrade
   and model/tool call end. Call-end records include failures; separate call-error
   records would duplicate them. Iteration, stage, and call-start details remain
   in tracing. Usage is written once per model call, on call end.
   Model call audits retain the same CallID as Usage; tool call audits retain
   ToolCallID from the tool request. Interaction-level events leave call IDs empty.
   ToolCallID is interpreted within its interaction, not as a global identifier.
4. Missing provider usage or pricing is represented by a missing value or
   `unknown`; it is never converted to zero. OpenAI-compatible HTTP middleware
   observes the nullable usage object before Genkit allocates a zero value,
   including final SSE usage-only chunks. Explicit provider zero remains zero.
   All three token fields must be present and nonnegative. Per-call observations
   are isolated and only usage metadata is retained. Response/line parsing uses
   a transient 1 MiB bound; oversized records remain unknown.
5. Usage and audit writes are tenant and subject scoped at the Store boundary.
   The Memory and GORM stores expose the same `UsageAuditStore` contract.
6. Usage and audit records use an independent 30-day default retention and
   bounded per-backend capacities. Cleanup requires the internal system
   capability, cannot remove records inside the retention window, and runs in
   bounded batches (Memory bounds scanned records with a continuing cursor;
   GORM selects expired rows using the timestamp index). GORM reserves capacity
   and inserts records in one transaction; cleanup releases slots transactionally.
   Session deletion does not silently erase the audit trail.
7. Recorder writes preserve trusted identity but use an independent five-second
   persistence deadline, so model cancellation does not cancel final accounting.
   Writes remain best effort on storage failure or capacity exhaustion. Schema
   revision is `v1`, metadata/ownership policy revision is
   `subject-owned-metadata-v1`, and recorder implementation revision is
   `usage-audit-recorder-v1`. These identify contracts, not the model version.

## Consequences

Deployments configure `auth.jwks_url`, `auth.issuer`, `auth.audience`, and the
trusted fallback tenant. Invalid configuration is a startup configuration error,
distinct from a request authentication failure. Startup fails if Admin JWKS
cannot be loaded. The shared JWKS cache and failed-refresh cooldown are 30
seconds; failed refreshes reject expired keys. Publish a new public key at least
30 seconds before activating its signing key, and retain old keys until issued
tokens expire. Missing audit identity emits a metadata-only warning and skips
persistence. Source-level token attribution and the Context Compiler remain
later-phase work.
