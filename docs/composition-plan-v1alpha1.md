# Project composition plan `v1alpha1`

## 1. Purpose and commands

`engkit.composition-plan/v1alpha1` is the immutable JSON preview for either a
new project or an existing workspace. Planning never writes files.

```text
engkit project inspect [--root PATH]
engkit project plan-create --destination PATH --profile PROFILE_ID@VERSION [--set NAME=VALUE]
engkit project plan-compose [--root PATH] --profile PROFILE_ID@VERSION [--set NAME=VALUE]
```

Future execution commands are reserved as:

```text
engkit project create --destination PATH --profile PROFILE_ID@VERSION [--set NAME=VALUE] --approve PLAN_ID
engkit project compose [--root PATH] --profile PROFILE_ID@VERSION [--set NAME=VALUE] --approve PLAN_ID
engkit project recover [--root PATH] --approve PLAN_ID
```

Execution and recovery are not authorized by this design milestone.

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
| `executable` | `false` until the separately approved execution milestone. |
| `verification` | Post-execution validation required from a future executor. |
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

Operations are limited to portable directory creation and exact file creation
or replacement that is owned by the selected profile and visible in the plan.
Removal of unrelated content is never inferred. Provider rendering, Spec Kit
initialization, dependency installation, script execution, network access, and
credential access are outside this contract.

## 4. Approval, help, and exit behavior

A future executor must rebuild the plan from current state and accept only the
complete matching `plan_id`. Changed catalog content, variables, target state,
or rendered bytes invalidates approval. Execution must use JSON plans, JSONL
journals, atomic JSON checkpoints, same-filesystem staging, verification, and
fail-closed recovery.

Every command family and subcommand accepts `help`, `-h`, and `--help`, writes
usage to standard output, and exits `0`. Usage mistakes exit `2`; validation,
conflict, or execution failures exit `1`. Normal command results are
deterministic JSON.
