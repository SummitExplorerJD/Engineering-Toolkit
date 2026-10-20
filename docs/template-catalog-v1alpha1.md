# Template catalog contracts `v1alpha1`

## 1. Purpose and layout

The initial catalog is reviewed repository content packaged with one Engineering
Toolkit release. It is immutable at runtime, provider-neutral, and available
offline. Catalog sources are never discovered as installed skills.

```text
catalog/
  catalog.yaml
  templates/TEMPLATE_ID/VERSION/
    template.yaml
    files/
  profiles/PROFILE_ID/VERSION/
    profile.yaml
    files/
```

All paths are normalized slash-separated relative paths. Symbolic links,
special files, path traversal, case-insensitive collisions, and Windows-reserved
components are invalid.

## 2. Catalog schema

`catalog.yaml` uses `engkit.catalog/v1alpha1` and contains:

| Field | Contract |
| --- | --- |
| `schema_version` | Exactly `engkit.catalog/v1alpha1`. |
| `id` | Lowercase hyphenated catalog identifier. |
| `version` | Exact Semantic Version without a leading `v`. |
| `minimum_engkit` | Minimum exact CLI version. |
| `templates` | Unique, sorted template entries. |
| `profiles` | Unique, sorted profile entries. |

Each entry contains `id`, exact `version`, relative `path`, and a
`sha256:` directory digest. The digest covers a sorted inventory of every
regular file below that entry. The packaged catalog digest and CLI version are
reported by every catalog query and composition plan.

## 3. Skill-template schema

`template.yaml` uses `engkit.skill-template/v1alpha1`. Unknown fields, aliases,
multiple YAML documents, duplicate keys, and files larger than 1 MiB are
rejected.

| Field | Contract |
| --- | --- |
| `schema_version` | Exactly `engkit.skill-template/v1alpha1`. |
| `id` | Lowercase hyphenated template identifier matching its catalog entry. |
| `version` | Exact Semantic Version matching its catalog entry. |
| `description` | Non-empty normalized text. |
| `variables` | Unique variable declarations sorted by name. |
| `skill` | Static canonical manifest fields, excluding provenance and resources. |
| `files` | Complete source-to-destination resource inventory. |

The nested `skill` object declares `id`, `version`, `description`,
`instructions`, capabilities, inputs, outputs, permissions, compatibility,
dependencies, and ownership according to
[`engkit.skill/v1alpha1`](skill-manifest-v1alpha1.md). Rendering derives
provenance from the exact catalog and template identity and derives resource
digests from rendered bytes.

Variables use lowercase snake-case names and one of `string`, `integer`, or
`boolean`. Every declaration has `name`, `type`, `description`, and `required`.
An optional `default` must match the declared type. Strings may define
`min_length`, `max_length`, an RE2 `pattern`, or unique `allowed_values`.
Integers may define inclusive `minimum` and `maximum`. Unknown, missing
required, duplicate, or invalid values fail before rendering.

A file entry contains:

- `source`: regular file below `files/`;
- `destination`: fixed canonical-package resource path;
- `kind`: canonical resource kind;
- `media_type`: `text` or `binary`; and
- `source_sha256`: digest of the unrendered source bytes.

Text resources must be valid UTF-8 and may contain `{{engkit.NAME}}` tokens.
Substitution occurs once using the canonical text form of the resolved value:
strings are inserted verbatim, integers use base-10 without a leading plus or
zero padding, and booleans are `true` or `false`. Binary resources cannot
contain placeholders. Unresolved or undeclared tokens fail closed. Source and
destination paths never permit substitution.

The renderer writes declared resources first, hashes the rendered bytes, and
emits `skill.yaml` last. The result must pass the existing canonical skill
validator before it can enter a mutation or composition plan.

## 4. Authoring and read-only CLI contracts

```text
engkit catalog inspect
engkit templates list
engkit templates inspect --id TEMPLATE_ID --version VERSION
engkit templates scaffold --output PATH --id TEMPLATE_ID --version VERSION
```

Catalog and template queries emit deterministic JSON and do not write files.
Scaffolding is a bounded authoring operation that targets an explicit absent
directory and creates only the documented starter inventory. It refuses an
existing path and never overwrites, merges, executes, imports, or contacts a
network. Project and workspace composition remain plan-and-approve operations.

The implemented JSON result schemas are:

- `engkit.catalog/v1alpha1` for `catalog inspect`, with an additional derived
  `sha256` field covering the packaged `catalog.yaml` bytes;
- `engkit.template-list/v1alpha1` for the sorted catalog template inventory;
- `engkit.template-inspection/v1alpha1` for one validated template and its
  catalog directory digest; and
- `engkit.template-scaffold-result/v1alpha1` for the normalized output path and
  created-file inventory.

Strict reading, offline inspection, and bounded scaffold publication are
implemented in `v0.2.1B`. Template rendering remains a later milestone.
