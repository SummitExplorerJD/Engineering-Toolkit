# Milestone 13B1 Context Index Schema Packet

## Approved outcome

The documentation-only B1 packet fixes the optional real-source configuration,
eligibility policy, SQLite v2 schema, chunk identity, freshness, application
results, CLI proposal, exact B2 file inventory, validation, risks, and rollback.
It implements nothing and performs no real-project indexing.

## Compatibility decisions

- `engkit.config/v1alpha1` remains the configuration identifier. An absent
  `context` mapping preserves current documentation-only behavior.
- SQLite advances from `user_version = 1` to `2` only by explicit full rebuild.
  A v1 index is preserved until successful v2 atomic publication.
- Current public search fields remain required. Source inventory and exact read
  use separate versioned application results.
- The existing `github.com/ncruces/go-sqlite3 v0.35.4` module includes FTS5;
  B2 may activate that extension without changing `go.mod` or `go.sum`.

## Resource and token decisions

The hard source ceilings are 20,000 candidates, 1 MiB per file, 128 MiB total,
50,000 chunks, 8 KiB and 120 lines per chunk, and 512 MiB for the published
database. Chunks never overlap. Search defaults to five 512-byte snippets with
a 4 KiB combined content budget; explicit indexed reads default to 8 KiB and
cannot exceed 32 KiB.

Stable source and chunk digests support reuse without resending unchanged text.
Token savings remain approximate evidence derived from deterministic byte
comparisons; no tokenizer or model dependency is introduced.

## Security decisions

Only explicit roots are expanded. Fixed allowlists, built-in directory and
secret path exclusions, user prefix exclusions, linked-path refusal, regular
file checks, UTF-8 validation, NUL refusal, and resource ceilings apply before
content enters SQLite. Excluded paths and content are not retained; only counts
by stable reason are stored.

The fixed path policy cannot detect every project-specific secret. B2 must make
that limitation explicit and cannot claim that indexing is a secret scanner.
Git ignore files are not interpreted, and Git is not invoked. Project-specific
local or ignored trees require explicit `exclude_paths` entries.

## B2 boundary

The only proposed implementation files are the 17 paths enumerated in
`docs/context-index-v0.4.0-v1alpha1.md`. MCP code, dependency manifests,
catalogs, workspace initialization output, provider integrations, Ollama, TUF,
release workflows, and real-project validation remain outside B2.

B2 must start with focused synthetic tests, preserve the last complete index on
every failure, and prove format, vet, full tests, six-target CGO-disabled builds,
and token-efficiency byte ratios before handoff.

## Preserved approval boundaries

B1 authorizes no code or configuration change, SQLite write or deletion,
real-project access, MCP execution, model or provider call, network operation,
Git mutation, workflow, tag, installation, or release action. B2 requires a new
exact implementation approval.

## Rollback

Revert the B1 contract and its linked requirements, architecture, roadmap, MCP
contract, and milestone context. No runtime, database, project, or external state
was changed by B1.
