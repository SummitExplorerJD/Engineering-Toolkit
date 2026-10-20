# Engineering Toolkit Roadmap

Each milestone is independently approved. Completion of one milestone does not authorize the next.

## Milestone 0: Repository discovery

**Status:** Complete

- Inspect existing files and Git relationships.
- Verify the Go module and local toolchain.
- Preserve the synchronized baseline.

## Milestone 1: Documentation and governance

**Status:** Complete

- Establish requirements, architecture, ADRs, roadmap, repository instructions, and agent conventions.
- Define a safe Spec Kit adoption procedure without running initialization.
- Add no executable code or dependencies.

**Exit gate:** Documentation review, link checks, `git diff --check`, and confirmation that only the approved files changed.

## Milestone 2: CLI foundation

**Status:** Complete

- Select minimal dependencies.
- Establish `cmd/` and `internal/` module boundaries.
- Define versioned YAML, JSON, and JSONL contracts.
- Implement read-only workspace inspection and plan preview first.

**Approval inputs:** Exact file list, dependency proposal, schema drafts, tests, risks, and rollback plan.

## Milestone 3: Workspace state and recovery

**Status:** Complete for the approved scope

- Implement controlled workspace initialization.
- Add approval binding, JSONL journals, atomic checkpoints, and recovery.
- Publish `.engkit` from a same-filesystem staging directory after revalidation.

**Delivered boundary:** No SQLite, Spec Kit, provider adapter, dependency, or Git change.

## Milestone 3B: Optional contextual index

**Status:** Complete

- Select a SQLite driver and build strategy.
- Define the derived schema, rebuild process, freshness reporting, privacy defaults, and migrations.
- Prove that deleting the database cannot remove canonical project state.

**Approval inputs:** Dependency proposal, schema, privacy analysis, rebuild tests, migration policy, and cross-platform build impact.

**Delivered boundary:** A CGO-free SQLite projection indexes metadata for canonical artifacts and searchable content only for Markdown documentation. The database is replaceable and rebuildable; no canonical state depends on it.

## Milestone 4: Agent and Spec Kit adapters

**Status:** Complete

- Implement the capability model.
- Add selective Codex, Claude Code, and OpenCode adapters.
- Initialize or integrate official Spec Kit after preview and conflict review.

**Approval inputs:** Capability matrix, generated paths, ownership boundaries, upstream version, dry-run diff, and removal strategy.

**Delivered boundary:** The CLI exposes a versioned capability catalog, read-only
provider-path inspection, and deterministic Spec Kit preparation plans for Codex,
Claude Code, and OpenCode. Plans require an explicit upstream release tag and are
non-executable. Official Spec Kit `v1.2.0` is installed for Codex from the
approved inventory. Workflow execution, upgrades, removal, additional provider
generation, and external mutations remain separately gated.

### Milestone 4B: Isolated Codex inventory

**Status:** Complete

- Rehearsed official Spec Kit `v1.2.0` with the `codex` integration in an
  isolated copy on Windows.
- Recorded the exact 30-file inventory, sizes, hashes, ownership, removal
  strategy, script review, and provider risks.
- Confirmed that 44 pre-existing files remained byte-for-byte unchanged.

**Delivered boundary at completion of 4B:** The real repository still had no
`.specify` directory or generated `speckit-*` skills. This inventory became the
approval contract for Milestone 4C.

### Milestone 4C: Real Codex installation

**Status:** Complete

- Installed the approved 30-file Spec Kit `v1.2.0` Codex inventory.
- Confirmed that no pre-existing file was modified or deleted.
- Verified 27 deterministic hashes and the three timestamp-bearing structures.
- Ran `specify check`; Codex CLI was available and the diagnostic passed.
- Kept all generated workflows unexecuted.

**Delivered boundary:** `speckit-taskstoissues` is installed for upstream
manifest integrity but cannot be invoked without separate approval for GitHub
mutation. No commit, push, feature specification, constitution adoption, or
generated workflow execution was authorized.

## Milestone 5: Secure distribution

**Status:** Stage 5A internal implementation complete; activation awaiting approval

- Define release targets and packaging.
- Add TUF metadata consumption, signature verification, expiration, rollback protection, and key-rotation procedures.
- Design installation and recovery without enabling publication.

**Approval inputs:** Threat model, dependency choice, key roles, offline-root procedure, CI proposal, and compromise recovery plan.

**Delivered boundary:** The threat model, six-platform target matrix,
single-repository TUF role policy, offline-root ceremony, rotation and incident
procedures, client boundary, dependency comparison, and staged CI proposal are
documented. The internal verified-download core now implements policy, bounded
transport, TUF verification, durable metadata refresh, and atomic target-cache
publication against test repositories. No production key, trust root,
repository endpoint, public CLI command, CI workflow, installer, self-update,
publication, commit, or push was created.

**Next gate:** Define and approve Milestone 5B production activation inputs:
bootstrap root, repository descriptor, CLI contract, operational readiness,
and release-hosting controls. Installation and self-replacement remain later
independent approvals.

### Milestone 5A: Verified-download implementation packet

**Status:** Complete for the approved internal scope

- Defined the internal update and go-tuf adapter boundaries.
- Recorded exact dependency, module-graph, CLI-schema, transport-limit, path,
  test, file, and rollback proposals.
- Kept the production root, repository endpoint, public CLI activation,
  installation, CI, signing, and publication outside the slice.
- Found five reachable Go standard-library vulnerabilities in Go 1.26.4 during
  the isolated dependency rehearsal; all report fixes in Go 1.26.9.
- Updated the module policy to Go 1.26.9 and verified tests, vet, and a clean
  repository vulnerability scan.
- Repeated the isolated candidate audit with zero reachable or imported-package
  vulnerabilities; five required-module findings have no called vulnerable
  symbols and remain review inputs.
- Implemented strict stable-channel, semantic-version, platform, target-path,
  signed-custom-metadata, and no-downgrade policy.
- Implemented bounded HTTPS transport, a single-repository go-tuf adapter,
  last-known-good metadata preservation, and non-executable content-addressed
  cache publication.
- Passed unit, integration, full repository, static-analysis, vulnerability,
  and six-platform cross-compilation checks. Race instrumentation was attempted
  but the installed Windows GCC failed before compilation.

**Next gate:** Milestone 5B requires separate approval and must not infer a
production root, endpoint, CLI activation, installation, signing, CI,
publication, commit, or push from completion of this internal slice.

### Milestone 5B: Production activation planning

**Status:** Activation packet prepared; implementation blocked pending
production trust and repository inputs

- Defined the exact production input bundle for the bootstrap root, immutable
  repository descriptor, release identity, hosting controls, ownership, and
  operational readiness.
- Fixed the proposed public boundary at `engkit update check` and
  `engkit update download`; installation and execution remain unavailable.
- Defined staging, failure, root-rotation, CDN/proxy, cross-platform,
  incident-response, and rollback validation requirements.
- Prohibited runtime trust overrides from workspace files, `.engkit`, YAML,
  environment variables, command-line flags, and writable caches.
- Added no root, endpoint, key, network behavior, CLI code, dependency, CI,
  installer, signing workflow, or publication path.

**Next gate:** Supply and independently verify the canonical production
`root.json`, its SHA-256 digest and ceremony evidence, exact metadata and target
HTTPS base URLs, repository ownership, initial stable release identity, and
hosting/incident controls. Then review the final exact file diff and explicitly
approve implementation and the narrowly scoped network behavior.

## Milestone 6: Hardening and first release

**Status:** Awaiting approval

- Run cross-platform tests and security review.
- Finalize operations, support, compatibility, and release documentation.
- Separately approve commit, push, signing, and publication.

### Milestone 6A: Release-candidate preparation

**Status:** Prepared locally; execution and publication remain unapproved

- Added a local six-platform release build that injects a canonical version,
  disables CGO, trims build paths, and produces hashes and a JSON manifest.
- Added a manual, read-only GitHub Actions candidate workflow pinned to reviewed
  commits of official GitHub actions.
- Added the first-release checklist, draft `v0.1.0` notes, changelog, and
  security-reporting policy.
- Kept tag creation, GitHub settings, workflow execution, GitHub Pages,
  production TUF activation, signing, publication, commit, and push outside the
  implemented scope.

**Next gate:** Decide whether `v0.1.0` is manual-update-only or TUF-enabled,
review the complete initial commit, and separately approve commit/push and any
GitHub-side configuration. Publication remains blocked until the release
checklist is complete.

### Milestone 6B: GitHub TUF staging preparation

**Status:** Prepared locally; GitHub configuration and deployment remain
unapproved

- Added reviewed public staging metadata without private key material.
- Added a manual GitHub Pages workflow pinned to official action commit SHAs.
- Added a fail-closed assembly step that rebuilds all six executables and
  requires exact signed lengths and SHA-256 digests.
- Added a staging deployment runbook and preserved production activation as a
  separate approval boundary.
- Kept commit, push, Pages enablement, workflow execution, production root
  embedding, GitHub Release creation, and publication outside this milestone.

**Next gate:** Review the complete local diff and separately approve commit and
push. After the reviewed commit reaches GitHub, separately approve enabling
Pages and running the manual staging deployment before recording its URL and
published root digest.

## Planned post-`v0.1.0` releases

These release scopes are directional and remain subject to independent design,
implementation, Git, and publication approvals. See
[`future-capabilities.md`](future-capabilities.md) for the consolidated boundary
between implemented, planned, deferred, and open work.

### `v0.1.1`: installation safety and hardening

**Status:** Completed and published on October 19, 2026

- Add user-scoped PowerShell and POSIX installation helpers.
- Require exact versions and SHA-256 verification.
- Preserve an existing installation on failure and require explicit upgrade and
  PATH decisions.
- Test all six supported operating-system and architecture targets.
- Keep new workspace, skill, self-update, signing, package-manager, and
  production TUF behavior outside this patch release.

**Published evidence:** PowerShell and POSIX bootstrap helpers, release
packaging, additive manifest metadata, and installer transaction tests passed
on Windows, Linux, and macOS for AMD64 and ARM64. Candidate `v0.1.1-rc.3` and
the stable `v0.1.1` artifacts were independently reviewed. The immutable
`v0.1.1` GitHub Release points to commit
`0f7042fedb5571ad695de00abe8eab1ebebbe8ca`, and a fresh installation from the
public release URL completed successfully.

### `v0.2.0`: skill authoring and lifecycle

**Status:** Published on 2026-10-20; canonical design, read-only operations,
mutation planning, approved transactional local lifecycle, and template/profile
composition design completed

- Define the provider-neutral skill contract and ownership model.
- Add read-only discovery, inspection, and validation first.
- Add plan-and-approve creation, update, and removal operations.
- Record skill mutations through plans, journals, and checkpoints.
- Do not silently execute scripts, install remote skills, or select one provider
  format as canonical without an accepted decision.

Planned implementation gates:

1. **`v0.2.0A` — canonical design:** accepted in ADR 0007. Canonical packages
   use `.agents/skills/SKILL_ID/skill.yaml` plus `instructions.md`; existing
   Spec Kit and provider-native directories remain external.
2. **`v0.2.0B` — schema and read-only operations:** implemented locally. The
   complete `engkit.skill/v1alpha1` schema and deterministic discovery,
   inspection, and validation do not execute resources or mutate workspaces.
3. **`v0.2.0C` — mutation planning:** implemented locally. Immutable create,
   update, and remove plans bind exact package state without applying changes.
4. **`v0.2.0D` — approved local lifecycle:** apply approved mutations with
   ownership checks, journals, checkpoints, verification, and recovery. The
   transactional execution contract is accepted in ADR 0008 and implemented
   locally with failure-injection coverage.
5. **`v0.2.0E` — template and profile composition design:** accepted in
   [ADR 0009](adr/0009-skill-templates-and-project-profiles.md). Templates,
   explicit project profiles, rendered canonical skills, and provider output
   have separate ownership. Exact schemas and implementation remain separately
   approved.

**Published evidence:** The immutable `v0.2.0` tag points to release commit
`4a26c9a`. The six-platform release workflow completed, the GitHub Release was
published with its manifest and checksums, and an existing Windows `v0.1.1`
installation upgraded successfully and reported `engkit v0.2.0`.

### `v0.2.1`: template authoring and project-profile composition

**Status:** Planned

- Define versioned skill-template, project-profile, catalog, and composition-plan
  schemas.
- Add an authoring scaffold that writes only to an explicit absent output
  directory, plus a deterministic offline renderer.
- Recommend compatible profiles from read-only evidence while requiring
  explicit profile selection.
- Plan and apply new-project creation or existing-workspace composition through
  approval, journals, checkpoints, verification, and recovery.
- Ship an independently approved first-party template set and pilot profile.

### `v0.2.2`: selective provider synchronization

**Status:** Planned

- Render approved canonical skills selectively for Codex, Claude Code, and
  OpenCode.
- Preview conflicts and report unsupported or lossy mappings.
- Preserve canonical data and provider-owned files outside the approved scope.
- Keep official Spec Kit interoperation explicit and independently owned.

