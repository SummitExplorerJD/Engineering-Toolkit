# Provider synchronization execution `v1alpha1`

## 1. Status and boundary

This document defines the accepted `v0.2.2C1` transaction design. No executor
or mutation CLI is implemented by this design slice. Implementation,
conformance, Git operations, and release publication require separate approval.

The future executor will apply only exact, executable provider plans for one
canonical skill and one explicitly selected provider. It will never execute a
skill resource, invoke a provider CLI, contact a network, access credentials,
resolve dependencies, initialize Spec Kit, or generate provider runtime policy.

## 2. CLI contract

The proposed implementation surface is:

```text
engkit skills plan-sync [--root PATH] --provider PROVIDER --id SKILL_ID
engkit skills sync [--root PATH] --provider PROVIDER --id SKILL_ID --approve PLAN_ID
engkit skills plan-unsync [--root PATH] --provider PROVIDER --id SKILL_ID
engkit skills unsync [--root PATH] --provider PROVIDER --id SKILL_ID --approve PLAN_ID
engkit skills recover-sync [--root PATH] --provider PROVIDER --id SKILL_ID --approve PLAN_ID
```

Execution requires `--id`; multi-skill plans remain useful for review but are
not executable in the initial transaction scope. `sync` and `unsync` rebuild
the applicable plan immediately before accepting approval. The complete
approved ID must match, the plan must be executable, and no blocking finding
may remain.

Sync plans require `provider-synchronization` and `provider:PROVIDER` approval
scopes. Unsync plans require `provider-unsynchronization` and
`provider:PROVIDER`. Approval of one operation or provider never authorizes the
other.

`plan-unsync` does not require the canonical package to remain present. It
derives the removal set exclusively from a valid ownership record and exact
observed output digests. Missing, malformed, overlapping, or changed ownership
state produces a non-executable plan.

`recover-sync` never rebuilds a new plan. It validates the persisted plan,
journal, checkpoint, lock, staging, backups, live outputs, and ownership record
identified by the provider, skill ID, and complete approved plan ID.

## 3. Cross-operation interlock

Provider synchronization and canonical skill mutation share the canonical
skill lock. A provider transaction acquires locks in this fixed order:

1. `.engkit/recovery/skills/locks/SKILL_ID.lock/`;
2. `.engkit/recovery/provider-sync/locks/PROVIDER/SKILL_ID.lock/`.

This order prevents source changes while rendering or publishing and avoids
deadlock. Normal execution never removes an existing lock based on age or
process ID.

Canonical skill update and removal must fail closed while any supported
provider ownership path for that skill is present or ambiguous. A malformed,
linked, or unreadable record blocks mutation rather than being ignored. The
user must explicitly unsynchronize each provider, mutate the canonical package
through its separately approved plan, and then synchronize the desired
providers again. This initial rule prevents stale derived packages and prevents
a canonical directory replacement from silently deleting the Codex `SKILL.md`
sidecar. Automatic cascade updates or removals are outside `v1alpha1`.

## 4. Persisted artifacts

The plan key is the hexadecimal portion of the approved `sha256:` plan ID.
Provider and skill IDs are normalized path components.

```text
.engkit/
  plans/provider-sync/PROVIDER/SKILL_ID/PLAN_KEY.json
  journals/provider-sync/PROVIDER/SKILL_ID/PLAN_KEY.jsonl
  checkpoints/provider-sync/PROVIDER/SKILL_ID/PLAN_KEY.json
  recovery/provider-sync/PROVIDER/SKILL_ID/PLAN_KEY/
    staged/
    backup/outputs/
    backup/ownership.json
  recovery/provider-sync/locks/PROVIDER/SKILL_ID.lock/
  providers/PROVIDER/SKILL_ID.json
```

Plans, journals, checkpoints, and retained backups are durable transaction
evidence. Recovery data is ignored by Git but is not disposable cache. The
initial contract defines no automatic retention, purge, or garbage collection.

## 5. Journal, checkpoint, ownership, and result

Journal lines use `engkit.journal-event/v1alpha1`. Sequence begins at one and
increases by exactly one. Event IDs derive from the full plan ID and sequence.
Every complete line ends with a newline and is synchronized before the
checkpoint may advance past it. A malformed or truncated line proves nothing.

Required event families are `approval.accepted`,
`provider_sync.execution_started`, `provider_sync.staged`,
`provider_sync.output_started`, `provider_sync.output_published`,
`provider_sync.outputs_verified`, `provider_sync.ownership_published`,
`provider_sync.verification_succeeded`, `provider_sync.execution_completed`,
`provider_sync.execution_failed`, `provider_sync.recovery_started`,
`provider_sync.recovery_completed`, and `provider_sync.recovery_refused`.

Checkpoints use `engkit.provider-sync-checkpoint/v1alpha1` and contain:

- the plan ID, operation, provider, skill ID, status, and latest journal sequence;
- relative plan, journal, staging, backup, ownership, and lock paths;
- the next output index and completed output paths;
- exact before and after digests for every touched output and ownership record;
- directories created by the current transaction; and
- an update timestamp used only for diagnostics.

Valid statuses are `prepared`, `staged`, `publishing`, `outputs_published`,
`ownership_published`, `verified`, `complete`, `rolled_back`, and `failed`.
Checkpoint replacement uses a sibling temporary file, synchronization, and
same-directory replacement. It cannot claim progress unsupported by the
durable journal and observed filesystem.

The ownership record remains `engkit.provider-ownership/v1alpha1`. A successful
sync record lists exactly the live generated files and their digests and binds
them to the source package, adapter contract, and publishing plan. Unsync moves
the live record to retained backup after all owned outputs have been moved to
backup. No live record remains after successful unsynchronization.

Successful execution or recovery emits
`engkit.provider-sync-execution-result/v1alpha1` with the root, plan ID,
operation, provider, skill ID, final status, changed paths, and relative paths
to the persisted evidence and retained backups.

## 6. Preconditions and staging

Before creating transaction evidence, execution must:

1. resolve the workspace and initialized `.engkit` state without following an
   unsafe link or junction;
2. rebuild and compare the exact approved plan;
3. require one source skill for sync or one valid ownership record for unsync;
4. reject every blocking finding and non-executable mapping;
5. revalidate all target and ownership observations;
6. reject Spec Kit, external, canonical, and unowned paths outside the exact
   generated-file set; and
7. acquire the canonical skill lock followed by the provider-skill lock.

After locking, the executor repeats target observations before persisting the
plan, journal, and checkpoint. For sync, it writes every rendered byte to
same-filesystem staging with exclusive creation and verifies the approved
digest. It also stages the deterministic next ownership record. For unsync,
there are no newly rendered provider bytes.

All live targets, staging paths, backups, and ownership records must be on the
workspace filesystem. Cross-device publication or copy fallback is prohibited.

## 7. Publication algorithm

Outputs are processed in stable plan order. `unchanged` performs no write but
is verified. `create` uses a capability-checked atomic no-replace operation.
`replace` first moves the exact owned file to its unique backup and then moves
the staged file into place with the same no-replace guarantee. `remove` moves
the exact owned file to backup; it does not delete it. Before each transition,
the current path must still match the approved observation and recorded digest.

After all outputs match their planned final state, the executor verifies them
as a set. It then publishes the ownership record:

- creation uses atomic no-replace publication;
- replacement first retains the exact prior record as `backup/ownership.json`
  and publishes the staged record without replacement;
- unsync moves the exact live record to `backup/ownership.json` only after all
  owned outputs are safely backed up.

Publishing ownership last prevents a record from claiming output that was not
published. An interruption before that step leaves an intentionally
recoverable mismatch between live files and the prior record. Normal planning
fails closed on that mismatch; only explicit recovery may resolve it.

The initial executor never removes provider package directories. It records
directories created during the transaction so rollback may remove only those
that are still empty, but successful sync and unsync leave empty directories
in place. This avoids claiming directory ownership.

## 8. Recovery

Recovery trusts no single artifact. It compares the persisted plan, complete
journal, checkpoint, staged digests, retained backups, live output digests,
ownership record, and both locks. Unknown or conflicting content is preserved
and recovery is refused.

| Operation | Observed safe state | Recovery result |
| --- | --- | --- |
| Sync | All outputs and the new ownership record match | Verify and complete. |
| Sync | Outputs match after-state; ownership is prior or absent; staged record is exact | Publish ownership, verify, and complete. |
| Sync | A verified prefix was published; every remaining live path matches before-state | Restore exact backups, remove only exact transaction-created files, restore prior ownership, and roll back. |
| Sync | No output was published and staging is exact or absent | Roll back and retain evidence. |
| Unsync | Owned outputs and live ownership are absent; backups match before-state | Verify and complete. |
| Unsync | A verified prefix was backed up; remaining paths match before-state | Restore outputs and ownership, verify, and roll back. |
| Either | Any live, staged, backup, record, journal, checkpoint, or lock state is ambiguous | Refuse and preserve all evidence. |

Rollback removes a created file only when the journal and checkpoint identify
it and its current digest equals the approved after digest. It restores a
replaced or removed file only from an exact retained backup and only while the
live destination is absent. A refused recovery retains both locks.

## 9. Verification and security rules

- Verification checks every selected output, the ownership record, canonical
  source digest for sync, and preservation observations for unrelated paths.
- Codex verification separately proves that `skill.yaml`, `instructions.md`,
  declared resources, and external sibling packages are unchanged.
- Claude Code and OpenCode verification proves that unowned files under the
  derived package directory are unchanged.
- Journal payloads never contain rendered content, credentials, or environment
  dumps.
- Runtime permission files, provider configuration, `AGENTS.md`, `CLAUDE.md`,
  `.specify`, and official `speckit-*` content are never generated or adopted.
- A failed verification is never reported as success, even after publication;
  it remains an explicit recovery case.
- Recovery performs no rendering, network request, provider invocation,
  canonical mutation, or ownership discovery beyond persisted evidence.

## 10. Implementation verification gate

The separately approved implementation must cover:

- exact and mismatched approvals with zero pre-approval writes;
- create, unchanged, replace, stale-output removal, and full unsync;
- ownership creation, replacement, backup, malformed-record refusal, and
  changed-owned-output refusal;
- shared canonical lock and provider lock ordering, contention, and stale-lock
  refusal;
- canonical update and removal refusal while any provider ownership is live;
- failure injection before and after every output and ownership transition;
- every recovery-table branch and ambiguous-state preservation;
- Codex sidecar preservation of canonical and Spec Kit content;
- isolated-provider preservation of unowned content;
- Windows portable paths, link and junction refusal, case collisions, and
  no-replace capability checks;
- no network, credential, script, dependency, provider CLI, or Spec Kit action;
  and
- Windows, Linux, and macOS behavior on AMD64 and ARM64 in CI.

