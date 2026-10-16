# TUF Staging Rehearsal Runbook

## Purpose

This runbook validates the Engineering Toolkit verified-download path before a
production root, repository endpoint, or public update command exists. It uses
only disposable Ed25519 keys, synthetic target bytes, an in-process HTTPS test
server, and temporary filesystem state.

The rehearsal does not authorize or perform production signing, publication,
installation, executable replacement, CI changes, commit, or push. It does not
create a reusable signing repository. The production activation gate remains
defined by the
[Milestone 5B production activation packet](../milestone-05b-activation-packet.md).

## Safety properties

- Test private keys exist only in process memory.
- Test metadata, targets, and cache files exist only under Go test temporary
  directories and are removed by the test runner.
- The HTTPS server binds only for the duration of the test and serves only
  generated fixtures.
- No production URL, credential, root, or release artifact is read or written.
- Downloaded targets remain non-executable and are never launched.
- Failed refreshes preserve last-known-good trusted metadata.

Do not modify the rehearsal to accept a production private key or credential.
Production key ceremonies must use the separately approved offline procedure
in the [secure distribution plan](../secure-distribution-plan.md).

## Rehearsal command

Run from the repository root:

```text
go test -v ./internal/update/tufclient -run '^TestStagingRehearsal$'
```

The environment must permit loopback HTTPS sockets and atomic rename operations
inside the Go temporary directory. A restricted sandbox may reject those OS
operations even when the implementation is correct; in that case, rerun on an
approved local development host and retain the complete output.

The command must not be given network credentials. It does not require access
to the public Internet.

## Scenarios and expected results

| Scenario | Expected result |
| --- | --- |
| Six-platform stable target inventory | All canonical paths match the approved target layout |
| Valid metadata refresh | Root, timestamp, snapshot, and targets version 1 are trusted |
| Valid target download | Signed bytes enter the content-addressed cache with no execute bits |
| Corrupted timestamp signature | Refresh fails and last-known-good timestamp bytes remain unchanged |
| Tampered target bytes | Download fails and no target is published |
| Expired metadata | Refresh fails with `metadata_expired` |
| Timestamp rollback | Version 2 is accepted; replayed version 1 fails with `rollback_detected` |
| Dual-threshold root rotation | Sequential root version 2 is accepted |
| Insufficient root threshold | Rotation fails with `trust_failed` |

The valid-download case logs the disposable bootstrap-root SHA-256 digest and
the verified target digest. These values are evidence for that run only and
must never be copied into a production descriptor.

## Evidence checklist

Record the date, host operating system, architecture, Go version, command,
complete test output, and final pass/fail result. Before accepting the evidence:

- [ ] The command selected only `TestStagingRehearsal`.
- [ ] Every listed subtest passed.
- [ ] The run used no production root, endpoint, key, or credential.
- [ ] No generated file was copied out of the test temporary directory.
- [ ] `go test ./internal/update/...` also passed on the same host.
- [ ] `go test ./...`, `go vet ./...`, and `govulncheck ./...` passed or every
      environment limitation was recorded without claiming success.
- [ ] `git diff --check` passed and the final file list was inspected.

Passing this rehearsal is necessary but not sufficient for production. A
staging service that mirrors the chosen production hosting topology must still
be exercised with disposable keys after the exact origin/CDN design exists.

## Failure handling

If a security scenario unexpectedly succeeds, stop the activation work. Keep
the failing output, classify whether the defect is in policy, transport, TUF
metadata handling, or cache publication, and propose a separately reviewed fix.

If the test fails only because loopback sockets or atomic renames are blocked,
record the host restriction and rerun unchanged on an approved environment. Do
not disable HTTPS, signature checks, expiration, rollback protection, hash
verification, or atomic publication to make the rehearsal pass.

## Removal

Before production activation, this rehearsal is removable by deleting:

```text
internal/update/tufclient/staging_test.go
docs/runbooks/tuf-staging-rehearsal.md
.agents/context/milestone-05b-staging-rehearsal.md
```

No production code, dependency, public schema, CLI command, or persistent user
state depends on these files.
