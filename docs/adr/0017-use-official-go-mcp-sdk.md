# ADR 0017: Use the official Go SDK for the read-only MCP server

## Status

Accepted for the `v0.3.0D1` design slice on 2026-10-26. Adding the module,
changing `go.mod` or `go.sum`, implementing or executing the server, configuring
a harness, Git operations, and publication require separate approval.

## Context

ADR 0015 places MCP beside the CLI as a read-only presentation adapter. The
implementation needs correct JSON-RPC framing, protocol negotiation, typed
tool schemas, cancellation, and standard-input/output transport. Implementing
those protocol mechanics locally would create an unnecessary compatibility and
security surface.

The dependency must remain compatible with the repository's Go 1.27.3 policy,
must support the required standard-input/output server boundary, and must not
force use of its optional HTTP, authentication, OAuth, sampling, elicitation,
or remote-client features.

## Decision

Use `github.com/modelcontextprotocol/go-sdk` at the exact version `v1.8.0` for
the first MCP implementation. The module is the official Go SDK, provides the
required server and standard-input/output primitives, and declares Go 1.25.

The adapter shall explicitly advertise tools only, register its fixed tool set
before serving, and use `mcp.StdioTransport`. Protocol handlers shall call
application-owned interfaces and shall not depend directly on SQLite,
filesystem transaction stores, provider projections, or CLI subprocesses.

The version is pinned. A later upgrade requires dependency review and separate
approval. `v0.3.0D2` must inspect the complete module-graph delta, licenses, and
vulnerability results before accepting the dependency change.

## Consequences

- Protocol framing, negotiation, schemas, and cancellation use the maintained
  official implementation.
- Engineering Toolkit retains ownership of tool policy, root confinement,
  output limits, safe errors, and application boundaries.
- The module graph will grow in `v0.3.0D2`; unused SDK features remain
  unconfigured and unimported but their transitive dependency impact must still
  be reviewed.
- Fixed registration and explicit capabilities prevent accidental exposure of
  resources, prompts, logging, elicitation, sampling, or list-change behavior.
- Removing the MCP adapter and dependency removes the feature without a state
  migration because the server owns no canonical state.

## Alternatives considered

- **Implement JSON-RPC and MCP locally:** rejected because it duplicates
  protocol mechanics and increases compatibility and security risk.
- **Wrap `engkit` CLI subprocesses:** rejected because it parses presentation
  output, weakens cancellation, and duplicates typed error translation.
- **Use an unofficial Go library:** rejected because the official SDK covers
  the required transport and server surface without a compensating benefit.
- **Use an HTTP transport initially:** rejected because it introduces binding,
  authentication, origin, lifecycle, and remote-exposure decisions outside the
  approved scope.
- **Track the latest SDK release dynamically:** rejected because builds and
  protocol behavior must be reviewable and reproducible.

## References

- [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Server package documentation](https://go.sdk.modelcontextprotocol.io/server/)
- [SDK protocol and transport documentation](https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/protocol.md)
- [Exact `v1.8.0` module declaration](https://raw.githubusercontent.com/modelcontextprotocol/go-sdk/v1.8.0/go.mod)
