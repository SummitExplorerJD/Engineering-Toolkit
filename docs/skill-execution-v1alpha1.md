# Skill mutation execution `v1alpha1`

## 1. Status and boundary

This document defines the implemented `v0.2.0D` local mutation contract.

The executor applies only local plans produced from canonical
`engkit.skill/v1alpha1` packages. It does not import provider-native skills,
render provider files, run skill resources, resolve dependencies, access
credentials, contact networks, or initialize a workspace.

Execution requires an already initialized workspace with a valid `.engkit`
state directory. Existing Spec Kit and provider-native directories remain
external and cannot be created, updated, removed, adopted, or recovered by this
executor.

## 2. Proposed CLI contract

```text
engkit skills create [--root PATH] --source PACKAGE_PATH --approve PLAN_ID
engkit skills update [--root PATH] --id SKILL_ID --source PACKAGE_PATH --approve PLAN_ID
engkit skills remove [--root PATH] --id SKILL_ID --approve PLAN_ID
engkit skills recover [--root PATH] --id SKILL_ID --approve PLAN_ID
```

`create`, `update`, and `remove` rebuild the applicable mutation plan from the
current filesystem immediately before accepting approval. `--approve` must
contain the complete, exact `plan_id` from that rebuilt plan. The command fails
before writing any execution artifact when the approved ID no longer matches.

Newly generated skill mutation plans use `executable: true`. Because this field
participates in the plan digest, plans produced by `v0.2.0C` are invalid and
must be regenerated. The experimental plan schema remains
`engkit.skill-mutation-plan/v1alpha1`; the field already represents executor
availability rather than a new operation contract.

`recover` never creates a new mutation plan. It loads the persisted plan and
checkpoint identified jointly by `--id` and `--approve`, validates their
relationship, inspects the target and recovery directories, and applies the
deterministic recovery table in section 8. Approval of another plan, a partial
ID, or a plan for another skill is rejected.

## 3. Persisted artifacts

The filesystem remains authoritative. SQLite is not involved. A plan key is
the hexadecimal portion of its `sha256:` plan ID; using the key avoids the
colon that is invalid in Windows filenames.

```text
.engkit/
  plans/skills/PLAN_KEY.json
  journals/skills/PLAN_KEY.jsonl
  checkpoints/skills/PLAN_KEY.json
  recovery/skills/PLAN_KEY/
    staged/
    backup/
  recovery/skills/locks/SKILL_ID.lock/
```

The plan, journal, and checkpoint are durable audit artifacts. The recovery
directory is ignored by Git but is not ordinary cache: it may contain the only
rollback copy during or after an update or removal. An incomplete recovery
directory must not be deleted or reused automatically.

The initializer adds `/recovery/*` to the generated `.engkit/.gitignore`
contract. Existing initialized workspaces may receive
that one exact ignore entry as part of the first approved skill execution, only
after their existing file has been parsed and verified as a regular file. The
implementation must not replace unrelated user entries.

Completed update and remove transactions retain their backup in the unique
plan recovery directory. `v0.2.0D` does not add automatic retention, garbage
collection, or a purge command. Deleting a completed backup is an explicit
manual maintenance decision and makes post-completion rollback unavailable.

## 4. Artifact schemas

### Journal

Each journal line uses `engkit.journal-event/v1alpha1` and contains:

- `schema_version`, `event_id`, `plan_id`, `sequence`, `timestamp`, `type`,
  and a non-secret `payload`;
- a sequence beginning at 1 and increasing by exactly one;
- an event ID derived from the complete plan ID and sequence;
- a final newline for every complete event.

Events are appended and flushed before advancing the checkpoint that refers to
them. A malformed or truncated final line is reported and is never interpreted
as completion. Required event types are `approval.accepted`,
`skill.execution_started`, `skill.staged`, `skill.backup_published`,
`skill.target_published`, `skill.verification_succeeded`,
`skill.execution_completed`, `skill.execution_failed`,
`skill.recovery_started`, `skill.recovery_completed`, and
`skill.recovery_refused`. Events that do not apply to an operation are omitted.

### Checkpoint

Each checkpoint uses `engkit.skill-checkpoint/v1alpha1` and contains:

- `plan_id`, `kind`, `skill_id`, and `status`;
- normalized target, staging, backup, plan, and journal paths;
- expected before and after package digests when applicable;
- the latest complete journal sequence and an update timestamp.

Valid statuses are `prepared`, `staged`, `backup_published`,
`target_published`, `verified`, `complete`, `rolled_back`, and `failed`.
Checkpoint updates use a sibling temporary file, file synchronization, and
same-directory replacement. A checkpoint never claims a later state than the
latest durable journal event and observed filesystem state support.

### Result

Successful mutation and recovery commands emit
`engkit.skill-execution-result/v1alpha1` with the root, plan ID, operation,
skill ID, final status, and relative paths to the persisted plan, journal,
checkpoint, and retained backup when present.

## 5. Locking and concurrency

The executor creates the skill lock directory atomically. A lock is scoped to
one canonical skill identity, allowing unrelated skills to proceed without
sharing mutable transaction state. The lock records the plan ID, process ID,
start time, and command for diagnostics, but those facts do not prove that a
process is alive.

An existing lock causes normal execution to fail closed. The executor does not
delete a lock based only on age or process ID because PID reuse and suspended
processes are platform-dependent. `recover` owns stale-lock evaluation and may
remove the lock only after it has validated the matching durable artifacts and
completed or safely rolled back the transaction.

## 6. Preconditions and staging

Before creating any execution artifact, the executor must:

1. resolve and validate the workspace and `.engkit` paths without following an
   unsafe symbolic link;
2. rebuild the plan from current state and compare the complete plan ID;
3. validate the exact canonical source and target rules from the plan;
4. confirm every planned path remains inside the intended package root; and
5. acquire the per-skill lock.

After locking, it persists the approved plan with exclusive creation and
records the initial journal and checkpoint. For create and update, it copies
the complete source package to `staged/` without following symbolic links,
using exclusive file creation and restrictive permissions. It then validates
and snapshots the staged package again. The staged digest must equal the
approved `source_sha256` before the target can change.

All staging, backup, and target paths must be on the workspace filesystem.
Cross-device rename fallback is prohibited because it would weaken the
publication boundary.

## 7. Publication algorithm

### Create

The target must still be absent. The executor renames the complete staged
package to the target, verifies the resulting package and digest, records
completion, and releases the lock. It never merges into an existing directory.

### Update

The target must still match the approved before digest. The executor renames
the complete target to `backup/`, advances the journal and checkpoint, then
renames `staged/` to the target. If the second rename fails while the target is
absent, it attempts to restore `backup/` to the target and records the observed
outcome. It never performs an in-place file-by-file merge.

### Remove

The target must still match the approved before digest. The executor renames
the complete target to `backup/`, verifies that the target is absent and that
the backup digest matches, records completion, and retains the backup. It does
not recursively delete the approved package as its publication step.

Each operation verifies the final filesystem state against the complete
approved package digest and canonical validator before recording `complete`.
The ordered file operations in the plan remain the human-reviewable change
contract; the executor publishes their exact aggregate result as one package
directory transition.

## 8. Interruption recovery

Recovery trusts neither the checkpoint alone nor the presence of a directory.
It compares the persisted plan, complete journal, checkpoint, target digest,
staged digest, and backup digest. Unknown or conflicting content is preserved
and recovery fails closed with actionable paths.

| Operation | Observed safe state | Recovery result |
| --- | --- | --- |
| Create | Target has the expected after digest | Verify and complete. |
| Create | Target is absent; staging is expected or absent | Remove only known staging state, mark rolled back, release lock. |
| Create | Target exists with any other state | Refuse; preserve all paths. |
| Update | Target has the expected after digest and backup has the before digest | Verify and complete; retain backup. |
| Update | Target is absent and backup has the before digest | Restore backup, verify before digest, mark rolled back. |
| Update | Target has the before digest and no conflicting backup | Mark rolled back after verification. |
| Update | Any digest or path is ambiguous | Refuse; preserve all paths. |
| Remove | Target is absent and backup has the before digest | Verify and complete; retain backup. |
| Remove | Target has the before digest and no conflicting backup | Mark rolled back after verification. |
| Remove | Target or backup has any other state | Refuse; preserve all paths. |

Known staging content may be removed only after its inventory and digest match
the approved after state. Unknown content is never recursively deleted by
recovery. A refused recovery leaves the lock and all evidence intact for manual
inspection.

## 9. Failure and security rules

- Approval validation and plan revalidation precede writes.
- Paths, symlinks, case-insensitive collisions, and reserved Windows names fail
  closed before publication.
- Existing external, provider-native, invalid, or conflicting skill state is
  never converted into a managed package implicitly.
- Journal payloads and diagnostics contain no package contents, credentials, or
  environment dumps.
- Declared skill permissions do not authorize script or process execution.
- A failed verification cannot be reported as success even when publication
  already occurred; it remains recoverable through the checkpoint.
- Recovery never downloads replacement content or depends on the original
  source path remaining available.

## 10. Implementation verification

The implementation includes tests for:

- exact and mismatched approval with zero pre-approval writes;
- source or target changes between planning and execution;
- create, update, remove, and retained backup contents;
- file-to-directory and directory-to-file changes through package swapping;
- lock contention and stale-lock refusal;
- failure injection before and after every publication rename;
- every recovery-table branch, including ambiguous-state refusal;
- journal ordering, truncation detection, and checkpoint replacement;
- containment, symlink, case-collision, and Windows filename handling;
- preservation of all existing Spec Kit/provider-native directories; and
- Windows, Linux, and macOS behavior on AMD64 and ARM64 in CI.

The implementation adds no dependency, network access, provider
synchronization, remote installation, script execution, automatic backup
cleanup, Git operation, or release publication.
