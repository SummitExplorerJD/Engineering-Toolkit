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

For explicitly selected real workspaces, contextual indexing may include
bounded textual source files beneath configured workspace-relative source
roots. Source selection, exclusions, byte limits, chunk boundaries, language
hints, line ranges, and content digests shall be deterministic and recorded in
the rebuildable projection. The index shall exclude linked files, binary or
oversized content, dependency and generated-output trees, repository metadata,
and known secret-bearing paths by default.

Retrieval shall optimize for minimum useful context rather than maximum result
volume. Search returns compact ranked snippets first; exact indexed excerpts
require a separate bounded read. Callers shall be able to constrain path,
artifact kind, result count, and total response bytes.

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

The initial adapter shall use only the fixed loopback Ollama endpoint
`http://127.0.0.1:11434` and an allowlist of exact model tags. Workspace files,
environment variables, and command-line arguments shall not override that
endpoint. The adapter shall not install or start Ollama, pull models, discover
remote endpoints, access credentials, invoke tools, or fall back to a remote
model. A deterministic non-model path shall remain available for every
correctness-critical operation.

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

A later read-only MCP expansion may expose context status, indexed-source
inventory, filtered search, indexed excerpt reads, and a bounded workspace tree
through application-owned use cases. It shall use staged retrieval and compact
defaults to reduce model-context consumption. It shall not expose a database
path or SQL interface, rebuild the index, read arbitrary unindexed files, invoke
a model, or broaden the authority of the launching harness.

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

Context operations shall enforce deterministic byte and item budgets at index,
query, and protocol boundaries. Default MCP results shall favor a small number
of compact summaries and snippets, and shall require an explicit follow-up for
larger excerpts. Validation shall measure indexed files and bytes, query result
bytes, and approximate context reduction against a bounded full-file baseline;
it shall not claim exact model-token counts without a pinned tokenizer.

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

## 23. `v0.3.0D1` MCP contract and dependency acceptance criteria

- The only initial command is `engkit mcp serve --transport stdio --root PATH`;
  both flags are explicit, one process is bound to one validated root, and no
  listening socket or network request is permitted.
- Standard output is reserved exclusively for newline-delimited protocol
  messages. Bounded safe diagnostics use standard error and disclose neither
  request content, secrets, nor paths outside the bound root.
- The fixed initial inventory contains only `engkit_doctor`,
  `workspace_inspect`, `context_search`, `catalog_inspect`,
  `profiles_recommend`, `skills_list`, `skills_validate`, and `plan_inspect`.
- Tools delegate to application-owned read-only use cases and reuse their
  versioned results. MCP handlers do not open SQLite, transaction stores,
  provider projections, or CLI subprocesses directly.
- Tool schemas reject unknown properties and bound strings, paths, arrays, plan
  files, context results, diagnostics, and complete responses. Linked or
  escaping paths, stale indexes, unknown schemas, and invalid plan identities
  fail closed.
- The server advertises tools only. Logging, resources, prompts, subscriptions,
  completion, elicitation, sampling, server actions, mutation, approval,
  recovery, model invocation, and provider-runtime invocation remain disabled.
- The selected dependency is the official
  `github.com/modelcontextprotocol/go-sdk` pinned to `v1.8.0`. Adding it and
  accepting its reviewed module-graph delta require separate `v0.3.0D2`
  approval.
- The design adds no executable code, dependency, persistence, migration,
  harness configuration, Git operation, or release action.

## 24. `v0.3.0E1` local-intelligence contract acceptance criteria

- The initial operation may rank only profile candidates already found eligible
  by deterministic policy; model output cannot add a candidate, alter
  compatibility, create a plan, select a profile, approve, or execute.
- The Ollama endpoint is fixed at `http://127.0.0.1:11434`. The client disables
  proxies and redirects, sends no credentials, follows no endpoint override,
  performs no retry, and never installs, starts, pulls, mutates, or silently
  substitutes a model.
- Local intelligence is disabled by default. The initial allowlist contains
  only `phi4-mini:latest`, `qwen3.5:0.8b`, and `granite4:3b`; no model is an
  implicit default, and the observed local digest is result evidence.
- The first model-backed operation requires a fresh SQLite projection and uses
  no implicit canonical-file fallback. Missing, stale, unsupported, malformed,
  oversized, cancelled, or timed-out model behavior preserves deterministic
  profile ordering.
- Goals, excerpts, requests, responses, deadlines, candidate counts, and result
  strings have exact bounds owned by the local-intelligence contract. Secret
  patterns, ignored state, credentials, environment values, private keys,
  transaction data, and unapproved paths are excluded before prompt assembly.
- Requests use a versioned prompt, no tools, no images, no thinking, no
  streaming, temperature zero, immediate unload, and strict JSON Schema output.
  Unknown fields and candidate identifiers fail closed.
- `engkit.advice-result/v1alpha1` stores digests rather than raw goals, excerpts,
  prompts, or model responses. It contains no approval field and is never an
  executor input.
- E1 inventory may establish executable, daemon, tag, size, and digest facts but
  does not establish model compatibility. Each exact model digest requires a
  separately approved synthetic smoke test before a compatibility claim.
- E1 adds no executable code, dependency, public schema implementation,
  configuration change, SQLite migration, model invocation, Git operation, or
  release action.

## 25. `v0.3.1A` post-release validation planning acceptance criteria

- `v0.3.1` is a compatibility-preserving maintenance line. It may correct only
  defects reproduced against published `v0.3.0`; it does not add commands,
  schemas, providers, models, profiles, persistence formats, or distribution
  authorities.
- Validation begins from immutable tag `v0.3.0`, exact release commit
  `2bd28273e072b2a33647fb5d7ba09968e381f27a`, and the 12 published uploaded
  assets. GitHub-generated source archives and attestations are not counted as
  uploaded release assets.
- A fresh Windows AMD64 installation uses the public release in an isolated,
  disposable user-writable directory without administrator rights or PATH
  mutation. It verifies downloaded hashes, exact reported version, embedded
  catalog identity, and cleanup without replacing the existing user
  installation.
- Installed-binary acceptance uses synthetic workspaces only. It covers doctor,
  catalog and profile inspection, unified-init planning and refusal, disabled
  and unavailable local-advice fallback, and bounded MCP standard-input/output
  behavior without real repository content.
- Installer validation covers fresh installation, explicit upgrade, bounded
  failure injection, rollback to exact prior bytes, and removal of transaction
  directories. Destructive cleanup is limited to the exact disposable root and
  requires separate execution authority.
- Validation records exact commands, exit status, version, asset and executable
  digests, catalog identity, scenario results, and bounded failure diagnostics.
  It records no credentials, prompts, model responses, environment values, or
  unrelated absolute paths.
- Ollama and model-backed smokes remain outside the initial validation run. No
  model may be installed, pulled, updated, replaced, invoked, or reclassified;
  the existing exact-digest observations and compatibility claims remain
  unchanged.
- A failed scenario must be reproduced and classified before a correction is
  proposed. Any correction requires its own affected-file inventory, tests,
  risks, rollback, and approval. Passing validation produces no source change
  and does not justify an empty patch release.
- New model compatibility, allowlist changes, MCP tools, profiles, automatic
  updates, production TUF activation, signing, notarization, marketplaces, and
  package-manager distribution remain future minor-release work.
- Planning acceptance does not authorize network access, installation,
  workspace mutation, failure injection, cleanup, code changes, Git operations,
  workflow execution, tagging, or release publication.

## 26. `v0.4.0A` real-workspace context and MCP design acceptance criteria

- The primary product outcome is lower context consumption for repository
  understanding: callers inspect compact workspace facts, search bounded
  snippets, and request only the exact indexed excerpts needed for a task.
- Source indexing is opt-in through explicit workspace-relative roots. The
  design must define deterministic textual-file eligibility, built-in excluded
  paths, user exclusions, symlink refusal, per-file and aggregate limits, chunk
  boundaries, and freshness evidence before implementation.
- SQLite remains an optional, local, disposable projection. A schema change is
  handled by rebuilding into a new database and atomically replacing the old
  projection, never by treating derived rows as canonical or irreplaceable.
- The proposed read-only MCP additions are `context_status`,
  `context_sources`, enhanced `context_search`, `context_read`, and
  `workspace_tree`. Exact schemas, byte limits, pagination, error behavior, and
  equivalence with application use cases must be accepted before code changes.
- Staged retrieval is mandatory. Status and tree results contain no file
  contents; search defaults to compact snippets; `context_read` can return only
  indexed text from the same fresh projection and requires an exact path and
  bounded line range or chunk identity.
- MCP results shall not duplicate the same excerpt in structured and textual
  compatibility fields when the negotiated client contract permits a compact
  representation. C1 must test both structured-content clients and clients that
  require text compatibility before changing the existing response convention.
- Result budgets are expressed in deterministic UTF-8 bytes and item counts.
  Token savings are evaluated as an approximate ratio against a declared
  bounded baseline because model tokenizers differ and no tokenizer dependency
  is introduced by the design slice.
- Real-workspace validation begins with a local MCP test client and explicitly
  selected projects. Retained evidence contains counts, timings, digests,
  budget utilization, and sanitized outcomes, but no source text, prompts,
  credentials, unrelated paths, or environment values.
- Starting an AI harness against a real project is a separate disclosure gate:
  MCP excerpts can enter that harness's model context even when the Engineering
  Toolkit server itself performs no network request.
- The initial expansion remains read-only. MCP index rebuild, plan approval or
  execution, recovery, canonical-file mutation, provider invocation, Ollama,
  remote transport, resources, prompts, and sampling remain outside scope.
- `v0.4.0A` changes documentation only. It authorizes no Go or configuration
  change, SQLite schema change, dependency operation, real-project indexing,
  MCP execution, model call, network access, Git operation, or release action.
- Production TUF activation moves to a later independently approved release so
  update trust and real-workspace context do not share one implementation and
  risk boundary.

## 27. `v0.4.0B1` context-index schema acceptance criteria

- `engkit.config/v1alpha1` gains only an optional `context` mapping with 1 to 32
  explicit source roots and up to 128 prefix exclusions. Existing configuration
  remains valid and continues documentation-only indexing when the mapping is
  absent.
- Discovery uses a fixed textual extension and basename allowlist, mandatory
  VCS, dependency, generated-output, state, and secret path exclusions, explicit
  user prefix exclusions, linked-path refusal, valid UTF-8, and exact size and
  aggregate limits. It does not interpret globs, environment variables, Git
  ignore rules, or content as configuration.
- SQLite advances to `user_version = 2` through complete rebuild and atomic
  replacement, never an authoritative in-place migration. A v1 projection is
  unsupported for reads but remains preserved until successful v2 publication.
- Exact source bytes and normalized LF text have separate digests. Stable,
  non-overlapping chunks bind path, source digest, ordinal, inclusive line
  range, and content digest; one chunk is at most 8 KiB and 120 lines.
- Hard rebuild ceilings are 20,000 candidates, 1 MiB per eligible file,
  128 MiB total eligible source, 50,000 stored chunks, and a 512 MiB published
  database. Aggregate overflow fails before publication rather than creating a
  partial index.
- The v2 database separates metadata, sources, chunks, sanitized exclusion
  counts, and an FTS5 external-content search table. FTS5 comes from the already
  pinned SQLite module and adds no module-version change.
- Status, source inventory, filtered search, and indexed read are
  application-owned, versioned, freshness-checked, and bounded. Retrieval never
  exposes SQL, an absolute database path, excluded paths, or arbitrary unindexed
  files.
- B2 is limited to the exact source and test inventory in the index contract;
  it changes no MCP handler, dependency version, catalog, initialization output,
  provider, model, TUF behavior, Git state, workflow, or release artifact.
- B1 changes documentation only and authorizes no configuration edit, database
  operation, implementation, real-project indexing, MCP execution, model call,
  network access, Git operation, or publication.

## 28. `v0.4.0C1` compact MCP expansion acceptance criteria

- The fixed server inventory expands from eight to twelve tools by adding
  `context_status`, `context_sources`, `context_read`, and `workspace_tree` and
  enhancing the existing `context_search`. No tool rebuilds or mutates the
  index.
- Every expanded context tool accepts an optional `response_mode`. `compact` is
  the default and places the complete typed result only in
  `structuredContent`; its text content is a bounded summary with no excerpt
  duplication. `text` places the complete JSON result only in text content and
  omits `structuredContent` for compatibility clients.
- Safe tool errors use bounded text content only, set the protocol error flag,
  and expose no partial structured result, SQL, database path, credential,
  environment value, excluded path, or unrelated absolute path.
- Encoded result ceilings are 16 KiB for status, 64 KiB for source inventory,
  32 KiB for search, 40 KiB for indexed read, and 64 KiB for workspace tree.
  Application item, depth, line, and content-byte limits remain independently
  enforced before protocol encoding.
- Source inventory and workspace-tree continuation bind the exact fresh
  projection fingerprint. A rebuild invalidates earlier continuation inputs
  rather than mixing pages from different projections.
- `workspace_tree` is synthesized only from paths in the fresh indexed-source
  inventory. It does not traverse the canonical filesystem and cannot reveal
  excluded or unindexed names.
- The adapter retains schemas derived by the pinned official MCP SDK but uses a
  reviewed low-level handler because the typed helper inserts a duplicate JSON
  text fallback when structured content is otherwise sufficient. It does not
  reimplement protocol framing or negotiation.
- C2 conformance must compare application results with both protocol response
  modes, prove excerpt bytes occur in only one representation, exercise stale
  continuation, limits, truncation, cancellation, safe errors, and output
  isolation, and retain the existing tools-only stdio capability boundary.
- C2 is limited to the exact source and test inventory recorded by the C1
  contract. It adds no module version, network transport, resource, prompt,
  sampling, Ollama call, provider configuration, index rebuild tool, canonical
  mutation, or real-project execution.
- C1 changes documentation only. It authorizes no Go change, MCP execution,
  database operation, real-project indexing, model call, dependency operation,
  network access, Git operation, workflow, installation, or publication.

