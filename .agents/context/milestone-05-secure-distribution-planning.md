# Milestone 05 Context: Secure Distribution Planning

## Original planning scope

- Plan release targets, TUF client boundaries, roles, thresholds, expiry,
  offline-root handling, rotation, recovery, and future CI publication flow.
- Compare dependencies without changing `go.mod` or `go.sum`.
- Keep implementation, keys, signed metadata, CI, installation, publication,
  commit, and push outside the approved scope.

## Material findings

- The candidate release matrix is Windows, Linux, and macOS on `amd64` and
  `arm64`, using raw executables as updater targets.
- The first implementation slice should end at verified cache download.
  Cross-platform installation and Windows self-replacement remain separate.
- The bootstrap root belongs in the application binary, not writable cache or
  project `.engkit` state.
- A single TUF repository without delegation or TAP 4 minimizes the initial
  trust and path-handling surface.
- Root and targets use proposed two-of-three thresholds. Snapshot and timestamp
  are separate one-of-one online roles with short validity windows.
- Official `go-tuf/v2` is the accepted client dependency, pinned to `v2.4.2`
  behind the internal adapter boundary.
- TUF protects repository metadata and target authenticity but does not make a
  compromised, correctly signed build safe.

## Security observation

The go-tuf TAP 4 multi-repository client had a path-traversal advisory affecting
versions through `v2.4.0`, fixed after that release. The initial design excludes
TAP 4 regardless of the version selected. Version `v2.4.2` additionally fixes
empty target hashes and duplicate-public-key threshold counting.

## Remaining approvals after Stage 5A

- Bootstrap-root bytes and production key ceremony.
- Production repository descriptor and public CLI activation.
- CI, signing, target hosting, metadata publication, and incident automation.
- Platform installers, code signing, notarization, and self-replacement.

## Milestone 5A packet findings

- The isolated dependency rehearsal retained SQLite v0.35.4 and proposed
  go-tuf/v2 v2.4.2 plus x/mod v0.30.0.
- The Go module build list increased from 16 to 93 modules. Twelve new external
  modules were present in the compiled dependency closure.
- govulncheck v1.9.0 found five reachable standard-library vulnerabilities in
  Go 1.26.4, all reported fixed in Go 1.26.9.
- The approved module policy update to Go 1.26.9 cleared the toolchain gate on
  2026-10-16. Repository tests, vet, and govulncheck passed.
- A fresh isolated go-tuf v2.4.2 and x/mod v0.30.0 rehearsal found zero
  reachable and zero imported-package vulnerabilities. Five required-module
  findings had no called vulnerable symbols and remain review inputs.
- The dependency graph, ADR-0005, exact file set, and stop boundary were
  explicitly accepted on 2026-10-16.
- Stage 5A implemented strict policy, platform state paths, bounded HTTPS
  transport, go-tuf refresh and lookup, last-known-good metadata preservation,
  and atomic non-executable content-addressed target publication.
- Tests, vet, vulnerability analysis, and six-platform cross-compilation
  passed. Race instrumentation remains pending because the installed Windows
  GCC failed before compilation with an illegal system-DLL relocation.
- The exact dependency, file, transport, JSON-contract, test, and rollback
  proposal is recorded in docs/milestone-05a-implementation-packet.md.
