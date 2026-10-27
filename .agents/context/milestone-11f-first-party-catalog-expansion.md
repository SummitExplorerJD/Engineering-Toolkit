# Milestone 11F: first-party catalog expansion

## Approved slice

The initial F slice was implemented locally on 2026-10-27. It expands the
embedded offline catalog from one to three project profiles without adding a
new template:

- `go-cli@0.1.0` for Go command-line applications;
- `go-library@0.1.0` for reusable Go libraries; and
- the existing immutable `go-service@0.1.0` pilot.

Both new profiles select the existing immutable `project-guide@0.1.0` template,
require the bounded `project_name` variable, support `new-project` and
`existing-workspace` modes, and declare the `go` ecosystem. Their source-layout
policy is `preserve`; they do not create `src/`, `cmd/`, a module, or application
source.

## Exact catalog evidence

The catalog identity is `engineering-toolkit@0.3.0` and remains compatible
with the existing `engkit.catalog/v1alpha1` contract.

- `go-cli@0.1.0` directory SHA-256:
  `sha256:85b8fb53573a69775d493a23dc04c1a571661746e8213397e4ab4e59dfe9c019`.
- `go-library@0.1.0` directory SHA-256:
  `sha256:3b37b7baf208dbf30de7625511ce0f659ae844188b421c6e5cae0a9bcfbe0953`.

Each profile contains only `profile.yaml` and four text templates:
`AGENTS.md.tmpl`, `go.md.tmpl`, `project-overview.md.tmpl`, and
`verify.md.tmpl`. The profile manifests bind every asset by SHA-256. Neither
profile declares integration requests, scripts, dependencies, network access,
credentials, provider output, or executable permissions.

## Product boundary

The CLI profile documents command entry points, separation of domain behavior
from presentation and process exits, stdout/stderr contracts, stable flags and
exit codes, and cross-platform verification. The library profile documents
exported API compatibility, package boundaries, examples, caller-owned error
handling, and avoidance of global mutable state.

Deterministic inspection marks all three profiles compatible when Go ecosystem
evidence is present. Recommendation and optional local advice may rank them but
cannot select one. Planning still requires an exact explicit profile identity
and normal approval.

## Validation

- Catalog loading verified exact identities, ordering, asset hashes, and
  directory hashes.
- Focused catalog, profile recommendation, composition-plan, and CLI tests
  passed.
- Plans for both new profiles were executable, deterministic, read-only, and
  contained no `src` directory operation.
- `go test ./...` passed.
- `go vet ./...` passed.
- CGO-disabled CLI builds passed for Windows, Linux, and macOS on AMD64 and
  ARM64.

## Preserved exclusions

The slice adds no production Go behavior, public schema change, SQLite
migration, dependency, new template, executable resource, marketplace, remote
catalog, automatic profile selection, provider synchronization, Spec Kit
mutation, Git operation, CI change, release, or publication. Every additional
catalog entry remains separately inventoried and approved.
