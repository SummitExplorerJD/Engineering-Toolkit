# ADR 0014: Local intelligence is advisory and model-optional

## Status

Accepted for the `v0.3.0A` design slice on 2026-10-23. Configuration schema
changes, SQLite changes, network access, local model inventory or invocation,
implementation, Git operations, and publication require separate approval.

## Context

Engineering Toolkit can use workspace context to accelerate profile, skill, and
verification decisions. A local model can improve ranking and explanations, but
model output is non-deterministic and may be malformed, stale, or influenced by
untrusted repository text. Making it canonical or executable would conflict
with deterministic planning, explicit approval, filesystem ownership, and the
optional nature of SQLite.

## Decision

Keep deterministic policy authoritative. Place local-model support behind an
optional application adviser interface. The adviser consumes only bounded,
secret-filtered excerpts obtained through a context port after deterministic
eligibility checks. It returns a versioned advisory report that cannot be used
as approval or accepted directly by an executor.

Use an explicitly enabled loopback Ollama adapter as the initial implementation
direction. Require an exact configured model tag from an allowlist, bounded
requests and responses, deadlines, cancellation, structured-output validation,
redirect refusal outside loopback, and no tool execution. Never install, start,
or pull Ollama or a model. Preserve a deterministic model-free path for every
correctness-critical operation.

Keep filesystem artifacts canonical. SQLite may accelerate the context port but
must prove freshness or allow an explicit canonical fallback. Model and protocol
adapters never receive a raw database handle.

## Consequences

- Engineering Toolkit remains usable without Ollama, a model, or SQLite.
- Model quality may change recommendation ordering but cannot change policy or
  grant mutation authority.
- Local operation reduces external disclosure, but bounded prompt construction
  and secret exclusion remain mandatory.
- Users must install and manage Ollama models independently.
- Exact model compatibility requires environment-specific validation before a
  release can claim support.

## Alternatives considered

- **Let the model generate executable plans:** rejected because output is
  non-deterministic and cannot establish approval or trustworthy preconditions.
- **Store knowledge canonically in SQLite:** rejected because the filesystem is
  the accepted source of truth and the index must remain rebuildable.
- **Automatically pull a missing model:** rejected because it adds unapproved
  network, disk, licensing, provenance, and supply-chain effects.
- **Allow remote OpenAI-compatible endpoints initially:** rejected because it
  expands credential and data-disclosure boundaries before local behavior is
  proven.
- **Require a model for profile selection:** rejected because correctness and
  automation must retain a deterministic offline path.
