# Milestone 03B Context: Optional Contextual Index

## Approved scope

- Add one SQLite driver suitable for cross-platform builds without CGO.
- Build a derived index from canonical workspace files.
- Report whether the index is fresh or stale.
- Search indexed documentation.
- Demonstrate deletion and full reconstruction without loss of canonical state.

## Dependency decision

Use `github.com/ncruces/go-sqlite3/driver` at module version `v0.35.4` through Go's `database/sql` interface.

Reasons:

- It is CGO-free and does not require a platform C compiler.
- Its upstream test matrix includes Windows, macOS, Linux, and additional platforms.
- Its required dependency surface is smaller than the evaluated pure-Go alternative.
- The dominant `mattn/go-sqlite3` alternative requires CGO and GCC.

The driver uses a Wasm-derived SQLite engine and documents higher per-connection memory than native alternatives. Engineering Toolkit limits the database pool to one open connection for this local index.

## Derived schema

- `metadata`: schema version, source fingerprint, rebuild timestamp, and artifact count.
- `artifacts`: repository-relative path, kind, size, modification time, SHA-256 digest, and optional content.
- `artifacts_kind_idx`: lookup index by artifact kind.
- SQLite `user_version` is `1`; the external contract remains `engkit.context-index/v1alpha1`.

## Privacy boundary

- Markdown files under the approved documentation roots may store searchable content up to 1 MiB per file.
- YAML configuration, plans, journals, and checkpoints store only metadata and hashes.
- Symlinks and non-regular files are not indexed.
- The database and cache are excluded by the generated `.engkit/.gitignore`.
- The index is not a secret store and never becomes an authoritative source.

## Reliability behavior

- Rebuild occurs in a new database in `.engkit/context`.
- The new database is closed before publication.
- Replacement backs up the prior database, publishes the new one, and restores the backup if publication fails.
- Freshness compares a deterministic fingerprint of canonical paths, kinds, hashes, sizes, and modification times.
- Search output explicitly reports freshness.

## Verification evidence

- Tests cover rebuild, read-only status, search, staleness after source modification, replacement, deletion, recovery, and exclusion of sensitive configuration content.
- Windows read-only DSNs use the driver-compatible `file:C:/...` form.
- `go test ./...`, `go vet ./...`, `gofmt`, and module normalization are required before milestone completion.

## Remaining decisions

- Full-text search is deferred; the initial implementation uses case-insensitive SQLite substring matching over documentation content.
- Future schema versions require an explicit migration or rebuild decision. Because the database is derived, deletion and rebuild should remain the preferred migration path.

