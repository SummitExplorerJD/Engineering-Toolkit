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

## Completion evidence

- Candidate `v0.2.2-rc.2` passed release workflow run `39975623529` after the
  macOS test helper resolved the physical temporary root before validation.
- Stable `v0.2.2` passed release workflow run `39977017785` on Windows, Linux,
  and macOS for AMD64 and ARM64; the aggregate provider-conformance comparison
  also passed.
- The immutable release tag points to
  `25c14d88e9871b78d674956d370a84768db9b90e` and contains 12 assets.
- The downloaded stable ZIP digest is
  `sha256:ae460b7f129b4daf5814b82ad4d9a0376c2514d37ea5d39669348a4b29f4f599`;
  all internal checksum entries were independently verified.
- A local Windows upgrade reports `engkit v0.2.2` from
  `C:\Users\julit\AppData\Local\Programs\engkit\engkit.exe`, and embedded
  catalog inspection succeeds.

## Deferred work

- D2 native discovery with exact Codex, Claude Code, and OpenCode versions.
- Provider installation, authentication, network access, and credentials.
- Public evidence schemas or production-package changes.
