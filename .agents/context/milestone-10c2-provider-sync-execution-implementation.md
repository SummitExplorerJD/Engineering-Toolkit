# Milestone 10C2: provider-synchronization execution implementation

## Outcome

The `v0.2.2C2` implementation applies one exact, approved provider projection
for one canonical skill. It adds explicit sync, unsync, and recovery commands,
durable transaction evidence, retained backups, ownership-last publication,
and a canonical-mutation interlock.

## Module boundary

- `providersync` remains responsible for deterministic planning, unsync
  authority, materialized bytes, and ownership records.
- `providersyncexec` owns staging, dual locking, publication, verification,
  journaling, checkpoints, retained backups, and recovery.
- `skillexec` checks provider ownership before canonical update or removal.
- `cli` rebuilds plans and routes exact approval IDs; it contains no transaction
  policy.

## Safety decisions

- A provider transaction acquires the canonical skill lock before its
  provider-and-skill lock.
- Creation and replacement require same-filesystem atomic no-replace support;
  unsupported filesystems fail closed.
- Ownership is published only after all outputs verify.
- Recovery renders nothing and trusts only mutually consistent durable and live
  evidence.
- Successful unsync retains exact backups and leaves empty provider directories.

## Deferred work

- Multi-skill execution and implicit cascade operations.
- Provider-native runtime invocation or policy generation.
- Backup retention and garbage collection.
- `v0.2.2D` provider conformance and pilot validation.
- Git operations and release publication.
