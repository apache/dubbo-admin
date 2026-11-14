# The AI provider compatibility layer

`ai/component/models/compat.go` spells a request the way a given endpoint
expects. An endpoint that advertises itself as OpenAI-compatible still
diverges: the same thinking switch is `enable_thinking` on Qwen and `thinking`
on DeepSeek-V3.1, and `reasoning_effort` on OpenAI proper.

This document records the decisions behind that layer and, just as importantly,
**which of the declared capabilities actually reach the request today**. The
code refers to the sections here, so the reasoning survives a refactor instead
of living in a single line of Go comment.

## What reaches the request

`ApplyRequest` is the only place the compat block turns into bytes on the
wire. It reads four fields:

| Field | What it does |
| --- | --- |
| `thinking_format` | Picks the spelling of the thinking switch: `enable_thinking`, `thinking:{type}`, `reasoning_effort`, `reasoning:{effort}`, a top-level string, or `chat_template_kwargs`. |
| `thinking_budget_field` + `thinking_budget` | Caps thinking tokens under a named key. The cap only travels when a positive number is configured. |
| `chat_template_key` | The inner key of `chat_template_kwargs` that carries the thinking flag. |
| `max_tokens_field` | `max_tokens` or `max_completion_tokens`. |

Everything else in `Compat` is **reserved**: parsed and validated, but not
applied. See [Reserved fields](#reserved-fields).

## The effort level is not always carried

`reasoning_effort` is an abstract level — `minimal`, `low`, `medium`, `high`,
`xhigh`. Only the formats that carry a level keep it: `openai` sends
`reasoning_effort`, `openrouter` sends `reasoning.effort`. The formats whose
switch is a plain boolean — `qwen`, `qwen-chat-template`, `chat-template`,
`deepseek` — receive "thinking on" and nothing more, so the level is dropped
on the way out.

That is a property of the endpoint, not a mistake in the config, but it is
worth knowing before reading a request log: `reasoning_effort: high` on a
Qwen endpoint produces `enable_thinking: true`, and the endpoint picks its own
budget.

## Field naming by capability

A field states what the endpoint supports, and its doc comment states what
happens when it is disabled. It does not state how to achieve it — that is
the job of `thinking_format`, which is about spelling rather than capability.

Two capabilities that are often confused are therefore two fields:
`supports_forced_tool_choice` says whether a forced choice is accepted at all,
and `supports_forced_tool_choice_with_thinking` says whether it is accepted
while the model reasons. See
[Separate capability bits](#separate-capability-bits).

## Conservative defaults

An unset field picks the spelling the widest set of endpoints accepts, not the
one OpenAI happens to recommend today. `max_tokens_field` defaults to
`max_tokens` rather than `max_completion_tokens`, because the older spelling
is understood by more endpoints.

There is no base-URL sniffing. A gateway in front of a provider defeats it:
the URL says nothing about which vendor is behind it, and a wrong guess is
silent. The provider is configuration, so the quirks are configuration too.

## Two-level merge

A provider block and a model block merge field by field. The model overrides
only the fields it actually sets, and the fields it leaves unset keep the
provider value. There are two levels and not three — a separate profile layer
would be more structure than one protocol needs.

Tri-state booleans (`*bool`) make "unset" distinguishable from "explicitly
false" during the merge. Without that, a model could not turn off a provider
flag the provider had turned on: `override || base` would treat an explicit
false as unset.

## Thinking field probe order

`ThinkingFields` lists `reasoning`, `reasoning_content` and `reasoning_text`,
most specific first, and the reader takes the first non-empty one.

This exists because vLLM renamed `reasoning_content` to `reasoning` and kept
the old name only as an input alias. A client that reads the old name from a
response gets an empty string — no error, no log line. Probing an ordered list
is immune to that rename, and to the endpoints that return both spellings with
identical content: taking the first non-empty value rather than concatenating
avoids duplicating it into the prompt.

The reader also reports **which** field the content came from, so a later turn
can replay it through the same field.

## Separate capability bits

`SupportsForced(thinking)` takes two inputs rather than one, because
endpoints differ in which restriction applies: some reject `tool_choice:
"required"` outright, others only while the model is thinking. Requests count
as thinking when the model reasons by default, not only when thinking was
explicitly requested.

The two bits stay separate so that an endpoint can be described exactly:
"a forced choice is fine, unless the model is thinking".

## Startup validation

`Validate` runs at startup and fails there rather than letting a
misconfiguration surface as an unexplained 400 in the middle of a
conversation. Every message names the offending provider and says how to fix
it.

The checks are the cross-field constraints a JSON Schema cannot express:
an unknown `thinking_format`, a `thinking_budget` without the field name that
carries it, a `chat_template_key` on a format that does not use one, a
`thinking_format: qwen` combined with `supports_strict_mode: true` — two
individually reasonable settings that together always produce a 400.

A field that is configured but not applied is refused here too. That check
runs last, so a mistake with a more specific explanation is reported with
that explanation.

## Reserved fields

These fields are parsed and validated, but nothing applies them to the request
yet. The reason is the same in every case: the part of the request they would
change is built by genkit's `compat_oai` plugin, and that plugin takes a typed
request whose shape this layer cannot reach.

Configuring one is **rejected at startup**. A configuration that promises a
behaviour the request does not have is worse than one that refuses to start,
so the layer does not accept a switch that does nothing.

| Field | What it would do | Why it is reserved |
| --- | --- | --- |
| `thinking_field` | Which response field carries the thinking content. | `compat_oai` maps only the content and the tool calls of a response, so the reasoning field never reaches this layer. |
| `send_back_thinking` | Replays thinking content on later turns. DeepSeek-compatible endpoints answer 400 (`reasoning_content ... must be passed back`) when a tool-calling assistant message omits the field. | The replay would write into the message list, which the plugin builds. |
| `thinking_tags` | The XML delimiters used when thinking is embedded in the text stream. | Belongs to the replay, which is not wired up. |
| `ignore_leading_whitespace` | Drops leading whitespace from streamed content. | The whitespace is dropped while the response is streamed, and the plugin streams it. |
| `supports_developer_role` | Whether the endpoint accepts the `developer` role. | The message roles are chosen by genkit's prompt options. |
| `supports_strict_mode` | Whether tool definitions may carry `strict`. | The tool definitions are built by genkit's tool options. |
| `supports_store` | Whether the `store` field is accepted. LiteLLM-style proxies commonly reject it. | The plugin assembles the request and does not expose the field. |
| `supports_forced_tool_choice` | Whether `tool_choice: "required"` or a named tool is accepted. | The agent never forces a tool choice — it lets the endpoint pick — and the tool-choice value belongs to the plugin. |
| `supports_forced_tool_choice_with_thinking` | Whether a forced choice is accepted while the model reasons. | Same. |
| `requires_tool_result_name` | Whether tool-result messages must carry `name`. | The message list is built by the plugin. |
| `requires_assistant_after_tool_result` | Whether a synthetic assistant message is needed between tool results and the next user message. | Same. |
| `backfill_thinking_field` | Injects an empty thinking field into assistant messages that carry tool calls but are missing it — the cheap alternative to replaying the original text. | Same. |

The fields are kept in the struct rather than deleted, so the design is not
lost and wiring one up later is a matter of reading it. The merge rules stay
one-to-one with the configuration either way.

## What was measured

Measured against DashScope compatible-mode with `qwen3.5-plus`, 2026-10-04.

**Thinking is on by default.** A request that never mentions thinking still
came back with `reasoning_content`, having burned 1648 reasoning tokens out of
the same `max_tokens` pool as the answer. Only `enable_thinking: false`
suppresses it. This is why the cap is sent even when reasoning was not
requested: gating it on an explicit effort would leave the default case
uncapped.

**`enable_thinking` works without streaming.** `enable_thinking: true` with
`stream: false` returned HTTP 200. The "only supports streaming" limit
applies to self-hosted open-weight Qwen3, not to this hosted model, so the
non-streaming agent call is fine.

**A forced tool choice is ignored, not rejected.** `tool_choice: "required"`
with a question that needs no tool produced a prose answer and no tool call —
identical to `auto`. Neither an error nor a tool call. An agent that relies on
forcing gets silence. This is more dangerous than an outright rejection, and
it is why the forced-choice bits are recorded rather than assumed.

**Only `system` is accepted.** `role: developer` returned HTTP 400
("developer is not one of ['system','assistant','user','tool','function']"),
so the plain `system` role is the only correct choice.

**The Qwen family rejects `strict`.** Tool definitions carrying `strict` are
refused, which is why `thinking_format: qwen` and `supports_strict_mode: true`
cannot be combined.

**The empty thinking block ahead of a tool call** was observed with Ollama and
Qwen3, not with this endpoint. It is what `ignore_leading_whitespace` exists
for, and it is unverified here.

## What is not implemented

The read side of the thinking story is not implemented either. `ExtractThinking`
knows how to pull thinking content out of a raw response and which field it
came from, but nothing calls it: `compat_oai` does not hand the reasoning
field over. Neither reading thinking back nor replaying it on a later turn
works today.

The three ways out, none of them taken here:

1. teach genkit's `compat_oai` to map the reasoning field — an upstream change;
2. define the model ourselves instead of going through the plugin — gives up
   the plugin's conveniences;
3. keep a separate channel for the thinking content — a second request shape
   to maintain.

The first is the cheapest and the least duplicative, but it is not this
layer's to make.
