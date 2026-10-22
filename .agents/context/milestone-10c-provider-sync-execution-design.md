# Milestone 10C1: provider-synchronization execution design

## Outcome

The accepted `v0.2.2C1` design defines one-skill, one-provider transactional
sync, unsync, and recovery. It introduces no executor or mutation CLI.

The transaction contract uses exact plan approval, same-filesystem staging,
stable per-file publication, retained backups, ownership publication last,
durable JSONL journals, JSON checkpoints, and explicit fail-closed recovery.

## Cross-operation decision

Provider transactions acquire the canonical skill lock before a
provider-and-skill lock. Canonical update and removal must refuse while any
supported provider ownership path is present or ambiguous. Users explicitly
unsync providers, perform the separately approved canonical mutation, and
resync selected providers afterward. This prevents stale derived packages and
silent removal of the Codex `SKILL.md` sidecar.

## Excluded from this slice

- Runtime code and CLI routing.
- Multi-skill execution.
- Automatic cascade sync or unsync.
- Directory ownership or cleanup.
- Backup garbage collection.
- Provider CLI, network, credential, dependency, script, or Spec Kit actions.
- Git operations and release publication.

## Next approval gate

Prepare and separately approve the exact `v0.2.2C2` implementation packet,
including Go module boundaries, schema structs, canonical-mutation interlock,
failure injection, changed files, validation, risks, and rollback.
