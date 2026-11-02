# Context Compiler core (Issue #1539, PR 1)

The production `AgentComponent` constructs the Compiler with the conversation
Store, template registry and registered tool definitions. Every Act and forced
Answer call uses its complete compiled request. The SDK adapter has no separate
system prompt or tool bindings. Agent construction requires a `CompilationStore`;
the old constructor without storage returns an explicit migration error. Existing
unit fixtures can still construct private test agents to exercise loop behavior.

## Configuration

The Agent spec and `schema/json/agent.schema.json` accept:

```yaml
context_compiler_mode: enabled
context_tokens: 32768
context_safety_margin: 1024
max_context_bytes: 8388608
max_collect_bytes: 8388608
context_window_turns: 10
max_tokens: 3000
template_id: react
template_version: "1"
template_variables:
  Language: Chinese
# Optional: restrict the runtime tool registry and require caller scopes.
allowed_tools: [lookup_service]
tool_required_scopes:
  lookup_service: dubbo:read
# Optional: paths to archived JSON Template definitions (see below).
template_archive_files: []
```

Set `context_tokens` to the configured model's actual context capacity. Input
estimates use `utf8-json-bytes-v1`: serialized UTF-8 bytes plus protocol framing,
including the rendered system/phase strategy, tool names/descriptions/input
schemas, messages and native tool pairs. This deliberately conservative estimate
is not a model tokenizer or a guarantee of an exact token ceiling. Output reserve
and safety margin are subtracted before selection; `max_tokens` is also forwarded
to the provider. Actual provider usage remains in separate fields from the input
estimate. Missing provider usage/cost stays unknown.

The storage Turn quota and recent input window remain separate. Increasing the
window or enabling the Compiler does not increase storage quotas.

`minimal` is the explicit rollback mode: retain the Compiler, template pin,
identity checks, sanitization and mandatory budget; omit completed history and
use the tool-free Answer strategy. It supports basic chat without reactivating
an unbudgeted prompt path. It neither deletes stored records nor changes their
retention. Return to `enabled` after reviewing the captured request fixtures and
resolving the rollout issue; do not leave this mode as an alternate assembly path.

## Templates and migration

Template definitions contain `ID`, `Version`, `Shared`, `Act`, `Answer`,
`AllowedTools` (names) and `Variables` (a name-to-`Required`/`MaxBytes` map).
Only literal text and `{{.Variable}}` are supported; there are no functions,
methods, loops, includes, file/network/environment access or recursive rendering.
Current Agent variables are server-owned strings, required and limited to 4096
bytes each. User text is never evaluated as a template.

The persisted Session template reference includes ID, version and a SHA-256
content identity. The Session API stores the current reference atomically when
creating a new Session. Existing Sessions without a reference are atomically pinned
on their first Compiler-backed interaction. Subsequent attempts cannot replace
that reference. Changing content requires a new named version; keep the exact old
JSON definition in `template_archive_files` while its Sessions remain active.
The archived definition includes both phase policies and the original variable
rules and tool allowlist, not just the shared policy file. A missing version or
missing required variable fails explicitly; it never falls back to new policy.
The Compiler selects values from the server configuration using only the pinned
template's declared variable names, then applies that version's required/length
rules. Adding a variable to a newer version therefore does not break archived
Sessions. Archives pin rules and text, not historical variable values: retain
configuration values needed by old versions. Direct Template rendering still
rejects undeclared input variables.
Archived versions still intersect their tool allowlists with the current runtime
and caller permission sets. The registry rejects conflicting content under the
same ID/version.

GORM migration adds `template_ref`, the `ai_tool_batches` table, and independent
estimate/template/compiler/estimator fields to Usage events. Existing message
rows are retained. Old tool result envelopes are labeled legacy evidence, even
when the SDK JSON codec discarded their original content type. No missing
arguments or call IDs are invented. New batches are rendered once, from the
canonical ledger, as native assistant calls and tool results.

## Canonical tool ledger

One model response creates one ordered batch containing its original call IDs,
names, JSON arguments, states and results, plus Session/Turn/Interaction and model
call association. The entire pending batch commits before any tool executes.
Terminal updates use a batch-version CAS and require the original active Turn;
released or replaced attempts cannot write back. Tool execution also checks the
actual allowlist from the most recent compiled request.

A batch has at most 64 calls and a 1 MiB encoded payload ceiling, and consumes
Session/tenant byte capacity and backend entity/batch capacity. Oversized results
or failed persistence abort the interaction instead of recording success without
evidence. Best-effort interruption recording uses a fresh bounded context. If it
cannot commit, pending remains an unknown execution outcome. Failure and lease
cleanup release the Turn but retain this ledger under the Session's quotas and
lifecycle. Session tombstones immediately block reads/writes, and normal Session
cleanup removes the ledger. Failed records therefore never bypass retention or
accumulate outside quota accounting.

At interaction start, both backends query released-attempt batches independently
of completed/current Turns, using the current Turn as an upper boundary. The
snapshot contains at most 16 recent batches and 1 MiB of payload (also capped by
`max_collect_bytes`). GORM preflights payload sizes before loading them. Rebuilding
the Agent or reopening GORM does not lose this path, including same-key retries.
Completed-history batches are excluded to avoid duplicate injection.

Recovery does not replay calls. Recovered batches are structured, sanitized
evidence, preserving successful results and marking pending/interrupted outcomes
as unknown. A mandatory notice remains even if the storage window or Compiler
budget omits optional details; absence of details never establishes non-execution.
Window/byte omissions report `RECOVERY_LIMIT`. Nothing becomes an orphaned
native request or a fabricated successful result. Exactly-once external tool
execution is not promised, including after a new model decision on a retried
interaction.

## Sources, selection and reporting

Register a `ContextSource` during composition using `Compiler.Register` and a
server-owned `SourcePolicy`. See `compiler_test.go`'s extension fixture: adding
it does not modify the Compiler loop. Registry order, allowed content kinds,
scope/classification, mandatory status and failure behavior are fixed by the
server. `RecentSuffix` applies continuity only within that Source; independent
Sources and independent items can skip oversized candidates without blocking
fitting history. Rendering order remains registry order and chronological order.
Source identity/scope declarations cannot grant privileges. The Compiler
checks ownership, source/version and validity, and gives each Source its own
copy of the collection snapshot. Content cannot become a system message.

The first Sources consume bounded Input, History and Tool snapshots. Whole
completed Turns are selected as a recent suffix. A tool response batch is never
split. Current input and current tool evidence are mandatory; they cannot be
silently truncated. Tool *result data* has deterministic depth/collection/string
limits and explicit truncation/source markers, while call IDs and arguments
remain protocol-complete. Canonical records are never rewritten by this view.

`max_collect_bytes` limits raw Store snapshots and Source groups before rendering;
`max_context_bytes` independently limits rendered Compiler messages plus tool
schemas. Both default to 8 MiB. The collection hard limit and Store quotas remain
in force even if results could compress further. A 100 KiB result can fit a 16 KiB
request budget when the collection limit allows it: sanitize and bound result data
before measuring its rendered view. The same result limits apply to native,
incomplete and recovered batches. IDs, arguments, statuses and batch pairing are
preserved; oversized arguments can still make mandatory content fail.

Store envelopes and omission/report metadata do not consume the rendered message
allowance. Metadata is bounded by Turn limits and `MaxItems`. The Compiler reserves
mandatory bytes/tokens first, then selects whole optional groups against both
dimensions. Required content that cannot fit fails explicitly; optional content
is omitted without treating an oversized collection envelope as a Source failure.

Store queries enforce the Turn count and byte bounds before GORM loads message
payloads. Optional history that exceeds the byte cap is returned as count-bounded
omission metadata; current input that exceeds it fails. Reports identify these
Turns as `SOURCE_BYTE_LIMIT`, and token-pressure omissions as
`GROUP_BUDGET_EXCEEDED`. Reports include IDs, versions, classifications,
sanitization/truncation/omission decisions and estimates, never original text.
Their versions/estimate are available on the model-call trace, and full decisions
are available as sanitized debug metadata;
Usage stores the separate estimate and relevant versions. No dedicated report
table or management UI is introduced.

The history tool retains its old name for compatibility but now requires the
server's prepared active Turn, reads at most three recent Turns within 64 KiB,
and fails if that bounded contract is unavailable. It never calls `AllMemory`.

## Verification and remaining scope

Run from `ai/`:

```sh
go test -count=1 ./...
go test -count=1 -race ./component/context ./component/agent/react ./store/...
```

Tests include shared Memory/GORM contracts, persistent restart and physical
cleanup, CAS races, failed/partial/cancelled batches, legacy evidence, Source
extension/identity/role checks, immutable template versions, group budgets,
minimal rollback and zero provider calls on mandatory overflow. A local HTTP
provider fixture checks the actual OpenAI-compatible request after Genkit:
one system message, phase strategy, tools/schema, call-ID pairing, output limit
and history restored by a fresh Agent. Tests need no external model credentials.

Summary persistence, cursor-based summary recovery, cross-session Memory,
frontend controls, resource scopes, dynamic plugins, a model-specific tokenizer,
and a general cross-source evidence resolver remain outside this PR. New
Summary/Memory Sources must use these same policies and budget/SDK boundary.
