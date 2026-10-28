# Real-Workspace Context and MCP Contract `v1alpha1`

## 1. Status and authority

This document defines the approved `v0.4.0A` direction, the `v0.4.0B1` index
contract, and the separately approved documentation-only `v0.4.0C1` MCP
expansion contract. B2 now implements the local application operations; C1
fixes how they may be presented without duplicating source excerpts. Its primary
goal is to reduce the amount of repository content consumed by an agent while
preserving source attribution, freshness, and deterministic failure.

It does not authorize Go or configuration changes, a SQLite schema change,
indexing a real project, starting an MCP client or server, attaching an AI
harness, accessing a network, changing Git, or publishing a release.

## 2. Design outcome

The normal retrieval sequence is:

1. inspect context status or a bounded workspace tree without file contents;
2. search a fresh projection and receive small ranked snippets;
3. select one or more exact paths or chunk identities; and
4. read only the bounded indexed excerpts needed for the task.

Returning an entire repository, a whole source root, or a full file by default
is not supported. SQLite accelerates discovery but never owns project truth.

## 3. Source-selection contract

Indexing source code is opt-in. The accepted configuration accepts an ordered set
of normalized workspace-relative source roots. A root cannot be absolute,
empty, linked, or escaping. The normalized `.` root is allowed only when
selected explicitly and remains subject to all mandatory exclusions and limits.
Overlapping roots are deduplicated deterministically.

The exact allowlist, prefix-only user exclusions, path rules, mandatory
exclusions, and limits are fixed in
[`context-index-v0.4.0-v1alpha1.md`](context-index-v0.4.0-v1alpha1.md). They
cover:

- `.git` and other version-control metadata;
- `.engkit/context` and generated Engineering Toolkit state;
- dependency, package-cache, build-output, coverage, and generated-code trees;
- environment files, credential stores, private keys, certificates with private
  material, authentication files, and common secret-bearing names;
- symlinks, junctions, sockets, devices, non-regular files, and binary content;
  and
- any file or aggregate inventory that exceeds the accepted limits.

Workspace exclusions may narrow the inventory but cannot override mandatory
exclusions. Eligibility is evaluated before content is stored. The rebuild
result reports bounded exclusion counts by reason without exposing excluded
content or secret-bearing names.

## 4. Projection and chunk contract

Each accepted file contributes path, artifact kind, language hint, byte size,
modification evidence, SHA-256 digest, and deterministic textual chunks. Each
chunk binds its parent digest, ordinal, line range, content digest, and bounded
text. Chunking is line-oriented and stable for identical input bytes and policy.
No parser, embedding model, remote service, or tokenizer is required.

The accepted hard ceilings are 1 MiB per file, 20,000 candidates, 128 MiB of
eligible source input, 50,000 stored chunks, 8 KiB and 120 lines per chunk, and
a 512 MiB published database. B2 proved the boundary with synthetic or injected
fixtures rather than indexing a real project.

Schema incompatibility does not trigger an in-place migration. Rebuild writes a
complete new database, verifies its schema and inventory fingerprint, and
atomically replaces the prior disposable projection. Failure preserves the
last complete projection but marks it unusable when its freshness evidence no
longer matches canonical input.

## 5. Token-efficiency and representation contract

Deterministic UTF-8 byte counts are the contract because providers use different
tokenizers. A validation report may estimate token reduction only when it names
the estimator; the product does not depend on or persist provider token counts.

Every v0.4 context tool accepts optional `response_mode` with enum `compact` or
`text`; omission selects `compact`.

- In `compact` mode, the complete typed result appears once in
  `structuredContent`. `content` contains exactly one text block of at most 256
  UTF-8 bytes summarizing counts, bounds, and continuation state. It contains no
  snippet or excerpt bytes and is not a JSON serialization of the result.
- In `text` mode, `content` contains exactly one complete JSON text block and
  `structuredContent` is absent. This supports clients that do not surface
  structured tool results without sending the same result twice.
- Errors use one safe text block of at most 1,024 UTF-8 bytes, set `isError`, and
  contain no structured partial result.

The official SDK's typed helper automatically inserts a JSON text fallback when
`content` is unset. C2 therefore retains the SDK-derived input and output
schemas but installs a reviewed low-level handler that validates input, encodes
the typed application result, and selects exactly one representation. It does
not alter protocol framing or implement MCP locally.

Application and protocol budgets are:

| Operation | Default | Application ceiling | Encoded MCP ceiling |
| --- | ---: | ---: | ---: |
| Status | no content | no content | 16 KiB |
| Source inventory | 50 items | 200 items | 64 KiB |
| Search | 5 x 512-byte snippets | 20 items / 16 KiB content | 32 KiB |
| Indexed read | 8 KiB | 32 KiB content | 40 KiB |
| Workspace tree | 100 entries / depth 3 | 500 entries / depth 8 | 64 KiB |

Every result reports applied limits and truncation. Continuation inputs bind the
fresh projection fingerprint and fail after a rebuild. Projection, source, and
chunk digests allow clients to reuse unchanged content.

Token-efficiency validation compares the complete serialized MCP response for
`compact` and `text` modes, proves that neither duplicates excerpt bytes, and
compares staged retrieval with a declared bounded full-file baseline. A smaller
response that omits required evidence does not pass.

## 6. Exact read-only MCP surface

The existing eight-tool inventory becomes twelve tools: four additions plus an
additive expansion of existing `context_search`. All input objects reject
unknown fields. Paths are normalized workspace-relative strings of at most 512
UTF-8 bytes.

### `context_status`

Input contains only optional `response_mode`. The result is
`engkit.context-status/v1alpha1`. It contains no source text, SQL, absolute
database path, or environment value. Encoded output is limited to 16 KiB.

Compact summary form is `context <freshness>: <sources> sources, <chunks>
chunks` plus `; rebuild required` when unavailable.

### `context_sources`

Input fields are optional `path_prefix`, `kind`, `language`, `limit`, `after`,
`projection`, and `response_mode`. `limit` defaults to 50 and is 1 through 200.
`after` requires the exact projection fingerprint from the prior page. The
result is `engkit.context-sources/v1alpha1`, contains metadata only, and is
limited to 64 KiB encoded.

Compact summary form reports returned source count and whether continuation is
available; it contains no paths.

### `context_search`

Input requires `query` of 1 through 512 UTF-8 bytes and accepts optional
`path_prefix`, `kind`, `language`, `limit`, `max_content_bytes`, and
`response_mode`. `limit` defaults to 5 and is 1 through 20.
`max_content_bytes` defaults to 4 KiB and is 1 through 16 KiB. The result remains
the additive `engkit.context-index/v1alpha1` response and is limited to 32 KiB
encoded.

Compact summary form reports result count, applied snippet bytes, and
truncation; it contains no query or snippet text.

### `context_read`

Input requires `path` and exactly one selector: `chunk`, or both `start_line`
and `end_line`. It accepts optional `max_bytes` and `response_mode`. A line range
contains at most 400 inclusive lines. `max_bytes` defaults to 8 KiB and is 1
through 32 KiB. The result is `engkit.context-read/v1alpha1` and is limited to
40 KiB encoded.

Compact summary form reports the path, returned line range, applied bytes, and
continuation state but no excerpt text.

### `workspace_tree`

Input accepts optional `path_prefix`, `max_depth`, `limit`, `after`,
`projection`, and `response_mode`. `max_depth` defaults to 3 and is 1 through 8;
`limit` defaults to 100 and is 1 through 500. `after` requires the exact current
projection fingerprint.

The result schema is `engkit.workspace-tree/v1alpha1`. It contains projection
fingerprint, normalized prefix, applied depth and limit, entries ordered by
path, truncation, and continuation. Each entry contains only `path` and enum
`directory` or `source`. Directories are synthesized from paths already present
in the fresh indexed-source inventory. The tool does not traverse or read the
canonical filesystem and therefore cannot reveal excluded or unindexed names.
Encoded output is limited to 64 KiB.

Compact summary form reports entry count and continuation state; it contains no
paths.

All five tools delegate to application ports. MCP handlers receive neither a
database handle nor unrestricted filesystem access. Index rebuild remains a
specialized CLI operation outside MCP.

## 7. Safety and disclosure boundary

The local server remains standard-input/output only, read-only, tools-only, and
closed to external network access. It does not rebuild indexes, invoke Ollama,
execute skills or plans, access credentials, grant approval, or mutate canonical
or provider-owned files.

Indexed source is untrusted data. Snippets can contain prompt injection even
when they contain no recognized secret. A local MCP conformance client keeps
data on the host, but an AI harness can place returned excerpts in a local or
remote model context. Every real-project harness pilot therefore requires a
separate approval naming the project and disclosure behavior.

## 8. Validation gates

Synthetic conformance must cover eligibility, mandatory exclusions, binary and
linked files, exact chunk stability, rebuild interruption, stale projection,
filters, ordering, pagination, budgets, truncation, cancellation, safe errors,
protocol output isolation, and application-result equivalence.

The local real-project pilot must begin with a read-only inventory packet for
each selected project. Evidence may retain version, platform, policy digest,
projection digest, counts, sizes, durations, query labels, byte budgets,
reduction ratios, and pass/fail outcomes. It retains no source text, snippets,
prompts, model responses, credentials, environment values, or unrelated paths.

## 9. Proposed `v0.4.0C2` implementation packet

C1 does not authorize C2. If separately approved, C2 is limited to these exact
files:

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

The implementation would add the tree application result and reader, synthesize
it through the SQLite source inventory, extend injected MCP dependencies for
status, sources, search, read, and tree, and select the approved response mode
through a low-level tool handler. `go.mod`, `go.sum`, configuration schemas,
SQLite tables, canonical workspace files, and the original eight tool names
remain unchanged.

Focused tests must cover every input schema, unknown fields, defaults, maximum
and over-limit values, result equivalence, both response modes, absence of
excerpt duplication, encoded ceilings, tree synthesis, projection-bound
pagination, stale continuation, cancellation, safe errors, standard-output
isolation, and the unchanged twelve-tool capability inventory. Validation then
runs formatting, `go test ./...`, `go vet ./...`, six CGO-disabled cross-builds,
`git diff --check`, and final changed-file inspection.

Principal risks are clients that ignore structured content, accidental source
duplication through SDK fallback behavior, disclosure of excluded names through
tree synthesis, continuation across rebuilt projections, and regressions in the
existing eight tools. Rollback reverts only the C2 source inventory, restoring
the original eight-tool server. It does not delete or migrate a database and
does not modify canonical project files.

## 10. Deferred capabilities

- MCP-driven index rebuild or any other mutation.
- Automatic, background, or file-watcher rebuild.
- Full-file or arbitrary unindexed file reads.
- Embeddings, semantic vector search, symbol parsers, or tokenizer dependencies.
- MCP resources, prompts, sampling, elicitation, or network transports.
- Ollama or other model access through MCP.
- Provider-specific configuration or parity claims.
- Production TUF activation and self-update.

## 11. Rollback

The C1 gate changes documentation only. Reverting its additions to this
contract, ADR 0020, and their linked requirements, architecture, roadmap, MCP
contract, and milestone context removes the MCP expansion packet without
touching the implemented B2 application/index behavior, canonical files,
SQLite databases, MCP clients, models, providers, Git configuration,
installations, or releases.
