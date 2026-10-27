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

**Status:** Completed and published on 2026-10-22

- Define versioned skill-template, project-profile, catalog, and composition-plan
  schemas.
- Add an authoring scaffold that writes only to an explicit absent output
  directory, plus a deterministic offline renderer.
- Recommend compatible profiles from read-only evidence while requiring
  explicit profile selection.
- Plan and apply new-project creation or existing-workspace composition through
  approval, journals, checkpoints, verification, and recovery.
- Ship an independently approved first-party template set and pilot profile.

Planned implementation gates:

1. **`v0.2.1A` — exact contracts:** accepted in
   [ADR 0010](adr/0010-template-profile-v1alpha1-contracts.md). The catalog,
   skill-template, project-profile, and composition-plan schemas plus future CLI
   families are specified. No runtime behavior is implemented by this slice.
2. **`v0.2.1B` — strict readers and authoring scaffold:** implement offline
   catalog, template, and profile parsing, deterministic inspection, and
   bounded creation of an explicit absent template directory. Implemented
   locally with an intentionally empty embedded catalog and scoped CLI help.
3. **`v0.2.1C` — rendering and composition planning:** implement bounded
   substitution, profile recommendations, and non-executable plans for new and
   existing workspaces. Implemented locally with in-memory canonical validation,
   deterministic target inventories, and no composition mutation entry point.
4. **`v0.2.1D1` — composition lifecycle design:** accepted in
   [ADR 0011](adr/0011-transactional-project-composition.md). The additive
   execution boundary, new-project tree publication, workspace no-replace
   publication, artifacts, locks, verification, and recovery tables are exact.
   No executor is implemented by this slice.
5. **`v0.2.1D2` — approved composition lifecycle implementation:** apply
   matching plans through the accepted transaction contract and complete the
   cross-platform failure-injection gate. Implemented locally with additive
   new-project and initialized-workspace execution, durable evidence, exact
   recovery, and capability-checked no-replace publication.
6. **`v0.2.1E` — first-party pilot content:** independently review and package
   the initial templates and one pilot profile. Implemented locally with
   `project-guide@0.1.0` and `go-service@0.1.0`; the pilot produces only the
   provider-neutral engineering workspace layer and no application code.

**Published evidence:** The immutable `v0.2.1` tag points to commit
`38b7aab7b24253fee3e55ff1dbfae000dced4a4e`. Candidate `v0.2.1-rc.1` and the
stable package passed the release workflow on Windows, Linux, and macOS for
AMD64 and ARM64. All published asset digests matched the independently reviewed
manifest and `SHA256SUMS`; a local Windows upgrade reported `engkit v0.2.1` and
successfully inspected the embedded pilot catalog.

### `v0.2.2`: selective provider synchronization

**Status:** Published in `v0.2.2`; D1 conformance complete; OpenCode D2 native
discovery passed on Windows x64; Codex and Claude Code D2 remain pending

- Render approved canonical skills selectively for Codex, Claude Code, and
  OpenCode.
- Preview conflicts and report unsupported or lossy mappings.
- Preserve canonical data and provider-owned files outside the approved scope.
- Keep official Spec Kit interoperation explicit and independently owned.

Planned implementation gates:

1. **`v0.2.2A` — exact synchronization contract:** accepted in
   [ADR 0012](adr/0012-selective-provider-synchronization.md). The plan schema,
   provider targets, per-file ownership, mapping-quality rules, conflict
   behavior, and Spec Kit boundary are specified. No runtime behavior is
   implemented by this slice.
2. **`v0.2.2B` — read-only rendering and planning:** implemented. Deterministic
   in-memory rendering, target inspection, conflict reporting, and immutable
   plans are available through `engkit skills plan-sync`. No provider file or
   ownership record is written.
3. **`v0.2.2C1` — transactional synchronization design:** accepted in
   [ADR 0013](adr/0013-transactional-provider-synchronization.md). It defines
   single-skill sync and unsync, dual locking, ownership-last publication,
   durable evidence, verification, and recovery. No executor is implemented.
4. **`v0.2.2C2` — approved transactional implementation:** implemented. The
   executor, CLI routing, canonical-mutation interlock, durable evidence,
   explicit recovery, and failure-injection coverage follow the accepted C1
   contract.
5. **`v0.2.2D` — provider conformance and pilot:** the contract is accepted in
   [provider conformance and pilot validation
   `v1alpha1`](provider-conformance-v1alpha1.md). The D1 harness validates exact
   pilot bytes, transactions, preservation, and equivalence through native
   binaries on the six supported release targets without invoking provider
   runtimes. D1 passed on Windows, Linux, and macOS for AMD64 and ARM64, and the
   aggregate portable-evidence comparison passed. A separately approved
   OpenCode `1.17.13` D2 native-discovery smoke passed on Windows x64. Codex and
   Claude Code discovery still require later approval for exact versions,
   commands, authentication, network behavior, evidence, and cleanup.

**Published evidence:** Candidate `v0.2.2-rc.2` and stable `v0.2.2` passed the
release workflow on all six targets, including provider lifecycle tests and the
aggregate conformance comparison. The immutable `v0.2.2` tag points to commit
`25c14d88e9871b78d674956d370a84768db9b90e`; its 12 release assets match the
reviewed manifest and checksums. A local Windows upgrade reports
`engkit v0.2.2` and successfully inspects the embedded catalog.

**Next gate:** The
[`D2 native-discovery approval packet`](provider-native-discovery-d2.md)
records the completed Windows x64 D2-I inventory, the first environment-blocked
OpenCode attempt, and a later successful OpenCode `1.17.13` D2-S run. Native
discovery found the synchronized `project-guide` with exact metadata and
content, preserved global provider state, accepted only OpenCode's deterministic
`.opencode/.gitignore` initialization side effect, and completed transactional
cleanup. Codex `0.163.0-alpha.10.2` remains blocked pending a bounded
non-mutating discovery surface, and Claude Code is not installed. Any further
provider inventory, installation, or D2 execution requires separate approval;
the OpenCode result grants no standing provider, authentication, network,
credential, or configuration authority.

### `v0.3.0`: diagnostics, unified initialization, and local intelligence

**Status:** `v0.3.0A`, `v0.3.0C1`, `v0.3.0D1`, and `v0.3.0E1` completed;
`v0.3.0B`, `v0.3.0C2`, `v0.3.0D2`, `v0.3.0E2`, and the approved
`v0.3.0F` catalog slice are implemented locally; release qualification is not
approved

This release introduces a safe operational front door and optional local
decision assistance without changing the canonical filesystem, approval, or
transaction model.

Planned implementation gates:

1. **`v0.3.0A` — requirements and architecture:** completed. Defines doctor, unified init,
   context retrieval, advisory local models, MCP harness access, catalog growth,
   security limits, portability, and acceptance criteria. This gate changes
   documentation only.
2. **`v0.3.0B` — doctor contract and implementation:** implemented locally.
   Adds read-only probes, stable findings, machine-readable output, scoped help,
   and tests. Ollama detection is limited to executable presence in `PATH`; no
   daemon, model, repair, installation, provider-runtime, or network action is
   included.
3. **`v0.3.0C1` — unified-init contract:** completed. The accepted contract
   fixes explicit target classification, exact aliases to existing plans and
   plan IDs, ambiguous-state refusal, non-interactive CLI behavior, and
   specialized recovery authority. This gate changes no executor or schema.
4. **`v0.3.0C2` — unified-init implementation:** implemented locally. The
   read-only classifier and CLI facade reuse existing planning, approval,
   transaction, evidence, and recovery paths without a new persisted schema.
5. **`v0.3.0D1` — MCP contract and dependency decision:** completed. The
   accepted contract fixes the explicit-root standard-input/output command,
   eight read-only tools, strict bounded schemas, application ownership, error
   mapping, output isolation, conformance requirements, and the official Go MCP
   SDK pinned to `v1.8.0`. It adds no code or dependency.
6. **`v0.3.0D2` — read-only MCP implementation:** implemented locally. The
   explicit-root standard-input/output server exposes the eight approved
   application operations, advertises tools only, rejects invalid or oversized
   requests and results, and includes protocol, application-equivalence,
   cancellation, path, plan-identity, and cross-platform coverage. It adds no
   mutation, approval, recovery, provider-runtime, local-model, or network
   capability.
7. **`v0.3.0E1` — local-intelligence contract and inventory:** completed. The
   fixed loopback endpoint and all three candidate tags were inventoried without
   model invocation. The accepted contract fixes the initial profile-ranking
   task, strict structured response, privacy filters, size limits, deadline,
   model-digest evidence, and deterministic fallback. Installation and tag
   presence are not compatibility claims; model smoke execution and
   implementation remain separately gated.
8. **`v0.3.0E2` — advisory implementation:** implemented locally. The explicit
   `advise profiles` command composes deterministic eligibility, fresh bounded
   SQLite context, strict privacy filtering, an optional fixed-loopback Ollama
   adapter, and a non-authoritative versioned result. Local intelligence is
   disabled by default, malformed or unavailable model behavior preserves the
   deterministic order, and model output cannot approve or execute a plan.
   Synthetic smoke evidence on Ollama `0.40.4` supports the observed exact
   digests of `phi4-mini:latest` and `granite4:3b`; `qwen3.5:0.8b` remains an
   allowlisted candidate but did not pass the accepted structured-response
   contract and therefore receives no compatibility claim.
9. **`v0.3.0F` — first-party catalog expansion:** the approved initial slice is
   implemented locally. Catalog `engineering-toolkit@0.3.0` adds independently
   reviewed `go-cli@0.1.0` and `go-library@0.1.0` profiles, both selecting the
   existing immutable `project-guide@0.1.0` template. They preserve source
   layout and generate only provider-neutral workspace guidance, conventions,
   context, and verification assets. No application source, new template,
   executable resource, marketplace, remote package, dependency, or automatic
   selection is included. Further catalog entries require another independent
   inventory and approval.
10. **`v0.3.0RC` — release qualification:** validate Windows, Linux, and macOS
    on AMD64 and ARM64, including optional-integration absence and failure paths,
    before any candidate or stable publication.

Every gate requires its own affected-file inventory, tests, risks, rollback,
and approval. Completing the documentation gate does not authorize Go changes,
dependencies, schema publication, SQLite migrations, local network access,
model calls, MCP execution, Git operations, or release publication.

