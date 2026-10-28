# Milestone 13C1 MCP Expansion Contract

## Accepted boundary

The documentation-only C1 gate fixes the additive read-only MCP surface and its
non-duplicating response convention. The existing eight tools become twelve by
adding context status, source inventory, indexed read, and workspace tree while
expanding context search filters and budgets.

C1 performs no Go, configuration, SQLite, dependency, MCP runtime, real-project,
model, network, Git, workflow, installation, or release operation.

## Representation decision

All five expanded context tools accept optional `response_mode`. Default
`compact` returns the complete application result only as structured content
and a source-free summary no longer than 256 UTF-8 bytes. Compatibility `text`
returns the complete JSON result only in one text block. Safe errors use bounded
text only. Conformance must prove that excerpt bytes are never duplicated.

The pinned official SDK remains responsible for schemas and protocol behavior.
Its typed helper automatically creates a JSON text fallback, so the proposed C2
implementation uses a reviewed low-level tool handler only for input
validation, application invocation, encoded ceilings, and representation
selection.

## Safety and budget outcome

- Status contains no source content and is limited to 16 KiB encoded.
- Source inventory is metadata-only and is limited to 200 items and 64 KiB.
- Search defaults to five 512-byte snippets and is limited to 20 items, 16 KiB
  of content, and 32 KiB encoded.
- Indexed read defaults to 8 KiB and is limited to 32 KiB of content and 40 KiB
  encoded.
- Workspace tree defaults to 100 entries and depth 3, is limited to 500 entries,
  depth 8, and 64 KiB encoded, and derives names only from fresh indexed paths.
- Pagination binds the exact projection fingerprint and fails after rebuild.

MCP remains tools-only, stdio-only, read-only, and without model, provider,
credential, network, rebuild, resource, prompt, sampling, approval, execution,
or recovery authority.

## Proposed C2 inventory

C2 requires another explicit approval and is limited to:

- `internal/contextview/context.go`
- `internal/contextview/context_test.go`
- `internal/contextsearch/search.go`
- `internal/contextsearch/search_test.go`
- `internal/index/sqlite/index.go`
- `internal/index/sqlite/index_test.go`
- `internal/mcpserver/server.go`
- `internal/mcpserver/server_test.go`
- `internal/mcpserver/conformance_test.go`
- `internal/cli/run.go`
- `internal/cli/run_test.go`
- `.agents/context/milestone-13c2-mcp-expansion-implementation.md`

It changes no module version, configuration schema, SQLite table, canonical
workspace file, provider configuration, model behavior, or original tool name.

## Required validation

Focused tests cover schemas, unknown fields, bounds, response equivalence,
compact and text modes, non-duplication, encoded ceilings, tree synthesis,
fresh pagination, stale continuation, cancellation, safe errors, output
isolation, and fixed tool inventory. Full validation adds formatting,
`go test ./...`, `go vet ./...`, six CGO-disabled cross-builds,
`git diff --check`, and changed-file inspection.

## Risks and rollback

Risks are compatibility clients that ignore structured content, accidental SDK
fallback duplication, excluded-name leakage, continuation across projections,
and regression of existing tools. C1 rollback reverts only its documentation.
If C2 is later approved and fails, revert its exact source inventory to restore
the eight-tool server; no database deletion or migration is required.
