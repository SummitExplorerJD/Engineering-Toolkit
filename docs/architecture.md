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
  skill-mutation and project-composition transaction designs.
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
ownership, conflicts, and ordered operations. The separately approved executor
implements the accepted journals, checkpoints, verification, and recovery
design. Provider rendering consumes
canonical output downstream and cannot
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
and regular-file inventory rules. The CLI embeds the reviewed
`project-guide@0.1.0` template and `go-service@0.1.0` pilot profile as immutable
offline catalog content.

The implemented `composition` plan layer depends on catalog, template,
profile, workspace-inspection, and canonical-skill validation ports. It emits
deterministic plans and materializes exact approved output bytes without owning
filesystem publication. The separate `compositionexec` module consumes only a
rebuilt matching executable plan through the approved journal, checkpoint,
staging, verification, and recovery contract.

## 16. Project-composition transaction boundary

The implemented `v0.2.1D1`/`v0.2.1D2` boundary keeps planning and execution in
separate modules. The composition executor consumes only a freshly rebuilt,
fully matching executable plan with no blocking conflicts. Its first scope is
additive: existing matching files are no-ops, while replacement, removal, and
ownership adoption remain unsupported.

New-project creation stages a complete initialized project in a hidden sibling
directory on the destination filesystem. It publishes the complete tree with a
directory rename; an approved empty destination is first retained as an exact
sibling backup. Transaction evidence begins inside the staged `.engkit` tree
and moves with the project, avoiding a successful project whose audit state is
owned only by an external sidecar.

Existing-workspace composition requires prior workspace initialization. It
uses one workspace composition lock, recovery-local staging, aggregate
directory publication for canonical skill packages, and atomic no-replace file
publication through same-filesystem hard links for other profile assets. A
capability probe fails before live mutation when that primitive is unavailable.
This path is recoverable but not globally
atomic across separate workspace outputs. It never replaces the workspace
root, `.git`, or unowned content.

Both kinds use durable plans, JSONL journals, atomic JSON checkpoints,
same-filesystem staging, digest verification, and explicit fail-closed
recovery. The complete design is specified in
[`composition-execution-v1alpha1.md`](composition-execution-v1alpha1.md) and
[ADR 0011](adr/0011-transactional-project-composition.md).

## 17. Selective provider-synchronization boundary

Provider synchronization is a downstream projection of a valid canonical skill
package. It never changes `skill.yaml`, `instructions.md`, or declared canonical
resources. The core planning layer consumes canonical package bytes and an
immutable adapter contract, then produces provider-specific bytes in memory.
Filesystem inspection and eventual publication remain infrastructure concerns.

The `providersync` package owns deterministic planning and exact byte
materialization. `skills` supplies stable canonical snapshots; the provider planner
renders exact bytes, performs `Lstat`-style observations, validates optional
per-file ownership evidence, and emits `engkit.provider-sync-plan/v1alpha1`.
The CLI exposes read-only sync and unsync planning separately from execution.

The initial target layout is deliberately provider-specific:

| Provider | Generated skill root | Shared instruction behavior |
| --- | --- | --- |
| Codex | `.agents/skills/SKILL_ID/` | Adds a derived `SKILL.md` beside the canonical package without owning the directory |
| Claude Code | `.claude/skills/SKILL_ID/` | Creates an isolated derived skill package |
| OpenCode | `.opencode/skills/SKILL_ID/` | Creates an isolated derived skill package and does not generate commands |

Every generated file has a separate ownership record below
`.engkit/providers/PROVIDER/SKILL_ID.json`. The record binds the adapter contract,
canonical source digest, output path, and last published digest. It is
operational evidence, not provider configuration and not a substitute for the
canonical package.

Because the Codex target overlaps the canonical package, a regular root
`SKILL.md` is treated as provider-owned sidecar state. Canonical validation and
snapshot hashing exclude only that exact file; symbolic links, special files,
and every other undeclared package file remain invalid. Provider planning still
observes and conflict-checks the sidecar independently.

Planning classifies every mapping as `exact`, `lossy`, or `unsupported` and
every destination as `create`, `unchanged`, `replace`, `remove`, `preserve`, or
`conflict`. Only exact plans containing create, unchanged, replace, or removal
of an exact previously owned digest may become executable. Replacement and
removal never follow symbolic links and never claim an unrecorded file.

`SKILL.md` contains only provider-supported discovery metadata and the rendered
canonical instructions. Passive declared resources may be copied byte-for-byte
when their relative layout is supported. Codex reuses canonical sibling
resources without adopting them; isolated Claude Code and OpenCode targets use
derived copies with individual ownership. Scripts, non-empty permission
requests, unresolved dependencies, and provider-incompatible resources block
the initial executable contract rather than being omitted. Provider runtime
permission files are never generated.

Official Spec Kit output remains externally owned even where it shares
`.agents/skills` or another provider root. Synchronization owns individual
planned paths only, never `.agents/skills`, `.claude/skills`,
`.opencode/skills`, `.specify`, or an entire generated skill directory. The
complete proposed contract is specified in
[`provider-sync-plan-v1alpha1.md`](provider-sync-plan-v1alpha1.md) and
[ADR 0012](adr/0012-selective-provider-synchronization.md).

## 18. Provider-synchronization transaction boundary

The `v0.2.2C2` implementation adds an execution boundary without merging
provider mutation into canonical skill logic. A provider transaction consumes
one rebuilt executable plan for one provider and one skill. It stages exact
bytes, publishes file-level transitions with retained backups, verifies the
complete provider view, and publishes per-file ownership last.

The transaction uses the existing canonical skill lock followed by a
provider-and-skill lock. This freezes the canonical source and establishes one
lock order across modules. Canonical skill update and removal must refuse while
any supported provider ownership path is present or ambiguous; explicit unsync
removes projections before canonical mutation, and later sync recreates only
the selected provider views. No module automatically cascades across approval
scopes.

Durable state is namespaced below `.engkit/{plans,journals,checkpoints,
recovery}/provider-sync/PROVIDER/SKILL_ID/`. Recovery compares the persisted
plan, journal, checkpoint, locks, staging, backups, live output, and ownership
record. It never trusts a directory or checkpoint alone and never deletes or
adopts ambiguous content.

This boundary retains per-file ownership: parent provider directories remain
unowned and successful unsync leaves empty directories in place. The exact
contract is specified in
[`provider-sync-execution-v1alpha1.md`](provider-sync-execution-v1alpha1.md)
and [ADR 0013](adr/0013-transactional-provider-synchronization.md). The
`providersyncexec` package implements this boundary; CLI routing remains a thin
layer. It invokes no provider runtime, network, credential, script, dependency,
or Spec Kit operation.

## 19. Provider-conformance boundary

Provider conformance is verification around the existing adapter and
transaction boundaries, not a fourth provider integration layer. The D1 harness
creates disposable workspaces from the embedded pilot catalog, drives the
public composition and provider-sync CLI, and compares exact portable outputs.
Production packages continue to own rendering and transaction policy; the
harness owns only fixtures, orchestration, assertions, and evidence capture.

Static contract checks validate provider paths and `SKILL.md` discovery
metadata before publication. Lifecycle checks validate sync, unchanged replay,
unsync, refusals, ownership, recovery boundaries, and preservation. The same
scenario runs through native binaries on Windows, Linux, and macOS for AMD64
and ARM64. Cross-platform comparison excludes only operational evidence fields
whose existing contracts are intentionally machine-local; it never excludes
rendered bytes, relative paths, findings, or generated-file digests. Plan IDs
remain exact approval identifiers within one resolved workspace and are not
treated as cross-host content identifiers.

Native provider discovery is a separate D2 boundary outside the automated
offline harness. A provider CLI adapter is not added to the modular monolith.
Any discovery smoke invokes an external runtime only after a separate approval
binds the exact executable version, command, authentication, network behavior,
workspace, evidence, timeout, and cleanup. Runtime results remain diagnostic
evidence and never become canonical state or ownership authority.

The full conformance plan is specified in
[`provider-conformance-v1alpha1.md`](provider-conformance-v1alpha1.md). It adds
no public schema and changes no accepted adapter or transaction contract.

## 20. Diagnostics boundary

Doctor is an application-level read model assembled from narrow probes. Core
and workspace probes validate platform facts, configuration, canonical
artifacts, transaction evidence, and ownership records. Optional probes describe
SQLite freshness, local-model readiness, provider projections, and MCP
configuration. A probe returns versioned findings and cannot repair the state it
observes.

Findings use stable codes and separate `pass`, `warning`, `error`, and
`unavailable` outcomes. An unavailable optional integration does not make a
healthy workspace invalid. Remediation is descriptive data for a user or
automation to review; it is never a shell command executed by doctor. The exact
proposed contract is documented in
[`doctor-init-v1alpha1.md`](doctor-init-v1alpha1.md).

## 21. Unified initialization boundary

The top-level initialization entry point is a facade over existing workspace and
composition use cases. It inspects one target and selects at most one operation:
new-project creation for an absent or empty destination with an exact profile,
workspace initialization for an existing uninitialized directory without a
profile, or composition for an initialized workspace with an exact profile.

An existing non-empty uninitialized workspace plus a requested profile is
deliberately not converted into a compound transaction. The user first approves
workspace initialization and later approves composition. The facade reuses the
selected plan schema, approval digest, executor, journal, checkpoint, recovery,
and verification contract rather than creating an alternate mutation path.

Classification is deterministic and filesystem-only. The facade requires an
explicit target, treats an empty directory as having zero entries, and treats
partial or invalid `.engkit` state as ambiguous rather than uninitialized. It
does not consult SQLite, a model, MCP, or provider discovery. Absent targets
without a profile, initialized targets without a profile, unsafe paths, and
compound requests fail before a lifecycle is selected.

The selected planner's JSON and plan ID pass through unchanged. Apply repeats
classification, rebuilds the plan from current state, and invokes the matching
existing executor with exact approval. The facade adds no envelope schema,
journal, checkpoint, lock, staging area, recovery command, or migration.
Specialized commands remain public and are the recovery authority. The exact
contract is specified in
[`doctor-init-v1alpha1.md`](doctor-init-v1alpha1.md) and
[ADR 0016](adr/0016-unified-initialization-facade.md).

## 22. Context and local-intelligence boundary

Canonical filesystem artifacts remain the source of truth. A context port
returns bounded excerpts with source paths, content digests, and freshness
evidence. The current SQLite implementation may satisfy retrieval when its
source inventory is current; otherwise the application refuses stale indexed
claims or uses an explicit canonical-filesystem fallback. Neither local advice
nor MCP receives a raw database handle.

Deterministic policy evaluates compatibility, conflicts, ownership, and
executable-plan rules before any model call. An optional adviser may rank
already eligible choices, classify non-authoritative evidence, or explain a
result. Its structured response is an advisory report, not a plan. Selecting a
recommendation creates normal explicit inputs for a later deterministic planner
and requires the same approval as a manually selected input.

The initial Ollama adapter is infrastructure behind an application-owned
adviser interface. It accepts only an explicitly enabled loopback HTTP endpoint
and one configured exact model tag from an allowlist. Requests exclude known
secret-bearing artifacts, use bounded context, disable tool execution, apply
deadlines, and never discover, install, start, or pull a model. The permitted
initial tags are user-configurable candidates:

```text
phi4-mini:latest
qwen3.5:0.8b
granite4:3b
```

Availability and compatibility are observed by doctor; the presence of a tag in
configuration is not proof that it is installed or suitable. The full proposed
boundary is documented in
[`local-intelligence-mcp-v1alpha1.md`](local-intelligence-mcp-v1alpha1.md) and
[ADR 0014](adr/0014-local-intelligence-and-ollama.md).

## 23. MCP harness boundary

The MCP server is another presentation adapter beside the CLI. It translates
protocol requests into the same application ports and returns their versioned
results. It does not import storage implementations into protocol handlers and
does not give a harness direct access to SQLite, canonical files, ownership
records, or transaction internals.

The first transport is standard input/output. Protocol frames are the only data
written to standard output; logs and diagnostics go to standard error. The
server opens no socket and inherits no authority merely because a harness
started it. The initial tools are read-only diagnostics, inspection, search,
catalog, recommendation, skill validation, and plan inspection. Mutation tools,
elicited approval, resources, prompts, subscriptions, and remote transports are
deferred to separately accepted contracts.

This architecture interprets *harness* as the agent host that launches and
communicates with the MCP process, including a compatible Codex, Claude Code, or
OpenCode environment. Provider-specific discovery files remain adapter-owned
projections and are not part of the protocol core. See
[ADR 0015](adr/0015-mcp-harness-boundary.md).

## 24. First-party catalog growth boundary

Catalog growth uses the existing embedded, offline, digest-bound catalog. Each
template or profile is an independent immutable version with its own review,
tests, compatibility evidence, and release decision. Candidate families such as
architecture review, testing strategy, release review, security review,
documentation maintenance, generic projects, services, libraries, and CLI
applications are inventory categories rather than accepted packages.

No catalog milestone enables remote marketplaces, floating versions, automatic
selection, executable skill resources, or permission grants. Local intelligence
may recommend only entries already accepted into the packaged catalog.

