# Milestone 16A `v0.6.0` Real-Workspace MCP Adoption Planning

## Approved planning scope

`v0.6.0A` defines a documentation-only planning boundary for adopting the
existing SQLite v2 context projection and twelve-tool read-only MCP server in
explicitly selected real workspaces. It does not authorize implementation,
workspace configuration, indexing, database rebuild, MCP execution, provider
configuration, model invocation, network access, dependency changes, Git
operations, workflows, installation, tagging, or publication.

The owning documents are [requirements](../../docs/requirements.md),
[architecture](../../docs/architecture.md), and the
[roadmap](../../docs/roadmap.md). Existing implementation contracts remain in
the [context-index specification](../../docs/context-index-v0.4.0-v1alpha1.md)
and [MCP context specification](../../docs/context-mcp-v0.4.0-v1alpha1.md).

## Authoritative baseline

The planning baseline is published `v0.5.0` plus the completed `v0.5.1`
validation line:

- canonical workspace files remain authoritative;
- SQLite `user_version = 2` is optional, local, disposable, freshness-checked,
  and rebuilt rather than migrated as authoritative state;
- `engkit context rebuild`, `status`, `sources`, `search`, and `read` expose the
  accepted explicit-root context lifecycle;
- `engkit mcp serve --transport stdio --root PATH` exposes exactly twelve
  read-only tools through application ports;
- `compact` and `text` protocol modes avoid duplicate excerpt representation;
- mandatory and user exclusions, symlink refusal, size limits, stale refusal,
  and bounded indexed reads are implemented and synthetically qualified; and
- a local real-workspace protocol pilot already demonstrated staged byte
  reductions without attaching a model-backed host.

This gate does not reopen the validated `v0.5.0` bytes or create a `v0.5.1`
release. It plans a later minor release because adoption contracts may add new
commands, configuration, provider projections, or application operations.

## User outcome

A user should be able to select one real workspace deliberately, understand
what will be indexed and excluded, establish or refresh the disposable
projection, connect a supported local MCP host through an inspectable
configuration, verify protocol health, and use staged retrieval without
returning whole files or repository trees unnecessarily.

The workflow must remain useful without Ollama, a model-backed host, remote MCP
transport, automatic background processes, or provider runtime execution.

## Proposed adoption lifecycle

The later B1 contract should evaluate this lifecycle without treating the
names below as accepted CLI syntax:

1. inspect one explicit root and summarize eligible source roots, mandatory
   exclusions, user exclusions, projected limits, and existing freshness;
2. preview any required workspace configuration or host projection without
   writing it;
3. bind an exact approval to any accepted configuration mutation;
4. run an explicit CLI projection rebuild and verify publication, integrity,
   inventory, and freshness;
5. preview and, under a separate approval, publish one provider-owned MCP host
   configuration that launches the existing stdio server for the exact root;
6. validate locally with content-free status, tree, bounded search, and exact
   read requests; and
7. stop before any model-backed host unless a separate disclosure packet is
   approved.

Planning must decide whether existing workspace initialization and provider
transaction machinery can be reused without weakening module boundaries. It
must not invent a second approval mechanism or allow host configuration to
become canonical workspace state.

## Freshness decision inventory

Three strategies require explicit comparison:

| Strategy | Benefit | Principal risk | Planning disposition |
| --- | --- | --- | --- |
| Explicit CLI rebuild | Deterministic, bounded, and compatible with the current read-only MCP contract | Users must notice and resolve stale state | Compatibility baseline |
| Approval-bound rebuild operation | Can make adoption more guided while retaining explicit authority | Adds a public plan, executor, transaction, recovery, and schema surface | Candidate for B1 design only |
| Automatic or watched rebuild | Lowest manual friction | Hidden mutation, background lifetime, races, resource use, and privacy surprises | Deferred unless independently designed |

`v0.6.0A` does not accept MCP-triggered rebuild, automatic refresh, filesystem
watching, polling, or background services. Missing or stale projections
continue to fail closed and identify the explicit CLI rebuild path.

## Host-configuration boundary

A future host adapter may describe the exact executable, arguments, stdio
transport, workspace root, and provider-owned destination needed for Codex,
Claude Code, or OpenCode. B1 must define:

- supported host and version evidence;
- exact generated paths and ownership records;
- preservation and conflict behavior for user- or provider-owned content;
- preview, approval, publication, verification, removal, and recovery;
- argument-safe process configuration with no shell interpolation;
- absence of embedded credentials, tokens, prompts, or environment capture;
  and
- behavior when the host format cannot express the accepted root or command
  safely.

Planning and configuration publication never start the host, authenticate,
invoke a provider runtime, or send MCP results to a model.

## Efficiency and disclosure evidence

The authoritative efficiency measurements are:

- UTF-8 bytes and item counts for each MCP result;
- applied content bytes after result limits;
- maximum line and response sizes;
- projection and chunk identities used to avoid repeated retrieval;
- a declared bounded full-file baseline for the same task; and
- the reduction ratio between staged retrieval bytes and that baseline.

No tokenizer dependency or exact token claim is accepted in A. A later gate may
propose one only with an exact tokenizer, version, input representation, test
vectors, and compatibility policy.

Evidence tiers are:

1. synthetic protocol conformance with no real workspace content;
2. local real-workspace protocol validation that retains metrics but no source
   content or raw frames; and
3. optional model-backed validation, which may disclose selected MCP results
   and therefore requires a separate named-project and provider approval.

Retained evidence contains no source text, snippets, prompts, model responses,
credentials, authentication state, environment values, private paths, raw MCP
frames, SQLite files, or unrelated roots.

## Threats and stop conditions

Later packets must fail closed on an ambiguous root, linked boundary, unknown
configuration field, mandatory-exclusion conflict, unsupported schema, stale
projection, unsafe provider destination, unowned conflict, response-budget
violation, protocol output contamination, or disclosure outside the approved
tier.

A pilot failure is evidence, not implementation authority. It must be reduced
and classified as a product defect, test defect, environment limitation, host
compatibility condition, or external-service condition before any correction
is proposed.

## Later gate requirements

`v0.6.0B1` must present exact public interfaces, versioned schemas, affected
source and test files, module ownership, dependency impact, compatibility,
focused and full validation, risks, and rollback. SQLite schema changes, MCP
mutation, provider configuration writes, and new dependencies each require an
explicit independent decision rather than being inferred from this packet.

`v0.6.0C1` must name one real workspace, exact local client, bounded queries,
disclosure limits, evidence paths, cleanup, and stop conditions. `v0.6.0C2`
must additionally name the provider, exact host version, authentication and
network behavior, content allowed to leave the machine, and revocation or
cleanup procedure.

## Validation of this planning gate

Planning review must confirm the exact four-file documentation inventory,
cross-document links and terminology, the distinction between proposals and
accepted behavior, and the absence of execution authority. Run
`git diff --check` and inspect the final changed-file list. Product tests are
not required because this gate changes no executable code.

## Risks and rollback

The main planning risks are accidental authorization of MCP mutation,
automatic indexing, provider configuration writes, or real-project disclosure.
The packet mitigates them through separate gates, explicit roots, fail-closed
freshness, preview-before-mutation, and named disclosure tiers.

Before any commit, rollback reverts only this context file and the associated
additions to `docs/requirements.md`, `docs/architecture.md`, and
`docs/roadmap.md`. No code, schema, configuration, SQLite projection,
dependency, installation, credential, provider state, or external system
requires rollback.
