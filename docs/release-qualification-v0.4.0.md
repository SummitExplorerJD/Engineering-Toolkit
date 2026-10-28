# `v0.4.0` Release Qualification

## Status and authority

This document defines the non-publishing qualification gate for `v0.4.0`.
Preparation and a local Windows AMD64 smoke for `v0.4.0-rc.1` completed on
2026-10-28. Candidate `v0.4.0-rc.2` subsequently satisfied the complete
six-platform gate at commit `1de90e29242ecb6ec5171a8463203b3a20b3d657` in
[GitHub Actions run 40770854564](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40770854564).

This gate does not authorize a commit, push, workflow run, tag, GitHub Release,
installer execution outside an isolated test root, model invocation, provider
runtime, network-backed harness, or publication. Qualification applies to one
exact candidate commit and version. A replacement candidate runs the complete
gate and never reuses a failed candidate's binaries or evidence.

## Supported matrix

The existing manual release-candidate workflow builds and tests:

| Operating system | Architectures |
| --- | --- |
| Windows | AMD64, ARM64 |
| Linux | AMD64, ARM64 |
| macOS | AMD64, ARM64 |

All release binaries use `CGO_ENABLED=0`. Existing native version, installer,
provider-conformance, aggregate comparison, vulnerability, and TUF-staging
checks remain unchanged. The v0.4.0 smoke is additive and runs only when the
workflow input starts with `v0.4.0`.

## Synthetic qualification scenarios

`scripts/test-v040-release-binary.ps1` creates one isolated workspace containing
only synthetic names and content. It validates these ten scenario identifiers:

1. `context-v2-rebuild-and-freshness`
2. `mandatory-and-user-exclusions`
3. `bounded-search-and-indexed-read`
4. `cli-and-mcp-result-equivalence`
5. `twelve-tool-stdio-inventory`
6. `indexed-workspace-tree`
7. `compact-result-non-duplication`
8. `text-compatibility-single-representation`
9. `strict-input-and-path-refusal`
10. `stale-projection-refusal`

The fixture configures the workspace root explicitly, includes one eligible
source, one user-excluded source, and one mandatory secret-path exclusion, then
publishes a SQLite v2 projection. CLI assertions verify fresh status, bounded
source inventory, search, indexed read, and stale refusal after a controlled
fixture change.

The MCP assertion uses standard input/output only. It negotiates the protocol,
requires a tools-only capability set and the exact twelve names, then compares
status, search, and read identities with the CLI results. Compact responses must
contain structured data once and a source-free summary; text compatibility must
contain one JSON text block and no structured result. Tree entries are checked
against excluded prefixes. Unknown fields and an escaping path must fail.

The smoke never invokes a model, Ollama daemon, provider runtime, credential,
network service, or real-project content. Raw MCP responses and synthetic
excerpts are processed in memory and are not retained as evidence.

## Evidence contract

Each native job writes exactly one bounded
`engkit.v040-release-qualification/v1alpha1` record. It contains:

- exact candidate version and public binary filename;
- context policy and projection digests;
- source, chunk, exclusion, tool, and tree-entry counts;
- applied compact search and read byte counts; and
- the ten fixed scenario identifiers.

It contains no source path, source text, snippet, excerpt, query text, prompt,
model response, credential, environment value, temporary root, or unrelated
absolute path. Workflow upload selects only
`v040-release-qualification.json`; the isolated fixture is not uploaded.

## Repository and package validation

Before requesting the candidate workflow, the reviewed source must pass:

```text
go test ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
go test -v ./internal/update/tufclient -run '^TestStagingRehearsal$' -count=1
```

The release builder must retain the existing package contract: six native
binaries, `install.ps1`, `install.sh`, `LICENSE`, `release-manifest.json`, and
`SHA256SUMS`. The package contains 11 files and 10 checksum entries. A later
publication packet may additionally wrap those 11 exact files in the workflow
ZIP, producing the same 12 uploaded-asset shape used by v0.3.0.

Every manifest size and digest, checksum entry, executable format, installer
line ending, embedded version, and per-platform qualification record requires
independent review after the workflow succeeds.

## Candidate acceptance

`v0.4.0-rc.2` qualifies only if the validation/build job, all six native smoke
jobs, the ten v0.4.0 scenarios on every tuple, all installer transactions, all
provider-conformance jobs, and the aggregate provider comparison pass for one
exact commit. Temporary workflow artifacts are one candidate package, six
provider records, and six v0.4.0 qualification records.

Signing, notarization, production TUF activation, release notes, stable build,
tag creation, asset upload, local installation, and publication remain separate
decisions. D2 remains skipped: candidate qualification does not attach a real
project to a model-backed harness.

## Observed local preparation

On 2026-10-28:

- the PowerShell parser accepted the new smoke script;
- the local release builder produced all six CGO-disabled
  `v0.4.0-rc.1` binaries and the unchanged 11-file package;
- the existing native release smoke passed for Windows AMD64;
- the new Windows AMD64 qualification passed all ten scenarios;
- sanitized local evidence reported 12 tools, 4 indexed sources, 3 chunks, and
  3 aggregate exclusions; and
- the temporary package, synthetic workspace, index, evidence directory, and
  binaries were removed.

No GitHub Actions workflow, network service, Git operation, tag, release,
installation, provider runtime, Ollama daemon, or model was invoked. Local
success does not substitute for native execution on the other five tuples.

## Observed workflow qualification

On 2026-10-28, the first `v0.4.0-rc.1` workflow attempt exposed a harness exit
status defect after every v0.4.0 scenario had passed. That candidate was
discarded. Commit `1de90e29242ecb6ec5171a8463203b3a20b3d657` clears the
expected negative-command status after validating it. A later successful run
under the same `rc.1` identifier demonstrated the correction but did not revive
the discarded candidate.

Fresh candidate `v0.4.0-rc.2` then passed run `40770854564` at the corrected
commit. The validation/build job, all six native smoke jobs, all installer and
provider-conformance checks, and the aggregate provider comparison completed
successfully. The workflow produced one candidate package, six provider
artifacts, and six v0.4.0 qualification artifacts.

Independent artifact review confirmed:

- all 13 downloaded workflow artifact digests matched GitHub metadata;
- the candidate package digest was
  `7c021629ea28d16f70137d69de9c156f6710b750d97fa1d351f512b621821e16`;
- the package contained the expected 11 files and all 10 `SHA256SUMS` entries
  matched their contents;
- all six qualification records named `v0.4.0-rc.2`, reported 12 tools, 4
  sources, 3 chunks, 3 exclusions, 6 tree entries, and the same 10 scenarios;
  and
- all six provider artifacts contained portable and runtime evidence for
  Codex, Claude, and OpenCode on the declared operating-system and architecture
  tuple.

The downloaded review copies were removed. No candidate artifact was renamed
or retained for stable publication. Tagging, a stable `v0.4.0` rebuild, release
asset upload, publication, and local installation remain separate approvals.

## Failure and rollback

Any failed tuple blocks the candidate. Preserve bounded logs, discard its
artifacts, correct source or harness under a new approval where required, and
advance to the next `-rc.N` identifier. Never rename a candidate binary, reuse
failed evidence, move an existing tag, or replace an immutable asset.

Before workflow execution, rollback reverts only the v0.4.0 smoke script,
workflow steps, this contract, roadmap status, and milestone context. No Go
source, public schema, SQLite table, workspace configuration, dependency,
provider, model, installation, Git ref, or external service requires recovery.
