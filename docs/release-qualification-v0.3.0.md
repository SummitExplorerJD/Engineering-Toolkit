# `v0.3.0` Release Qualification

## Status and authority

This document defines the non-publishing qualification gate for `v0.3.0`. It
was satisfied by `v0.3.0-rc.1` on 2026-10-27. It does not authorize a commit,
push, stable workflow run, tag, GitHub Release, installer execution outside an
isolated test root, model invocation, or publication.

Qualification applies to the exact candidate commit and version. A replacement
candidate must run the complete gate again and must not reuse evidence from a
failed candidate.

## Supported matrix

The manual release-candidate workflow must build and run native tests on:

| Operating system | Architectures |
| --- | --- |
| Windows | AMD64, ARM64 |
| Linux | AMD64, ARM64 |
| macOS | AMD64, ARM64 |

All release binaries use `CGO_ENABLED=0`. Each native job verifies the injected
version, installation transaction, provider conformance, and the `v0.3.0`
qualification scenarios below.

## Synthetic qualification scenarios

`scripts/test-v030-release-binary.ps1` creates isolated workspaces containing
only synthetic names and content. It verifies:

- catalog `engineering-toolkit@0.3.0` contains exact profiles
  `go-cli@0.1.0`, `go-library@0.1.0`, and `go-service@0.1.0`;
- unified initialization produces the existing workspace and project plan
  schemas and refuses a non-empty ambiguous target;
- Go profile recommendation returns all three compatible profiles without
  changing the workspace;
- disabled local intelligence preserves deterministic ordering;
- enabled local intelligence with no fresh SQLite projection falls back before
  attempting an Ollama request;
- a model outside the exact allowlist is rejected;
- an integration probe with an isolated empty `PATH` reports Ollama unavailable
  while retaining the read-only MCP capability;
- the MCP stdio server negotiates the protocol, advertises only tools, exposes
  exactly the eight approved read-only tools, and stops after input reaches
  end-of-file.

The smoke script neither uses repository excerpts as prompts nor starts,
installs, downloads, replaces, or invokes an Ollama model. Real-model evidence
remains limited to the separately approved synthetic smoke observations
recorded for milestone `v0.3.0E2`.

## Repository validation

Before requesting a candidate workflow run, the reviewed source must pass:

```text
go test ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
go test -v ./internal/update/tufclient -run '^TestStagingRehearsal$' -count=1
```

The release builder must produce six binaries, installers, `LICENSE`,
`release-manifest.json`, and `SHA256SUMS`. Checksums, manifest entries, version
output, line endings, and executable formats must be independently reviewed.

## Candidate acceptance

A candidate qualifies only when all six native jobs and the aggregate provider
comparison pass for the same commit and candidate version. The retained
artifacts must include one bounded `v030-release-qualification.json` file per
platform tuple. Evidence records scenario identifiers and public artifact
identity only; it contains no prompts, responses, repository excerpts,
credentials, or absolute workspace paths.

Signing and notarization limitations, production TUF activation, exact release
notes, tag creation, and publication remain separate decisions. Qualification
does not convert an observed mutable Ollama tag into a general compatibility
claim.

### Accepted `v0.3.0-rc.1` evidence

GitHub Actions run
[`40537840017`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40537840017)
passed from commit `08dac40b4b7f7d2549ce318dcf5e446036347fda`:

- the validation and build job passed tests, vet, vulnerability analysis, the
  TUF staging rehearsal, packaging, and artifact upload;
- Windows, Linux, and macOS passed natively on AMD64 and ARM64;
- all six installer transactions, provider-conformance jobs, and `v0.3.0`
  qualification smokes passed;
- the aggregate provider-conformance comparison passed; and
- 13 temporary artifacts were retained: one candidate package, six provider
  records, and six release-qualification records.

Independent inspection verified all candidate checksums, manifest hashes and
sizes, six executable formats, two installers, LF-only `install.sh`, and six
qualification records containing the same eight synthetic scenarios. The
candidate package artifact digest is
`sha256:22385f54badee13611514d774b793cbfb73cc653d0d731423227e40001416548`.

This evidence qualifies the source and release process but does not make the
`-rc.1` binaries stable assets. Stable publication requires a fresh `v0.3.0`
build from the exact approved release commit and separate verification.

## Failure and rollback

Any failed tuple blocks the candidate. Preserve its logs, discard its candidate
artifacts, correct the source or harness under a new approval when required,
and use the next `-rc.N` identifier. Do not replace an immutable artifact or
move a published tag.

Before any workflow run, rollback is limited to removing the version-specific
smoke script and document and reverting their workflow and roadmap references.
No schema, canonical workspace, SQLite database, model, or external service
requires recovery.
