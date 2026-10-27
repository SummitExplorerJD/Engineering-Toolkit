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

## Publication gate completion

- The release-documentation commit and push were separately approved.
- Stable `v0.3.0` workflow run `40568038154` passed from the exact reviewed
  release commit.
- The stable manifest, checksums, native evidence, provider evidence, and
  package contents passed independent review.
- The annotated tag, draft, asset upload, publication, and local upgrade each
  received separate approval before execution.

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

Candidate acceptance did not authorize reuse of candidate binaries as stable
assets. At candidate acceptance, no tag or GitHub Release existed; the later
stable workflow rebuilt version `v0.3.0` from the separately reviewed release
commit.

## Stable publication evidence

Manual workflow run
[`40568038154`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40568038154)
completed successfully on 2026-10-27 for stable version `v0.3.0` from exact
commit `2bd28273e072b2a33647fb5d7ba09968e381f27a`. The build job, all six native
smoke jobs, and aggregate provider-conformance comparison passed. The run
retained one stable package plus six provider-conformance and six qualification
artifacts.

Independent review confirmed the stable package's 11 files, all 10
`SHA256SUMS` entries, manifest sizes and hashes, six CGO-disabled binaries, two
installers, PE/ELF/Mach-O identities, LF-only `install.sh`, exact embedded
version, and all six bounded qualification records. The workflow ZIP became
`engkit-v0.3.0.zip`; its 11 entries matched the 11 individual release assets
byte for byte, and its digest is
`sha256:550d92230c1562d8d9f12ac5fcd5bed0284bfb1e2d06e0703529c1a07cac80fe`.

Annotated tag `v0.3.0` resolves to the exact stable commit. The
[`v0.3.0` GitHub Release](https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.3.0)
was published on 2026-10-27 as non-draft, non-prerelease, immutable, and latest.
Its 12 uploaded assets have the exact approved inventory, sizes, and SHA-256
digests.

The existing user-scoped Windows AMD64 installation was transactionally
upgraded from `v0.2.2` to `v0.3.0` using the verified stable asset directory.
The installed executable matched approved digest
`sha256:8775446dce378fe29ca8d19895684208706a169996d738873fd7d3b0174e86ce`,
reported exact version `v0.3.0`, exposed catalog `engineering-toolkit@0.3.0`,
and left no installer transaction directory. This was not the distinct fresh
network-installation check described in the publication packet.
