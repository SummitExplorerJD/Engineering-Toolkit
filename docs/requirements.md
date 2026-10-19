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
shall be separate operations. Installation shall require its own immutable plan
and approval and shall never be implied by a metadata refresh.

### FR-013: Extensibility

The system shall allow new workspace capabilities and agent adapters to be added through internal interfaces without splitting the initial product into separately deployed services.

### FR-014: Skill lifecycle

The system shall support a provider-neutral, versioned skill model with explicit
identity, capabilities, inputs, outputs, permissions, compatibility, ownership,
and provenance. Skill creation, update, removal, and provider synchronization
shall be previewable, deterministic, and separately approved before mutation.
Unsupported or lossy provider mappings shall be reported rather than silently
discarded.

### FR-015: Project creation and composition

The system shall distinguish initializing an existing workspace from creating a
new project. Future project creation shall use an explicitly selected,
versioned profile; inspect ecosystem compatibility before choosing a source
layout; preview all generated paths; and compose workspace conventions, skills,
workflows, provider integrations, and optional Spec Kit assets without silently
replacing existing files.

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

### NFR-007: Performance

Common local inspection commands should complete interactively on typical repositories. Optional indexing shall improve discovery without being required for correctness.

### NFR-008: Compatibility

Persisted schemas and CLI machine output shall be versioned. Breaking changes require migration guidance and a separately approved compatibility decision.

### NFR-009: Maintainability

Core policy and domain behavior shall be testable without invoking provider CLIs, network services, SQLite, or terminal rendering.

### NFR-010: Observability

Errors shall distinguish user input, policy refusal, environmental failure, integration failure, and internal defects. Diagnostic output shall be actionable and safe to share.

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

