# Milestone 13RC Release Qualification Context

## Approved preparation scope

The separately approved RC preparation adds one synthetic native-binary smoke,
connects it to the existing manual six-platform workflow for `v0.4.0` inputs,
and records the qualification contract. It changes no Go code, dependency,
public schema, SQLite format, workspace configuration, installer, package
layout, model behavior, provider adapter, Git state, or release.

## Qualification boundary

The candidate identifier is `v0.4.0-rc.1`. The workflow retains read-only
repository permissions and the existing pinned action revisions. Existing test,
vet, vulnerability, TUF staging, packaging, native version, installer, and
provider-conformance steps remain unchanged.

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

## Remaining approvals

Preparation does not authorize committing or pushing these changes, running the
workflow, downloading artifacts, tagging, creating a draft, uploading assets,
publishing a release, or updating a local installation. Each remains a separate
approval boundary.

## Rollback

Revert the new v0.4.0 smoke, its two conditional workflow steps, qualification
contract, roadmap status, and this context record. A failed candidate advances
to the next `-rc.N`; its bytes and evidence are never reused.
