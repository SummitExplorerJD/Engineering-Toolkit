# Milestone 07: v0.1.1 bootstrap installers

## Approved boundary

The approved slice adds user-scoped bootstrap installation scripts for Windows,
Linux, and macOS; packages them with release artifacts; and tests installation
transactions across AMD64 and ARM64 runners. It does not add a CLI command,
activate production TUF, publish a release, sign executables, integrate package
managers, commit, or push.

## Implementation decisions

- Exact versions are mandatory; `latest` is not supported.
- Existing installations require explicit upgrade intent.
- Binary verification precedes version execution and publication.
- Staging occurs under the destination directory for same-filesystem rename and
  recovery.
- PATH modification is opt-in and idempotent.
- Local asset mode supports candidate testing and reviewed offline bundles.
- The release manifest adds installer metadata without changing its
  `v1alpha1` schema identifier.

## Observed local evidence

- A local `v0.1.1-rc.1` candidate built all six release executables.
- The Windows AMD64 installer transaction test passed fresh install, checksum
  rejection, replacement refusal, injected post-backup failure recovery,
  explicit upgrade, PATH idempotence, and installed-version verification.
- The PowerShell network path downloaded, verified, and installed the published
  `v0.1.0` executable into an ignored temporary directory without changing
  PATH or the user's existing installation.
- POSIX execution requires Linux and macOS CI runners because the local Windows
  sandbox cannot start its WindowsApps `bash.exe` shim.
- The disposable TUF rehearsal originally anchored generated metadata to
  2026-10-16 and expired after 24 hours. It now anchors the in-memory repository
  to the current test time while preserving the explicit expired-metadata case;
  no checked-in signed metadata or trust material changed.

## Remaining gate

Local repository tests, vet, module-tidiness checks, vulnerability analysis,
six-target compilation, manifest and checksum validation, documentation-link
validation, and the Windows AMD64 installer transaction pass. After commit and
push are separately approved, run the release-candidate workflow and record the
Linux, macOS, and Windows ARM64 installer results before preparing
`v0.1.1-rc.1` for human testing.

