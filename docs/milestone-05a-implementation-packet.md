# Milestone 5A Implementation Approval Packet

- **Prepared:** 2026-10-16
- **Status:** Implemented on 2026-10-16 for the approved internal scope
- **Approved scope:** Internal verified-download core only

## 1. Purpose and approval boundary

This packet defines the exact next implementation slice for authenticated TUF
metadata refresh, release selection, and verified download into an
application-owned cache.

Approval of this packet would not authorize:

- a production bootstrap root or private key;
- a production repository URL or network access from the released CLI;
- a public `engkit update` command;
- executable installation or self-replacement;
- CI, signing, metadata creation, publication, commit, or push.

The implementation must stop at an internal client exercised against an
in-process test repository. Production activation is Milestone 5B and requires
the separately approved trust anchor, repository descriptor, CLI diff, and
operational readiness review.

## 2. Proposed dependencies

The exact direct additions are:

```text
github.com/theupdateframework/go-tuf/v2 v2.4.2
golang.org/x/mod v0.30.0
```

`go-tuf/v2` provides the maintained TUF client workflow. `x/mod/semver`
provides strict comparison of signed release versions so rollback policy does
not depend on a local version parser.

An isolated `go get` and `go mod tidy` rehearsal, starting from the current
repository module files, produced these additional indirect entries:

```text
github.com/cenkalti/backoff/v5 v5.0.3
github.com/google/go-containerregistry v0.20.7
github.com/opencontainers/go-digest v1.0.0
github.com/secure-systems-lab/go-securesystemslib v0.11.0
github.com/sigstore/protobuf-specs v0.5.0
github.com/sigstore/sigstore v1.10.6
golang.org/x/crypto v0.55.0
golang.org/x/term v0.45.0
google.golang.org/genproto/googleapis/api v0.0.0-20250825161204-c5933d9347a5
google.golang.org/protobuf v1.36.11
```

The existing SQLite and `golang.org/x/sys v0.47.0` versions remain unchanged.
The Go module build list grows from 16 to 93 modules: 77 new modules. Package
reachability analysis shows 12 new external modules in the compiled dependency
closure. The difference exists because Go retains the module graph of direct
dependencies even when many packages are not linked into the application.

This is a material supply-chain cost. Dependency approval must explicitly
accept the complete graph, not only the visible direct requirements.

## 3. Compatibility and license observations

- The repository declares Go `1.26.9`; go-tuf v2.4.2 declares Go `1.25.0`.
  The language-version requirement is compatible.
- go-tuf v2.4.2 is a tagged, Apache-2.0 release.
- The official updater API exposes root-rotation, metadata-size, local-cache,
  remote-URL, custom-fetcher, target lookup, and verified-download boundaries
  needed by this design.
- The updater is not concurrency-safe. Engineering Toolkit will create one
  client per operation and hold one process-local operation lock.
- TAP 4 multi-repository APIs, signing APIs, and repository-authoring APIs are
  excluded.

Full license enumeration and notice-file generation are required in the
implementation validation; this packet does not claim that transitive-license
review is complete.

## 4. Security gate discovered during rehearsal

`govulncheck v1.9.0` reported five reachable standard-library findings in the
installed Go `1.26.4` toolchain:

| Finding | Area | Fixed version |
| --- | --- | --- |
| `GO-2026-6617` | HTTP/2 HPACK encoder race | Go 1.26.9 |
| `GO-2026-6613` | HTTP/1 CONNECT desynchronization | Go 1.26.9 |
| `GO-2026-6612` | HTTP/2 flow-control accounting | Go 1.26.9 |
| `GO-2026-6611` | HTTP/2 CPU exhaustion | Go 1.26.9 |
| `GO-2026-6603` | HTTP/2 trailer memory exhaustion | Go 1.26.9 |

The scan also reported 14 imported-package and nine required-module findings
without reachable call paths in the rehearsal. They still require review when
the real code exists.

**Resolution (2026-10-16):** the separately approved module policy now declares
Go 1.26.9, and the Go toolchain selector successfully executed that version.
`go test ./...`, `go vet ./...`, and `govulncheck v1.9.0 ./...` passed for the
repository. A fresh isolated rehearsal with go-tuf v2.4.2 and x/mod v0.30.0
also passed its build and reported zero reachable vulnerabilities and zero
vulnerabilities in imported packages. It reported five findings only in
required modules with no called vulnerable symbols. Those non-reachable module
findings remain review inputs; they do not reinstate the toolchain stop
condition.

## 5. Internal API and ownership

The `internal/update` domain owns:

- strict `stable` channel policy;
- supported OS and architecture tuples;
- SemVer validation and no-downgrade comparison;
- expected target-path construction;
- validation of signed custom target metadata;
- response schemas and error classification; and
- application state and cache path selection.

The `internal/update/tufclient` adapter owns:

- constructing the go-tuf updater from injected trusted-root bytes;
- refreshing one single-repository metadata chain;
- exact target lookup;
- target hash and length verification;
- bounded download into a caller-provided staging path; and
- extraction of trusted metadata versions and expirations.

The adapter must not choose a channel or version, obtain a production root,
read workspace `.engkit`, install an executable, invoke a shell, create signing
metadata, or publish anything.

Proposed ports:

```go
type Repository struct {
    MetadataURL string
    TargetsURL  string
    Root        []byte
}

type Platform struct {
    OS   string
    Arch string
}

type Client interface {
    Refresh(context.Context) (TrustedState, error)
    Target(context.Context, string) (Target, error)
    Download(context.Context, Target, string) (VerifiedFile, error)
}
```

The concrete types may gain private fields, but public JSON field names and
security semantics may not change during implementation without returning to
approval.

## 6. Repository and target policy

- Only HTTPS repository descriptors compiled into the application are allowed.
- There is no CLI flag, environment variable, YAML setting, or workspace file
  that can override a production repository or trust root.
- The initial channel allow-list contains only `stable`.
- Supported platform tuples are the six pairs in
  `docs/secure-distribution-plan.md`.
- Target paths must match exactly:

```text
releases/stable/<semver>/<os>-<arch>/engkit[.exe]
```

- Versions use canonical `vMAJOR.MINOR.PATCH` Go-module SemVer syntax.
- Pre-release versions are not eligible for `stable`.
- The candidate must be greater than or equal to the running version. A lower
  version is a rollback-policy error even if TUF metadata authorizes it.
- Target custom metadata must match
  `engkit.release-target/v1alpha1`, version, channel, OS, architecture, and
  executable format.
- TUF consistent snapshots are mandatory; delegation and TAP 4 are rejected.

## 7. Transport and resource limits

| Control | Proposed value |
| --- | ---: |
| Root metadata maximum | 512 KiB |
| Timestamp metadata maximum | 64 KiB |
| Snapshot metadata maximum | 1 MiB |
| Targets metadata maximum | 4 MiB |
| Target maximum | 128 MiB |
| Sequential root rotations per refresh | 32 |
| Delegations | 0 |
| DNS/connect timeout | 10 seconds |
| TLS handshake timeout | 10 seconds |
| Response-header timeout | 15 seconds |
| Whole metadata request timeout | 30 seconds |
| Whole target request timeout | 10 minutes |
| Redirects | At most 3, same HTTPS host only |
| Retries | At most 2 after the first attempt |

Retries apply only to idempotent GET requests and transient transport,
`408`, `429`, and `5xx` responses. Signature, expiration, rollback,
policy, length, and hash failures are never retried. Responses are read through
hard byte limits. URLs containing user information, fragments, non-HTTPS
schemes, unexpected hosts, or ambiguous escaped paths are rejected before a
request.

The User-Agent is `engkit/<version> update-client/1`. Proxy behavior must be
explicitly tested; credentials are never sourced from workspace files.

## 8. State and atomicity

The bootstrap root is injected by the application and remains outside the
writable cache. Stage 5A uses only test roots held in test memory and temporary
directories.

Metadata refresh uses a private staging directory next to the trusted metadata
directory. New trusted state is published only after go-tuf accepts the full
refresh. A failed refresh preserves the prior trusted state.

Target download uses a unique same-directory temporary file with user-only
permissions. After go-tuf verification, the implementation synchronizes and
closes the file, rechecks the destination, and atomically renames it to a
content-addressed cache name. It never marks the file executable in Stage 5A.

Concurrent processes use a platform-safe lock with a bounded wait. If a robust
cross-process lock cannot be delivered without another dependency, Stage 5A
must refuse concurrent publication using exclusive temporary-file creation and
record the limitation rather than invent an unsafe lock.

## 9. Planned CLI activation contract

These commands are specified now but are not exposed in Stage 5A:

```text
engkit update check [--channel stable]
engkit update download [--channel stable] [--version vMAJOR.MINOR.PATCH]
```

Like existing inspection commands, successful output is always JSON.
`check` refreshes trusted metadata but downloads no target. `download`
writes only verified application cache. Neither command installs a binary.

Check result:

```json
{
  "schema_version": "engkit.update-check/v1alpha1",
  "status": "update_available",
  "channel": "stable",
  "current_version": "v0.1.0",
  "candidate": {
    "version": "v0.2.0",
    "os": "windows",
    "arch": "amd64",
    "target_path": "releases/stable/v0.2.0/windows-amd64/engkit.exe",
    "length": 12345678,
    "sha256": "<64 lowercase hexadecimal characters>"
  },
  "metadata": {
    "root_version": 1,
    "timestamp_version": 4,
    "snapshot_version": 4,
    "targets_version": 4,
    "timestamp_expires": "2026-10-17T00:00:00Z"
  }
}
```

Allowed check statuses are `current` and `update_available`. There is no
successful `unverified` status.

Download result:

```json
{
  "schema_version": "engkit.update-download/v1alpha1",
  "status": "verified_cached",
  "channel": "stable",
  "target": {
    "version": "v0.2.0",
    "os": "windows",
    "arch": "amd64",
    "target_path": "releases/stable/v0.2.0/windows-amd64/engkit.exe",
    "length": 12345678,
    "sha256": "<64 lowercase hexadecimal characters>"
  },
  "cache_path": "<absolute application cache path>",
  "verified_at": "2026-10-16T00:00:00Z"
}
```

Usage errors retain exit code 2. Network, trust, policy, and filesystem failures
retain exit code 1 and use stable error categories:
`update_not_configured`, `transport_failed`, `trust_failed`,
`metadata_expired`, `rollback_detected`, `target_incompatible`,
`target_invalid`, and `cache_failed`. A future JSON error envelope requires
its own cross-command compatibility decision.

## 10. Exact Stage 5A file set

### New implementation files

```text
internal/update/model.go
internal/update/policy.go
internal/update/policy_test.go
internal/update/state.go
internal/update/state_windows.go
internal/update/state_darwin.go
internal/update/state_linux.go
internal/update/state_test.go
internal/update/tufclient/client.go
internal/update/tufclient/client_test.go
internal/update/tufclient/transport.go
internal/update/tufclient/transport_test.go
internal/update/tufclient/testrepo_test.go
```

`testrepo_test.go` generates ephemeral Ed25519 test keys in memory and writes
only to `t.TempDir()`. No private test key or signed production-like
repository is committed.

### Modified files

```text
go.mod
go.sum
docs/adr/0005-use-official-go-tuf-client.md
docs/roadmap.md
README.md
.agents/context/milestone-05-secure-distribution-planning.md
```

No file under `cmd/`, `internal/cli/`, `.specify/`, or installed
Spec Kit skills is modified in Stage 5A.

## 11. Required tests

- Accept valid root, timestamp, snapshot, targets, and target bytes.
- Reject insufficient thresholds, duplicate-key threshold inflation, empty
  target hashes, wrong lengths, wrong hashes, and malformed metadata.
- Reject expired, rolled-back, freeze-replayed, and mix-and-match metadata with
  a controlled clock.
- Accept sequential dual-threshold root rotation and reject skipped or
  excessive root versions.
- Reject all delegations, TAP 4 configuration, non-HTTPS URLs, unsafe
  redirects, traversal, encoded traversal, absolute paths, and wrong platform
  metadata.
- Enforce every byte, time, retry, and redirect limit.
- Preserve last-known-good metadata and target cache after injected failures.
- Prove download staging is same-filesystem, non-executable, atomically
  published, and collision-safe.
- Validate Windows, macOS, and Linux state paths without reading workspace
  `.engkit`.
- Run race-enabled tests around operation serialization where supported.

Validation commands after the toolchain gate is cleared:

```text
gofmt -d internal/update
go test ./internal/update/... ./internal/cli/...
go test -race ./internal/update/...
go test ./...
go vet ./...
govulncheck ./...
git diff --check
```

Cross-compilation must at least compile the update packages for the six target
tuples. Native filesystem and locking smoke tests remain a Milestone 6 release
gate.

## 12. Rollback and removal

Before production activation, Stage 5A is removable by deleting
`internal/update`, removing the two direct requirements, running
`go mod tidy`, and reverting the documentation status. No workspace schema,
CLI command, production trust state, or user file depends on it.

Any test cache lives under Go or test temporary directories and is not part of
repository state. Rollback must verify that the module graph returns to the
pre-Stage-5A build list and that `go test ./...` still passes.

## 13. Approval checklist

Implementation may begin only after all are true:

- [x] Go 1.26.9 or later supported patch release is available and approved.
- [x] A fresh isolated dependency rehearsal reproduces the proposed module diff.
- [x] `govulncheck` has zero reachable findings or an approved exception.
- [x] The 77-module build-list increase and 12-module compiled increase are
      explicitly accepted.
- [x] ADR-0005 is approved for transition from Proposed to Accepted.
- [x] The exact file set and Stage 5A stop boundary are approved.
- [x] Production root, endpoint, CLI activation, installation, CI, signing,
      publication, commit, and push remain unapproved.

## 14. Implementation evidence

The approved slice added the 13 files listed in Section 10 and the exact two
direct dependencies from Section 2. The implementation provides strict policy,
platform state selection, bounded HTTPS transport, a single-repository go-tuf
adapter, isolated metadata refresh, last-known-good preservation, and atomic
non-executable content-addressed target publication.

Observed validation on Go 1.26.9:

- update-package, CLI, and full repository tests passed;
- `go vet ./...` and `git diff --check` passed;
- `govulncheck v1.9.0 ./...` reported zero reachable and zero imported-package
  vulnerabilities, with five findings only in required modules that have no
  called vulnerable symbols;
- update test binaries compiled for Windows, Linux, and macOS on both `amd64`
  and `arm64` with CGO disabled;
- all 92 non-main modules in the build list exposed a root license or notice
  file after full module download; and
- race-enabled testing was attempted, but the installed Windows GCC failed with
  an illegal system-DLL relocation before package compilation. This environment
  limitation remains a Milestone 6 validation item.

No production network descriptor, root, key, command, installer, signing,
publication, commit, or push was added.

