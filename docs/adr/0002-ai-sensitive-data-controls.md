# ADR 0002: AI Sensitive-Data Controls

## Status

Accepted for Issue #1539 Phase 1, PR 2.

## Context

AI inputs and outputs may contain credentials, private configuration, user data,
and tool evidence. The same content currently crosses canonical conversation
storage, model calls, logs, traces, and RAG indexes, but those destinations do
not have the same purpose or retention boundary.

## Decision

- Classify data as canonical data, operational telemetry, derived index data,
  or an ephemeral model-input view.
- Preserve canonical conversation messages without applying the generic
  Sanitizer. Session ownership remains the access-control boundary. Retention
  belongs to PR 3.
- Apply the versioned baseline Sanitizer to all AI log attributes and dynamic
  error strings. Findings contain stable rule IDs, paths, and counts, never the
  matched secret or a reversible fingerprint.
- Keep normal OpenTelemetry spans metadata-only. `capture_content` is fixed to
  `none`; model messages, model output, tool arguments, and tool results require
  a future controlled diagnostic channel with explicit authorization and TTL.
- Build a deep-copied, filtered message view immediately before each model call.
  Never write that view back to canonical conversation storage.
- Apply the same ephemeral model-input policy to RAG query understanding,
  embedding, and reranking providers. Provider calls must never receive the
  raw user query, even when they are outside the ReAct path.
- Allow only documented metadata fields into RAG indexes. Reject a complete
  document or chunk, including its external identifier, when secret detection
  matches instead of silently changing the indexed evidence. Rejection errors
  identify only the batch position and rule, never the external identifier.
- Keep baseline redaction and derived-data rejection always enabled. Feature
  flags cannot disable them.
- Treat secret detection as defense in depth, not as a replacement for identity,
  authorization, encryption, or retention.

## Initial Rules

The first rule version covers authorization and cookie headers, token/API-key/
password assignments, private-key PEM blocks, JWTs, kubeconfig credential
fields, sensitive URL parameters, and bounded encoded-secret inspection.

## Encryption Boundary

PR 2 does not introduce application-managed field encryption because the
project has no key-provider, rotation, or ciphertext-migration contract. TLS and
storage encryption remain deployment requirements. Application-managed
encryption requires a separate ADR and migration plan.

## Consequences

- Existing tracing configurations using `truncated` or `full` content capture
  fail validation and must switch to `none`.
- A document containing a detected secret fails indexing with a stable error;
  operators must clean the source rather than receiving a modified index copy.
- The Sanitizer may produce false positives. Stable rule IDs and match counts
  support diagnosis without exposing the matching value.
