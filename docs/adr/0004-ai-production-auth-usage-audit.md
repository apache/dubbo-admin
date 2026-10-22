# ADR 0004: AI Production Authentication and Privacy-Safe Accounting

## Status

Accepted for Phase 1 PR4.

## Decisions

1. Production HTTP traffic uses a verified JWT bearer token. The initial adapter
   accepts HS256 with a deployment-provided secret and validates signature,
   issuer, audience, expiry, and the `sub`/tenant claims. Identity is never
   read from ordinary request headers or request bodies.
2. Static authentication remains loopback-only development support. Enabling
   the `production_auth` flag requires JWT mode.
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
   `unknown`; it is never converted to zero.
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

The JWT secret, issuer, and audience must be supplied through deployment
configuration. A future JWKS or signed-gateway adapter can implement the same
`security.Authenticator` interface. Source-level token attribution and the
Context Compiler remain later-phase work.
