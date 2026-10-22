# ADR 0003: AI Interaction Idempotency and Session Lifecycle

## Status

Accepted for Issue #1539 Phase 1, PR 3.

## Decision

- `POST /api/v1/ai/chat/stream` requires an `Idempotency-Key` of 1 to 128 printable ASCII characters.
- The server hashes a versioned canonical representation of the parsed Session ID and message. Identity is not accepted from the request body.
- `(tenant, owner, session, idempotency_key)` is unique. Reusing a key with a different hash returns `IDEMPOTENCY_CONFLICT`.
- A Session has at most one running Interaction. A different request returns `SESSION_BUSY`; a retry of the running request returns `INTERACTION_IN_PROGRESS`.
- A completed retry replays the persisted final result without invoking the model.
- Interaction acquisition happens synchronously before SSE headers are written or model work starts.
- The HTTP handler owns a newly prepared Interaction until the Agent accepts it by returning initialized Channels. Synchronous panic or nil/invalid Channels leaves ownership with the Handler and marks the Interaction failed immediately. After acceptance, the Agent owns every terminal transition, including startup rejection and panic paths, and must eventually close the Channels.
- Running Interactions use a lease. An expired lease may be reclaimed by the same idempotent request after partial Turn data is removed.
- The Turn quota counts completed Turns, not retained failed Interaction records. Failed Interactions remain available for idempotent retry without permanently consuming the Session's Turn capacity.
- Client disconnect does not cancel model work. Process restart recovery is lease based.
- Session deletion first commits `deletion_pending`, which immediately blocks reads and writes. Canonical and derived cleanup proceeds asynchronously and idempotently.
- Retention uses the same deletion path as an explicit delete.
- The configured retention period is a Store policy. Direct creates and migrations fill missing expiration timestamps from that policy, and legacy fallback checks use the same value.
- A derived artifact may be registered only while its tenant-matched canonical Session is active. Registration and deletion lock the same Session record. Cleanup handlers are selected by artifact kind, must be idempotent, and reconciliation also scans artifacts whose canonical Session is missing or inconsistent.
- A failed derived-artifact cleanup records its attempt, error code, and next retry time. `DerivedArtifact.NextRetryAt` is the authoritative scheduling field in both Store implementations, so a failing artifact temporarily leaves the eligible batch and cannot permanently starve later cleanup work. Cleanup scans treat malformed canonical source IDs, missing Sessions, and tenant mismatches as orphaned artifacts instead of silently skipping them.
- `CleanupJob` is the canonical-source progress projection, not a second task scheduler. Its attempt count records source-level failure observations, its error code is the most recently observed failure, and its next retry time is the earliest retry across remaining artifacts. A nil job retry time means at least one artifact is immediately eligible or no artifacts remain; canonical cleanup alone transitions the job to `completed`.
- Cleanup workers use keyset pages ordered by `(sort_time, tenant_id, id)` and retain the cursor between ticks. Session and artifact scans are tenant-aware through this deterministic tuple; an empty page resets the cursor for the next full sweep. MemoryStore checks cancellation between candidates and Page APIs are the cancellable worker path.
- The `max_in_memory_entities` limit applies only to MemoryStore. Persistent backends are bounded by Session, Turn/Event, per-Session byte, and per-tenant byte quotas.
- Session state, CleanupJob, and DerivedArtifact records are the authoritative cleanup work state. The outbox is a producer-side transactional notification boundary for a future publisher; `published_at` means a consumer has acknowledged publication and is not set merely because local cleanup completed. Periodic reconciliation remains the crash-recovery mechanism in PR 3.
- The UI interaction controller, rather than the HTTP adapter, creates the idempotency key and preserves it with the request context snapshot for logical retries.

## Consequences

The HTTP layer owns request-key validation and canonical hashing. Stores own atomic idempotency, concurrency, quota, and deletion decisions. The Agent consumes a prepared Interaction and reports its terminal state; it does not invent production Interaction IDs.
