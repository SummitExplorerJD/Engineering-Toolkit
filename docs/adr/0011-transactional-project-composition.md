# ADR 0011: Transactional project composition

## Status

Accepted in `v0.2.1D1` and implemented for `v0.2.1D2`.

## Context

`v0.2.1C` produces deterministic plans for creating a new project or composing
an existing workspace, but deliberately marks those plans non-executable. The
two operations have different crash and ownership boundaries. A new project can
be assembled as one complete tree and published into its parent filesystem. An
existing workspace contains unrelated user files and cannot be replaced as one
directory.

Composition must work on Windows, Linux, and macOS without silently replacing
unowned content. It must preserve the exact approval binding, durable evidence,
explicit recovery, and fail-closed behavior already established for canonical
skill mutations.

## Decision

The executor will rebuild the complete plan from current catalog bytes,
arguments, variables, and target state. It accepts only the full matching
`plan_id`, rejects blocking conflicts, and writes nothing before approval and
lock acquisition succeed. Plans become executable only when the separately
approved executor is present; that change participates in the plan digest, so
older previews cannot be applied.

The first executor is additive only. It creates paths that were approved as
missing and treats already matching regular files as no-ops. It does not adopt,
replace, or remove existing content. Provider synchronization, Spec Kit
mutation, dependency installation, scripts, network access, and credentials
remain outside this boundary.

New-project creation uses a target-scoped sibling lock and a hidden sibling
staging root on the destination filesystem. The complete project, including an
initialized `.engkit` state tree and its transaction evidence, is validated in
staging. An absent destination receives the project through one directory
rename. An approved empty destination is first renamed to an exact sibling
backup and then receives the staged tree. Evidence paths inside the staged tree
are root-relative, so they remain valid after publication. Successful evidence
therefore lives inside the new project; rolled-back staging is retained beside
the destination for inspection instead of being recursively deleted.

Existing-workspace composition requires an initialized workspace. It uses one
workspace composition lock and stages every rendered output below
`.engkit/recovery/composition/PLAN_KEY/`. Complete canonical skill directories
are published by same-filesystem directory rename when their targets are
absent. Other additive files use an atomic no-replace publication primitive.
The implementation must capability-check that primitive before changing the
workspace and fail closed when the filesystem cannot provide it. It may not
fall back to a replacing rename or an in-place live write.

Both operation kinds persist the approved plan, append-only JSONL journal, and
atomically replaced JSON checkpoint. Recovery is explicit and approval-bound.
It validates the plan, journal, checkpoint, lock, staging state, target state,
and exact digests. It completes a fully published state, rolls back only exact
known partial output, or refuses ambiguous and changed content while preserving
all evidence.

The exact commands, artifact locations, state transitions, recovery tables,
and implementation verification gate are defined in
[`composition-execution-v1alpha1.md`](../composition-execution-v1alpha1.md).

## Consequences

- A new project is never exposed as a partially assembled tree.
- Existing-workspace publication is recoverable but not globally atomic across
  multiple unrelated output paths.
- The additive-only boundary avoids guessing ownership and removes no user
  content.
- Empty-destination creation has a short target-absent interval between two
  directory renames, covered by the retained exact backup and recovery state.
- Rolled-back new-project staging and completed workspace recovery artifacts
  consume disk until explicit maintenance is designed and approved.
- Existing workspaces must run `engkit workspace init` before composition.
- The first-party catalog remains an independent gate; executor tests may use
  fixtures without shipping catalog content.

## Rejected alternatives

- **Merge a new project file by file:** rejected because users could observe a
  partially assembled project after interruption.
- **Replace an existing workspace directory:** rejected because `.git`, source,
  and unrelated user content are outside profile ownership.
- **Overwrite matching paths after a preflight check:** rejected because the
  check and replacement race could destroy external content.
- **Write target files in place with exclusive creation:** rejected because a
  reader could observe partial bytes before synchronization completes.
- **Treat an empty destination as equivalent to an absent path without a
  backup:** rejected because removing the directory creates an unrecorded crash
  window.
- **Automatically resume any partial transaction:** rejected because recovery
  is material and must remain explicit and attributable.
- **Store new-project evidence only outside the project:** rejected because a
  successful project should carry its own durable workspace record.
