# Project composition execution `v1alpha1`

## 1. Status and boundary

This document defines the accepted `v0.2.1D1` design and implemented
`v0.2.1D2` execution contract.

The initial executor applies only additive outputs from a rebuilt matching
`engkit.composition-plan/v1alpha1`. It neither adopts nor replaces existing
content and never executes resources, installs dependencies, renders provider
files, initializes Spec Kit, accesses credentials, or contacts a network.

## 2. CLI contract

```text
engkit project create --destination PATH --profile ID@VERSION [--set NAME=VALUE] --approve PLAN_ID
engkit project compose [--root PATH] --profile ID@VERSION [--set NAME=VALUE] --approve PLAN_ID
engkit project recover [--root PATH] --approve PLAN_ID
```

`create` and `compose` rebuild the plan immediately before accepting approval.
The complete approved ID must match, every precondition must still hold, the
plan must have `executable: true`, and no blocking conflict may remain. Plans
created while no executor exists stay invalid because `executable` participates
in the digest.

For `project.create`, `project recover --root` identifies the intended project
destination even when it is currently absent. For `workspace.compose`, it
identifies the initialized workspace. Recovery loads persisted evidence; it
does not create a new plan or depend on the original catalog remaining
available.

## 3. Common artifact schemas

Plan keys use the hexadecimal portion of the `sha256:` plan ID.

Journal lines use `engkit.journal-event/v1alpha1`. Sequence starts at one and
increases exactly by one; event IDs derive from the full plan ID and sequence.
Every line is appended and synchronized before its checkpoint transition. A
truncated or malformed line never proves progress.

Checkpoints use `engkit.composition-checkpoint/v1alpha1` and record:

- the plan ID, plan kind, target root, status, and latest journal sequence;
- root-relative plan, journal, staging, backup, and lock locations;
- exact approved target-before and output digests;
- the next operation index, applied output paths and digests, and directories
  created by the transaction; and
- an update timestamp used only for diagnostics, never for plan identity or
  stale-lock authority.

Valid statuses are `prepared`, `staged`, `publishing`, `published`, `verified`,
`complete`, `rolled_back`, and `failed`. Checkpoints use sibling temporary
files, synchronization, and same-directory replacement. They cannot claim a
state later than the durable journal and observed filesystem support.

Successful execution or recovery emits
`engkit.composition-execution-result/v1alpha1` with the root, plan ID, kind,
final status, and paths to the plan, journal, checkpoint, retained staging or
backup, when applicable.

Required event families are `approval.accepted`,
`composition.execution_started`, `composition.staged`,
`composition.operation_started`, `composition.operation_completed`,
`composition.target_published`, `composition.verification_succeeded`,
`composition.execution_completed`, `composition.execution_failed`,
`composition.recovery_started`, `composition.recovery_completed`, and
`composition.recovery_refused`. Inapplicable events are omitted.

## 4. Common safety and locking rules

Before creating an artifact, execution must:

1. resolve the exact target and reject linked or non-directory ancestors;
2. rebuild the plan and compare the complete approved ID;
3. reject non-executable plans and every blocking conflict;
4. revalidate catalog, profile, template, variable, ownership, and path rules;
5. confirm every planned target precondition; and
6. atomically acquire the operation lock.

Lock records contain the plan ID, command, process ID, and start time for
diagnostics. Age and process existence do not authorize lock removal. Normal
execution refuses an existing lock; only matching explicit recovery can
release it after validating durable evidence.

All staging, backups, targets, and publication operations must remain on one
filesystem. Cross-device copy fallback is prohibited. Paths are rechecked for
links, case-insensitive collisions, reserved names, special files, and root
escape before publication.

## 5. New-project creation

For destination `PARENT/NAME`, the transaction uses deterministic sibling
locations derived from the portable destination name and plan key:

```text
PARENT/.NAME.engkit-create.lock/
PARENT/.NAME.engkit-create-PLAN_KEY/       # complete staged project
PARENT/.NAME.engkit-empty-backup-PLAN_KEY/ # only for an approved empty target
```

The staged project contains the rendered profile outputs and initialized
workspace state:

```text
.engkit/
  config.yaml
  .gitignore
  plans/composition/PLAN_KEY.json
  journals/composition/PLAN_KEY.jsonl
  checkpoints/composition/PLAN_KEY.json
  recovery/composition/PLAN_KEY/
```

The planner includes the deterministic workspace bootstrap
contract in project creation preconditions and verification. Dynamic journal
events and timestamps remain execution evidence rather than rendered profile
outputs.

The executor assembles and validates the complete tree in the sibling staging
directory. If the approved destination was absent, it publishes with one
directory rename. If the approved destination was empty, it first renames that
exact still-empty directory to the sibling backup and then renames staging to
the destination. It verifies all outputs and initialized workspace state before
recording completion and releasing the lock. The empty backup may be removed
only after successful verification and only while still exactly empty.

Internal evidence paths are stored relative to the project root, so moving the
staged tree does not invalidate them. An interrupted recovery discovers exactly
one authoritative evidence root: the sibling stage before publication or the
destination after publication. Finding both, neither when a lock exists, or an
unexpected destination causes refusal.

### New-project recovery

| Observed safe state | Result |
| --- | --- |
| Destination has the complete expected project and staging is absent | Verify and complete; remove the valid lock and exact empty backup. |
| Destination is absent and staging has the complete expected project | Mark rolled back, retain staging and evidence, restore the exact empty backup when present, and release the valid lock. |
| Destination is the approved empty directory and staging is complete | Mark rolled back, retain staging and evidence, and release the valid lock. |
| Destination, staging, backup, or evidence has any other state | Refuse and preserve every path and lock. |

Rolled-back staging is retained because it contains the only journal and
checkpoint. No recovery path recursively deletes it. Cleanup is a future
explicit maintenance operation.

## 6. Existing-workspace composition

Composition requires a valid initialized `.engkit` workspace and uses:

```text
.engkit/
  plans/composition/PLAN_KEY.json
  journals/composition/PLAN_KEY.jsonl
  checkpoints/composition/PLAN_KEY.json
  recovery/composition/PLAN_KEY/
    staged/
  recovery/composition/locks/workspace.lock/
```

One workspace-wide lock serializes composition plans because profiles can
overlap `AGENTS.md`, `.agents`, and source-layout paths. Unrelated skill
transactions retain their separate locks, but composition must reject any
planned canonical skill target currently controlled by an incomplete skill
transaction.

All output bytes are written and validated under staging first. Before any live
change, the executor capability-checks same-filesystem atomic no-replace file
publication using a disposable probe inside the recovery directory. It fails
without target mutation when that guarantee is unavailable.

Complete canonical skill directories whose targets are absent are published by
directory rename. Other files are published atomically without replacement
from validated staging. Directories are created in parent-first order and are
recorded individually. Already matching files produce no write. Any changed,
newly occupied, linked, or ambiguous target stops execution; replacing rename
and in-place write are forbidden.

The transaction is recoverable rather than globally atomic. Readers may see a
prefix of completed additive operations after interruption, but never partial
file bytes or an incomplete canonical skill package.

### Existing-workspace recovery

| Observed safe state | Result |
| --- | --- |
| All expected outputs match and the journal supports publication | Verify and complete. |
| Publication is incomplete and every applied output still has its exact approved digest | Remove only those exact transaction-created outputs, remove only recorded empty directories, and mark rolled back. |
| No output was published and staging is exact or absent | Mark rolled back. |
| Any applied output changed, an unknown path occupies a target, or evidence disagrees | Refuse; preserve outputs, staging, evidence, and lock. |

Canonical skill directories are removed during rollback only as complete
validated packages with the approved digest. Files are removed only when both
the journal/checkpoint claim their publication and current bytes match the
approved after digest. A directory is removed only when recorded as created by
the transaction and currently empty.

## 7. Verification and failure rules

Verification inventories every planned output, validates every canonical skill
package, compares exact digests and owners, confirms all unrelated observed
paths remain unchanged, validates `.engkit` state, and records the final changed
path list. A verification failure can never be reported as success and remains
recoverable through persisted evidence.

Journal payloads contain no rendered file content, credentials, or environment
dumps. Recovery performs no catalog lookup, network request, rendering, or
script execution. Unknown content is never deleted, overwritten, or adopted.

## 8. `v0.2.1D2` implementation verification

The implementation adds no dependency. Its automated coverage verifies:

- exact approval and mismatched approval with zero pre-approval writes;
- absent and explicitly empty new-project destinations;
- complete-tree publication and initialized workspace state;
- initialized-workspace enforcement and partial canonical-skill conflicts;
- additive workspace publication when atomic no-replace links are available,
  with fail-closed capability refusal when they are not;
- interruption and explicit recovery both before and after project publication;
- changed-output refusal during workspace recovery; and
- deterministic planning, materialization, and CLI routing.

The repository test and vet suites pass, including the loopback TUF transport
tests. The release builder compiles the CLI for Windows, Linux, and macOS on
AMD64 and ARM64. Broader failure-injection and race coverage remains required
before declaring the composition contract stable.

The implementation gate excludes first-party catalog content, provider
synchronization, Spec Kit mutation, remote catalogs, scripts, dependency
installation, backup cleanup, Git operations, and release publication.
