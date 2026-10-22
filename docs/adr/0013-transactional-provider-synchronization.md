# ADR 0013: Provider synchronization uses explicit unsync and dual locking

## Status

Accepted for the `v0.2.2C1` design slice on 2026-10-22. Implementation,
provider conformance, Git operations, and publication require separate
approval.

## Context

Read-only plans can safely describe provider projections, but publication spans
generated files and a per-file ownership record. A crash between those writes
can leave files that no longer match the prior record. Codex adds another
constraint: its generated `SKILL.md` is inside the canonical package directory,
while canonical skill update currently replaces that directory as a complete
unit. Concurrent or uncoordinated mutation could therefore publish stale output
or silently remove a provider sidecar.

## Decision

Execute one provider and one canonical skill per transaction. Rebuild the exact
plan before accepting approval, stage all rendered bytes on the workspace
filesystem, publish outputs in stable order with per-file backups, verify the
complete output set, and publish the ownership record last. Persist a durable
JSON plan, JSONL journal, JSON checkpoint, staging, backups, and explicit lock
state. Recovery validates all evidence and either completes, restores the exact
before-state, or refuses while preserving ambiguous content.

Acquire the existing canonical skill lock before a provider-specific skill
lock. Canonical skill update and removal fail while any supported provider
ownership path is present or ambiguous; malformed, linked, and unreadable
records are blocking. Removing a projection is a separate approved `unsync`
transaction derived from a valid ownership record. Users unsync providers
before canonical mutation and explicitly resynchronize afterward. No operation
silently cascades across providers.

Never claim a provider package directory. Successful unsync retains output and
ownership backups and leaves empty directories in place. Directory cleanup,
multi-skill execution, automatic cascading, and backup garbage collection are
future decisions.

## Consequences

- Approval remains selective by provider and skill.
- Ownership never claims bytes that have not been published and verified.
- Interrupted publication is recoverable without trusting the ownership record
  alone.
- Codex sidecars cannot race with complete canonical package replacement.
- Canonical updates require explicit unsync and later resync, which is safer but
  more operationally verbose.
- Empty provider directories and retained backups require future explicit
  maintenance rather than automatic deletion.

## Alternatives considered

- **Publish ownership first:** rejected because the record could claim files
  that were never published.
- **Automatically synchronize every provider during canonical mutation:**
  rejected because it combines independent approval scopes and adapter risks.
- **Let canonical update preserve Codex output implicitly:** rejected because a
  canonical transaction would then mutate provider-owned state without a
  provider plan.
- **Use only a provider lock:** rejected because the canonical source could
  change during rendering or publication.
- **Execute multi-skill plans initially:** rejected because partial publication
  and recovery would broaden the first transaction contract unnecessarily.
- **Delete generated directories on unsync:** rejected because per-file
  ownership does not establish directory ownership.

