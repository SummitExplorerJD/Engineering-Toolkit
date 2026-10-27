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

- Approve the exact documentation commit and push separately.
- Run the stable `v0.3.0` workflow from the resulting reviewed commit.
- Review the stable manifest, checksums, native evidence, and provider evidence.
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

During local validation, no GitHub Actions workflow, Ollama daemon, model,
provider runtime, remote network service, Git operation, tag, or release was
invoked.

## Accepted candidate evidence

The manual `Release candidate` workflow run
[`40537840017`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40537840017)
completed successfully on 2026-10-27 for `v0.3.0-rc.1` from commit
`08dac40b4b7f7d2549ce318dcf5e446036347fda`. The run completed the build job,
all six native smoke jobs, and aggregate provider-conformance comparison in
10 minutes 25 seconds. The only annotation was a non-failing GitHub-hosted
runner capacity notice for macOS ARM64.

The run retained 13 temporary artifacts through 2026-11-10:

- one candidate package with artifact digest
  `sha256:22385f54badee13611514d774b793cbfb73cc653d0d731423227e40001416548`;
- six platform-specific provider-conformance records; and
- six `engkit.release-qualification/v1alpha1` records, one for each supported
  operating-system and architecture tuple.

Independent review confirmed 11 files inside the candidate package, 10 valid
`SHA256SUMS` entries, six manifest binaries, two manifest installers, matching
manifest sizes and hashes, CGO disabled, PE/ELF/Mach-O executable identities,
and LF-only `install.sh`. Every qualification record names `v0.3.0-rc.1`,
catalog `0.3.0`, the same three exact Go profiles, and all eight approved
synthetic scenarios.

Candidate acceptance does not authorize reuse of candidate binaries as stable
assets. The stable workflow must rebuild version `v0.3.0` from the separately
reviewed documentation commit. No tag or GitHub Release was created.
