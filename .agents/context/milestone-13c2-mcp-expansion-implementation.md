# Milestone 13C2 MCP Expansion Implementation

## Implemented boundary

The separately approved C2 slice expands the fixed read-only MCP inventory from
eight to twelve tools. It adds `context_status`, `context_sources`,
`context_read`, and `workspace_tree`, and extends `context_search` with bounded
path, kind, language, result-count, and content-byte filters.

All context handlers delegate to application-owned search and reader ports. The
CLI composition root injects SQLite-backed implementations; MCP handlers do not
receive a database handle or unrestricted filesystem reader. Index rebuild
remains outside MCP.

## Compact representation

The five context operations accept optional `response_mode`:

- default `compact` returns the complete typed result once in
  `structuredContent` and one source-free summary of at most 256 UTF-8 bytes;
- `text` returns the complete JSON result once in text content and omits
  `structuredContent`; and
- safe tool errors return one bounded text block, set `isError`, and contain no
  structured partial result.

Tool-specific encoded ceilings fail closed at 16 KiB for status, 64 KiB for
sources, 32 KiB for search, 40 KiB for read, and 64 KiB for tree. Tests prove
that indexed excerpts are not duplicated between representations.

## Indexed tree and continuation

`workspace_tree` synthesizes directory entries only from normalized paths in a
fresh SQLite source inventory. It never traverses canonical files. Depth,
entry-count, path-prefix, and response budgets are bounded. Tree and source
continuations require the exact current projection fingerprint and fail after a
rebuild.

Search now reports applied result and byte budgets plus truncation. Source
inventory reports its applied page limit and truncation. Indexed reads report
the applied byte ceiling. These are additive application-result fields.

## Observed validation

- Focused context contract, context search, MCP server, SQLite index, and CLI
  suites passed.
- MCP conformance covers the twelve-tool inventory, strict schemas, both
  response modes, excerpt non-duplication, application-result equivalence,
  encoded ceilings, safe root scrubbing, cancellation, and output isolation.
- SQLite tests cover indexed-only tree synthesis, excluded-path absence,
  pagination, and stale projection refusal.
- `go test ./...` passed outside the Windows sandbox.
- `go vet ./...` passed.
- CGO-disabled `go build -trimpath ./cmd/engkit` passed for Windows, Linux, and
  macOS on AMD64 and ARM64. The verified temporary build directory was removed.
- `go.mod` and `go.sum` did not change.

The first sandboxed focused run encountered the known Windows denial for atomic
renames in sandbox-owned temporary directories. The approved out-of-sandbox
SQLite rerun passed. One unrelated CLI project-publication rename was denied
transiently on the first out-of-sandbox run; its immediate isolated rerun and
the subsequent complete CLI and repository suites passed.

No real project, model, Ollama daemon, provider harness, credential, network,
dependency operation, Git mutation, workflow, installation, or release was
used.

## Remaining gates

D1 requires a separate approval that names each real project and inventories
its project-specific exclusions before indexing. Any model-backed harness
remains the later D2 disclosure gate. C2 does not authorize either execution.

## Rollback

Revert the C2 source inventory and this record to restore the original
eight-tool server. No SQLite table or configuration schema changed in C2, so
rollback requires no database deletion or migration and does not modify
canonical project files.
