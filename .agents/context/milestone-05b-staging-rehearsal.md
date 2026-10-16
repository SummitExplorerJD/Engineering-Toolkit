# Milestone 5B-A Staging Rehearsal Context

## Scope

Milestone 5B-A adds a repeatable local TUF rehearsal with disposable keys. It
does not activate the public CLI or add a production root, endpoint, signing
workflow, installer, CI configuration, commit, or push.

## Implementation

- `internal/update/tufclient/staging_test.go` composes the existing test
  repository and verified-download client into one operator-visible rehearsal.
- `docs/runbooks/tuf-staging-rehearsal.md` defines execution, evidence,
  expected results, failure handling, and removal.
- Test private keys remain in process memory. Generated metadata and target
  bytes stay in Go temporary directories.

## Covered scenarios

- Canonical target paths for Windows, Linux, and macOS on `amd64` and `arm64`.
- Valid metadata refresh and verified non-executable cache publication.
- Corrupted metadata signature with last-known-good preservation.
- Tampered target rejection without publication.
- Expiration and timestamp rollback rejection.
- Accepted dual-threshold root rotation and rejected insufficient threshold.

## Remaining production gate

The local rehearsal does not reproduce an external origin or CDN. After a
hosting design is selected, a second staging run must use that topology with
disposable keys. Production activation still requires every input and approval
listed in `docs/milestone-05b-activation-packet.md`.

## Observed validation

On 2026-10-16, the focused rehearsal passed every scenario using Go test with
`TEMP` and `TMP` redirected to an authorized workspace directory. This was
necessary because the desktop sandbox blocks atomic renames in its default
Windows package temporary directory. The test used loopback HTTPS only.

The following checks passed in the same environment:

- focused `TestStagingRehearsal` execution;
- all tests under `internal/update/...`;
- the complete repository test suite; and
- `go vet ./...`.

The disposable root digest printed by the rehearsal is intentionally not
recorded here because a new key set is generated for every run and it has no
production trust meaning.
