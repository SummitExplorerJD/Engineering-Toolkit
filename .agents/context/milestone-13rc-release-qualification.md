# Milestone 13RC Release Qualification Context

## Approved preparation scope

The separately approved RC preparation adds one synthetic native-binary smoke,
connects it to the existing manual six-platform workflow for `v0.4.0` inputs,
and records the qualification contract. It changes no Go code, dependency,
public schema, SQLite format, workspace configuration, installer, package
layout, model behavior, provider adapter, Git state, or release.

## Qualification boundary

The accepted candidate identifier is `v0.4.0-rc.2`. The workflow retains
read-only repository permissions and the existing pinned action revisions.
Existing test, vet, vulnerability, TUF staging, packaging, native version,
installer, and provider-conformance steps remain unchanged.

Each native tuple additionally runs `scripts/test-v040-release-binary.ps1` in a
synthetic workspace. The smoke covers SQLite v2 rebuild and freshness, mandatory
and user exclusions, bounded source/search/read behavior, stale refusal, exact
twelve-tool stdio MCP inventory, indexed tree synthesis, compact
non-duplication, text-only compatibility, strict input and path rejection, and
CLI/MCP result equivalence.

## Evidence boundary

One `engkit.v040-release-qualification/v1alpha1` JSON record per tuple retains
only public binary identity, version, policy and projection digests, bounded
counts, applied byte limits, fixed scenario identifiers, and outcomes. Raw MCP
frames, synthetic paths and content, queries, snippets, excerpts, prompts,
credentials, environment values, and temporary roots are not retained or
uploaded.

The workflow upload selects only the bounded JSON record. It does not upload
the synthetic workspace or disposable SQLite index. No real repository content
or model-backed harness participates in RC qualification.

## Observed local validation

On 2026-10-28:

- PowerShell parsing completed without diagnostics;
- the existing builder produced the unchanged 11-file `v0.4.0-rc.1` package
  with six CGO-disabled binaries;
- the existing Windows AMD64 native version smoke passed;
- the new Windows AMD64 v0.4.0 smoke passed all ten scenarios;
- bounded evidence reported 12 tools, 4 sources, 3 chunks, and 3 exclusions;
  and
- all temporary package, fixture, database, binary, and evidence files were
  removed after inspection.

No workflow, network, Git operation, installer, provider runtime, Ollama daemon,
model, tag, or release was used. The six native workflow tuples remain the
authoritative candidate gate.

## Observed candidate qualification

The first `v0.4.0-rc.1` run exposed a PowerShell exit-status defect after the
new qualification script had reported success. The candidate was discarded.
Commit `1de90e29242ecb6ec5171a8463203b3a20b3d657` resets the status left by an
expected negative command after validating the failure. A subsequent `rc.1`
run demonstrated the correction but did not revive that candidate identifier.

Fresh `v0.4.0-rc.2` passed GitHub Actions run `40770854564` on 2026-10-28. The
validation/build job, all six native jobs, installer transactions, portable and
runtime provider checks, aggregate provider comparison, and ten v0.4.0
scenarios per tuple completed successfully.

The run produced 13 temporary artifacts. Independent review matched every
artifact digest to GitHub metadata, matched all 10 package checksums across the
unchanged 11-file package, validated six bounded qualification records, and
validated portable and runtime records for all three providers on all six
tuples. The candidate package artifact digest was
`7c021629ea28d16f70137d69de9c156f6710b750d97fa1d351f512b621821e16`.
Downloaded review copies were removed after inspection.

## Stable publication evidence

Manual workflow run
[`40773213540`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40773213540)
completed successfully for stable version `v0.4.0` from exact commit
`a7bb270569a520628b2b7db3385ebf9b4a007b97`. The build job, all six native
smoke jobs, ten v0.4.0 scenarios per tuple, installer and provider checks, and
aggregate provider comparison passed. Independent review confirmed all 13
artifact digests, the 11-file package, all 10 package checksums, executable
formats, six qualification records, and six provider records. The exact stable
workflow ZIP has digest
`1999101eae28d16f70137d69de9c156f6710b750d97fa1d351f512b621821e16`.

Annotated tag `v0.4.0` resolves to the exact stable commit. The
[`v0.4.0` GitHub Release](https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.4.0)
was published on 2026-10-29 as non-draft, non-prerelease, immutable, and latest.
Public metadata reports the 12 approved uploaded assets in `uploaded` state
with matching names, sizes, and SHA-256 digests. GitHub's two generated source
archives are outside that uploaded-asset inventory.

## Remaining approvals

A distinct public redownload and fresh Windows network-installation check
remain separately gated. No public installer execution, installed version,
context rebuild, or installed-binary MCP result is claimed here.

## Rollback

Revert the new v0.4.0 smoke, its two conditional workflow steps, qualification
contract, roadmap status, and this context record. A failed candidate advances
to the next `-rc.N`; its bytes and evidence are never reused.
