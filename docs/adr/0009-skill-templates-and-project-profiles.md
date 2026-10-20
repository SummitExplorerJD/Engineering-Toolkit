# ADR 0009: Skill templates and explicit project profiles

## Status

Accepted for `v0.2.0E` design. Implementation requires separately approved
slices.

## Context

The canonical skill lifecycle can validate and install a complete local skill
package, but it does not author that package or decide which skills suit a
project. Treating one example skill as a universal template would conflate
authoring source, installed state, and project policy. Inferring a project type
and immediately generating files would also bypass the established preview and
approval boundary.

Engineering Toolkit must support two related audiences: catalog authors need a
safe starting structure for a new template, while engineers need a profile that
composes the appropriate skills and workspace assets for an explicitly selected
project type.

## Decision

Engineering Toolkit will use four distinct layers:

1. A **skill template** is versioned provider-neutral catalog source.
2. A **project profile** selects exact template versions and composition policy.
3. A **rendered canonical skill** is normal `engkit.skill/v1alpha1` workspace
   state managed by the existing lifecycle.
4. A **provider rendering** is downstream derived output and never canonical.

The initial first-party catalog will be reviewed repository content packaged
with the CLI, identified by the CLI/catalog version, and usable without network
access. Remote catalogs, arbitrary repository imports, and marketplaces are not
authorized by this decision.

Template rendering will support only declared typed variables and a bounded
placeholder syntax. Unknown variables, missing required values, invalid values,
undeclared output paths, unresolved placeholders, and output collisions fail
closed. Rendering will not provide loops, conditionals, functions, includes
outside the template, environment expansion, process execution, network access,
or credential access in the first contract. Generated resource hashes are
computed from rendered bytes and the canonical manifest is emitted last.

Profiles will be selected explicitly by identifier and version. Read-only
project inspection may recommend compatible profiles and explain its evidence,
but a recommendation never becomes selection or approval. Profiles may compose
skill templates, conventions, workflows, source-layout policy, and optional
integration requests. Each category retains its own ownership and validation
rules.

New-project creation and existing-workspace composition will remain distinct
user-facing operations. They share a deterministic plan contract that binds the
selected profile and catalog versions, template versions, resolved variables,
observed workspace facts, rendered digests, ownership, conflicts, operations,
and verification. No output is written before approval of the complete plan.

Exact schema fields, command names, catalog layout, and transaction mechanics
will be proposed and approved in later implementation slices. Candidate command
families such as `skills scaffold`, project creation, and workspace composition
are illustrative until their public contracts are accepted.

## Consequences

- The CLI can eventually generate different skill sets for Go, web, Python, or
  other explicitly selected project profiles without embedding provider policy
  in canonical skills.
- Catalog sources do not appear as installed skills and cannot collide with
  `.agents/skills/` discovery.
- A profile and all of its inputs can be reviewed and reproduced without a
  network dependency.
- Template expressiveness is intentionally limited to make output deterministic
  and keep templates non-executable.
- The catalog ships on the CLI release cadence until a separately trusted remote
  distribution design exists.
- Provider synchronization is postponed until canonical profile composition is
  implemented and verified.

## Rejected alternatives

- **Use an installed skill as the universal template:** rejected because it
  conflates catalog source with workspace state and cannot express profile
  selection cleanly.
- **Automatically choose and apply a profile:** rejected because detection can
  be ambiguous and does not constitute approval.
- **Adopt a general-purpose template engine:** rejected because functions,
  includes, and ambient inputs weaken determinism and expand the execution
  boundary.
- **Store first-party templates only on a remote service:** rejected because
  initial composition must remain offline, reproducible, and independent of
  credentials or service availability.
- **Generate provider-native skills directly:** rejected because it would make
  one provider format canonical and bypass the established canonical package.
