# Project profile `v1alpha1`

## 1. Purpose

`engkit.project-profile/v1alpha1` describes an explicitly selected,
provider-neutral project composition. A profile is catalog source, not installed
workspace state, and cannot grant permissions or authorize mutation.

## 2. Profile fields

Unknown fields, aliases, multiple YAML documents, duplicate keys, and profile
files larger than 1 MiB are rejected.

| Field | Contract |
| --- | --- |
| `schema_version` | Exactly `engkit.project-profile/v1alpha1`. |
| `id` | Lowercase hyphenated identifier matching its catalog entry. |
| `version` | Exact Semantic Version matching its catalog entry. |
| `description` | Non-empty normalized text. |
| `compatibility` | Declarative ecosystem and workspace constraints. |
| `variables` | Unique typed project inputs sorted by name. |
| `skills` | Exact template selections and variable bindings. |
| `workspace_directories` | Optional fixed portable directories to create. |
| `workspace_assets` | Optional fixed-path conventions and workflow resources. |
| `source_layout` | Explicit source-layout policy. |
| `integration_requests` | Optional capability requests for later adapters. |

Profile variables use the same types, constraints, canonical text forms, and
validation rules as template variables. A skill selection contains an exact
`template_id`, exact `template_version`, and bindings for every required
template variable. A binding is either a literal typed value or a reference to
one declared profile variable; it cannot evaluate an expression.

Workspace directories are unique fixed paths. The initial project-workspace
surface may declare `.agents/context`, `.agents/conventions`, `.agents/skills`,
and `.agents/workflows`; source-layout directories are derived separately from
`source_layout` compatibility evidence.

Each workspace asset has a regular `source` below the profile's `files/`
directory, a fixed portable `destination`, a `kind` of `agent-instructions`,
`context`, `convention`, or `workflow`, a `media_type`, and a `source_sha256`.
`agent-instructions` targets the root `AGENTS.md`; the other kinds remain below
their matching `.agents/` directories. Assets follow the same single-pass text
substitution rules as templates. They cannot target canonical skill package
files, provider-owned directories, `.engkit/` execution state, or existing
external Spec Kit content.

`source_layout.policy` is one of `preserve`, `ecosystem-default`, or `src`.
The plan must record compatibility evidence and refuse `src` when the detected
ecosystem cannot support it safely. An integration request names a provider and
capability but remains non-executable metadata for the later provider phase.

## 3. Inspection and selection

```text
engkit profiles list
engkit profiles inspect --id PROFILE_ID --version VERSION
engkit profiles recommend [--root PATH]
```

All three commands are read-only and emit deterministic JSON. Recommendations
contain observed evidence, compatibility results, and reasons. They never
select a profile, write a workspace, initialize Spec Kit, render provider
files, or become approval. Subsequent planning requires an explicit
`PROFILE_ID@VERSION` selection.

`profiles list` emits `engkit.profile-list/v1alpha1`. `profiles inspect` emits
`engkit.profile-inspection/v1alpha1` with the validated profile and its exact
catalog directory digest. `profiles recommend` emits
`engkit.profile-recommendation/v1alpha1`, including the inspected root,
catalog identity, observed evidence, compatibility result, and reasons for
every packaged profile. It is implemented in `v0.2.1C` and does not rank,
select, or approve a profile.

## 4. Ownership boundary

Rendered canonical skills are owned by their canonical manifests and existing
skill lifecycle. Profile-owned `AGENTS.md`, context, convention, and workflow
assets are tracked separately by the composition plan. Existing unowned files,
provider-native content, and Spec Kit content remain external and unchanged
unless a later independently approved contract explicitly owns those paths.

Strict reading, catalog-backed inspection, and evidence-based recommendation
are implemented. First-party profile content remains a separately approved
milestone.
