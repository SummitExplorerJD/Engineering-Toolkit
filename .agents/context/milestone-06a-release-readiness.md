# Milestone 6A Release Readiness Context

## Scope

Milestone 6A prepares a non-publishing `v0.1.0` candidate workflow. It does not
authorize commit, push, tags, GitHub settings changes, workflow execution,
GitHub Releases, GitHub Pages, production TUF keys, signing, or publication.

## Decisions

- Release binaries remain raw executables for Windows, Linux, and macOS on
  `amd64` and `arm64`.
- Version injection uses the existing `internal/cli.Version` linker variable.
- Candidate builds use CGO-disabled, trimmed Go builds and produce
  `SHA256SUMS`, `release-manifest.json`, and `LICENSE` beside the six binaries.
- The manual GitHub workflow has `contents: read` and uploads only a temporary
  Actions artifact retained for 14 days.
- Official GitHub Actions are pinned to full commits corresponding to reviewed
  release tags.
- Production publication stays blocked until the release checklist is complete.

## Current readiness observation

Before this milestone, the CLI compiled as `v0.1.0` for all six target tuples
and the Windows `amd64` binary reported the expected version. However, only
three repository files were tracked at `HEAD`; the implementation and
documentation remained uncommitted. Therefore the project was buildable but
not publishable.

## Remaining decisions

- Whether `v0.1.0` is manual-update-only or embeds the production TUF root.
- GitHub Pages source and deployment mechanism.
- Native smoke-test hosts and evidence retention.
- Windows signing and macOS signing/notarization disposition.
- Exact commit, push, tag, and publication approvals.

## Observed validation

On 2026-10-16:

- the release script generated six raw executables, `LICENSE`, a six-target
  JSON manifest, and eight independently verified `SHA256SUMS` entries;
- the Windows `amd64` artifact reported `v0.1.0` through the public version
  command;
- the script refused to mix a new candidate into a non-empty output directory;
- `actionlint v1.7.12` accepted the candidate workflow;
- `go mod tidy -diff`, the complete test suite, and `go vet ./...` passed; and
- `govulncheck v1.9.0 ./...` reported zero vulnerabilities in called code and
  imported packages. Five findings exist only in required modules with no
  called vulnerable symbols, matching the earlier Milestone 5A audit.

The candidate output used for this validation lives outside the repository and
is not approved for publication. The ignored `.engkit-test-temp` Go build cache
could not be removed because the desktop sandbox rejected recursive deletion;
it remains excluded by `.gitignore` and is not a release input.
