# ADR 0019: Expand bounded context before adding MCP authority

## Status

Accepted for the documentation-only `v0.4.0A` design gate on 2026-10-27.
Implementation, configuration, SQLite schema, real-project validation, harness
attachment, Git operations, and publication require separate approval.

## Context

The first SQLite projection indexes a limited canonical inventory and searches
documentation content. The first MCP server exposes eight useful read-only
application operations, but a real source project still requires a caller to
load files through unrelated mechanisms. That can send substantially more text
to an agent than is needed for one question.

Simply exposing unrestricted file reads or increasing result limits would make
MCP more convenient but would increase token consumption, disclosure, secret,
prompt-injection, and denial-of-service risk. Giving MCP a raw SQLite handle
would also invert the existing application boundary.

## Decision

Expand the application context boundary in stages: explicit source selection,
deterministic bounded chunks in a disposable SQLite projection, compact filtered
search, and exact follow-up reads of already indexed excerpts. Add read-only MCP
tools only after their application contracts, byte budgets, and conformance
tests are independently approved.

Optimize defaults for the smallest useful response. Status, inventory, and tree
operations return metadata only. Search returns a few short snippets. Larger
excerpts require another explicit request and remain subject to a hard byte
budget. UTF-8 bytes and item counts are authoritative; token savings are
measured approximately because no provider tokenizer becomes a product
dependency. Stable digests support client-side reuse, and the protocol must
avoid duplicating the same excerpt across structured and textual representations
when negotiated-client compatibility permits it.

Real projects are first exercised through a local MCP client without a model.
Attaching a model-backed harness is a separate approval because returned source
text can leave the server process and enter a provider context.

Keep MCP read-only. It cannot rebuild the index, mutate canonical state, approve
or execute plans, invoke models, access credentials, or open a network
transport. Keep production TUF activation outside `v0.4.0`.

## Consequences

- Agents can discover relevant project context without receiving whole files or
  trees by default.
- Every excerpt remains attributable to a path, line range, source digest, and
  fresh projection.
- Explicit roots and mandatory exclusions require a future configuration and
  schema decision before source indexing can be implemented.
- SQLite schema replacement is safe because the projection is rebuildable, but
  large projects require bounded rebuild and failure tests.
- Search relevance and byte reduction must both pass; minimizing bytes alone is
  not sufficient.
- Read-only local MCP does not eliminate disclosure risk once an external
  harness consumes its results.

## Alternatives considered

- **Expose arbitrary file reads through MCP:** rejected because it bypasses
  indexing policy, freshness, exclusions, and staged retrieval.
- **Return complete matching files:** rejected because it defeats the primary
  token-efficiency goal and magnifies prompt-injection exposure.
- **Add embeddings immediately:** deferred because it adds model, dependency,
  persistence, determinism, and resource questions before bounded lexical
  retrieval is measured.
- **Add MCP index rebuild now:** deferred because it introduces mutation and
  agent-triggered resource consumption into the first real-project pilot.
- **Combine the work with production TUF activation:** rejected for this release
  because trust activation and repository-context disclosure have independent
  risks, inputs, tests, and rollback paths.

## References

- [Real-workspace context and MCP contract](../context-mcp-v0.4.0-v1alpha1.md)
- [MCP server contract](../mcp-server-v1alpha1.md)
- [Architecture](../architecture.md#25-token-efficient-real-workspace-context-boundary)
- [Requirements](../requirements.md#26-v040a-real-workspace-context-and-mcp-design-acceptance-criteria)

## Rollback

The A gate changes documentation only. Reverting this ADR and its linked
contract and planning additions removes the decision without changing Go,
configuration, SQLite, workspaces, MCP runtime behavior, providers, models,
Git state, or releases.
