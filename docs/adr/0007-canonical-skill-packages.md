# ADR 0007: Canonical provider-neutral skill packages

## Status

Accepted and implemented for the `v0.2.0` canonical package and mutation
planning slices. Transactional mutation execution is specified by ADR 0008.

## Context

Engineering Toolkit must manage reusable skills without making Codex, Claude
Code, OpenCode, or Spec Kit the owner of canonical workspace state. Provider
formats use different discovery paths and contracts, and this repository
already contains official Spec Kit Codex-facing skills under `.agents/skills/`.
Treating every directory in that location as Engineering Toolkit-owned would
create an overwrite and upgrade hazard.

Skill metadata and long-form instructions also have different editing and
validation needs. A single provider-shaped `SKILL.md` would couple canonical
state to whichever adapter was implemented first. Storing canonical skills
under `.engkit` would mix portable authored content with operational plans,
journals, checkpoints, indexes, and caches.

## Decision

Use a provider-neutral package rooted at:

```text
.agents/skills/SKILL_ID/
  skill.yaml
  instructions.md
  references/
  scripts/
  templates/
  assets/
```

`skill.yaml` is the opt-in ownership marker and declarative manifest. Its
initial schema identifier is `engkit.skill/v1alpha1`. The directory name and
manifest identity must match a normalized, lowercase, hyphenated identifier.
The manifest declares version, description, capabilities, inputs, outputs,
requested permissions, compatibility, dependencies, ownership, provenance,
resource inventory, and content-integrity data. `instructions.md` owns the
provider-neutral instructional prose.

Only a directory with a valid `skill.yaml` using an Engineering Toolkit schema
is a canonical managed package. A directory that contains `SKILL.md` or another
provider artifact but no canonical manifest is provider-native or externally
owned. Discovery may report it, but Engineering Toolkit must not adopt, edit,
remove, or render over it implicitly. Official Spec Kit assets remain owned by
Spec Kit unless a future explicit import plan is approved.

Provider adapters may later render derived files from a canonical package, but
those files are not canonical state. Generated outputs require recorded
ownership and content digests, conflict detection, and an approved plan before
replacement or removal. Unsupported or lossy mappings fail closed and are
reported.

Read-only discovery, inspection, and validation are the first implementation
slice. These operations do not execute scripts, resolve remote dependencies,
access credentials, contact networks, or mutate the workspace. Skill scripts
and other executable resources are data during `v0.2.0`; declaring a permission
is a request for future policy evaluation, not an execution grant.

Validation must reject ambiguous identities, duplicate canonical identities,
absolute or escaping resource paths, undeclared managed resources, invalid
schema versions, and integrity mismatches. Symlinks that resolve outside the
package or workspace boundary are invalid. Diagnostics must distinguish
canonical, provider-native, generated, conflicting, and invalid packages.

Creation, update, and removal remain downstream slices of `v0.2.0`. Each
mutation uses an immutable JSON plan, explicit approval bound to that plan,
guarded execution, a JSONL journal, and an atomic JSON checkpoint. Import,
export, provider synchronization, remote installation, and script execution
are not authorized by this decision.

## Consequences

- Canonical authored content remains portable and reviewable under `.agents`.
- The explicit manifest prevents existing Spec Kit or provider-native skills
  from being mistaken for Engineering Toolkit-owned state.
- YAML carries structured intent while Markdown remains comfortable for long
  instructions.
- Provider adapters can evolve without changing the canonical package format.
- The `v1alpha1` schema, validators, canonical serialization, and migration
  rules must be specified and tested before mutation commands are implemented.
- Co-locating canonical packages and Codex-facing outputs requires strict
  per-file ownership and conflict checks in the later synchronization release.

## Rejected alternatives

- **Canonical `SKILL.md` with frontmatter:** rejected because it privileges the
  format already used by Codex and Claude-oriented tooling and couples metadata
  evolution to provider-facing prose.
- **Canonical packages under `.engkit/skills`:** rejected because `.engkit`
  owns operational state, while portable authored agent assets belong under
  `.agents`.
- **Treat every `.agents/skills` directory as managed:** rejected because it
  would capture existing Spec Kit and user-owned provider assets implicitly.
- **Automatically import provider-native skills:** rejected because conversion
  may be lossy and changes ownership without an approved plan.
- **Execute declared scripts during validation:** rejected because validation
  must remain deterministic, read-only, and safe for untrusted workspaces.
