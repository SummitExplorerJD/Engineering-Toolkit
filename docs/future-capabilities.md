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

**Status:** Canonical package design, read-only lifecycle, mutation planning,
approved transactional local lifecycle, and template/profile composition design
completed.

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

Implemented execution interface:

```text
engkit skills create [--root PATH] --source PACKAGE_PATH --approve PLAN_ID
engkit skills update [--root PATH] --id SKILL_ID --source PACKAGE_PATH --approve PLAN_ID
engkit skills remove [--root PATH] --id SKILL_ID --approve PLAN_ID
engkit skills recover [--root PATH] --id SKILL_ID --approve PLAN_ID
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
- Execution will rebuild the current plan before approval, stage a complete
  package, publish with same-filesystem directory renames, and retain update or
  remove backups under `.engkit/recovery/`.
- Per-skill locks, durable plans, append-only journals, atomic checkpoints, and
  explicit digest-based recovery govern every mutation.

Remaining decisions for separately approved implementation slices include:

- Canonical serialization and migration rules beyond the implemented
  `v1alpha1` read-only contract.
- Exact skill-template, project-profile, catalog, and composition-plan schemas.
- Future runtime trust and permission UX for executable resources.
- Import, export, and migration behavior for existing provider-native skills.

No provider's current format will become the canonical model merely by being
implemented first.

### `v0.2.1`: template authoring and project-profile composition

This release is intended to generate project-appropriate canonical skills and
workspace assets from an explicitly selected, versioned profile.
[ADR 0009](adr/0009-skill-templates-and-project-profiles.md) fixes the ownership
and safety boundary; exact schemas and commands remain separate approval
decisions.

Planned capabilities:

- Create an editable skill-template draft without installing it.
- List and inspect first-party templates and profiles shipped with the CLI.
- Inspect a project read-only and recommend compatible profiles with evidence.
- Create a new project from an explicitly selected profile.
- Extend an existing initialized workspace through a reviewed composition plan.
- Choose `src/` only after ecosystem compatibility checks.
- Materialize approved `.agents` conventions, context, skills, and workflows.
- Render canonical skill packages offline with declared typed variables.
- Preview every output, conflict, ownership decision, and resolved variable.
- Apply the approved composition through journals, checkpoints, verification,
  and recovery.

Candidate interfaces, not yet public contracts, include:

```text
engkit skills scaffold --id SKILL_ID --output PATH
engkit catalog list
engkit profiles recommend [--root PATH]
engkit project plan-new --profile PROFILE --destination PATH
engkit project new --profile PROFILE --destination PATH --approve PLAN_ID
engkit workspace plan-compose --profile PROFILE [--root PATH]
engkit workspace compose --profile PROFILE --approve PLAN_ID [--root PATH]
```

The first implementation will use a reviewed, versioned catalog packaged with
the CLI. Remote catalogs, arbitrary template imports, and automatic catalog
updates remain deferred. Detection can recommend a profile but cannot select or
apply one.

The initial implementation slices are expected to be:

1. exact schemas and catalog layout;
2. read-only catalog inspection, scaffold, and deterministic rendering;
3. project inspection and composition planning;
4. approved composition execution and recovery; and
5. independently reviewed first-party templates and a pilot profile.

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

### `v0.2.2`: selective provider synchronization

This release is intended to render or synchronize approved canonical skills for
Codex, Claude Code, and OpenCode after canonical project composition exists.

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
- Preview Spec Kit initialization or upgrade conflicts before execution.
- Preserve clear ownership boundaries between Engineering Toolkit, providers,
  users, and official Spec Kit assets.

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
- [ADR 0009](adr/0009-skill-templates-and-project-profiles.md) for the template
  and profile composition boundary, followed by an update or superseding
  decision if an exact public interface changes it.
- Versioned schema proposals with migration and compatibility policy.
- Exact path inventories and conflict behavior.
- Security analysis for executable resources and remote provenance.
- Tests, rollback strategy, and cross-platform validation plan.

