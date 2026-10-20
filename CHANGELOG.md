# Changelog

All notable changes to Engineering Toolkit will be documented in this file.
The project follows Semantic Versioning while the public surface evolves.

## [Unreleased]

## [0.2.0] - 2026-10-20

### Added

- Experimental provider-neutral `engkit.skill/v1alpha1` manifest contract.
- Read-only skill discovery, inspection, and validation commands with strict
  resource integrity and ownership classification.
- Deterministic, non-executable create, update, and remove plans bound to exact
  source and target package digests.
- Transactional skill-execution design with exact approval binding,
  same-filesystem publication, durable recovery artifacts, and fail-closed
  interruption handling.
- Approved local `skills create`, `skills update`, `skills remove`, and
  `skills recover` execution with per-skill locks, retained backups, journals,
  checkpoints, and post-publication validation.
- Accepted architecture for offline skill templates and explicit project
  profiles planned for `v0.2.1`.

### Fixed

- Windows 8.3 short-path aliases are no longer misclassified as symbolic-link
  or junction ancestors during approved skill mutations.

## [0.1.1] - 2026-10-19

### Added

- User-scoped PowerShell and POSIX bootstrap installers with exact-version,
  architecture, local-asset, upgrade, and opt-in PATH handling.
- Release packaging and six-platform CI coverage for installation transactions.

### Security

- Mandatory SHA-256 and staged-binary version verification before publication.
- Same-filesystem installation with explicit replacement intent and restoration
  of an existing executable after a failed upgrade.

## [0.1.0] - 2026-10-16

### Added

- Initial modular-monolith CLI foundation.
- Workspace inspection, initialization planning, approval, journaling, and
  checkpoint recovery.
- Optional rebuildable SQLite context index.
- Selective coding-agent capability inspection and Spec Kit planning.
- Internal TUF verified-download core and local staging rehearsal.
- Manual release-candidate build and validation workflow.
- Reproducible six-platform build script and checksum manifest.

### Security

- Fail-closed update metadata and target verification.
- Bounded HTTPS transport, rollback protection, last-known-good metadata, and
  non-executable content-addressed target caching.

[Unreleased]: https://github.com/SummitExplorerJD/Engineering-Toolkit/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/SummitExplorerJD/Engineering-Toolkit/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.1.1
[0.1.0]: https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.1.0
