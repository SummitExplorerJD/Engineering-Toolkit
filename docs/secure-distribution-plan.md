# Secure Distribution Plan

## 1. Decision boundary

This document completes the planning gate for Milestone 5. It does not approve
implementation, dependency changes, key generation, CI changes, signing,
publication, installation, or self-update.

The [Milestone 5B production activation packet](milestone-05b-activation-packet.md)
defines the next review gate for a production bootstrap root, immutable
repository descriptor, public check/download commands, hosting controls, and
operational readiness. That packet is prepared but remains blocked on concrete
production inputs; it does not activate network behavior.

The first implementation slice should stop after authenticated discovery and a
verified download into application-owned cache. Replacing the installed
executable is a later slice because Windows requires a separate process or
installer to replace a running executable safely.

## 2. Release target matrix

The candidate v1 release matrix is deliberately small and uses raw executables
as updater targets. Manual archives may be added later as separate verified
targets; they are not inputs to the updater, avoiding archive traversal and
ambiguous extraction behavior.

| OS | Architecture | TUF target leaf | Execution note |
| --- | --- | --- | --- |
| Windows | `amd64` | `engkit.exe` | Authenticode may be added as defense in depth |
| Windows | `arm64` | `engkit.exe` | Requires native smoke-test coverage |
| Linux | `amd64` | `engkit` | Installer must set mode `0755` after verification |
| Linux | `arm64` | `engkit` | Requires native smoke-test coverage |
| macOS | `amd64` | `engkit` | Code signing and notarization are separate platform controls |
| macOS | `arm64` | `engkit` | Code signing and notarization are separate platform controls |

Canonical target paths are proposed as:

```text
releases/<channel>/<version>/<os>-<arch>/engkit[.exe]
```

Only `stable` is in the initial channel allow-list. Every target carries TUF
length and SHA-256 fields plus application-readable custom metadata:

```json
{
  "schema": "engkit.release-target/v1alpha1",
  "version": "0.1.0",
  "channel": "stable",
  "os": "windows",
  "arch": "amd64",
  "format": "executable"
}
```

The client validates both the target path and custom metadata. Unknown fields
remain signed but are not granted semantics until a schema revision is
approved. Pre-release version ordering, minimum supported version, and channel
promotion rules require a public contract before implementation.

## 3. TUF repository policy

Use one repository and the four top-level TUF roles. Do not enable TAP 4
multi-repository support or target delegation in the first release. Delegation
is reserved for a future need such as independently operated release channels.
Enable consistent snapshots.

The following values are a project policy proposal, not generated metadata:

| Role | Keys | Threshold | Custody | Maximum validity |
| --- | ---: | ---: | --- | ---: |
| `root` | 3 | 2 | Offline, separate operators and devices | 365 days |
| `targets` | 3 | 2 | Release operators; offline except during an approved release | 90 days |
| `snapshot` | 1 | 1 | Online release automation | 7 days |
| `timestamp` | 1 | 1 | Online publication automation | 24 hours |

Ed25519 is the proposed initial key type because the official Go library
supports it and no FIPS requirement is currently recorded. A future FIPS
requirement must reopen this choice before keys exist.

Expiration is intentionally short for online roles and longer for offline
roles. Operators must renew metadata before half of its validity window has
elapsed. Clients fail closed on expiration; clock correction and newly signed
metadata are the recovery mechanisms, not an insecure bypass.

## 4. Offline-root ceremony

The procedure must be rehearsed with disposable keys before production:

1. Designate three root-key custodians and two release approvers. Record no
   private-key material in the repository, CI, issue tracker, or chat.
2. Generate each root key on a separate offline device using an approved tool
   and encrypted removable backup. Record public keys and independently verify
   their fingerprints through a second channel.
3. Construct unsigned root metadata on a clean ceremony workstation. Verify
   role keys, thresholds, expiration, specification version, and consistent
   snapshot policy against a printed checklist.
4. Obtain at least two root signatures on isolated devices. Transfer only the
   signed metadata and public material back to the ceremony workstation.
5. Verify all signatures and the canonical bytes with two independent
   operators. Archive the ceremony record and publish only the signed root.
6. Embed the approved bootstrap root bytes into the release build and verify
   that the binary yields the expected root digest.
7. Store each key and backup in separate controlled locations. Test recovery
   from one backup without exposing another custodian's key.

No general-purpose CI identity may hold a root or targets private key. Online
snapshot and timestamp credentials must be scoped to their roles and
repository paths.

## 5. Rotation and compromise recovery

### Planned rotation

- Rotate one root key at a time while the old root threshold remains available.
- Increment root versions sequentially. Each new root is signed by the old
  threshold and the new threshold before publication.
- Refresh targets, snapshot, and timestamp metadata after any role-key change.
- Rehearse rotation at least annually and before a root expiration window is
  at risk.

### Lost or compromised online key

Stop publication, preserve evidence, generate a replacement under a reviewed
incident plan, update the authorizing root metadata with the offline root
threshold, and publish a complete new metadata set. Treat repository
availability as secondary to preserving trust.

### Lost or compromised targets key

If fewer than two targets keys are compromised, the threshold prevents
unilateral publication. Revoke affected public keys in a new root, replace
them, review every target authorized since the suspected compromise, and
publish fresh targets, snapshot, and timestamp metadata.

### Root compromise

One compromised root key is rotated using the remaining threshold. Compromise
of two root keys exceeds the protection claim and requires out-of-band trust
recovery, including a newly distributed `engkit` trust anchor. The normal
updater must not silently recover from this event.

### Repository or CDN compromise

Freeze publication, restore known-good target bytes and metadata, rotate any
credentials exposed through the hosting incident, and publish fresh timestamp
metadata. TUF should preserve authenticity, but availability may be lost.

### Build-system compromise

TUF cannot reject a malicious artifact approved by the targets threshold.
Suspend releases, rotate exposed automation credentials, audit source and
build inputs, rebuild independently, and require a new release authorization.

## 6. Planned client boundary

The future `internal/update` package owns target policy and exposes ports for:

- a trusted-clock source;
- metadata and target transport with strict limits;
- trusted metadata persistence;
- platform detection;
- verified target staging; and
- structured diagnostics.

The TUF library adapter implements only metadata refresh and target download.
It must not select channels, compare application versions, choose filesystem
destinations, install files, invoke shells, or access workspace state.

Proposed read-only and cache-writing flow:

1. Load the compiled bootstrap root and last-known-good metadata.
2. Refresh metadata through the TUF client with bounded transport.
3. Resolve the exact allow-listed platform target.
4. Validate signed custom metadata against application policy.
5. Download into a unique application-cache temporary file.
6. Verify target length and digest through the TUF client.
7. Synchronize and atomically publish the verified cache entry.
8. Return a versioned JSON result describing the verified file and evidence.

Installation never occurs implicitly. A later installation plan must bind the
verified target digest, current executable identity, destination, platform
strategy, rollback artifact, checkpoint, and approval ID.

## 7. Dependency evaluation

| Candidate | Advantages | Risks and costs | Decision |
| --- | --- | --- | --- |
| `github.com/theupdateframework/go-tuf/v2` | Official Go project; TUF client API; consistent snapshots; role, signature, and delegation support; active security fixes | Adds a security-sensitive transitive graph; APIs and minimum Go version must be pinned and audited | **Proposed**, exact minimum `v2.4.2`, pending dependency approval |
| Legacy `github.com/theupdateframework/go-tuf` | Earlier Go adoption and examples | Upstream declares it deprecated and prone to maintenance errors | Reject |
| `python-tuf` subprocess | Reference implementation and mature client design | Adds Python runtime, packaging, process, and cross-platform failure modes | Reject for runtime; retain as behavioral reference |
| Custom TUF implementation | Small apparent direct dependency surface | High cryptographic and protocol risk; duplicates conformance and security work | Reject |

Version `v2.4.2` is the current minimum proposal because it includes fixes for
empty target hashes and duplicate-public-key threshold counting and follows the
TAP 4 path-traversal fix in `v2.4.1`. The initial client still excludes TAP 4.
The implementation approval must recheck the latest release, Go compatibility,
license, vulnerability advisories, transitive modules, and API surface before
changing `go.mod`.

## 8. CI and publication proposal

CI remains unimplemented. Its future design separates these jobs:

1. Build deterministic platform binaries and attach hashes, test results, SBOM,
   and provenance as review evidence.
2. Require an approved release record before targets signing.
3. Sign targets with a two-person threshold outside general build jobs.
4. Generate snapshot and timestamp metadata with role-scoped automation.
5. Validate the complete repository with a clean client and expired/rollback
   negative tests.
6. Publish immutable target paths before metadata, publishing timestamp last.
7. Re-read the public repository and verify it from the embedded bootstrap root.

Publication credentials, branch protections, artifact hosting, provenance
format, code signing, notarization, retention, and emergency shutdown are open
Milestone 6 decisions.

## 9. Implementation approval packet

The exact packet is recorded in
`docs/milestone-05a-implementation-packet.md`. It includes:

- the exact `go.mod` and `go.sum` diff and dependency audit;
- proposed CLI commands and versioned JSON output schemas;
- embedded-root source path and reproducible digest procedure;
- transport limits, endpoint allow-list, cache paths, and permissions;
- test matrix and conformance fixtures;
- exact files to add or modify;
- rollback plan for removing the update client; and
- confirmation that keys, signing, CI, publication, and installation remain out
  of scope unless separately approved.

Its initial dependency rehearsal discovered a security gate in Go 1.26.4. The
approved update to Go 1.26.9 and a fresh repository and isolated-candidate scan
cleared that gate on 2026-10-16. Stage 5A still requires explicit dependency,
ADR, file-set, and stop-boundary approval before implementation starts.

