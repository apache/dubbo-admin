# Conversation summary (Issue #1539 MVP PR 2)

The summary service runs once per interaction after the active Turn and mandatory
request budget are checked, before the first main model call. It fixes a terminal
history snapshot, advances the oldest uncovered range, and reuses that snapshot
throughout ReAct. Summary calls use a separate tool-free template and the same
Compiler/SDK adapter budget boundary, without recursively invoking the Agent.

## Configuration

In the Agent component spec:

```yaml
context_window_turns: 10
conversation_summary:
  enabled: true                 # default false; independent rollback switch
  # model: provider/model       # defaults to the Agent model
  # context_tokens: 32768       # defaults to Agent capacity; set for another model
  timeout_seconds: 10           # total snapshot/read/generate/commit deadline
  max_calls: 2                  # normal advancement, including fragment calls
  recovery_max_calls: 8         # bounded recovery after lag/partial progress
  fragment_bytes: 8192
  max_output_bytes: 8192
  output_tokens: 2048
  trigger_headroom_turns: 2
```

Both model capacities must reflect the deployed models' actual context windows.
Output reserve, safety margin, instructions, previous summary, citations and new
evidence all count against the auxiliary request. Estimates remain conservative
`utf8-json-bytes-v1` estimates, separate from provider usage. Configure output size
and fragment size so the previous summary plus new evidence fits. Impossible
auxiliary budgets degrade to recent complete Turns; mandatory main-request
overflow fails before any auxiliary or main model invocation.

The Memory component's `max_turns` is still a storage quota, not the input window.
For long-conversation fixtures set it above the test length (for example 200).
Canonical messages are retained; summarization does not reset counters or provide
unlimited storage. Terminal journal metadata has an independent 10,000-attempt
per-Session ceiling; in-memory entity limits also apply. Summary/checkpoint payloads
consume Session/tenant byte capacity and have an additional 128 KiB hard ceiling.

## Coverage and recovery

`ai_history_entries` records immutable terminal positions independently from Turn
IDs, including failed/released attempts whose messages were removed. New positions
are allocated in the same transaction/lock as completion or release. Cursors use
keyset queries bound to the Session, active attempt and fixed upper position;
neither contiguous Turn IDs nor offset pagination defines coverage. Metadata
queries determine a recent-window target without loading old payloads. Token/byte
pressure in the main preflight can bring that target forward.

Only the next uncovered Turn is processed. Its records are ordered as user input,
canonical tool batches, then the assistant answer. Records are packed into one
batch under `fragment_bytes`, including citation labels and evidence framing.
A short user/answer Turn, including small tool batches, needs one model call and
one summary commit; only oversized Turns create partial checkpoints. Each record
keeps its own citation and authority. Multiple Turns are not packed together.
Turns without canonical batches retain their original record order. Large tool
results are structurally limited before fragmenting the temporary evidence view; canonical protocol records remain
complete. Large GORM messages are read with binary `SUBSTR` byte bounds. Records
are inspected with bounded reads before emitting their first fragment; sensitive
records become explicit omission evidence. Completed inspection metadata is bound
to the immutable message identity or tool ledger version, the sanitizer rules and
the evidence transformation version. GORM saves it on the canonical row; Memory
keeps it in a Session-scoped record map. Subsequent fragments and commit validation
reread only the bounded fragment, preserving cross-fragment secret detection
without repeated full scans. Rule/version changes require a new inspection;
timeouts and errors never produce a reusable pass. Fragment boundaries preserve UTF-8.

Each inspection reserves 64 bytes of Session/tenant capacity (and one Memory
entity), once per record. Rechecking a changed rule/version replaces that metadata
without charging again. GORM migration preserves this accounting. Inspection is
committed by the read transaction before model generation and can survive a failed
summary commit. It contains no canonical content, needs no separate artifact, and
is removed with its canonical record/Session. Memory terminal positions are indexed
when completion/release occurs; opening a snapshot never backfills the history.

The service validates six structured arrays, field/count/size limits, sensitive
content and source citations. Confirmed facts require user-assertion or successful
tool provenance; assistant/unknown/uncertain-tool citations cannot be promoted.
Intermediate progress and its citations persist separately from the last complete
summary. Coverage advances only when all records/fragments of a Turn are complete.
Failed, interrupted and pending tool outcomes remain explicitly uncertain and are
never replayed by this service.

Each bounded fragment commits independently. Commit recomputes the source digest
and cursor transition, validates the active attempt/lease and expected summary
version, and atomically updates payload, coverage, capacity and derived-artifact
registration. Session Touch/version changes do not invalidate immutable history.
Timeouts, invalid output or CAS/storage failure stop this interaction's advancement;
the last valid summary remains usable. Later requests start at the oldest pending
position. Recovery can outpace new history when its bounded processing capacity is
sufficient; it does not promise to catch up with arbitrary input growth.

## Final selection and rollback

The registered SummarySource returns an untrusted evidence block, not system
policy or current user input. Compiler v2 first selects summary plus uncovered
history. Only a finally selected summary suppresses covered Turns and recovery
details with a known covered terminal position. Recovery positions are separate
from complete-Turn history gap accounting. Unknown legacy positions remain eligible
for selection; the mandatory no-automatic-replay notice is never suppressed.
If the summary does not fit, selection runs once more against the original bounded
history and recovery details, retaining the recent suffix whenever it fits.
Version/source revalidation failure also causes
bounded re-compilation without the summary; ownership/deletion failures still fail
the request. Reports contain per-item decisions, summary target/coverage/calls,
partial/backlog state, and omitted cursor intervals, without canonical content.

`enabled: false` and Compiler `minimal` mode skip both summary generation and
injection. They leave existing records under the same retention/cleanup rules and
keep ordinary Compiler-backed conversation working.

Auxiliary calls have distinct call IDs, `call_type=summary`, `auxiliary=true`,
Interaction identity, provider/estimated usage and unknown-cost semantics. They do
not masquerade as main model calls or tool executions. Existing streaming usage
continues to describe the main Agent; persisted auxiliary events are separate.

## Migration, deletion and PR 3 boundary

GORM migration adds `ai_history_entries`, `ai_summaries`, message role metadata and
Turn dependency metadata. It backfills completed Turns in completed-at/id order.
Previously released tool batches are retained as legacy failed-attempt evidence;
their original terminal order and deleted messages cannot be reconstructed. They
receive stable trailing migration positions rather than invented original order.
Historical messages without the new role metadata have unknown citation authority;
their serialized canonical role/content remains available as unconfirmed evidence.
Migration preserves existing summary/checkpoint byte accounting on restart.

Summary and checkpoint share a Session-derived artifact. Session deletion/expiry
immediately blocks reads and commits; a system-only cleanup handler removes the
payload idempotently before normal Session cleanup removes its terminal journal.
Delete-vs-commit follows the same Session lock as other lifecycle writes. Artifact
write failure rolls back the summary and its capacity reservation together.

Server-owned Memory ID/version/generation dependencies have storage fields on Turns,
terminal entries and summaries. Rolling/partial processing unions and preserves
them without accepting model-provided dependency claims. PR 3 will supply answer
dependency writes and Memory-state resolution/filtering; this PR does not expose
Memory CRUD, recall, automatic extraction or UI controls.

Pure in-memory state is lost on process restart; GORM preserves complete summaries
and intermediate cursors. No background summary queue or unbounded history read is
introduced. Model summarization still requires quality evaluation beyond schema
and provenance validation.

## Verification

From `ai/`:

```sh
go test -count=1 ./...
go test -race ./store/... ./component/context ./component/memory/... ./component/agent/react ./component/hooks
```

Shared Memory/GORM contracts cover coverage, packed short Turns with independent
citations and one model call, fragments, uncertain tools, ownership, CAS, invalid
commits, deletion and cleanup. GORM tests cover restart, inspection reuse and rule
invalidation, bounded payload-read counts for a 1 MiB record including all commits,
and injected artifact rollback. Cross-boundary secrets and interrupted inspections
have direct regressions. Service fixtures cover bounded timeout and catching up
while new Turns arrive. Compiler tests cover summary exclusion by tokens/bytes, disabled or
invalid summaries, covered recovery detail deduplication with mandatory notice,
and token/byte-budget fallback. An HTTP provider fixture checks actual SDK output,
summary injection without duplicate original Turns, tool-free auxiliary calls,
output limits, accounting and zero calls on mandatory overflow. CI needs no real
model credentials; production MySQL/PostgreSQL locking and model quality require
deployment validation in addition to SQLite contract fixtures.
