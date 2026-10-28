# Milestone 13A Real-Workspace Context Planning

## Approved outcome

`v0.4.0A` defines a documentation-only path to lower agent context consumption
while making SQLite retrieval and MCP useful on explicitly selected real source
projects. Production TUF activation moves to a later release so it retains a
separate trust and operational approval boundary.

The design uses staged retrieval: content-free status and tree metadata,
compact search snippets, and explicit exact indexed excerpts. Deterministic
UTF-8 byte limits are the contract; model-token reduction is approximate
validation evidence and introduces no tokenizer dependency. Stable content
digests support reuse, and C1 must eliminate duplicate structured/text excerpt
payloads where verified client compatibility permits it.

## Current baseline

- SQLite `user_version` is `1` and the public result remains
  `engkit.context-index/v1alpha1`.
- Current candidate discovery covers `AGENTS.md`, `README.md`, configuration,
  `docs`, `.agents`, plans, journals, and checkpoints.
- Only documentation-kind files store searchable content; source trees are not
  currently indexed.
- Search uses bounded case-insensitive substring matching and refuses a missing
  or stale index.
- MCP is stdio-only, tools-only, and read-only. It currently exposes eight
  application operations, including `context_search`, and does not receive a
  raw SQLite connection.
- The existing search result ceiling is 50 items and 1 MiB, which is a safety
  ceiling rather than a token-efficient default.

## Planned gate sequence

1. B1 fixes exact configuration, eligibility, exclusions, schema, limits,
   application results, dependencies, files, tests, risk, and rollback.
2. B2 implements and validates bounded source indexing only after B1 approval.
3. C1 fixes exact MCP schemas and budgets only after the application surface is
   stable.
4. C2 implements synthetic MCP conformance only after C1 approval.
5. D1 inventories and tests each explicitly selected real project with a local
   client and sanitized evidence.
6. D2 optionally attaches one exact model-backed harness under a separate
   disclosure, network, credential, evidence, and cleanup approval.

## Safety boundary

The planning slice authorizes no code, configuration, schema, database,
dependency, real-workspace, MCP runtime, model, provider, network, Git, workflow,
tag, or release change. It does not permit real repository content in a model
prompt or retained evidence.

Source code can contain secrets and prompt injection. Future mandatory
exclusions cannot be weakened by workspace configuration, and `context_read`
must remain limited to content already admitted into the same fresh projection.
MCP-driven rebuild, automatic watching, arbitrary file reads, embeddings, model
access, remote transport, and mutation remain deferred.

## Next approval packet

`v0.4.0B1` must present the exact configuration-schema delta, SQLite replacement
schema, source extension and path policies, hard limits, chunk algorithm,
freshness fingerprint, application result schemas, CLI changes, affected files,
focused and cross-platform tests, resource risks, compatibility impact, and
rollback. It must use synthetic inventories for design measurements and perform
no real-project indexing.

## Rollback

Revert the six `v0.4.0A` documentation changes. No runtime or external state was
changed by this gate.
