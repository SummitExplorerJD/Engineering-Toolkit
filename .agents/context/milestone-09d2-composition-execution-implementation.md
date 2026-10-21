# Milestone 09D2: Composition execution implementation

## Observed facts

- Composition journals modify managed `.engkit` state, so workspace planning
  must exclude that internal state from the unowned workspace tree digest while
  still validating initialization and preserving every other observed path.
- Publishing a canonical skill through a complete directory rename requires its
  target directory to be absent. A partial existing skill therefore becomes a
  blocking conflict rather than a file-by-file merge.
- Same-filesystem hard links provide an atomic no-replace publication primitive
  for regular files. Some sandboxed Windows filesystems deny hard-link creation;
  the executor detects that before live mutation and requires recovery of the
  retained failed transaction.

## Implemented boundary

- Valid composition plans are executable; blocking plans remain non-executable.
- `project create` publishes a complete initialized project from sibling staging
  and preserves an approved empty destination until verification succeeds.
- `project compose` requires initialization, uses a workspace lock, publishes
  complete canonical skills by rename, and publishes other files without
  replacement after a capability probe.
- Plans, journals, checkpoints, locks, staging, output digests, canonical skill
  validity, and checkpoint paths are validated during recovery.
- Unknown or changed output causes recovery refusal without deletion.

## Deferred decisions

- First-party templates and the pilot profile remain `v0.2.1E`; the packaged
  catalog is still intentionally empty.
- Replacement, removal, ownership upgrades, and transaction cleanup remain
  future contracts.
- Provider synchronization and Spec Kit mutation remain outside `v0.2.1`.
