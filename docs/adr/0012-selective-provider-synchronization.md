# ADR 0012: Selective provider synchronization uses derived per-file ownership

## Status

Accepted for the `v0.2.2A` design slice on 2026-10-22. Read-only planning,
execution, recovery, provider conformance, Git operations, and publication
require separate approval.

## Context

Engineering Toolkit owns provider-neutral canonical skills under
`.agents/skills/SKILL_ID/`, while Codex, Claude Code, and OpenCode discover
provider-facing `SKILL.md` packages in different roots. Codex shares the
canonical root. The repository also contains official Spec Kit skills under
that root, and provider roots may contain user-created or provider-native
content.

A directory-level ownership claim would therefore be unsafe. Copying only the
instruction text without accounting for resources, permissions, or dependencies
could also create a silently lossy provider package. The providers expose
similar Agent Skills structures, but their discovery paths and optional
metadata are not a promise of full semantic parity.

## Decision

Treat provider output as a deterministic derived projection of one exact
canonical package. Synchronization is explicit for one provider at a time and
uses these initial targets:

- Codex: `.agents/skills/SKILL_ID/`;
- Claude Code: `.claude/skills/SKILL_ID/`; and
- OpenCode: `.opencode/skills/SKILL_ID/`.

Generate a provider-facing `SKILL.md` with supported discovery metadata and the
canonical instructions. Copy a declared resource only when its relative layout
and semantics are supported without transformation. The initial executable
scope rejects scripts, non-empty permission requests, unresolved dependencies,
and any other unsupported or lossy mapping.

Record ownership per generated file below
`.engkit/providers/PROVIDER/SKILL_ID.json`. Each record contains the adapter
contract version, canonical source digest, output path, and last published
digest. A later operation may replace or remove a file only when the observed
digest equals that record. Changed or unrecorded files are conflicts and remain
untouched.

Plans are immutable JSON and bind the canonical package, adapter contract,
observed paths, prior ownership, exact output bytes and digests, mapping
quality, conflicts, risks, approvals, and verification. Lossy or unsupported
plans are reviewable but non-executable. Execution will use a separately
approved JSONL journal, JSON checkpoint, same-filesystem staging, and explicit
recovery design.

Spec Kit and provider-native content remain external. Sharing a parent
directory never transfers ownership. Synchronization does not create runtime
permission policy, execute resources, contact networks, access credentials,
resolve dependencies, initialize Spec Kit, or mutate canonical packages.

## Consequences

- Canonical authored state remains portable and provider-neutral.
- Codex can consume a generated file beside canonical files without the adapter
  owning the package directory.
- Claude Code and OpenCode outputs are isolated from canonical storage.
- Exact per-file records permit safe updates and removal while preserving
  external content.
- Some valid canonical skills will initially produce non-executable plans until
  their resources and policy semantics have an accepted mapping.
- Provider contract changes require conformance review and an adapter-contract
  revision rather than silent output drift.

## Alternatives considered

- **Use `.agents/skills` as one universal provider output:** rejected because it
  overlaps canonical and Spec Kit ownership and prevents selective providers.
- **Claim whole generated directories:** rejected because an existing provider
  package may contain user-owned files.
- **Render only instructions and ignore other manifest fields:** rejected
  because it silently creates lossy packages.
- **Generate provider permission configuration:** rejected because declared
  canonical permissions are requests, while enforcement belongs to each
  provider runtime.
- **Treat Spec Kit as another rendered provider:** rejected because Spec Kit is
  independently versioned upstream workflow infrastructure.
