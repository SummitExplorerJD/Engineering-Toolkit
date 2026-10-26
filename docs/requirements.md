# Engineering Toolkit Requirements

## 1. Purpose

Engineering Toolkit (`engkit`) will provide a consistent, auditable way to initialize, inspect, plan, execute, resume, and integrate engineering workspaces across supported operating systems and coding-agent environments.

## 2. Actors

- **Engineer:** creates and operates a workspace.
- **Reviewer or approver:** authorizes material changes and verifies evidence.
- **Agent adapter:** translates canonical workflows into provider-specific instructions.
- **Automation:** consumes stable machine-readable output without interactive prompts.
- **Release operator:** signs and publishes verified distributions.

## 3. Functional requirements

### FR-001: CLI lifecycle

The system shall expose a single `engkit` CLI with discoverable commands, consistent exit codes, machine-readable output where applicable, and non-interactive operation suitable for automation.

### FR-002: Workspace initialization

The system shall initialize a Project Workspace Standard without silently replacing existing files. It shall inspect compatibility before offering a `src/` layout and shall support a no-change preview.

### FR-003: Configuration

The system shall load declarative YAML configuration from `.engkit`, validate it against a versioned contract, report actionable errors, and avoid executing arbitrary code from configuration.

### FR-004: Plans

The system shall create and validate versioned JSON plans that describe intended operations, inputs, preconditions, affected paths, risks, required approvals, and verification steps.

### FR-005: Journals

The system shall append JSONL journal events for observable workflow transitions without rewriting prior events. Events shall be ordered, identifiable, and safe to replay for diagnostics.

### FR-006: Checkpoints and recovery

The system shall write JSON checkpoints atomically, resume interrupted workflows when safe, detect incompatible state, and provide an explicit recovery path.

### FR-007: Approval gates

The system shall pause before material changes identified by policy or plan. Approval records shall identify the approved plan and must not authorize unrelated later changes.

### FR-008: Context index

The system may maintain an optional SQLite database for contextual discovery, search, and acceleration. The index shall be rebuildable exclusively from canonical filesystem artifacts.

### FR-009: Agent adapters

The system shall provide selective adapters for Codex, Claude Code, and OpenCode. Each adapter shall declare supported capabilities and render only provider-appropriate assets.

### FR-010: Spec Kit interoperability

The system shall interoperate with the official Spec Kit project without vendoring or silently modifying upstream assets. Initialization and upgrades shall be previewable and separately approved.

### FR-011: Inspection and diagnostics

The system shall inspect workspace health, versions, schemas, journal integrity, checkpoint consistency, adapter state, and contextual-index freshness without mutating the workspace by default.

### FR-012: Distribution and updates

The system shall support authenticated release metadata, target verification, expiration checks, rollback protection, and key rotation following TUF roles and trust separation.

The system shall bootstrap from an application-bundled trusted root, verify the
complete metadata chain, select only an exact supported channel and platform
target, enforce target length and cryptographic hashes, and keep writable
metadata state outside project workspaces. Verified download and installation
shall be separate operations. Application-managed installation shall require
its own immutable plan and approval and shall never be implied by a metadata
refresh. A standalone bootstrap installer may operate before the CLI is
available, but it shall require an exact version, verify the selected
executable, require explicit replacement intent, and use a recoverable
user-scoped transaction.

### FR-013: Extensibility

The system shall allow new workspace capabilities and agent adapters to be added through internal interfaces without splitting the initial product into separately deployed services.

### FR-014: Skill lifecycle

The system shall support a provider-neutral, versioned skill model with explicit
identity, capabilities, inputs, outputs, permissions, compatibility, ownership,
and provenance. Skill creation, update, removal, and provider synchronization
shall be previewable, deterministic, and separately approved before mutation.
Unsupported or lossy provider mappings shall be reported rather than silently
discarded.

A canonical skill shall be explicitly identified by a versioned Engineering
Toolkit manifest rather than inferred from a provider-native file or directory
name. Discovery and validation shall distinguish managed canonical packages,
provider-native packages, generated outputs, conflicts, and invalid packages.
Read-only skill operations shall not execute package resources, contact remote
services, access credentials, or adopt external ownership.

### FR-015: Project creation and composition

The system shall distinguish initializing an existing workspace from creating a
new project. Project creation and existing-workspace composition shall use an
explicitly selected, versioned profile; inspect ecosystem compatibility before
choosing a source layout; preview all generated paths; and compose workspace
conventions, skills, workflows, provider integrations, and optional Spec Kit
assets without silently replacing existing files.

Read-only detection may recommend compatible profiles but shall not select or
apply one implicitly. A composition plan shall bind the exact profile version,
skill-template versions, resolved variables, observed workspace facts, rendered
content digests, conflicts, and affected paths. Applying a composition plan
shall require approval of that complete plan.

### FR-016: Skill templates and project profiles

The system shall distinguish an editable skill template from an installed
canonical skill. A template may produce a canonical skill package but shall not
be discovered as installed state merely because it is available in a catalog.

First-party templates and project profiles shall be versioned, provider-neutral,
reviewable repository sources distributed with the CLI. Rendering shall be
offline and deterministic. Template substitution shall use a bounded declarative
variable model with no arbitrary expressions, process execution, network access,
credential access, or environment-variable expansion. Missing, unknown, or
invalid variables shall fail closed.

Profiles shall identify exact template versions and may also declare
conventions, workflows, source-layout policy, and optional integration requests.
They shall not grant runtime permissions or bypass the existing plan, approval,
ownership, conflict, journal, checkpoint, or recovery contracts.

### FR-017: Selective provider synchronization

The system shall render an explicitly selected canonical skill for an explicitly
selected provider without making provider output canonical. Synchronization
shall bind the canonical package digest, adapter contract, observed destination
state, every intended output byte, and every prior Engineering Toolkit ownership
record in a deterministic plan before mutation.

The system shall own generated provider files individually rather than claiming
a provider root or package directory. It shall create an absent output, replace
or remove only an output whose observed digest matches its recorded ownership,
and preserve changed, unrecorded, provider-native, and Spec Kit-owned content.
Unsupported or lossy mappings shall remain reviewable but non-executable.

Provider synchronization shall not execute skill resources, resolve
dependencies, contact networks, access credentials, generate runtime permission
policy, initialize or upgrade Spec Kit, or infer approval from a prior skill or
project operation.

### FR-018: Unified initialization entry point

The system shall provide a discoverable top-level initialization entry point
that inspects one explicit target and selects one existing initialization or
composition lifecycle without weakening its contract. Planning shall remain
read-only. Applying an initialization shall rebuild the selected operation and
require approval of its complete current plan.

The unified entry point shall distinguish an absent or empty new-project target,
an existing uninitialized workspace, and an initialized workspace. It shall
refuse ambiguous requests and shall not silently chain independently approved
workspace initialization and profile composition operations.

The entry point shall require an explicit target and an exact profile version
when composition is requested. It shall return the selected existing plan
without a wrapper or changed plan ID, rebuild that plan before apply, and
delegate mutation and recovery to the selected existing executor. It shall not
own a new persisted plan type, journal, checkpoint, lock, or recovery command.

### FR-019: Workspace doctor

The system shall provide a read-only doctor operation that reports CLI,
platform, workspace, configuration, persisted-artifact, contextual-index,
catalog, provider-projection, local-intelligence, and MCP readiness through
versioned machine-readable findings. Each finding shall identify its stable
code, scope, severity, observed evidence, and an actionable remediation that
does not execute automatically.

Doctor shall distinguish unavailable optional capabilities from invalid required
state. It shall not install software, download models, rebuild indexes, recover
transactions, modify configuration, invoke agent runtimes, or contact non-local
services.

### FR-020: Local decision assistance

The system may use an explicitly enabled local model adapter to rank contextual
results, classify read-only evidence, explain conflicts, and recommend exact
profile, template, skill, or verification selections. Model output shall be
advisory and shall not be canonical state, an approval, an executable plan, or
authority to mutate a workspace.

The initial adapter shall support an explicitly configured loopback Ollama
endpoint and an allowlist of exact model tags. It shall not install or start
Ollama, pull models, discover remote endpoints, access credentials, invoke
tools, or fall back to a remote model. A deterministic non-model path shall
remain available for every correctness-critical operation.

### FR-021: MCP harness interface

The system shall provide an optional Model Context Protocol server over standard
input and output so an explicitly launched agent harness can call bounded
Engineering Toolkit use cases. The initial tool set shall be read-only and
shall delegate to application ports used by the CLI rather than reading SQLite,
provider directories, or internal storage directly.

MCP requests shall select an explicit workspace root, enforce the same path and
schema validation as the CLI, return versioned structured results, serialize
writes to standard output according to the protocol, and keep diagnostics on
standard error. The initial server shall not open a listening socket, expose
mutation tools, execute skill resources, access credentials, or grant approval.

### FR-022: First-party catalog growth

The system shall support incremental addition of independently reviewed,
versioned first-party templates and profiles. Each addition shall use the
existing exact-version, immutable digest, bounded rendering, and offline catalog
contracts. A catalog expansion shall not imply remote discovery, automatic
installation, executable resources, or permission grants.

## 4. Non-functional requirements

### NFR-001: Portability

The CLI shall support Windows, macOS, and Linux. Filesystem behavior shall account for path separators, case sensitivity, reserved names, permissions, and atomic-replacement differences.

### NFR-002: Safety

Read-only inspection shall be the default. Destructive or broad changes shall require an explicit target, preview, approval, and recoverable strategy where practical.

### NFR-003: Reliability

Interrupted operations shall not leave an apparently successful but incomplete state. Writes shall be atomic when supported and guarded by validation and checkpoints.

### NFR-004: Determinism

Given the same inputs, versioned configuration, and environment facts, planning and rendering shall produce equivalent canonical artifacts.

### NFR-005: Auditability

Material decisions and execution transitions shall be attributable through plans, approvals, journals, checkpoints, and observed verification evidence.

### NFR-006: Security

The system shall validate paths, avoid implicit code execution, protect trust metadata, minimize credential access, redact secrets from logs, and fail closed during update verification.

Distribution clients shall bound downloads, timeouts, redirects, retries, and
metadata sizes; reject traversal and ambiguous target paths; preserve the last
known-good trust state after failure; and provide no normal-mode bypass for
invalid signatures, expired metadata, or rollback detection.

Skill validation shall reject absolute or escaping resource paths, ambiguous or
duplicate canonical identities, unsupported schema versions, undeclared managed
resources, integrity mismatches, and symlinks that escape the permitted package
or workspace boundary. Declared permissions describe requested capabilities and
shall not authorize execution by themselves.

### NFR-007: Performance

Common local inspection commands should complete interactively on typical repositories. Optional indexing shall improve discovery without being required for correctness.

### NFR-008: Compatibility

Persisted schemas and CLI machine output shall be versioned. Breaking changes require migration guidance and a separately approved compatibility decision.

### NFR-009: Maintainability

Core policy and domain behavior shall be testable without invoking provider CLIs, network services, SQLite, or terminal rendering.

### NFR-010: Observability

Errors shall distinguish user input, policy refusal, environmental failure, integration failure, and internal defects. Diagnostic output shall be actionable and safe to share.

### NFR-011: Local intelligence isolation

Local-model prompts shall be constructed from bounded, explicitly selected,
non-secret context. Requests shall have size limits, deadlines, cancellation,
and deterministic failure behavior. Model unavailability, malformed output, or
timeout shall degrade to an actionable diagnostic or deterministic fallback and
shall not block correctness-critical read-only or planning operations.

### NFR-012: Protocol isolation

The MCP transport, local-model client, SQLite projection, and provider adapters
shall remain infrastructure modules behind application-owned interfaces. Core
domain tests shall not require an MCP client, Ollama daemon, model download,
SQLite database, provider CLI, credential, or network connection.

## 5. Initial non-goals

- A hosted control plane or multi-tenant service.
- A general-purpose plugin runtime executing untrusted code.
- Full behavioral parity among agent providers.
- SQLite as the authoritative state store.
- Automatic commits, pushes, deployments, or release publication.
- Replacing package managers, Git, coding-agent CLIs, or Spec Kit.

## 6. Milestone-1 acceptance criteria

- Requirements distinguish normative behavior from future implementation.
- Architecture assigns every requested artifact format to a clear owner.
- ADRs record the initial high-impact decisions and alternatives.
- Approval boundaries are consistent across repository instructions and workflows.
- Spec Kit initialization remains unexecuted pending separate approval.
- No executable code, dependency, Git state, or existing file is changed.

## 7. `v0.2.0A` skill-contract acceptance criteria

- The canonical package location and ownership marker are unambiguous.
- Existing Spec Kit and provider-native skills remain external and unchanged.
- Structured metadata and provider-neutral instructions have separate owners.
- Discovery, inspection, and validation are read-only and non-executing.
- Creation, update, and removal remain bound to plans, explicit approvals,
  journals, checkpoints, and separately approved implementation slices.
- Provider synchronization, remote installation, imports, and script execution
  remain outside the approved design milestone.

## 8. `v0.2.0C` skill-planning acceptance criteria

- Create, update, and remove plans are deterministic JSON and contain no
  timestamps or random identifiers.
- Plans bind exact source and target state through content and inventory hashes.
- Create refuses existing targets; update requires a higher version; remove
  accepts only valid canonical targets.
- External, invalid, conflicting, escaping, or symlinked state fails closed.
- File and directory type transitions are ordered without escaping the target
  package root.
- Every plan requires approval and remains explicitly non-executable.
- Planning performs no mutation, script execution, dependency resolution,
  provider rendering, network access, or credential access.

## 9. `v0.2.0D` skill-execution acceptance criteria

- Execution rebuilds the applicable plan from current filesystem state and
  requires approval of the complete matching plan ID before writing artifacts.
- Only canonical managed packages may be created, updated, removed, or
  recovered; provider-native, Spec Kit, conflicting, and invalid state remains
  unchanged.
- Create and update stage and validate a complete package before publication.
  Update and remove preserve an exact same-filesystem recovery backup instead
  of mutating or deleting the live package file by file.
- A per-skill lock, immutable persisted plan, append-only JSONL journal, and
  atomically replaced JSON checkpoint make every transition attributable.
- Explicit recovery compares actual target, staging, and backup digests with
  the persisted approved plan and refuses ambiguous content without deleting
  it.
- Completion is recorded only after canonical validation and complete digest
  verification of the final state.
- Execution and recovery remain local and do not run package resources,
  resolve dependencies, render providers, access credentials, or contact a
  network.

## 10. `v0.2.0E` template-and-profile design acceptance criteria

- Templates, profiles, rendered canonical skills, and provider-specific output
  have distinct ownership and storage boundaries.
- First-party catalog content is versioned with the CLI and renders offline.
- Profile selection is explicit; read-only project detection may only recommend
  compatible profiles with evidence.
- Rendering permits only declared, typed variables and performs no arbitrary
  evaluation, process execution, network access, credential access, or ambient
  environment expansion.
- Composition plans bind all selected versions, resolved inputs, observed
  workspace facts, rendered digests, conflicts, and affected paths before any
  mutation.
- Existing files, external Spec Kit skills, and provider-native assets remain
  unchanged unless a later exact plan owns and approves their paths.
- Authoring scaffolds, profile composition, execution, first-party catalog
  content, and provider synchronization remain separately approved
  implementation slices.

## 11. `v0.2.1A` template-and-profile contract acceptance criteria

- Catalog, template, profile, and composition-plan artifacts use explicit
  experimental schema identifiers and reject unknown or ambiguous YAML state.
- Catalog entries and profile selections bind exact versions and content
  digests; floating or implicit latest-version resolution is unavailable.
- Template and profile variables are declared, typed, validated, and rendered
  through bounded single-pass substitution in text resources only.
- Output paths are fixed declarations and cannot be produced through variable
  interpolation, ambient environment state, or executable template logic.
- Profile recommendation emits evidence but never selects a profile or grants
  approval.
- Profiles can declare the initial `AGENTS.md` and `.agents/context`,
  `.agents/conventions`, `.agents/skills`, and `.agents/workflows` workspace
  surfaces without treating those assets as installed canonical skills.
- Project creation and existing-workspace composition emit deterministic JSON
  plans that bind every input, output, conflict, owner, digest, precondition,
  operation, verification rule, and recovery rule before mutation.
- Catalog inspection, template inspection, profile inspection, recommendation,
  and composition planning are offline and perform no writes.
- Every CLI command family and subcommand provides successful `help`, `-h`, and
  `--help` behavior rather than interpreting help as an unknown subcommand.
- Parsers, scaffold execution, rendering, plan execution, first-party catalog
  content, and provider synchronization remain separate implementation gates.

## 12. `v0.2.1C` rendering-and-planning acceptance criteria

- Template and profile values are resolved by declared type, constraint, and
  canonical text form; unknown, duplicate, missing, or invalid inputs fail.
- Rendering is a single pass over declared text resources and does not write
  temporary or target files, execute code, read credentials, or contact a
  network.
- Every rendered canonical skill is validated in memory before it enters a
  composition plan.
- Recommendation records observed evidence and compatibility reasons without
  ranking, selecting, approving, or applying a profile.
- New-project and existing-workspace planning produce deterministic
  `engkit.composition-plan/v1alpha1` JSON with exact catalog, profile, template,
  variable, filesystem, output, conflict, precondition, operation,
  verification, and recovery data.
- Planning inventories the target, rejects links and special files, treats
  unowned collisions as blocking, preserves unrelated content, and performs no
  writes.
- Every composition plan has `requires_approval: true` and `executable: false`;
  applying or recovering a composition remains a separate approval gate.

## 13. `v0.2.1D1` composition-execution design acceptance criteria

- Execution rebuilds the complete current plan, requires the exact full
  approved ID, rejects blocking conflicts, and performs no write before lock
  acquisition and approval validation complete.
- The initial executor is additive only. Existing matching files are no-ops;
  replacement, removal, and implicit ownership adoption are unavailable.
- New projects are assembled as complete initialized workspace trees on the
  destination filesystem and published by directory rename. An approved empty
  destination is retained as an exact recovery backup during publication.
- Existing-workspace composition requires valid `.engkit` state, uses one
  workspace composition lock, publishes canonical skill packages as complete
  directories, and publishes other files through an atomic no-replace
  primitive that is capability-checked before mutation.
- Plans are JSON, journals are append-only JSONL, checkpoints are atomically
  replaced JSON, and all staging and backups stay on the target filesystem.
- Recovery is explicit and approval-bound. It completes an exact published
  state, rolls back only exact transaction-created output, or refuses ambiguous
  and changed content while preserving evidence and the lock.
- Project creation and workspace composition verify every output digest,
  canonical skill, workspace state artifact, and preservation of unrelated
  observed paths before reporting success.
- The design introduces no dependency, network access, credential access,
  script execution, provider synchronization, Spec Kit mutation, catalog
  content, Git operation, or release operation.
- Executor implementation and its Windows, Linux, and macOS failure-injection
  suite were accepted through the separate `v0.2.1D2` approval gate.

## 14. `v0.2.1D2` composition-execution acceptance criteria

- Valid additive plans are executable; blocking conflicts remain reviewable but
  non-executable, and pre-executor plan IDs cannot be reused.
- `project create`, `project compose`, and `project recover` implement exact
  approval binding, durable transaction evidence, verification, and explicit
  recovery.
- New-project publication moves a complete initialized project tree into place
  and supports both absent and approved-empty destinations.
- Existing-workspace composition stages all bytes, publishes complete canonical
  skill directories, and capability-checks atomic no-replace file publication
  before changing live outputs.
- Recovery validates persisted plan identity, journal sequence, checkpoint
  layout, locks, output digests, canonical skills, and transaction-created
  directories before completing or rolling back.
- The implementation has no new dependency, network access, credential access,
  provider synchronization, Spec Kit mutation, first-party catalog content,
  Git operation, or release operation.

## 15. `v0.2.1E` first-party pilot-content acceptance criteria

- The packaged offline catalog contains exactly `project-guide@0.1.0` and
  `go-service@0.1.0`, each bound to a complete immutable directory digest.
- The template renders a valid canonical skill with derived provenance and no
  executable resource, dependency, network permission, or provider output.
- The profile explicitly binds `project_name`, selects the exact template
  version, creates the approved `.agents` surfaces and `src/`, and renders only
  `AGENTS.md`, context, convention, and workflow assets.
- Planning remains read-only. Creation requires the exact approved plan ID and
  produces an initialized workspace whose installed skill validates.
- The pilot generates no application code, `go.mod`, script, dependency
  installation, provider synchronization, Spec Kit mutation, network access,
  credential access, Git operation, or release operation.
- Published template and profile versions are immutable; changed bytes require
  a new version and catalog digest.

## 16. `v0.2.2A` provider-synchronization design acceptance criteria

- Canonical packages remain the sole authored source; provider files are
  deterministic derived output with per-file ownership and SHA-256 digests.
- Codex, Claude Code, and OpenCode use explicit, independently selected adapter
  targets; selecting one provider never enables another.
- Plans bind the canonical package version and directory digest, adapter
  contract version, observed target state, exact output bytes and digests,
  conflicts, mapping quality, approvals, and verification steps.
- Output creation, replacement, and removal fail closed when a path is a
  symbolic link, special file, unowned existing file, changed owned file, or
  file/directory collision.
- Unsupported and lossy mappings are reported with stable codes and make a
  plan non-executable; no canonical field or declared resource is silently
  discarded.
- Provider permission configuration remains runtime-owned. A non-empty
  canonical permission request is not converted into repository policy.
- Executable resources, dependency resolution, network access, credentials,
  imports, and remote installation remain outside the initial synchronization
  implementation.
- Official Spec Kit files and manifests remain externally owned. Provider-root
  overlap does not transfer directory ownership to Engineering Toolkit.
- Planning, transactional execution, provider conformance testing, Git
  operations, and release publication remain separately approved gates.

## 17. `v0.2.2B` provider-synchronization planning acceptance criteria

- `engkit skills plan-sync` selects exactly one of `codex`, `claude`, or
  `opencode` and optionally one canonical skill ID.
- Planning is deterministic, offline, and read-only; it writes neither provider
  output nor `.engkit/providers` ownership records.
- Plans bind stable canonical snapshots, adapter contract and evidence date,
  destination observations, rendered digests, mapping findings, affected
  paths, risks, required approvals, and verification steps.
- Existing unowned, changed owned, symbolic-link, junction, special-file, and
  colliding targets fail closed with stable conflict codes.
- A regular root `SKILL.md` is a Codex provider sidecar, excluded from canonical
  validation and package digests but separately governed by provider ownership.
- Passive resources are referenced in place for Codex and copied in memory for
  isolated Claude Code and OpenCode plans. Scripts, permission requests, and
  unresolved dependencies make plans non-executable.
- Official Spec Kit and unrelated provider-native packages remain external and
  byte-for-byte untouched.
- Provider publication, removal, transaction evidence, recovery, conformance,
  Git operations, and release publication remain separately approved gates.

## 18. `v0.2.2C1` provider-synchronization execution design acceptance criteria

- Initial execution is restricted to one explicit provider and one canonical
  skill per approved transaction; multi-skill plans remain read-only.
- Sync, unsync, and recovery have separate explicit commands and exact plan-ID
  approval. Unsync authority comes only from a valid ownership record and exact
  observed owned digests.
- Provider transactions acquire the canonical skill lock before a
  provider-and-skill lock. Normal execution never infers stale-lock authority
  from age or process existence.
- Canonical skill update and removal fail while any supported provider
  ownership path is present or ambiguous, including malformed, linked, or
  unreadable records. Providers are unsynchronized and later resynchronized
  through separately approved operations; no cascade is implicit.
- Exact rendered bytes and the next ownership record are staged on the
  workspace filesystem. Outputs publish in stable order with retained backups,
  full output verification, and ownership publication last.
- Durable plans, JSONL journals, JSON checkpoints, same-filesystem staging,
  retained backups, and explicit recovery are required. Cross-device fallback
  is prohibited.
- Recovery validates all durable and live evidence, completes or restores only
  exact known states, and refuses while preserving ambiguous content and locks.
- Ownership remains per file. Successful unsync leaves empty directories in
  place and defines no automatic cleanup or backup retention policy.
- Provider CLIs, networks, credentials, resources, dependency resolution,
  runtime permission configuration, Spec Kit mutation, Git operations, and
  release publication remain outside the executor.
- Design acceptance does not authorize implementation. The exact code package,
  changed files, tests, risks, and rollback plan require separate approval.

## 19. `v0.2.2C2` provider-synchronization implementation acceptance criteria

- `skills plan-unsync`, `sync`, `unsync`, and `recover-sync` implement the
  accepted single-provider, single-skill CLI contract and require exact plan-ID
  approval before mutation.
- The executor rebuilds approved plans before publication, stages exact bytes,
  uses capability-checked no-replace publication, verifies all outputs, and
  publishes ownership last.
- Plans, JSONL journals, JSON checkpoints, dual locks, staging, and retained
  backups are durable and namespaced by provider, skill, and plan ID.
- Recovery completes an exact published final state, rolls back an exact known
  partial state, and preserves evidence and locks when any state is ambiguous.
- Canonical skill update and removal fail closed while any supported provider
  ownership record is present, malformed, linked, or unreadable.
- The implementation performs no provider CLI, network, credential, dependency,
  script, Spec Kit, Git, or release action.
- Provider conformance, pilot validation, Git operations, and publication remain
  separately approved gates.

## 20. `v0.2.2D` provider-conformance planning acceptance criteria

- Conformance uses the immutable `project-guide@0.1.0` pilot created from the
  explicit `go-service@0.1.0` profile in a fresh disposable workspace.
- Offline D1 tests validate exact `SKILL.md` paths, frontmatter, instructions,
  line endings, output digests, plan IDs, ownership, sync, unchanged replay,
  unsync, refusal behavior, and preservation for Codex, Claude Code, and
  OpenCode.
- Selecting one provider creates no output or ownership for another provider;
  canonical packages, unrelated provider-native files, official-style Spec Kit
  sentinels, and out-of-scope workspace files remain byte-for-byte unchanged.
- Windows, Linux, and macOS on AMD64 and ARM64 run the same native conformance
  scenario. Rendered bytes, relative paths, findings, and generated-file
  digests are identical across the six targets. Root-bound plan IDs are stable
  when repeated in one unchanged workspace but are not compared across hosts.
- A skipped platform, unsupported publication primitive, changed golden output,
  ambiguous provider contract, or preservation failure blocks conformance.
- D1 performs no provider invocation, provider installation, authentication,
  network access, credential access, skill execution, Spec Kit action, Git
  operation, or release operation.
- Native provider discovery is a separate D2 approval gate. It must bind exact
  provider versions, commands, host support, authentication and network
  behavior, evidence, timeouts, cleanup, and rollback before execution.
- D1 and D2 evidence do not adopt provider-native or Spec Kit content and do not
  change the canonical package, adapter contract, public CLI, or persisted
  schemas.
- Planning acceptance does not authorize the D1 harness, CI changes, native D2
  smoke tests, Git operations, or publication. Each requires its own exact
  implementation or execution packet.

## 21. `v0.3.0A` local-intelligence and harness design acceptance criteria

- `doctor`, unified `init`, local advice, and MCP have explicit boundaries,
  versioned outputs, failure behavior, and independently approved implementation
  slices.
- Unified initialization selects at most one existing transaction lifecycle and
  never collapses two approval scopes into one implicit operation.
- Doctor is read-only and emits stable findings without installing, repairing,
  rebuilding, recovering, downloading, or invoking provider runtimes.
- Filesystem artifacts remain canonical; SQLite is a freshness-checked,
  rebuildable retrieval projection and is never exposed as an MCP storage API.
- Local-model output is advisory, non-canonical, and unable to approve or execute
  a plan. Correctness-critical behavior has a deterministic model-free path.
- The initial Ollama boundary is loopback-only, explicitly enabled, bounded by
  timeout and input limits, and restricted to configured exact model tags.
- The initial MCP server uses standard input/output, exposes only read-only
  application use cases, opens no listening socket, and grants no approval or
  mutation authority.
- First-party catalog growth remains incremental, offline, exact-versioned, and
  independently reviewed; this design slice adds no catalog package implicitly.
- Dependency selection, public schema implementation, SQLite migration, network
  access, model execution, MCP execution, catalog content, Git operations, and
  release publication require separate approval.

## 22. `v0.3.0C1` unified-initialization design acceptance criteria

- `engkit init plan` and `engkit init apply` require one explicit `--target`;
  composition also requires one exact `--profile ID@VERSION` and validates
  repeatable typed `--set` values through the existing planner.
- Classification deterministically distinguishes absent, empty, uninitialized,
  initialized, invalid, linked, unreadable, and partial-state targets without
  writing, repairing, cleaning, or inferring a profile.
- The facade selects exactly one existing lifecycle: workspace initialization,
  new-project composition, or existing-workspace composition. An absent target
  without a profile, an initialized target without a profile, and a non-empty
  uninitialized target with a profile are refused.
- Partial or invalid `.engkit` state and unsafe filesystem targets fail closed
  and preserve evidence for doctor or specialized recovery.
- Planning emits the selected existing plan unchanged, including its schema and
  plan ID. Apply repeats classification, rebuilds that plan from current state,
  and requires its exact approval.
- The facade owns no plan schema, journal, checkpoint, lock, staging directory,
  backup, ownership record, or recovery command; specialized commands remain
  supported and authoritative.
- Planning and help are read-only and non-interactive. The facade does not use
  SQLite, Ollama, MCP, provider runtimes, credentials, network access, Spec Kit,
  Git, or release services.
- Design acceptance does not authorize implementation. Exact code changes,
  tests, risks, rollback, Git operations, and publication require separate
  approvals.

