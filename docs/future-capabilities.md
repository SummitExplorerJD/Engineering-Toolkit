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

## 2. Current baseline (`v0.1.0`)

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

## 3. Planned release sequence

The sequence below is provisional. Every release and material implementation
slice requires its own approval.

### `v0.1.1`: installation safety and hardening

The intended patch release improves distribution and installation without
adding a new public workspace or skill lifecycle.

Planned scope:

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

Proposed commands, not yet public contracts:

```text
engkit skills list
engkit skills inspect SKILL
engkit skills validate PATH
engkit skills plan-create
engkit skills create --approve PLAN_ID
engkit skills plan-update SKILL
engkit skills update SKILL --approve PLAN_ID
engkit skills plan-remove SKILL
engkit skills remove SKILL --approve PLAN_ID
```

Every mutation will follow the established lifecycle:

```text
request -> JSON plan -> review -> approval -> execution -> JSONL journal -> JSON checkpoint
```

Open decisions for this release include:

- `SKILL.md` with structured frontmatter versus `skill.yaml` plus
  `instructions.md`.
- Canonical schema identifiers and compatibility rules.
- Whether canonical skills live under `.agents/skills/` or another versioned
  workspace-owned location.
- Trust and permission UX for scripts and other executable resources.
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
      skill.yaml             # one candidate canonical format
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

This example is illustrative and must not be treated as an accepted persisted
schema.

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

