# MCP Server Contract `v1alpha1`

## 1. Status and authority

This document defines the approved `v0.3.0D1` contract for the first
Engineering Toolkit Model Context Protocol server and records the additive
documentation-only `v0.4.0C1` expansion. The original implementation shipped
in `v0.3.0`; the C1 additions are not implemented. Configuring a real-project
harness, changing implementation, committing, pushing, and publishing remain
separate approval boundaries.

The server is a read-only presentation adapter over application-owned use
cases. Canonical filesystem artifacts remain authoritative. SQLite remains an
optional, rebuildable projection and is never exposed as a protocol or storage
API.

## 2. Process and command contract

The initial command is:

```text
engkit mcp serve --transport stdio --root PATH
```

- `--root` is required. The command does not default to the current directory.
- `--transport` is required and accepts only the literal `stdio`.
- The root is resolved, canonicalized, and validated once before the protocol
  loop starts. A linked root or a root with a symbolic-link or junction
  ancestor is refused.
- One process serves one root and one client session. A request cannot replace
  or escape the root.
- Standard input and standard output carry newline-delimited JSON-RPC messages.
  Standard output contains no banner, log, progress indicator, or diagnostic.
- Bounded diagnostics are written to standard error. A diagnostic line is at
  most 8 KiB and does not include request bodies, context excerpts, secrets, or
  paths outside the bound root.
- Startup validation failure exits non-zero before protocol operation. After
  protocol startup, request failure is represented by the protocol and does
  not corrupt the stream.
- Request cancellation and deadlines propagate to the delegated application
  operation.
- The server opens no socket and performs no network request.

## 3. Server capabilities

The implementation registers its fixed tool set before serving and advertises
only tool capability. Logging, resources, prompts, subscriptions, completion,
elicitation, sampling, and server-initiated actions are disabled. Because the
tool inventory cannot change during a session, tool-list change notification is
disabled.

Protocol-version negotiation uses the versions supported by the exact selected
SDK release instead of hard-coding one client revision. `v0.3.0D2` must record
the negotiated versions covered by its conformance fixtures.

Every tool is annotated read-only, non-destructive, idempotent, and closed to
the external world. Input object schemas set `additionalProperties` to `false`.
All string limits are measured after UTF-8 encoding.

## 4. Result and error contract

On success, `structuredContent` is the exact versioned application result named
below. The SDK-generated JSON text representation is the compatibility content;
the MCP adapter does not add a success envelope or rename application fields.

Invalid JSON, a schema mismatch, an unknown property, or a missing required
argument is an invalid-parameters protocol error. A domain, policy, workspace,
or optional-capability failure is a tool result with `isError: true` and one
bounded, safe text block. It contains no partial structured result, stack trace,
SQL, database path, request excerpt, credential, or path outside the bound
root. Transport and server faults use JSON-RPC errors.

One complete success result is limited to 1 MiB after JSON encoding. A result
that exceeds the limit fails closed before any partial response is written.

## 5. Initial tools

### 5.1 `engkit_doctor`

Input:

```json
{
  "scope": "all"
}
```

`scope` is optional, defaults to `all`, and accepts `core`, `workspace`,
`integrations`, or `all`. The result is the existing
`engkit.doctor-result/v1alpha1` application result. The operation does not
repair any finding.

### 5.2 `workspace_inspect`

Input is an empty object. The result is the existing
`engkit.workspace-inspection/v1alpha1` application result for the bound root.

### 5.3 `context_search`

Input:

```json
{
  "query": "transaction recovery"
}
```

`query` is required, trimmed, and contains 1 through 512 bytes. The handler
calls an application-owned bounded-context use case; it does not import or
receive a SQLite implementation. The result is the existing
`engkit.context-index/v1alpha1` search response with at most 50 excerpts.

The initial server permits no canonical-filesystem fallback. A missing,
unsupported, or stale optional index returns a stable tool error and does not
make the server unavailable. Index creation or rebuild is outside this tool.

### 5.4 `catalog_inspect`

Input is an empty object. The result is the existing
`engkit.catalog/v1alpha1` embedded-catalog result. The tool performs no remote
catalog discovery.

### 5.5 `profiles_recommend`

Input is an empty object. The result is the existing
`engkit.profile-recommendation/v1alpha1` application result for the bound root.
Recommendations remain deterministic and non-authoritative.

### 5.6 `skills_list`

Input is an empty object. The result is the existing
`engkit.skill-list/v1alpha1` application result for canonical packages only.

### 5.7 `skills_validate`

Input:

```json
{
  "id": "project-guide"
}
```

`id` is optional. When present it must be a normalized canonical skill ID of 1
through 64 bytes. Omission validates all discovered canonical packages. The
result is the existing `engkit.skill-validation/v1alpha1` application result.
Validation does not execute package resources.

### 5.8 `plan_inspect`

Input:

```json
{
  "path": ".engkit/plans/skills/example.json"
}
```

`path` is required, uses slash-separated workspace-relative form, and contains
1 through 512 bytes. It must resolve beneath `.engkit/plans/`, name one regular
`.json` file no larger than 1 MiB, and contain no linked path component. The
parser accepts exactly one JSON document, rejects unknown fields, supports only
known Engineering Toolkit plan schemas, and verifies the schema's plan
identity. It never applies, approves, recovers, or rewrites the plan.

The result is:

```json
{
  "schema_version": "engkit.plan-inspection/v1alpha1",
  "root": "C:/workspace",
  "path": ".engkit/plans/skills/example.json",
  "plan_schema": "engkit.skill-mutation-plan/v1alpha1",
  "operation": "create",
  "plan_id": "sha256:...",
  "approval_required": true,
  "executable": true,
  "identity_valid": true,
  "operation_count": 2,
  "verification": ["..."],
  "recovery": ["..."]
}
```

`executable` is omitted when the source schema has no executable field.
`verification` and `recovery` contain bounded descriptive entries already owned
by the source plan and preserve their source order. Each array has at most 100
entries and each entry has at most 1,024 bytes. An invalid identity produces an
error rather than a result with `identity_valid: false`.

## 6. Application and infrastructure boundary

Handlers depend on application interfaces for doctor, workspace inspection,
bounded context search, catalog inspection, profile recommendation, canonical
skill listing and validation, and supported-plan inspection. The context and
plan use cases are application-owned even when their initial infrastructure is
SQLite or the local filesystem.

Handlers do not invoke the CLI as a subprocess, parse CLI presentation output,
open SQLite directly, traverse provider directories directly, or read
transaction stores outside the delegated use case. Domain policy remains in
the existing application and domain packages.

## 7. Dependency decision

Use the official Go SDK module
`github.com/modelcontextprotocol/go-sdk` pinned to exact version `v1.8.0` in
`v0.3.0D2`. The selected release provides the server, tool registration,
schema handling, protocol negotiation, cancellation, and standard-input/output
transport required by this contract and declares Go 1.25, which is compatible
with the repository's Go 1.27.3 policy.

The implementation shall construct one server with an explicit tools-only
capability set, register the fixed tools, and run `mcp.StdioTransport`. It shall
not import or activate SDK HTTP, OAuth, authentication, sampling, elicitation,
or remote-transport facilities.

Before accepting the dependency change, `v0.3.0D2` must inspect the complete
`go mod tidy` delta, licenses, vulnerability results, and transitive packages.
No floating version, unreviewed upgrade, or workspace-configured SDK selection
is permitted.

## 8. Harness configuration boundary

A harness configuration launches the exact installed `engkit` executable with
the command and explicit root described above. Provider-specific discovery or
configuration files are derived adapter artifacts, not MCP core state. This
contract does not authorize writing Codex, Claude Code, or OpenCode
configuration.

## 9. `v0.3.0D2` conformance requirements

- Golden protocol tests prove initialization, tool listing, schema rejection,
  success results, safe tool errors, cancellation, and clean shutdown.
- Standard-output isolation is tested with diagnostics and failed operations.
- Every tool result is compared with the same application use case invoked
  outside MCP.
- Path traversal, linked components, stale context, unknown plan schemas,
  invalid plan identities, oversized input, and oversized output fail closed.
- Tests prove the advertised capability set contains tools only and cannot
  change during a session.
- Tests prove no tool mutates the workspace, grants approval, executes a
  resource, invokes a provider, accesses a credential, or opens a network
  connection.
- Windows, Linux, and macOS builds remain valid for AMD64 and ARM64.

## 10. `v0.4.0C1` additive context contract

The proposed fixed inventory contains twelve tools. It preserves the original
eight and adds `context_status`, `context_sources`, `context_read`, and
`workspace_tree`; `context_search` gains bounded path, kind, language, result,
and content-byte filters. All five context operations delegate to
application-owned ports, require a fresh projection, and remain read-only.

Every expanded context input accepts optional `response_mode` with default
`compact` and compatibility value `text`:

- `compact` returns the complete typed result once in `structuredContent` and a
  source-free text summary of at most 256 UTF-8 bytes;
- `text` returns the complete JSON result once in text content and omits
  `structuredContent`;
- errors return bounded safe text only, set `isError`, and contain no partial
  structured result.

This explicit selection avoids duplicating excerpts through the official SDK
typed helper's automatic JSON text fallback. The future implementation retains
SDK-derived schemas and protocol handling, then uses a reviewed low-level tool
handler only for validation, application invocation, result encoding, and the
representation choice.

Encoded ceilings are 16 KiB for status, 64 KiB for sources, 32 KiB for search,
40 KiB for read, and 64 KiB for tree. `workspace_tree` is synthesized from
fresh indexed paths and never traverses the filesystem. Index rebuild remains
outside MCP. Exact inputs, outputs, pagination rules, application limits,
conformance requirements, and the proposed C2 inventory are defined in
[`context-mcp-v0.4.0-v1alpha1.md`](context-mcp-v0.4.0-v1alpha1.md).

## 11. Deferred capabilities

- Mutation, approval, and recovery tools.
- Canonical-file fallback for context search.
- Network transports or listening services.
- Resources, prompts, subscriptions, completion, logging, elicitation,
  sampling, and server-initiated actions.
- Direct local-model access and Ollama invocation.
- Provider-specific tool names or claims of provider parity.

Each item requires a separate threat analysis, contract, affected-file list,
tests, rollback plan, and approval.

## 12. Rollback

The future server owns no canonical or persistent state. Removing its command,
adapter, and dependency leaves canonical files, optional SQLite indexes,
specialized CLI commands, plans, journals, checkpoints, and provider
projections intact.
