# ADR 0020: Use compact MCP context results

## Status

Accepted for the documentation-only `v0.4.0C1` contract gate on 2026-10-28.
Implementation, MCP execution, real-project indexing, harness attachment, Git
operations, and publication require separate approval.

## Context

The first MCP server returns typed application results in both
`structuredContent` and a JSON text block. That convention maximizes basic
client compatibility, but it duplicates the same result. Once context search
and indexed reads return source excerpts, duplication directly conflicts with
the `v0.4.0` objective of reducing model-context bytes.

The pinned official Go SDK provides derived tool schemas and protocol handling.
Its typed tool helper also inserts a JSON text fallback when text content is
unset, so merely omitting the text field does not produce a single compact
representation. Client support for structured content also cannot be inferred
reliably from the negotiated protocol alone.

## Decision

Every expanded context tool accepts an explicit optional `response_mode`.
Omission selects `compact`, which returns the complete typed result only in
`structuredContent` and a bounded source-free text summary. `text` returns the
complete JSON result only as text and omits `structuredContent` for clients
that need the compatibility representation. Errors contain bounded safe text
only.

Retain the official SDK for protocol behavior and schema derivation. A future
C2 implementation may install a reviewed low-level tool handler solely to
validate the derived input, call the application port, enforce encoded budgets,
and select exactly one result representation. It must not replace SDK protocol
framing or add transport behavior.

Add `context_status`, `context_sources`, `context_read`, and `workspace_tree`,
and enhance `context_search`, for a fixed total of twelve tools. Tree entries
are synthesized from a fresh indexed-source inventory, never from unrestricted
filesystem traversal. All tools remain read-only; index rebuild remains a CLI
operation.

## Consequences

- Default structured clients receive source bytes once rather than twice.
- Compatibility clients must request `response_mode: "text"` explicitly.
- Both modes require conformance tests because smaller output alone does not
  prove equivalent evidence or correct client behavior.
- The low-level handler carries validation and encoding responsibility that the
  typed helper previously supplied automatically.
- Encoded response ceilings supplement application limits and can fail an
  otherwise valid application result closed.
- Projection-bound pagination and indexed-path-only tree synthesis prevent
  cross-rebuild mixing and excluded-name disclosure.

## Alternatives considered

- **Keep duplicate structured and text results:** rejected because every source
  excerpt could consume context twice.
- **Return structured content only for every client:** rejected because some
  clients expose only text tool content.
- **Infer client support automatically:** rejected because the available
  protocol negotiation does not establish how a harness presents tool results.
- **Fork or replace the MCP SDK:** rejected because the official pinned SDK
  already owns protocol and schema behavior; only result representation needs a
  narrower adapter.
- **Build the tree by walking the filesystem:** rejected because it could expose
  excluded or unindexed names and bypass projection freshness.

## References

- [Real-workspace context and MCP contract](../context-mcp-v0.4.0-v1alpha1.md)
- [MCP server contract](../mcp-server-v1alpha1.md)
- [Architecture](../architecture.md#25-token-efficient-real-workspace-context-boundary)
- [Requirements](../requirements.md#28-v040c1-compact-mcp-expansion-acceptance-criteria)
- [ADR 0017](0017-use-official-go-mcp-sdk.md)
- [ADR 0019](0019-expand-bounded-context-for-real-workspaces.md)

## Rollback

Revert this ADR and the linked C1 documentation additions. Because C1 changes
no implementation or persistent state, rollback leaves the existing eight-tool
server, B2 context application behavior, SQLite projection, canonical files,
dependencies, providers, models, Git configuration, installations, and releases
unchanged.
