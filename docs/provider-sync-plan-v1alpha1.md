# Provider synchronization plan `v1alpha1`

## Purpose and boundary

`engkit.provider-sync-plan/v1alpha1` is the immutable preview for rendering one
or more canonical skills to exactly one selected provider. The accepted
`v0.2.2C2` keeps planning read-only and lets the separately invoked executor
apply only one exact approved skill-and-provider plan.

The implemented read-only entry point is:

```text
engkit skills plan-sync [--root PATH] --provider PROVIDER [--id SKILL_ID]
```

The transactional implementation exposes these mutation, removal-planning, and
recovery entry points:

```text
engkit skills sync [--root PATH] --provider PROVIDER --id SKILL_ID --approve PLAN_ID
engkit skills plan-unsync [--root PATH] --provider PROVIDER --id SKILL_ID
engkit skills unsync [--root PATH] --provider PROVIDER --id SKILL_ID --approve PLAN_ID
engkit skills recover-sync [--root PATH] --provider PROVIDER --id SKILL_ID --approve PLAN_ID
```

This CLI spelling and the JSON schema are the experimental `v1alpha1`
contract. Incompatible changes require an explicit schema revision.

## Canonical source and provider targets

Only a valid `engkit.skill/v1alpha1` package is renderable. External,
conflicting, and invalid packages remain inspectable but cannot produce an
executable plan. A non-empty `compatibility.providers` list must include the
selected provider. An absent or empty list does not claim parity and requires
the adapter to evaluate every mapping.

For `operation: synchronize`, `sources` and `mappings` describe canonical
snapshots. For `operation: remove`, both arrays are empty: authority comes only
from the selected provider and skill identity, the exact observed ownership
record digest, and its exact owned output digests. Removal never depends on a
canonical package remaining present.

Initial output roots are:

| Provider | Root | Required primary file |
| --- | --- | --- |
| `codex` | `.agents/skills/SKILL_ID/` | `SKILL.md` |
| `claude` | `.claude/skills/SKILL_ID/` | `SKILL.md` |
| `opencode` | `.opencode/skills/SKILL_ID/` | `SKILL.md` |

The Codex root overlaps canonical storage. The adapter may own generated files
inside a canonical package but never the directory, `skill.yaml`,
`instructions.md`, or another canonical resource. Claude Code and OpenCode
packages remain derived copies and likewise use file-level ownership.

## Deterministic mapping

The provider `SKILL.md` begins with provider-supported YAML frontmatter
containing the canonical `id` as `name` and canonical `description`, followed by
the exact normalized contents of `instructions.md`. Adapter-specific optional
metadata is emitted only when its semantics are exact and documented.

Declared passive resources (`reference`, `template`, and `asset`) may be copied
byte-for-byte at their canonical relative paths. The planner must account for
every declared resource. For Codex, those resources already reside beside the
generated `SKILL.md`; they are referenced in place and never become generated
outputs or adapter-owned files. Claude Code and OpenCode receive derived copies
when the mapping is exact. Initial plans are non-executable when a package
contains:

- a `script` resource;
- a non-empty permission request;
- an unresolved canonical dependency;
- a provider-incompatible path, media type, or required semantic; or
- a field whose omission would change the advertised workflow.

The planner never executes, rewrites, or interprets resource contents.

## Plan shape

The plan contains these top-level fields:

| Field | Meaning |
| --- | --- |
| `schema_version` | Exactly `engkit.provider-sync-plan/v1alpha1` |
| `plan_id` | `sha256:` digest of canonical JSON with `plan_id` omitted |
| `operation` | `synchronize` or `remove` |
| `root` | Normalized absolute workspace root used for observation |
| `provider` | Exactly `codex`, `claude`, or `opencode` |
| `skill_id` | Exact canonical skill identity for executable single-skill plans |
| `adapter` | Contract identifier, version, evidence date, and target root |
| `sources` | Selected canonical identities, versions, directory digests, and manifest digests |
| `observations` | Target path kinds and digests plus prior ownership-record digests |
| `mappings` | Per-source mapping quality and stable findings |
| `outputs` | Exact destination, action, content digest, ownership, and source relation |
| `conflicts` | Stable blocking or advisory findings |
| `affected_paths` | Sorted unique paths the operation may write or remove |
| `risks` | Human-reviewable provider and workspace risks |
| `required_approvals` | Approval scopes required for future execution |
| `verification` | Ordered postconditions for output and preservation checks |
| `executable` | True only when every mapping and destination is safe and exact |

Arrays are sorted by normalized provider path and stable identifiers. Time,
environment variables, provider CLI availability, network state, and directory
enumeration order do not affect the plan ID.

## Mapping quality and findings

Each mapping is one of:

- `exact`: every required semantic and resource has an accepted representation;
- `lossy`: output could be produced only by dropping or weakening information;
- `unsupported`: the adapter has no accepted representation.

`lossy` and `unsupported` mappings set `executable` to false. Findings use
stable codes, including:

- `mapping.script_unsupported`;
- `mapping.permissions_unmapped`;
- `mapping.dependency_unresolved`;
- `mapping.provider_incompatible`;
- `mapping.resource_unsupported`;
- `target.unowned_existing`;
- `target.owned_changed`;
- `target.path_type`;
- `target.path_collision`; and
- `ownership.record_invalid`.

## Output actions and ownership

Every output action is one of:

- `create`: destination is absent;
- `unchanged`: destination digest already equals the planned digest;
- `replace`: destination equals the valid recorded prior digest;
- `remove`: destination equals the valid recorded prior digest and is no longer selected;
- `preserve`: destination is outside the selected owned scope; or
- `conflict`: safe ownership or publication cannot be proven.

Ownership records live at
`.engkit/providers/PROVIDER/SKILL_ID.json` and contain:

- schema version;
- provider and adapter-contract version;
- canonical skill identity, version, and directory digest;
- sorted generated paths with SHA-256 digests; and
- the synchronization plan ID that published them.

An ownership record grants authority over only its listed paths and exact
digests. It grants no authority over parent directories. Removal deletes empty
directories only when transaction evidence proves they were created by that
operation; otherwise it leaves them in place.

## Conflict and safety rules

Planning uses `Lstat`-style inspection and does not follow a destination
symbolic link. A symbolic link, junction, special file, escaping path,
file/directory collision, unowned file, changed owned file, malformed ownership
record, or overlapping ownership record is blocking.

Execution must rebuild the plan immediately before mutation and require
the caller's `--approve` value to equal the rebuilt `plan_id`. Publication must
use same-filesystem staging, atomic replacement only for exact owned files,
durable JSONL journal events, an atomically replaced JSON checkpoint, complete
digest verification, and explicit recovery. The accepted mechanics and
cross-operation lock boundary are specified in
[`provider-sync-execution-v1alpha1.md`](provider-sync-execution-v1alpha1.md)
and [ADR 0013](adr/0013-transactional-provider-synchronization.md). No executor
is invoked by planning; `providersyncexec` implements the separately approved
mutation boundary.

## Spec Kit boundary

`.specify`, official `speckit-*` packages, and files listed by a Spec Kit
integration manifest are external. They may be observed for collision detection
but never enter an Engineering Toolkit ownership record. A shared provider root
does not make its existing children managed.

Provider synchronization never invokes `specify`, a provider CLI, Git, a shell,
or a network service. Spec Kit initialization, upgrade, removal, and workflow
execution retain their independent plans and approvals.

## Initial conformance cases

Planning proves cases 1 through 6 and deterministic plan construction. The C2
transaction tests cover exact publication, replacement, stale-output removal,
unsync, stale-plan checks, and recovery. The D1 offline harness consolidates
pilot, lifecycle, preservation, and release-platform conformance; D2 native
provider discovery remains a later independent approval gate:

1. deterministic exact rendering for the instruction-only `project-guide` pilot;
2. selective output for one provider without touching the other two;
3. preservation of canonical files and unrelated provider-native files;
4. preservation of official Spec Kit packages under the shared Codex root;
5. refusal of changed, unowned, symbolic-link, junction, and colliding targets;
6. non-executable plans for scripts, permissions, dependencies, and lossy fields;
7. exact update and removal using a valid prior ownership record;
8. stale-plan rejection after canonical or destination changes; and
9. stable repeated plan IDs within one unchanged workspace and identical
   rendered bytes, relative output paths, findings, and output digests on
   Windows, Linux, and macOS.

The complete fixture, six-target matrix, evidence, pass criteria, and D1/D2
boundary are defined in
[`provider-conformance-v1alpha1.md`](provider-conformance-v1alpha1.md).
