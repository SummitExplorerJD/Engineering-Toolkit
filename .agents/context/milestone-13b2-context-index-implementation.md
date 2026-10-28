# Milestone 13B2 Context Index Implementation

## Implemented boundary

The separately approved B2 slice implements the optional real-source
configuration, deterministic source eligibility, non-overlapping chunks,
replaceable SQLite v2 projection, FTS5 search, source inventory, indexed reads,
CLI presentation, doctor evidence, and application-owned context contracts.

Existing `engkit.config/v1alpha1` files remain compatible. Source expansion is
disabled when `context` is absent. Existing canonical documentation and
metadata inventory remains available. The SQLite projection advances to
`user_version = 2` only through explicit rebuild and atomic replacement.

## Token-efficiency behavior

- Search defaults to five results, 512 bytes per snippet, and 4 KiB combined.
- Indexed reads default to 8 KiB and cannot exceed 32 KiB.
- Stored chunks do not overlap and bind exact paths, line ranges, source
  digests, content digests, and stable chunk identities.
- Source and projection digests let callers avoid requesting unchanged content.
- The synthetic bounded-budget fixture returned 153 content bytes from a
  21,630-byte declared full-file baseline while retaining the required match.
  That is a 99.29% byte reduction for the fixture; it is not a provider-token
  claim.

## Storage and dependency outcome

SQLite v2 separates metadata, sources, chunks, path-free exclusion summaries,
and an external-content FTS5 index. FTS5 is registered from the already pinned
`github.com/ncruces/go-sqlite3 v0.35.4` module. `go.mod` and `go.sum` did not
change, CGO remains disabled, and no migration treats derived rows as canonical.

Rebuild writes a complete staging database, verifies integrity and size, closes
it, and uses the existing backup-and-rename publication. Missing, stale, or
unsupported indexes fail closed and require an explicit rebuild.

## Safety behavior

Configured roots and exclusions use normalized workspace-relative paths.
Linked roots and entries, case-insensitive path collisions, non-regular files,
invalid UTF-8, NUL content, mandatory state/dependency/generated directories,
known secret path forms, user exclusions, and size limits are rejected or
excluded according to the accepted contract. Excluded paths and content are not
stored; only aggregate reason counts enter SQLite.

The fixed policy is not a secret scanner and does not interpret `.gitignore`.
Project-specific sensitive or ignored trees still require explicit
`exclude_paths` before a later real-project pilot.

## Public local commands

The existing rebuild, status, and search commands remain. Search now accepts
bounded path, kind, language, count, and content-byte filters. Read-only
`context sources` and `context read` commands expose metadata and exact indexed
content without opening arbitrary unindexed files.

No MCP handler changed in B2. The existing MCP `context_search` continues to use
the application search boundary with compact defaults. The broader MCP tool set
remains a separately approved C1/C2 gate.

## Observed validation

- Focused configuration, application-contract, SQLite, CLI, doctor, and local
  intelligence tests passed.
- The full `go test ./...` suite passed.
- `go vet ./...` passed.
- CGO-disabled `go build -trimpath ./cmd/engkit` passed for Windows, Linux, and
  macOS on AMD64 and ARM64. Temporary cross-build outputs were removed.
- The first sandboxed SQLite test attempt failed because Windows denied an
  existing atomic directory rename in the sandbox. The approved out-of-sandbox
  rerun passed.
- One full focused run observed the same transient Windows rename denial in an
  unrelated project-composition test; an immediate bounded rerun of that test
  and the new context lifecycle passed, and the later full repository suite
  passed.

No real project, Ollama model, provider runtime, MCP client, remote network,
dependency operation, Git operation, workflow, installation, or release was
used by validation.

## Remaining gates

C1 must bind the exact five-tool MCP schemas and compatibility representation
before any MCP implementation changes. D1 must inventory each proposed real
project, especially project-specific ignored or sensitive paths, before source
indexing. A model-backed harness remains a later disclosure gate.

## Rollback

Revert the B2 source changes and restore the v1 index implementation. Existing
configurations that omit `context` remain valid. A v2 database is disposable;
after separate deletion authority, remove only `.engkit/context/index.db` and
run the restored explicit rebuild. Canonical project files are unchanged.
