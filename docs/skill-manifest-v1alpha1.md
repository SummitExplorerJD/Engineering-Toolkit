# Canonical skill manifest `v1alpha1`

## 1. Purpose

An Engineering Toolkit canonical skill is a provider-neutral package at
`.agents/skills/SKILL_ID/`. A package is managed only when it contains a valid
`skill.yaml` whose `schema_version` is `engkit.skill/v1alpha1`. A directory
without that manifest is external state and is never adopted implicitly.

This contract is experimental. Incompatible changes may be introduced before a
stable schema, but every supported schema identifier remains explicit.

## 2. Package layout

```text
.agents/skills/release-review/
  skill.yaml
  instructions.md
  references/
  scripts/
  templates/
  assets/
```

Only `skill.yaml` is excluded from the resource inventory. Every other regular
file must be declared in `resources`. Symbolic links and special files are not
valid canonical package content. Empty optional directories are allowed.

## 3. Manifest fields

Unknown YAML fields, aliases, multiple YAML documents, and manifests larger
than 1 MiB are rejected.

| Field | Required | Contract |
| --- | --- | --- |
| `schema_version` | Yes | Exactly `engkit.skill/v1alpha1`. |
| `id` | Yes | Lowercase hyphenated identifier; must match `SKILL_ID`. |
| `version` | Yes | Exact Semantic Version without a leading `v`. |
| `description` | Yes | Non-empty normalized text. |
| `instructions` | Yes | Exactly `instructions.md` in this schema. |
| `capabilities` | No | Unique normalized capability tokens. |
| `inputs` | No | Unique typed input declarations. |
| `outputs` | No | Unique typed output declarations. |
| `permissions` | Yes | Requested capabilities only; never execution authority. |
| `compatibility.minimum_engkit` | Yes | Minimum exact Semantic Version without a leading `v`. |
| `compatibility.providers` | No | Provider names known to be renderable in a later release. |
| `dependencies` | No | Unique canonical skill IDs with exact versions. |
| `ownership.owner` | Yes | Non-empty accountable owner. |
| `ownership.maintainers` | No | Unique non-empty maintainer identifiers. |
| `provenance.source` | Yes | Source identifier recorded without resolving it. |
| `provenance.revision` | Yes | Source revision recorded without resolving it. |
| `resources` | Yes | Integrity inventory of all package files except `skill.yaml`. |

Input and output `type` values are `string`, `number`, `boolean`, `object`, and
`array`. Each value requires `name`, `type`, and `description`. Only inputs may
set `required: true`.

Permission lists are `filesystem_read`, `filesystem_write`, `network`, and
`process`. Values must be unique, non-empty, and free of surrounding
whitespace. Validation records the request but does not access any listed
resource or grant permission.

Resource fields are:

- `path`: normalized relative slash path with no traversal or backslashes.
- `kind`: `instructions`, `reference`, `script`, `template`, or `asset`.
- `sha256`: `sha256:` followed by 64 lowercase hexadecimal characters.

`instructions.md` must be declared with kind `instructions`. Other kinds are
restricted to their matching plural directory. Every declared digest is
verified against the local regular file.

## 4. Example

For an `instructions.md` containing exactly:

```markdown
# Release review

Review the proposed release and report blocking findings.
```

the corresponding minimal manifest is:

```yaml
schema_version: engkit.skill/v1alpha1
id: release-review
version: 0.1.0
description: Reviews a proposed release and reports blocking findings.
instructions: instructions.md
permissions: {}
compatibility:
  minimum_engkit: 0.2.0
ownership:
  owner: engineering
provenance:
  source: local
  revision: initial
resources:
  - path: instructions.md
    kind: instructions
    sha256: sha256:0ebc13b49a1b379fed818e73c6280277f7256198de9c72440022f35cff6678be
```

## 5. Read-only CLI behavior

```text
engkit skills list [--root PATH]
engkit skills inspect [--root PATH] --id SKILL_ID
engkit skills validate [--root PATH] [--id SKILL_ID]
```

All commands emit deterministic JSON. `list` and `inspect` classify packages as
`canonical`, `external`, `invalid`, or `conflict`. External directories are
reported but do not make validation fail. `validate` returns exit code `1`
after emitting its JSON result when an invalid or conflicting package is in the
selected scope. Usage mistakes return exit code `2`.

These commands do not write files, execute resources, contact networks, access
credentials, resolve dependencies, import provider content, or render provider
artifacts.
