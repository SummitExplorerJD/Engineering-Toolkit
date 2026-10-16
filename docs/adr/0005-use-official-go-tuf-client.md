# ADR-0005: Use the Official go-tuf v2 Client Behind an Internal Boundary

- **Status:** Accepted
- **Date:** 2026-10-15

## Context

ADR-0004 selects TUF as the distribution trust model but leaves the client
implementation open. Implementing the TUF verification state machine locally
would make Engineering Toolkit responsible for subtle signature, version,
expiration, delegation, and consistent-snapshot behavior. Running the Python
reference implementation as a subprocess would weaken the single-binary and
cross-platform goals.

The official go-tuf project provides a native Go client API and supports the
TUF metadata and cryptographic functions needed by the planned client. Its
legacy pre-v2 implementation is deprecated by upstream. Recent fixes also show
why the dependency must be pinned, audited, and kept behind a narrow boundary.

An isolated dependency rehearsal found that adding the updater and strict
SemVer comparison increases the Go module build list from 16 to 93 modules,
with 12 new external modules in the compiled dependency closure. This cost is
material and must be accepted explicitly. The same rehearsal found reachable
standard-library vulnerabilities in Go 1.26.4 that are fixed in Go 1.26.9.
The module policy was updated to Go 1.26.9 on 2026-10-16; a repeated isolated
scan found no reachable or imported-package vulnerabilities in the proposed
client dependency closure.

## Decision

Use `github.com/theupdateframework/go-tuf/v2` at pinned version `v2.4.2`.
Wrap it in an infrastructure adapter under `internal/update`; keep
release selection, channel policy, semantic-version rules, local paths, and
installation orchestration in Engineering Toolkit code.

Use `golang.org/x/mod/semver` at an independently pinned and reviewed version
for release comparison. Do not implement the client with a Go toolchain that
has reachable standard-library vulnerabilities.

Use only the single-repository updater API in the initial implementation. Do
not enable TAP 4 multi-repository support, repository-authoring APIs, signing,
key generation, or publication in the CLI runtime.

## Consequences

- TUF verification behavior is based on the maintained official Go project.
- Core application policy remains testable with fake metadata and transport
  ports without importing go-tuf.
- Dependency and advisory review becomes part of every client upgrade.
- The application must provide bounded transport, trusted bootstrap-root
  storage, durable metadata state, platform policy, and actionable errors.
- Acceptance of this ADR will not authorize key generation, signing,
  publication, installation, or self-update.

## Alternatives considered

- **Custom TUF client:** rejected because protocol and cryptographic edge cases
  create disproportionate security risk.
- **Legacy go-tuf:** rejected because upstream deprecates it in favor of v2.
- **python-tuf subprocess:** rejected because it introduces an external runtime
  and packaging boundary into a single-binary Go CLI.
- **Checksums or a generic signature library:** rejected by ADR-0004 because
  they do not implement the required repository, freshness, threshold, and
  rotation model.

## Approval condition

The exact dependency diff, API boundary, vulnerability review, conformance
plan, and removal path were approved on 2026-10-16 through
`docs/milestone-05a-implementation-packet.md`. Production trust roots,
endpoints, CLI activation, installation, signing, and publication remain
separate decisions.

