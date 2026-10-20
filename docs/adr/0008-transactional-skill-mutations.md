# ADR 0008: Transactional local skill mutations

## Status

Accepted and implemented for `v0.2.0D`.

## Context

`v0.2.0C` can describe an exact canonical skill creation, update, or removal,
but its plans are deliberately non-executable. Applying those plans introduces
destructive filesystem transitions, concurrent-command hazards, and crash
windows that do not exist during inspection or planning. A file-by-file update
could expose a partially updated package, and storing the only backup in the
ordinary cache would make recovery dependent on regenerable state.

The executor must work on Windows, Linux, and macOS. It cannot assume POSIX
advisory locks, open-file replacement semantics, or that a process ID proves a
lock owner is still alive.

## Decision

An execution command rebuilds the complete deterministic plan from current
state and accepts only the exact full plan ID through `--approve`. Approved
plans are persisted before target mutation. A change to source, target,
workspace identity, or executor capability produces a different plan and
requires new approval.

Skill publication uses complete directory transitions on the same filesystem.
Create renames a validated staged package into an absent target. Update first
renames the validated current target to a unique recovery backup and then
renames the validated staged package into place. Remove renames the target to
the recovery backup instead of recursively deleting it. The executor never
merges files into a live package.

Each skill identity has an atomically created lock directory. Durable JSON
plans, append-only JSONL journals, and atomically replaced JSON checkpoints
record execution. Staging and backups live under a distinct ignored
`.engkit/recovery/` tree rather than `.engkit/cache/`. Backups for completed
updates and removals are retained until explicit manual maintenance; automatic
cleanup is outside this milestone.

Recovery is explicit and approval-bound. It inspects durable artifacts and
actual package digests, then either verifies the approved final state, restores
an exact approved backup when the target is absent, recognizes an already
restored before state, or refuses ambiguous content without deleting it. Age
or process-ID heuristics never authorize removal of a stale lock.

The exact CLI, artifact fields, state transitions, recovery table, and
implementation verification gate are defined in
[`skill-execution-v1alpha1.md`](../skill-execution-v1alpha1.md).

## Consequences

- Users never approve a mutable plan reference or partial plan identifier.
- Update has a short target-absent interval between two directory renames, but
  that interval is detectable and recoverable.
- Complete package swapping avoids visible file-by-file partial state and
  naturally handles file/directory type transitions.
- Recovery does not depend on the original source package or network access.
- Retained backups consume local disk space until deliberately removed.
- The first execution in an older initialized workspace may append the exact
  `/recovery/*` ignore rule without replacing other `.gitignore` content.
- Plans produced before an executor exists remain unusable because changing
  `executable` changes their plan ID.

## Rejected alternatives

- **Apply file operations directly to the live target:** rejected because an
  interruption can expose a mixed package and complicates deterministic
  rollback.
- **Store backups in `.engkit/cache`:** rejected because cache is rebuildable
  and may be cleared while it contains the only rollback copy.
- **Delete removed packages recursively:** rejected because rename-to-backup is
  safer, faster, and recoverable on the same filesystem.
- **Automatically break old locks:** rejected because age and PID checks are
  not reliable ownership proof across supported platforms.
- **Accept a saved plan without rebuilding it:** rejected because source or
  target state may have changed after review.
- **Automatically recover during the next mutation:** rejected because
  recovery is itself material and must remain explicit and attributable.
