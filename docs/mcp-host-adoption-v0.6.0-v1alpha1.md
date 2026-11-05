# MCP host adoption v1alpha1

## 1. Status and scope

This document fixes the documentation-only `v0.6.0B1` contract for adopting
the existing Engineering Toolkit MCP server in one Codex trusted project. It
does not implement or execute the contract.

The accepted baseline is deliberately narrow:

- one explicit initialized workspace root;
- one host, `codex`;
- project-scoped `.codex/config.toml` only;
- the existing local STDIO server and twelve read-only tools;
- explicit context rebuild and fail-closed freshness; and
- create-only configuration publication with exact ownership.

User-global configuration, existing-file merge, Claude Code, OpenCode,
ChatGPT web, plugins, remote execution, network MCP, model invocation,
automatic rebuild, and real-project validation are excluded.

## 2. Host evidence and compatibility

The reviewed Codex MCP documentation states that trusted projects may scope
MCP servers through `.codex/config.toml`. A local STDIO entry accepts a required
`command` and optional `args`, environment settings, and `cwd`. Codex CLI, the
IDE extension, and the ChatGPT desktop app share Codex configuration, while
ChatGPT web does not read local Codex configuration.

The contract uses only `command`, `args`, `cwd`, `enabled`, `required`,
`enabled_tools`, and `default_tools_approval_mode`. The evidence was reviewed
on 2026-11-05 from the official Codex MCP documentation:

<https://developers.openai.com/codex/extend/mcp>

This is format evidence, not permission to start Codex, trust a project,
authenticate, use a model, or disclose workspace content.

## 3. Accepted lifecycle

1. The user explicitly selects an initialized workspace root.
2. `engkit mcp host plan` validates the root, current executable, target, host
   contract, current ownership, and recovery state without writing.
3. For configure, the planner renders exact TOML in memory. It returns a
   deterministic plan and may display the fragment separately for inspection.
4. The user approves the complete plan identity separately.
5. `engkit mcp host apply` rebuilds the plan from current state and requires an
   exact identity match before a transaction can publish anything.
6. The user explicitly runs `engkit context rebuild --root PATH` when the
   projection is missing, stale, or predates the `.codex` exclusion.
7. A later local protocol pilot may launch the existing server only after its
   own approval. A model-backed Codex pilot requires the separate C2 disclosure
   gate.

Planning, apply, removal, and recovery do not start `engkit mcp serve`, Codex,
Ollama, or any other provider process.

## 4. Exact CLI contract

The only new command forms proposed for B2 are:

```text
engkit mcp host plan --host codex --root PATH --operation configure
engkit mcp host plan --host codex --root PATH --operation remove
engkit mcp host apply --host codex --root PATH --operation configure --approve PLAN_ID
engkit mcp host apply --host codex --root PATH --operation remove --approve PLAN_ID
engkit mcp host recover --host codex --root PATH --approve PLAN_ID
```

All flags are required. Positional arguments and unknown flags fail with exit
status 2. `--host` accepts exactly `codex`; `--operation` accepts exactly
`configure` or `remove`. `--approve` accepts a complete lowercase
`sha256:<64-hex>` plan identity and is not accepted by `plan`. Command results
are JSON on standard output; diagnostics remain on standard error.

`plan` is read-only. `apply` reconstructs the applicable plan from the supplied
host, root, operation, current executable, filesystem, ownership, and recovery
state. `recover` loads the persisted transaction identified by the approval and
does not reconstruct a new plan.

## 5. Plan schema

The plan schema is `engkit.mcp-host-plan/v1alpha1`:

```json
{
  "schema_version": "engkit.mcp-host-plan/v1alpha1",
  "plan_id": "sha256:...",
  "operation": "configure",
  "root": "C:/workspace",
  "host": "codex",
  "host_contract_version": "codex-project-mcp/2026-11-05",
  "launcher": {
    "path": "C:/tools/engkit.exe",
    "sha256": "..."
  },
  "target": {
    "path": ".codex/config.toml",
    "action": "create",
    "before_sha256": "",
    "after_sha256": "..."
  },
  "observations": [],
  "conflicts": [],
  "affected_paths": [],
  "risks": [],
  "required_approvals": [],
  "verification": [],
  "recovery": [],
  "requires_approval": true,
  "executable": true
}
```

Paths use normalized slash-separated absolute or workspace-relative form as
shown. Digests use 64 lowercase hexadecimal characters; the plan identity uses
the `sha256:` prefix. Arrays are present even when empty and are sorted by
stable contract order. The identity is SHA-256 over canonical compact JSON with
`plan_id` omitted. Unknown fields are rejected when plans are inspected or
recovered.

Configure uses action `create`; remove uses `remove`. A conflict makes
`executable` false while preserving observations and remediation. Exact
TOML bytes are intentionally absent from the public plan. The planner returns
them only through an in-memory prepared value to the executor or a bounded
preview renderer.

## 6. Exact Codex projection

The created file contains one table and a terminal newline:

```toml
[mcp_servers.engineering_toolkit]
command = "<absolute-engkit-executable>"
args = ["mcp", "serve", "--transport", "stdio", "--root", "<absolute-root>"]
cwd = "<absolute-root>"
enabled = true
required = false
default_tools_approval_mode = "writes"
enabled_tools = ["engkit_doctor", "workspace_inspect", "context_status", "context_sources", "context_search", "context_read", "workspace_tree", "catalog_inspect", "profiles_recommend", "skills_list", "skills_validate", "plan_inspect"]
```

The renderer uses TOML basic-string escaping for quotation marks, backslashes,
and permitted control escapes after rejecting NUL and other disallowed control
characters. It passes arguments as an array and never constructs a shell
command. The root and executable must be absolute real paths without linked
ancestors. Apply verifies the executable digest again.

The explicit tool allowlist prevents later server additions from becoming
available implicitly. `writes` permits read-only annotated tools without a
write prompt while retaining host protection if a future tool is not marked
read-only. It does not weaken Engineering Toolkit's fixed read-only server.

## 7. Destination, conflicts, and ownership

The only provider destination is `.codex/config.toml` beneath the selected
root. Existing `.codex` may be reused only when it is a real directory. The
target and all existing ancestors must reject symbolic links, junctions, and
non-directory components.

Configure is executable only when the target does not exist. Any existing
file, directory, link, special file, unreadable entry, ownership record, active
transaction, or ambiguous path is a conflict. Engineering Toolkit does not
parse or merge an existing TOML file. The preview can show the complete
fragment for a user-managed edit, but apply remains unavailable.

The ownership schema is `engkit.mcp-host-ownership/v1alpha1` and is stored at
`.engkit/providers/codex/mcp-host.json`. It records the host and contract,
target path and digest, executable path and digest, and configuring plan ID.
It contains no TOML bytes, environment values, credentials, prompts, or model
state.

Remove is executable only when the ownership record validates, the target is a
regular unlinked file, and its digest equals the owned digest. Drift produces a
conflict and preserves both file and record. A successful remove deletes the
ownership record only after target absence is verified. `.codex` is removed
only if the configure transaction created it and it remains empty.

## 8. Transaction and recovery

The executor uses the existing rebuild-and-compare approval rule and a new
MCP-host transaction namespace:

```text
.engkit/plans/mcp-host/codex/<plan-key>.json
.engkit/journals/mcp-host/codex/<plan-key>.jsonl
.engkit/checkpoints/mcp-host/codex/<plan-key>.json
.engkit/recovery/mcp-host/locks/codex.lock
.engkit/recovery/mcp-host/codex/<plan-key>/
.engkit/providers/codex/mcp-host.json
```

Plan and ownership files are immutable JSON, journals are append-only JSONL,
and checkpoints are atomically replaced JSON. A transaction stages the exact
configuration, persists evidence before publication, uses exclusive creation,
flushes file and directory metadata where supported, verifies the target, and
then records ownership and completion.

Recovery accepts only the complete approved plan ID, validates every persisted
artifact and lock, and either completes the same operation or removes only
bytes proven to have been created by that transaction. It never deletes a
changed target, reconstructs a new plan, adopts unowned configuration, or
starts a host.

The execution result schema is
`engkit.mcp-host-execution-result/v1alpha1`. It reports plan ID, operation,
host, target, status, current digest when safe, journal, checkpoint, ownership,
and bounded verification outcomes. It contains no configuration bytes.

## 9. Context exclusion and freshness

`.codex` is added to the mandatory case-insensitive directory exclusions used
before contextual source eligibility. No file below `.codex` contributes a
source, chunk, search hit, tree entry, excluded path, or stored content.

The mandatory exclusion set participates through the context policy identifier,
which advances exactly from `engkit.context-policy/v1alpha1` to
`engkit.context-policy/v1alpha2`. That changes the configuration digest, so an
index built under the earlier policy becomes stale and fails closed. The user
restores availability with:

```text
engkit context rebuild --root PATH
```

SQLite remains `user_version = 2`; no migration, new table, fallback read, or
automatic rebuild is introduced.

## 10. Exact B2 implementation inventory

A later B2 approval may change only these production files:

- `internal/cli/run.go`
- `internal/index/sqlite/chunks.go`
- `internal/mcphost/model.go` (new)
- `internal/mcphost/plan.go` (new)
- `internal/mcphost/render.go` (new)
- `internal/mcphostexec/types.go` (new)
- `internal/mcphostexec/storage.go` (new)
- `internal/mcphostexec/execute.go` (new)
- `internal/mcphostexec/recover.go` (new)
- `internal/planinspect/inspect.go`

It may add or change only these focused tests:

- `internal/cli/run_test.go`
- `internal/index/sqlite/index_test.go`
- `internal/mcphost/plan_test.go` (new)
- `internal/mcphost/render_test.go` (new)
- `internal/mcphostexec/execute_test.go` (new)
- `internal/planinspect/inspect_test.go`
- `internal/mcpserver/conformance_test.go`

Documentation updated with implementation results is limited to this document,
`docs/requirements.md`, `docs/architecture.md`, `docs/roadmap.md`, and one B2
context record. Any additional production or test file, schema, dependency,
host, destination, or behavior requires a revised packet and approval.

No existing provider-skill adapter or executor is extended: its ownership keys,
rendering, and lifecycle are skill-specific. MCP-host modules may reuse common
validation patterns but do not make host state canonical or grant the MCP
server mutation authority.

## 11. B2 synthetic conformance

Focused tests must prove:

- deterministic plan identity and TOML bytes on Windows, Linux, and macOS;
- safe escaping and literal argument separation;
- current executable path and digest binding;
- exact twelve-tool allowlisting and read-only MCP annotations;
- refusal of existing, linked, unowned, unreadable, or drifting targets;
- approval absence, mismatch, stale plan, and concurrent lock refusal;
- atomic create, ownership publication, unchanged replay, exact remove,
  interruption points, and recovery;
- `.codex` exclusion at any depth and fingerprint-driven staleness;
- strict CLI inputs, JSON output, stderr isolation, and exit behavior;
- plan inspection with unknown-field and identity rejection; and
- unchanged existing MCP, context, provider, doctor, and workspace behavior.

Validation runs the smallest package tests first, then `go test ./...`,
`go vet ./...`, the repository vulnerability check, all six supported
CGO-disabled builds, `git diff --check`, and final changed-file inspection.
No test starts Codex, contacts a network, uses a real project, or writes outside
disposable synthetic roots.

## 12. Risks and rollback

The principal risks are accidental overwrite of user TOML, unsafe Windows path
escaping, stale executable binding, transaction interruption, adoption of
unowned bytes, future tool exposure, host-format drift, and unintended indexing
of host configuration. Create-only publication, literal argument arrays,
digest-bound approval, exact ownership, recovery, the tool allowlist, dated
host evidence, and mandatory `.codex` exclusion mitigate them.

B2 rollback removes the new commands and modules, restores the earlier
mandatory exclusion set, and rebuilds tests and binaries. An already created
project configuration may be removed only by the approved remove operation
while its digest matches ownership; otherwise it is preserved for manual
review. SQLite needs no downgrade, but a user may explicitly rebuild the
disposable projection under the restored source policy. No canonical workspace
artifact, model, credential, provider account, or remote system is rolled back.

## 13. Approval boundary

`v0.6.0B1` authorizes documentation only. B2 implementation, configuration
mutation, index rebuild, MCP or Codex execution, real-workspace access,
model-backed disclosure, dependencies, Git operations, workflows, and release
activity each remain separately gated.
