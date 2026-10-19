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
```

The exact filenames, schemas, ignore policy, and privacy defaults remain decisions for the CLI-foundation milestone. SQLite files and caches should normally be local and regenerable; portable plans and evidence need explicit retention policy.

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
- Locking and crash-consistency strategy per operating system.
- Installer and self-replacement strategy per operating system.
- Project-creation command shape and versioned profile format.
- Exact skill-schema fields, canonical serialization, and migration policy.
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

