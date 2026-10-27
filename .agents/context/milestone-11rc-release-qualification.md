# Milestone 11RC Release Qualification Context

## Approved scope

The approved `v0.3.0RC` preparation slice adds a synthetic native-binary smoke
harness, connects it to the existing six-platform manual release workflow, and
records the qualification contract. It changes no Go code, dependency, public
schema, SQLite format, workspace configuration, release asset format, or model
allowlist.

## Safety boundary

The qualification workspaces contain only synthetic content. Disabled advice
and missing-context fallback are exercised without an Ollama request. Provider
absence is produced by an isolated process `PATH`; no provider runtime is
invoked. MCP uses local standard input and output only and exposes no mutation
tool or listening socket.

The workflow remains manual with read-only repository permissions. Preparing
this slice does not authorize executing it, committing, pushing, tagging,
creating a release, or publishing artifacts.

## Evidence model

Each native platform job writes one bounded
`engkit.release-qualification/v1alpha1` JSON record. The record identifies the
candidate version, binary filename, catalog version, exact profile inventory,
and completed scenario names. It excludes prompts, model responses, repository
content, credentials, and absolute paths.

The existing provider-conformance aggregate remains authoritative for portable
provider projections. The new evidence is deliberately separate and covers the
`v0.3.0` doctor, unified-init, catalog, local-intelligence fallback, and MCP
surfaces.

## Remaining gates

- Obtain separate approval before dispatching the manual release-candidate
  workflow.
- Review all six native evidence records and aggregate provider evidence.
- Prepare exact release notes and publication inputs under a later approval.
- Obtain separate Git, tag, release, and publication approvals.

## Observed local validation

On 2026-10-27:

- the release builder produced all six CGO-disabled `v0.3.0-rc.1` binaries and
  the existing release package without changing the repository;
- the existing native release smoke passed for Windows AMD64;
- the new `v0.3.0` native qualification passed for Windows AMD64 and wrote the
  bounded evidence record;
- `go vet ./...` passed;
- `go test ./...` passed when rerun with the approved local filesystem and
  loopback permissions required by existing transaction and TUF tests;
- the first sandboxed test attempt failed because the sandbox denied atomic
  renames under its temporary directory and local TUF sockets, not because a
  product assertion failed;
- the PowerShell parser accepted the qualification script; and
- `actionlint` was not installed and was not downloaded, so workflow execution
  remains the authoritative validation of runner expressions and job wiring.

No GitHub Actions workflow, Ollama daemon, model, provider runtime, remote
network service, Git operation, tag, or release was invoked.
