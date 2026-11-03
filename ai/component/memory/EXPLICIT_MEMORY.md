# Explicit cross-session Memory (Issue #1539, PR 3)

Memory is a current-user fact store, distinct from the in-process conversation
backend. There is no automatic extraction, shared/resource scope, semantic search,
or UI in this PR. The memory panel and chat selectors belong to PR 4.

## Configuration

In the agent component spec, enable injection independently from summaries:

```yaml
explicit_memory:
  enabled: true                 # default false
  max_items: 16
  max_bytes: 16384
  max_tokens: 4096              # conservative compiler estimate, not tokenizer tokens
```

In the memory component spec, both backends share storage limits:

```yaml
explicit_memory:
  max_records: 128
  max_value_bytes: 4096
  max_user_bytes: 524288
  max_ttl_seconds: 31536000
```

Memory records consume the tenant byte quota and in-memory entity quota. Control
state (`generation`, `state_version`, and `enabled`) is stored per user and does
not consume the tenant content-byte quota, so disabling remains available when
the tenant is full. Values may omit expiry; when provided, expiry must be in the
future and within the TTL limit. Expired records are not recalled and still
occupy bounded storage until deleted or cleared. Disabling injection does not
delete data or bypass dependency validation. Management endpoints remain
available with injection disabled.

The AI service currently has no project directory integration. The MVP uses an
explicit server-owned catalog in the server component spec:

```yaml
memory_projects:
  - id: checkout-project         # stable ID, never a display name
    tenant_id: tenant-a
    owner_ids: [user-a, user-b]
```

Only listed tenant/user pairs can use this project in saved applicability or chat
matching. An empty catalog rejects all nonempty project references. This catalog
does not prove resource/tool authorization; tools retain their own permission
checks. `Router.SetMatchContextValidator` accepts a real directory/policy adapter
when one becomes available. The catalog is read at server startup.

## HTTP contract

All endpoints use the existing AI authentication and response envelope under
`/api/v1/ai`. Read `GET /memory-settings` before a mutation, or use the settings
returned with a list. Settings initially contain generation 1, state version 1,
and enabled true.

| Endpoint | Behavior |
| --- | --- |
| `GET /memories?limit=32&cursor=...` | Stable key/ID order; bounded list and next cursor |
| `POST /memories` | Explicit confirmation; create or return an identical fact |
| `PUT /memories/:id` | CAS update; requires current record version |
| `DELETE /memories/:id` | Requires generation, state_version and record version |
| `POST /memories/clear` | Requires generation and state_version; increments generation |
| `GET /memory-settings` | Current generation, state_version and enabled |
| `PUT /memory-settings` | Requires generation, state_version and enabled |

Example creation:

```json
{
  "generation": 1,
  "state_version": 1,
  "scope_type": "user",
  "memory_type": "task_context",
  "fact_key": "project.background",
  "value": "Checkout uses an application-level registry.",
  "applicability": {"projectRef": "checkout-project"}
}
```

The server supplies tenant, owner, scope ID, canonical key, confirmation status
and provenance. Unknown JSON fields and extra JSON documents are rejected. Chat
and Memory mutation request bodies are capped at 32 KiB. Secret findings, obvious
policy override syntax, invalid types/scopes/keys and overlong values are rejected.
This lexical validation is not a general prompt-injection detector: content stays
untrusted data, never system policy or tool authorization.

Supported fact categories:

| Type | Fact key | Applicability |
| --- | --- | --- |
| preference | response.language, response.style | Empty |
| task_context | project.background | Explicit project; optional task |
| diagnostic_constraint | diagnostic.priority, diagnostic.read_only | diagnosis task; optional project |

`diagnostic.read_only` accepts only the string `true`. It is advisory context that
can narrow a requested workflow; it never grants permissions or constitutes a
new hard authorization layer. Enforced tool permissions remain server policy.
Source Session is optional and immutable after creation. A source must belong to
the exact current user, even for a Session administrator.

Stable errors include `MEMORY_INVALID` (400), `MEMORY_VERSION_CONFLICT` (409),
`MEMORY_DISABLED` (409), `MEMORY_NOT_FOUND` (404), `RESOURCE_NOT_ACCESSIBLE` (404),
and `RESOURCE_EXHAUSTED` (429). Audit records identify the actor and operation,
without Memory values. They use the existing best-effort audit policy/flag.

## Identity, selection and chat compatibility

The canonical key is `v1:` plus SHA-256 of Go JSON encoding the fixed-order
`memory_type`, `applicability`, `fact_key` fields. Applicability orders projectRef
before taskType, trims surrounding whitespace, and omits empty members. Value is
trimmed but is not part of identity. Original identity fields are stored alongside
the key. Matching uses the same normalization. There are no unordered sets in
the first schema. Adding such fields requires a new normalization/key version.

Updating a fact preserves its record ID and advances version, with supersedes
pointing to the prior version. An identical save retains ID/version; creating a
different value under an occupied identity conflicts, so use PUT to correct it.
Identity changes conflict with an occupied target, instead of overwriting it.
Deleted IDs cannot be updated or reused. A new explicit creation gets a new ID.

Chat accepts an optional object:

```json
{"sessionID":"session-id","message":"help diagnose","matchContext":{"projectRef":"checkout-project","taskType":"diagnosis"}}
```

Task types are diagnosis, configuration and inspection. No selection recalls
only general preferences; the server does not inherit project/task from an older
request or infer project identity with a model. Relevant records are selected in
stable canonical-key/ID order, with separate count/byte/token limits. No keyword
fallback is enabled. An explicit current-user `fact_key=value` line excludes a
conflicting Memory candidate for this request and reports the reason; it does
not update stored Memory. Other language is not parsed into confirmed facts.

Absent/empty matching retains the existing v1 chat request hash exactly. Nonempty
normalized matching uses a v2 envelope. Equivalent normalized hints share a hash;
changing matching under an existing Idempotency-Key conflicts. Existing completed
idempotent replies remain replays of the original interaction, not new model calls.

## Consistency and retention

Both backends serialize all mutations on a bounded per-user aggregate. The memory
backend uses its existing Store mutex; GORM adds `ai_memory_ledgers`, queried by
hashed tenant/user identity and checked against the original identity. Aggregate
payload size is bounded (default 512 KiB of records, hard policy ceiling 2 MiB),
so neither recall nor management loads unbounded history or other users' data.
There is no cross-request content cache.

GORM uses a transaction, user-row write lock/revision CAS and tenant byte
reservation. Current locking reads avoid stale snapshots under MySQL repeatable
read. Source checks, content, capacity and cleanup registration commit together.
Source tombstones serialize on the same user row and remove associated Memory;
the cleanup handler remains idempotent. The in-memory backend likewise removes
associated Memory when the source is tombstoned; the existing worker finalizes
cleanup relationships. Explicit record deletion/clear also removes its relationship
metadata in the same atomic operation.

Clear removes records and advances generation. Disabled state advances its own
version and blocks writes. Saves carry both versions, so earlier requests cannot
commit after clear/disable. A fresh explicit save after clear is allowed. The
generation/state row survives clearing and GORM restart.

Only Memory actually selected within the final budget contributes direct
dependencies. Selected summaries and historical answers propagate their own
dependencies, including through tool batches and released-attempt recovery.
ReAct accumulates dependencies across calls and records them on the Turn before
Provider dispatch. The compiler revalidates just before dispatch, with one bounded
reselection when state changes. Repeated invalidation fails explicitly. No database
transaction is held across Provider IO; already sent input cannot be recalled.
Answers finishing after a mutation keep their original dependencies.

Invalid dependencies exclude entire historical model answers and associated tool
groups in a temporary view, retaining independent user statements. Summary reads
return an empty rebuild view with the physical CAS version intact; coverage and
fragment checkpoints no longer suppress history. Summary generation applies the
same filtering, and atomic commit rechecks dependencies, so it cannot restore an
old value after deletion. Rebuilding starts at the earliest cursor with existing
time/call limits. Canonical chat records are never rewritten by these filters.

The legacy `memory_all_by_session_id` tool is excluded from production ReAct's
authorized tool set because its result has no transitive Memory attribution.
History remains available through the bounded Source pipeline. Session admin
permission does not grant Memory management or inject either user's direct
Memory into another owner's Session.

Deleting Memory does not erase independently authored user statements, earlier
idempotent response replays, or copies already sent externally. To remove canonical
chat records, use Session deletion. Legacy data without dependency metadata cannot
be semantically attributed retroactively.

## Migration and verification

Run the existing GORM migration; it creates the bounded user ledger table and
recalculates tenant usage including Memory and dependency metadata. No canonical
message conversion or key backfill is needed. PR 2's dependency fields are now
resolved; the new state-version field prevents disable/re-enable from reviving
older dependent answers. Empty legacy dependencies preserve previous behavior.
The pure in-memory backend still loses all records/settings on process restart.

Shared backend tests cover identities, replacement, ownership, clear races,
source deletion, summary/fragment invalidation and atomic generation races. SQLite
tests check restart and injected artifact-registration rollback. Compiler and HTTP
Provider fixtures check matching, independent token budgets, message roles,
cross-session injection and transitive old-answer filtering. HTTP tests cover
matching validation, field propagation, stable errors and request-hash migration.
Production MySQL/PostgreSQL concurrency still needs deployment validation; SQLite
fixtures do not establish production locking behavior or model quality.
