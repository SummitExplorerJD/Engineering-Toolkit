# Local Intelligence and MCP Design `v1alpha1`

## 1. Status and scope

This document proposes the `v0.3.0D` and `v0.3.0E` contracts. It records the
security and ownership boundary before dependency selection or implementation.
It authorizes no network access, local model invocation, MCP execution, SQLite
migration, schema publication, or workspace mutation.

## 2. Decision pipeline

```text
canonical files
  -> bounded context port
  -> freshness-checked SQLite retrieval or canonical fallback
  -> deterministic eligibility and policy
  -> optional local adviser ranking and explanation
  -> advisory JSON report
  -> explicit user selection
  -> existing deterministic plan and approval lifecycle
```

The deterministic layer owns correctness. The adviser may reorder already
eligible alternatives or explain evidence; it cannot make an ineligible choice
executable. Non-mutating conveniences such as query classification and result
ranking may use model output automatically because they grant no persistence or
execution authority.

## 3. Context contract

The application context port returns a bounded list of excerpts. Each excerpt
includes a workspace-relative source path, source digest, content kind, bounded
text, and index-freshness evidence. Secret-like files, ignored local state,
credentials, environment variables, Git credential configuration, and
unapproved paths are excluded.

SQLite is one implementation of this port. Stale or unsupported index state is
reported. It may fall back to bounded canonical-file inspection only when the
caller explicitly permits that behavior. No model or MCP handler receives SQL,
a database path, or a database connection.

## 4. Ollama adapter

Proposed configuration extends `.engkit/config.yaml` only after a separately
approved schema decision:

```yaml
intelligence:
  enabled: false
  provider: ollama
  endpoint: http://127.0.0.1:11434
  model: qwen3.5:0.8b
  allowed_models:
    - phi4-mini:latest
    - qwen3.5:0.8b
    - granite4:3b
```

The listed tags are candidates, not guaranteed installed models and not an
implicit default. The adapter accepts loopback endpoints only in the initial
contract. It applies request byte, context item, response byte, and time limits;
supports cancellation; disables tool calls; requests structured output; and
rejects malformed or oversized responses.

It never installs or starts Ollama, pulls a model, follows a redirect to a
non-loopback host, sends credentials, discovers remote services, executes model
output, or silently substitutes a different model. Failure returns a stable
integration diagnostic and leaves deterministic commands usable.

The proposed advisory result schema is `engkit.advice-result/v1alpha1`. It binds
the selected model tag, adviser contract version, prompt-template version,
input excerpt digests, recommendations, reasons, uncertainty, warnings, and a
digest of the raw structured response. It contains no approval field and is not
accepted by an executor as a plan.

## 5. MCP server

The proposed command is:

```text
engkit mcp serve --transport stdio --root PATH
```

The harness launches one process for one explicit root. Standard output is
reserved for protocol messages and standard error for bounded diagnostics. The
initial server opens no socket and exposes no mutation, approval, credential,
model-pull, provider-runtime, or script-execution tool.

Initial tools are proposed as:

| Tool | Delegated application operation |
| --- | --- |
| `engkit_doctor` | Run selected read-only diagnostic probes |
| `workspace_inspect` | Inspect workspace facts |
| `context_search` | Search through the bounded context port |
| `catalog_inspect` | Inspect packaged catalog identity and entries |
| `profiles_recommend` | Produce deterministic compatible recommendations |
| `skills_list` | List canonical skill packages |
| `skills_validate` | Validate canonical packages without execution |
| `plan_inspect` | Parse and summarize an existing supported plan without applying it |

Tool schemas use bounded strings and arrays, reject unknown fields, and return
versioned results. The MCP layer does not duplicate domain policy. Cancellation
propagates through application contexts. A handler cannot change the configured
root through a request path that escapes the validated workspace.

## 6. Deferred MCP capabilities

- Mutation and recovery tools.
- Interactive approval or elicitation.
- Network transports and listening services.
- MCP resources, prompts, subscriptions, or server-initiated actions.
- Direct model access through MCP.
- Provider-specific tool names or claimed capability parity.

Each requires a new threat analysis and accepted decision. If mutation tools are
later added, they must accept exact plan approval and invoke the same executor;
MCP session presence shall never count as approval.

## 7. Validation plan

- Adviser tests with an in-memory fake; core tests require no daemon or model.
- Loopback validation, redirect refusal, deadlines, cancellation, size limits,
  malformed-response handling, and secret-exclusion tests.
- Context freshness and canonical-fallback tests with SQLite optional.
- MCP protocol golden tests proving standard-output isolation and stable tool
  schemas.
- Conformance tests proving each MCP result equals its CLI/application result.
- Tests proving no initial MCP tool mutates files or grants approval.
- Separate, explicitly approved local Ollama inventory and smoke tests for each
  exact candidate tag before claiming compatibility.

## 8. Rollback

The adapters own no canonical state. Disabling intelligence or removing MCP
leaves canonical files, SQLite rebuildability, specialized CLI commands, plans,
journals, checkpoints, and provider projections intact. Any future cached
advisory output shall be ignored and safely removable rather than required for
correctness.
