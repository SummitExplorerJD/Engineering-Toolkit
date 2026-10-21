# Project composition plan `v1alpha1`

## 1. Purpose and commands

`engkit.composition-plan/v1alpha1` is the immutable JSON preview for either a
new project or an existing workspace. Planning never writes files.

```text
engkit project inspect [--root PATH]
engkit project plan-create --destination PATH --profile PROFILE_ID@VERSION [--set NAME=VALUE]
engkit project plan-compose [--root PATH] --profile PROFILE_ID@VERSION [--set NAME=VALUE]
```

The accepted `v0.2.1D1` execution design reserves:

```text
engkit project create --destination PATH --profile PROFILE_ID@VERSION [--set NAME=VALUE] --approve PLAN_ID
engkit project compose [--root PATH] --profile PROFILE_ID@VERSION [--set NAME=VALUE] --approve PLAN_ID
engkit project recover [--root PATH] --approve PLAN_ID
```

`project recover --root` identifies the intended project destination for a
`project.create` plan, even when that destination is absent. Execution and
recovery are implemented by `v0.2.1D2`.

The inspection and planning commands are implemented in `v0.2.1C`.
`project inspect` emits `engkit.project-inspection/v1alpha1` and includes the
workspace inspection plus profile recommendations. The packaged catalog is
still empty, so successful selection of first-party content begins only after
the independently approved pilot-content milestone.

## 2. Plan fields

| Field | Contract |
| --- | --- |
| `schema_version` | Exactly `engkit.composition-plan/v1alpha1`. |
| `kind` | `project.create` or `workspace.compose`. |
| `plan_id` | SHA-256 of the canonical plan body excluding `plan_id`. |
| `target_root` | Absolute normalized destination or workspace root. |
| `catalog` | Exact catalog ID, version, digest, and CLI version. |
| `profile` | Exact profile ID, version, and directory digest. |
| `templates` | Exact selected template identities and digests. |
| `resolved_variables` | Typed profile and template values sorted by scope and name. |
| `workspace_facts` | Observed target, ecosystem, source-layout, and ownership facts. |
| `rendered_outputs` | Complete paths, owners, kinds, sizes, and SHA-256 digests. |
| `conflicts` | Deterministically sorted blocking or informational conflicts. |
| `preconditions` | Exact state that execution must observe again. |
| `operations` | Ordered proposed filesystem operations. |
| `requires_approval` | Always `true`. |
| `executable` | `true` only when the executor is available and no blocking conflict exists. |
| `verification` | Post-execution validation required from the executor. |
| `recovery` | Backup, journal, checkpoint, and ambiguity rules. |

Plans contain no timestamps, random identifiers, machine-specific temporary
paths, ambient environment values, or credentials. Identical catalog bytes,
arguments, and observed filesystem state produce the same `plan_id`.

## 3. Planning policies

For `project.create`, the destination must be absent or explicitly empty. The
plan records which state was observed and refuses any later difference. For
`workspace.compose`, every existing path is inventoried by ownership and digest;
unowned content is preserved and collisions are blocking.

Planning renders all selected templates and profile assets in memory, validates
every canonical skill, and records every resulting digest. It rejects missing
or unknown variables, incompatible source layouts, duplicate outputs,
case-insensitive collisions, escaping paths, links, special files, unsupported
integration requirements, and ownership ambiguity.

Operations in the initial executor are limited to portable directory creation
and exact additive file or canonical-package publication visible in the plan.
Existing matching files are no-ops; replacement, adoption, and removal are not
supported. Removal of unrelated content is never inferred. Provider rendering, Spec Kit
initialization, dependency installation, script execution, network access, and
credential access are outside this contract.

## 4. Approval, help, and exit behavior

The executor rebuilds the plan from current state and accepts only the
complete matching `plan_id`. Changed catalog content, variables, target state,
or rendered bytes invalidates approval. Execution must use JSON plans, JSONL
journals, atomic JSON checkpoints, same-filesystem staging, verification, and
fail-closed recovery.

Every command family and subcommand accepts `help`, `-h`, and `--help`, writes
usage to standard output, and exits `0`. Usage mistakes exit `2`; validation,
conflict, or execution failures exit `1`. Normal command results are
deterministic JSON.

The planner emits blocking conflicts as reviewable plan data and marks those
plans non-executable. Valid additive plans are executable. The transaction
contract is specified in
[`composition-execution-v1alpha1.md`](composition-execution-v1alpha1.md).
