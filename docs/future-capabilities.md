# Future Capabilities

This document records the intended product evolution after `v0.1.0`. It is a
planning aid, not a promise that the proposed commands or persisted formats
already exist. Normative behavior remains owned by `requirements.md`, accepted
ADRs, and approved implementation plans.

## 1. Status vocabulary

- **Implemented:** available in a released or verified implementation.
- **Planned:** intended for a future release, subject to design and approval.
- **Deferred:** explicitly outside the near-term release scope.
- **Open decision:** requires analysis and an accepted ADR or implementation
  approval before it becomes a contract.

## 2. Current baseline (`v0.1.1`)

### Implemented workspace initialization

`engkit workspace plan-init` inspects an existing directory and produces a
deterministic, read-only JSON plan. `engkit workspace init --approve PLAN_ID`
revalidates that plan and atomically publishes the initial `.engkit` state.

The current initializer creates:

```text
.engkit/
  .gitignore
  config.yaml
  plans/workspace-init.json
  journals/workspace-init.jsonl
  checkpoints/workspace-init.json
  context/
  cache/
```

This operation registers an existing directory as an Engineering Toolkit
workspace. It does not create an application, choose a language or framework,
generate `src/`, install skills, synchronize provider files, or execute Spec
Kit.

### Implemented supporting capabilities

- Read-only workspace and provider inspection.
- Optional rebuildable SQLite context index.
- Capability descriptions for Codex, Claude Code, and OpenCode.
- Non-executable, version-pinned Spec Kit preparation plans.
- Plans, approval binding, journals, and checkpoints for implemented mutations.
- Manual cross-platform release distribution.
- User-scoped PowerShell and POSIX bootstrap installation with exact-version,
  checksum, explicit-upgrade, PATH, and recovery controls.

## 3. Planned release sequence

The sequence below is provisional. Every release and material implementation
slice requires its own approval.

### `v0.1.1`: installation safety and hardening

**Status:** Implemented and published on October 19, 2026.

The patch release improves distribution and installation without adding a new
public workspace or skill lifecycle.

Delivered scope:

- User-scoped PowerShell installation for Windows.
- User-scoped POSIX shell installation for Linux and macOS.
- Explicit version selection; no implicit `latest` resolution initially.
- Operating-system and architecture detection.
- Mandatory SHA-256 verification against official release metadata.
- Atomic or recoverable replacement that preserves an existing installation on
  failure.
- Explicit upgrade behavior and explicit PATH modification.
- CI coverage for Windows, Linux, and macOS on AMD64 and ARM64.
- Packaging of the installation helpers with release artifacts.

Not included:

- Automatic or background updates.
- Self-replacement by the running executable.
- Production TUF activation.
- MSI, PKG, DEB, RPM, Homebrew, Winget, or other package-manager publication.
- Windows Authenticode signing or macOS signing and notarization.
- New project-generation or skill-management commands.

### `v0.2.0`: skill authoring and lifecycle

**Status:** Canonical package design, read-only lifecycle, and mutation planning
implemented locally; mutation execution has not started.

This release is intended to establish the canonical, provider-neutral skill
model and safe local lifecycle.

Planned skill concepts:

- Stable identity, version, description, and instructions.
- Declared inputs, outputs, capabilities, and permissions.
- Compatibility and dependency declarations.
- Optional `scripts/`, `references/`, `templates/`, and `assets/` resources.
- File ownership and conflict-detection rules.
- Provenance, content hashes, and update policy.
- Deterministic validation and rendering.

Implemented read-only commands:

```text
engkit skills list [--root PATH]
engkit skills inspect [--root PATH] --id SKILL_ID
engkit skills validate [--root PATH] [--id SKILL_ID]
```

Implemented non-executable planning commands:

```text
engkit skills plan-create [--root PATH] --source PACKAGE_PATH
engkit skills plan-update [--root PATH] --id SKILL_ID --source PACKAGE_PATH
engkit skills plan-remove [--root PATH] --id SKILL_ID
```

Proposed execution commands, not yet public contracts:

```text
engkit skills create --approve PLAN_ID
engkit skills update --approve PLAN_ID
engkit skills remove --approve PLAN_ID
```

Every mutation will follow the established lifecycle:

```text
request -> JSON plan -> review -> approval -> execution -> JSONL journal -> JSON checkpoint
```

Accepted design decisions:

- Canonical skills live under `.agents/skills/SKILL_ID/`.
- `skill.yaml` with schema `engkit.skill/v1alpha1` is the explicit ownership
  marker, and `instructions.md` contains provider-neutral instructions.
- Provider-native and Spec Kit directories without the canonical manifest are
  external state and remain unchanged by default.
- Read-only discovery, inspection, and validation do not execute resources,
  contact networks, access credentials, or mutate the workspace.

Remaining decisions for separately approved implementation slices include:

- Canonical serialization and migration rules beyond the implemented
  `v1alpha1` read-only contract.
- Execution-time locking, backup, journal, checkpoint, and recovery behavior.
- Future runtime trust and permission UX for executable resources.
- Import, export, and migration behavior for existing provider-native skills.

No provider's current format will become the canonical model merely by being
implemented first.

### `v0.2.1`: selective provider synchronization

This release is intended to render or synchronize approved canonical skills for
Codex, Claude Code, and OpenCode according to their declared capabilities.

Proposed commands, not yet public contracts:

```text
engkit skills plan-sync --provider PROVIDER
engkit skills sync --provider PROVIDER --approve PLAN_ID
```

Required behavior:

- Preview every generated, modified, conflicting, and removed path.
- Preserve provider-owned files that are outside the approved ownership set.
- Report unsupported capability mappings instead of claiming parity.
- Detect lossy conversion and require an explicit decision.
- Keep canonical workspace data independent from provider-specific output.

### `v0.2.2`: project, workflow, and Spec Kit composition

This release is intended to connect skills with workspace creation, reusable
workflows, and official Spec Kit interoperability.

Planned capabilities:

- Create a new project from an explicitly selected and versioned profile.
- Extend an existing initialized workspace through a reviewed plan.
- Choose `src/` only after ecosystem compatibility checks.
- Materialize the approved `.agents` structure.
- Compose skills into declarative workflows.
- Preview Spec Kit initialization or upgrade conflicts before execution.
- Preserve clear ownership boundaries between Engineering Toolkit, providers,
  users, and official Spec Kit assets.

The command name remains an open decision. Candidate interfaces include:

```text
engkit project new NAME
engkit workspace plan-init --profile PROFILE
engkit workspace init --approve PLAN_ID
```

A future managed workspace may contain:

```text
.agents/
  context/
  conventions/
  skills/
    SKILL_NAME/
      skill.yaml             # accepted canonical ownership marker
      instructions.md
      references/
      scripts/
      templates/
      assets/
  workflows/
.engkit/
  config.yaml
  plans/
  journals/
  checkpoints/
  context/
  cache/
src/                         # only when compatible with the selected ecosystem
AGENTS.md
README.md
```

The skill-package portion is defined by the `engkit.skill/v1alpha1` contract.
The wider project layout remains illustrative and is not an accepted project
creation schema.

## 4. Deferred capabilities

The following remain outside the planned `v0.2.x` core unless separately
designed and approved:

- Downloading and executing arbitrary remote skills.
- A public skill marketplace.
- Silent global skill installation.
- Automatic execution of skill scripts.
- Credential storage inside skills or workspace artifacts.
- Trust based only on a repository name, owner name, or download URL.
- Silent overwrites or lossy provider conversion.
- Automatic skill, provider, Spec Kit, or application updates.
- Hosted control-plane or multi-tenant workspace management.

## 5. Security and distribution direction

Future remote skill distribution must define provenance, signatures, permission
review, integrity verification, revocation, and recovery before activation.
Where applicable, release and skill distribution should reuse TUF principles:
trusted bootstrap material, signed metadata, expiration, rollback protection,
hash and length verification, and key rotation.

Workspace configuration must not override application trust roots or silently
authorize executable behavior. Verified download, installation, provider
synchronization, and script execution remain separate approval boundaries.

## 6. Required decision records

Before implementation, the applicable milestone should produce or update:

- Functional and non-functional requirements.
- Architecture boundaries and ownership rules.
- An ADR for the canonical skill format.
- An ADR for project creation and profile composition when the interface is
  selected.
- Versioned schema proposals with migration and compatibility policy.
- Exact path inventories and conflict behavior.
- Security analysis for executable resources and remote provenance.
- Tests, rollback strategy, and cross-platform validation plan.

