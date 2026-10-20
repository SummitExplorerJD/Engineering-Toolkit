# ADR 0010: Template and profile `v1alpha1` contracts

## Status

Accepted for `v0.2.1A` contract design. Runtime implementation remains subject
to separately approved milestones.

## Context

[ADR 0009](0009-skill-templates-and-project-profiles.md) separates catalog
templates, project profiles, rendered canonical skills, and provider output.
Implementation now needs exact experimental schema identifiers, storage rules,
rendering semantics, and CLI boundaries without making catalog content or
project detection an implicit authorization source.

## Decision

The first contract set uses these explicit identifiers:

- `engkit.catalog/v1alpha1` for a build-time inventory packaged with one CLI
  release;
- `engkit.skill-template/v1alpha1` for provider-neutral skill authoring source;
- `engkit.project-profile/v1alpha1` for explicit project composition policy;
  and
- `engkit.composition-plan/v1alpha1` for deterministic project creation and
  existing-workspace previews.

All human-authored contracts are strict YAML. Composition plans are canonical
JSON. Unknown fields, YAML aliases, multiple documents, duplicate keys,
unsupported schema versions, and invalid portable paths fail closed.

The packaged catalog is immutable at runtime and usable without network or
credential access. Catalog entries bind exact relative paths and SHA-256
directory digests. A template and every profile select exact versions; no
floating version ranges or implicit latest-version resolution are allowed.

Template substitution uses only `{{engkit.NAME}}`, where `NAME` is a declared
lowercase snake-case variable. Substitution is a single UTF-8 byte pass over
declared text resources. It is not allowed in paths or binary resources and
does not recursively evaluate inserted values. There are no loops,
conditionals, functions, includes, environment expansion, process execution,
network access, or credential access.

Read-only commands may inventory catalog content and recommend compatible
profiles with evidence. A user must still select an exact profile version.
Project creation and existing-workspace composition produce the same plan
schema with different kinds and preconditions. Neither planning operation
writes files. Later execution must rebuild the exact plan and require its full
`plan_id` before mutation.

The accepted command families are documented in the contract specifications.
Every command family and subcommand must accept `help`, `-h`, or `--help`
without treating the option as an unknown subcommand. Implementing this help
behavior is part of a later CLI milestone.

## Consequences

- Catalog, template, profile, and plan identities are reproducible and explicit.
- The initial implementation can be fully offline and provider-neutral.
- Project inspection remains advisory and cannot silently choose a profile.
- Text-only, single-pass substitution limits expressiveness but substantially
  reduces ambient input and execution risk.
- Profile assets can describe conventions, workflows, source-layout policy,
  and integration requests, but every resulting path remains visible in the
  plan and subject to category-specific ownership rules.
- Schema and CLI implementation, transactional execution, first-party content,
  and provider synchronization remain independent approval gates.

## Rejected alternatives

- **Floating catalog or template versions:** rejected because identical input
  commands could resolve to different content.
- **Path interpolation:** rejected because generated paths would be harder to
  bound and review safely across platforms.
- **General-purpose template engines:** rejected because evaluation features
  expand the execution boundary and weaken determinism.
- **Automatic profile selection:** rejected because recommendation evidence is
  not user intent or approval.
- **Direct provider-native generation:** rejected because canonical skills and
  workspace artifacts must remain the provider-neutral source of truth.
