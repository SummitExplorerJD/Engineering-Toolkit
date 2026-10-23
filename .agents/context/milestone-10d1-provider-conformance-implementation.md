# Milestone 10D1: provider-conformance implementation

## Outcome

The D1 harness validates the embedded `project-guide@0.1.0` pilot against the
accepted Codex, Claude Code, and OpenCode adapter contracts. It exercises exact
rendering, transactional sync and unsync, preservation, and six-platform
equivalence without invoking provider runtimes or the network.

## Module boundary

- `providersync` owns exact rendering, target selection, frontmatter checks,
  refusal behavior, and the reviewed golden fixture.
- `providersyncexec` proves the sync, unchanged replay, and unsync lifecycle
  while preserving canonical and unrelated content.
- `test-provider-conformance.ps1` composes the embedded pilot and drives the
  public CLI with each native release binary in a disposable workspace.
- `compare-provider-conformance.ps1` compares portable evidence and requires
  one runtime record for every supported operating-system and architecture pair.
- The release-candidate workflow runs and aggregates the same scenario across
  Windows, Linux, and macOS on AMD64 and ARM64.

## Safety decisions

- The harness creates and removes only its own GUID-named directory beneath
  `.engkit-test-temp`.
- Plan IDs are required to repeat in one unchanged workspace but are excluded
  from cross-host comparison because the root path is part of their contract.
- Portable evidence contains only stable relative paths, findings, and digests;
  runtime evidence contains platform and root-bound diagnostic fields.
- Provider-native, Spec Kit, canonical, and unrelated sentinel bytes must be
  preserved through sync and unsync.
- Missing filesystem capabilities, symlink or junction ambiguity, collisions,
  unowned output, and evidence differences fail closed.

## Validation boundary

Focused static tests, PowerShell parsing, and synthetic six-platform comparison
can run locally. The Windows sandbox cannot exercise atomic no-replace
publication or create symbolic-link fixtures, so those are not treated as local
passes. D1 completes only after the release-candidate workflow succeeds on all
six native targets and the aggregate portable comparison passes.

## Deferred work

- D2 native discovery with exact Codex, Claude Code, and OpenCode versions.
- Provider installation, authentication, network access, and credentials.
- Git commit, push, tags, release candidates, and stable release publication.
- Public evidence schemas or production-package changes.
