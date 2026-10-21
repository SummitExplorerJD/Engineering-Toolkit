# Engineering Toolkit Architecture

## 1. Architectural style

Engineering Toolkit is a modular monolith: one versioned CLI executable with cohesive internal modules, explicit dependency direction, and shared release lifecycle. The architecture optimizes for local reliability and comprehensible artifacts before distributed scale.

## 2. System context

`engkit` operates on a user-selected project workspace. It reads canonical configuration and workflow artifacts, produces plans, requests approval for material actions, applies approved operations, records journal events and checkpoints, and optionally rebuilds a local context index. Provider adapters generate or synchronize integration-specific assets. The updater verifies releases through TUF metadata before installation.

## 3. Logical modules

| Module | Responsibility | Must not own |
| --- | --- | --- |
| `cli` | Command parsing, output mode, exit-code mapping | Domain policy or persistence logic |
| `workspace` | Root discovery, compatibility checks, layout planning | Provider-specific rendering |
| `config` | YAML loading, merging, schema validation | Arbitrary execution |
| `planning` | Immutable operation plans and change sets | Applying filesystem changes |
| `approval` | Policy evaluation and approval binding | User-interface assumptions |
| `execution` | Apply approved operations and coordinate recovery | Provider-specific semantics |
| `journal` | Append-only JSONL event recording | Canonical current state |
| `checkpoint` | Atomic resumable state snapshots | Historical event ownership |
| `context` | Canonical context discovery and query contracts | SQLite implementation details |
| `index/sqlite` | Optional rebuildable SQLite projection | Sole ownership of project facts |
| `adapters` | Capability model and adapter orchestration | Core workflow policy |
| provider adapters | Codex, Claude Code, OpenCode rendering | Pretending unsupported capabilities exist |
| `skills` | Canonical skill contracts, validation, ownership, and lifecycle planning | Provider-specific storage or implicit script execution |
| `catalog` | Embedded offline catalog inventory and exact content binding | Runtime catalog updates or remote discovery |
| `templates` | Skill-template validation and bounded authoring scaffold | Provider rendering or project mutation policy |
| `profiles` | Project-profile validation and exact template selection | Implicit profile selection or execution |
| `schemautil` | Shared strict YAML and portable-path primitives | Domain-specific policy |
| `project` | Project-profile selection and new-workspace composition | Assuming one ecosystem or source layout |
| `speckit` | Official Spec Kit detection and controlled interoperation | Forking upstream contracts silently |
| `update` domain | Release channel, version, platform, and installation policy | TUF wire formats or network clients |
| TUF adapter | Metadata refresh and verified target download | Channel policy, installation, signing, or publication |
| `platform` | Filesystem, locking, process, and OS abstractions | Business decisions |
| release scripts | User-scoped bootstrap installation and release packaging | Application-managed updates or workspace mutation |

## 4. Dependency direction

Core domain modules define ports. Infrastructure implementations depend inward on those ports. The CLI composes modules at the boundary. Provider adapters, SQLite, Spec Kit, OS operations, and update transport are replaceable infrastructure and must not be imported by core policy packages.

Expected dependency flow:

`CLI -> application services -> domain ports <- infrastructure adapters`

## 5. Canonical artifact model

| Artifact | Format | Mutability | Role |
| --- | --- | --- | --- |
| Configuration | YAML | User-edited | Declarative intent and policy |
| Plan | JSON | Immutable once approved | Exact proposed operation contract |
| Journal | JSONL | Append-only | Execution history and evidence pointers |
| Checkpoint | JSON | Atomically replaced | Resume state for an active workflow |
| Context index | SQLite | Rebuildable | Search and derived relationships |
| Skill manifest | YAML | User-edited | Canonical identity, contract, ownership, and declared resources |
| Skill instructions | Markdown | User-edited | Provider-neutral instructional content |
| Skill template | YAML and declared text resources | Catalog-authored | Deterministic source for a rendered canonical skill |
| Project profile | YAML | Catalog-authored | Versioned selection and composition intent |
| Trust metadata | TUF JSON | Signed and versioned | Distribution trust decisions |

Every persisted artifact includes a schema version. Canonical serialization rules will be defined before signing or stable machine-output commitments.

## 6. Proposed repository implementation layout

The implementation milestone is expected to introduce:

```text
cmd/engkit/          CLI entry point
internal/app/        Use-case orchestration
internal/domain/     Plans, approvals, workflows, and policies
internal/workspace/  Workspace discovery and layout
internal/storage/    Files, journals, checkpoints, and SQLite projection
internal/adapters/   Provider and Spec Kit adapters
internal/update/     Update policy ports and TUF infrastructure adapter
internal/platform/   OS-specific boundaries
```

This repository will not add a `src/` wrapper around Go packages. `src/` belongs to managed project workspaces only when ecosystem compatibility and user configuration support it.

## 7. Proposed `.engkit` workspace layout

```text
.engkit/
  config.yaml
  plans/
  journals/
  checkpoints/
  context/
  cache/
  recovery/
```

SQLite files and caches are local and regenerable. The ignored `recovery/`
tree is not cache: it owns transaction staging, locks, and rollback copies that
must be preserved while a mutation is incomplete. Portable plans and evidence
require explicit retention policy.

## 8. Execution lifecycle

1. Discover and validate the workspace.
2. Load configuration and applicable policy.
3. Inspect current state without mutation.
4. Produce a deterministic JSON plan and human-readable summary.
5. Evaluate whether approval is required.
6. Bind approval to the plan digest and scope.
7. Execute guarded operations.
8. Append journal events and atomically advance checkpoints.
9. Run verification and record observed evidence.
10. Complete, recover, or roll back according to the plan.

## 9. Security boundaries

- Workspace paths are untrusted input and must remain within the resolved root unless explicitly allowed.
- Configuration and provider-generated content are data, not executable instructions.
- External tools are invoked through argument-safe process APIs with explicit working directories.
- Secrets never belong in plans, journals, checkpoints, context databases, or diagnostic bundles.
- TUF trust roots are pinned and updated only through verified rotation.
- The bootstrap root is bundled with the application, never loaded from a
  workspace or writable metadata cache.
- Verified download and executable installation are separate approval and
  recovery boundaries.
- Spec Kit and provider files are inspected and diffed before overwrite or migration.

## 10. Open decisions for later milestones

- CLI framework versus Go standard library parsing.
- YAML and JSON Schema libraries.
- Exact schema identifiers and migration policy.
- SQLite driver and build strategy.
- Acceptance and exact version of the proposed go-tuf v2 dependency.
- Locking and crash-consistency strategy for workflows other than the accepted
  skill-mutation transaction.
- Installer and self-replacement strategy per operating system.
- Exact skill-schema fields, canonical serialization, and migration policy.
- Exact template and profile schema fields, catalog update policy, and public
  command names.
- Runtime trust and permission UX for executable skill resources.

Each dependency choice requires a separate approval because it modifies `go.mod` and long-term compatibility.

## 11. Planned distribution data flow

`CLI -> update policy -> TUF client port <- go-tuf adapter -> bounded transport`

The first implementation slice ends at a verified, atomically published cache
entry. Application state uses platform-specific user state and cache locations,
not workspace `.engkit`. A later installer consumes an immutable approved plan
that binds the verified digest and destination. Repository authoring, keys,
signing, CI, and publication remain outside the runtime dependency graph.

## 12. Bootstrap installation boundary

The standalone `install.ps1` and `install.sh` release helpers are outside the
runtime dependency graph. They support first installation before the CLI is
available and explicit manual upgrades. They require an exact version, verify
the selected executable, stage within the destination filesystem, and preserve
an existing executable on failure.

These scripts do not activate the internal TUF client and are not an
implementation of a future `engkit update install` operation. An
application-managed installer remains downstream of TUF verification and the
plan, approval, journal, and checkpoint lifecycle.

## 13. Canonical skill boundary

Canonical skill packages live at `.agents/skills/SKILL_ID/` and are identified
only by a valid `skill.yaml` using an Engineering Toolkit schema. The initial
schema identifier is `engkit.skill/v1alpha1`; `instructions.md` owns the
provider-neutral instructional prose. Declared optional resources remain under
`references/`, `scripts/`, `templates/`, and `assets/` within the package.

Directories without the canonical manifest are external even when they reside
under `.agents/skills/`. This includes the current official Spec Kit
Codex-facing skills. Read-only discovery may classify and report those
directories but must not adopt or modify them. A later provider adapter may
render derived files only with explicit per-file ownership, prior-content
digests, conflict detection, and an approved plan.

The `skills` domain owns package identity, validation, compatibility,
provenance, and lifecycle planning. It does not execute scripts or own provider
storage semantics. Initial list, inspect, and validate operations perform no
network access, credential access, process execution, or workspace mutation.
Mutation follows the standard plan, approval, execution, journal, and
checkpoint lifecycle.

The complete experimental manifest fields and read-only CLI behavior are
specified in [`skill-manifest-v1alpha1.md`](skill-manifest-v1alpha1.md).
Immutable create, update, and remove previews use the separate
[`engkit.skill-mutation-plan/v1alpha1`](skill-mutation-plan-v1alpha1.md)
contract. The planning module exposes no execution function. The approved
execution design uses same-filesystem package-directory publication, per-skill
locks, durable plans, journals and checkpoints, and explicit digest-based
recovery as specified in
[`skill-execution-v1alpha1.md`](skill-execution-v1alpha1.md). Implementation
is owned by the separate `skillexec` module.

## 14. Template and project-profile composition boundary

A skill template is catalog source, not installed workspace state. First-party
template sources and project profiles are reviewed in this repository and
packaged with a specific CLI release. The initial catalog is local and offline;
remote catalogs, marketplace discovery, and runtime catalog updates are outside
the boundary.

A rendered skill is an ordinary canonical package governed by the existing
`engkit.skill/v1alpha1` validator and transactional lifecycle. The renderer
generates declared resources first, computes their content hashes, and emits the
canonical manifest last. It may substitute only schema-declared typed variables
through a bounded placeholder syntax. It has no general expression language and
does not read ambient environment variables, execute template resources, contact
a network, or access credentials.

A project profile selects exact template identifiers and versions and may
describe conventions, workflows, source-layout policy, and optional integration
requests. Read-only workspace classification produces evidence and compatible
profile recommendations. The user selects a profile explicitly; detection is
never approval.

Composition has two use cases with one policy boundary:

- new-project creation targets an absent or explicitly empty destination; and
- existing-workspace composition preserves all unowned content and reports
  conflicts before mutation.

Both produce a deterministic JSON plan that binds the profile, catalog version,
template versions, resolved variables, workspace facts, rendered file digests,
ownership, conflicts, and ordered operations. Execution remains a later,
separately approved transaction using journals, checkpoints, verification, and
recovery. Provider rendering consumes canonical output downstream and cannot
become the canonical template model.

## 15. Experimental template and composition contracts

The accepted `v1alpha1` contracts are split by ownership boundary:

- [`template-catalog-v1alpha1.md`](template-catalog-v1alpha1.md) defines the
  immutable packaged catalog, skill-template sources, bounded variables, and
  deterministic rendering inputs.
- [`project-profile-v1alpha1.md`](project-profile-v1alpha1.md) defines exact
  template selection, compatibility evidence, source-layout policy, workspace
  assets, and non-executable integration requests.
- [`composition-plan-v1alpha1.md`](composition-plan-v1alpha1.md) defines the
  shared immutable preview for new-project creation and existing-workspace
  composition.

Future implementation keeps parsing and rendering in provider-neutral core
modules. Catalog embedding and filesystem publication are infrastructure at the
CLI boundary. Rendered canonical skills pass through the existing `skills`
validator and lifecycle; profile-owned workspace assets require separate
composition ownership records and cannot be inferred as canonical skills.
Profile-owned workspace assets include the root `AGENTS.md` and declared files
below `.agents/context`, `.agents/conventions`, and `.agents/workflows`;
canonical packages remain exclusively below `.agents/skills`.

The implemented `catalog`, `templates`, and `profiles` packages provide strict
offline readers. `templates` also resolves typed variables and produces a
complete canonical package in memory. `skills` exposes a matching in-memory
validation boundary so planning does not need temporary filesystem state.
`schemautil` owns only shared YAML ambiguity rejection, portable path checks,
and regular-file inventory rules. The CLI embeds an empty catalog until
first-party content receives its own approval.

The implemented `composition` plan layer depends on catalog, template,
profile, workspace-inspection, and canonical-skill validation ports. It emits
deterministic, non-executable plans and exposes no mutation function. A later
executor may consume a rebuilt matching plan only after its journaling,
checkpoint, staging, verification, and recovery contract is independently
approved.

