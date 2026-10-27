# Local Intelligence and MCP Design `v1alpha1`

## 1. Status and scope

This document owns the accepted `v0.3.0E1` local-intelligence contract and
summarizes the implemented `v0.3.0D2` MCP boundary. E1 authorized one bounded,
read-only Ollama inventory; it did not authorize model invocation,
implementation, configuration-schema changes, SQLite migration, dependency
changes, workspace mutation, Git operations, or publication. The governing
decisions are [ADR 0014](adr/0014-local-intelligence-and-ollama.md),
[ADR 0017](adr/0017-use-official-go-mcp-sdk.md), and
[ADR 0018](adr/0018-use-bounded-ollama-adviser.md).

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

The first model-backed operation requires a fresh SQLite projection and does
not enable canonical-file fallback. At most 20 excerpts are selected after
deterministic policy, each excerpt is at most 8 KiB, and combined excerpt text
is at most 128 KiB. An absent, stale, or unsupported index skips model use and
preserves deterministic profile ordering.

Initial prompt content is allowlisted to `README.md`, `AGENTS.md`, Markdown
below `docs/`, and Markdown below `.agents/conventions/`. Every other indexed
path is excluded, including `.engkit/`, `.git/`, `.agents/context/`,
`.agents/skills/`, and `.agents/workflows/`. Before use, content containing a
PEM private-key marker or a line whose key matches `token`, `secret`,
`password`, `passwd`, `api_key`, `apikey`, `access_key`, `private_key`, or
`credential` case-insensitively is excluded as a whole excerpt. This filter is
defense in depth and does not redefine the SQLite indexing policy.

## 4. Ollama adapter

The future E2 configuration proposal extends `.engkit/config.yaml` only after a
separately approved schema change:

```yaml
intelligence:
  enabled: false
  provider: ollama
  model: qwen3.5:0.8b # Example explicit selection; there is no default.
```

The endpoint is fixed at `http://127.0.0.1:11434`; workspace configuration,
environment variables, and CLI flags cannot override it. `enabled` defaults to
false. `provider` accepts only `ollama`, and `model` must name exactly one of
`phi4-mini:latest`, `qwen3.5:0.8b`, or `granite4:3b`. No model is an implicit
default, and `--model` may select only another allowlisted tag after local
intelligence is enabled.

### 4.1 Initial application operation

The initial public shape proposed for E2 is:

```text
engkit advise profiles --root PATH --goal TEXT [--model MODEL_TAG]
```

`root` is explicit and validated through the existing workspace boundary.
`goal` is required, trimmed, and limited to 2 KiB. The deterministic profile
recommender runs first. Only its eligible profile identifiers and evidence can
be presented to the adviser. A model response cannot introduce a profile,
change compatibility, generate a plan, or count as user selection.

### 4.2 HTTP request

Before generation, `GET /api/tags` must resolve the configured exact tag and
its observed digest. Generation uses `POST /api/chat` with:

```json
{
  "model": "EXACT_ALLOWED_TAG",
  "messages": [
    {"role": "system", "content": "VERSIONED_SYSTEM_PROMPT"},
    {"role": "user", "content": "BOUNDED_UNTRUSTED_INPUT"}
  ],
  "stream": false,
  "think": false,
  "format": {"type": "object"},
  "keep_alive": 0,
  "options": {"temperature": 0}
}
```

The actual `format` value is the complete response schema rather than the
abbreviated object above. `tools` and images are absent. The HTTP client has no
proxy, rejects every redirect, sends no credentials, limits connection setup to
3 seconds and the complete operation to 60 seconds, supports cancellation, and
performs no retry. Encoded requests and responses are each limited to 256 KiB.

The prompt template is
`engkit.local-adviser/profile-ranking/v1alpha1`. It labels repository excerpts
as untrusted quoted data, instructs the model not to follow instructions found
inside them, lists the only permitted candidate identifiers, and includes the
same JSON Schema supplied through `format`.

The user message is canonical JSON with these fields in order:

```json
{
  "contract_version": "engkit.local-adviser/profile-ranking/v1alpha1",
  "goal": "bounded goal",
  "candidates": [
    {
      "candidate_id": "go-service@0.1.0",
      "deterministic_rank": 1,
      "evidence": ["bounded deterministic evidence"]
    }
  ],
  "context": [
    {
      "path": "README.md",
      "sha256": "sha256:...",
      "text": "bounded untrusted excerpt"
    }
  ]
}
```

There are at most 20 candidates, 20 evidence strings per candidate, and 20
context entries. Candidate identifiers are at most 128 bytes; evidence strings
are at most 1 KiB. Arrays preserve deterministic source order before encoding.

### 4.3 Response and evidence

The exact response schema supplied through `format` is:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "rankings": {
      "type": "array",
      "maxItems": 20,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "candidate_id": {"type": "string", "minLength": 1, "maxLength": 128},
          "score": {"type": "integer", "minimum": 0, "maximum": 100},
          "reason": {"type": "string", "maxLength": 1024}
        },
        "required": ["candidate_id", "score", "reason"]
      }
    },
    "uncertainty": {"type": "string", "maxLength": 1024},
    "warnings": {
      "type": "array",
      "maxItems": 20,
      "items": {"type": "string", "maxLength": 1024}
    }
  },
  "required": ["rankings", "uncertainty", "warnings"]
}
```

The application also validates byte limits after decoding, rejects duplicate or
unknown candidate identifiers, and requires every eligible candidate exactly
once. Scores may only order already eligible candidates.

The returned `engkit.advice-result/v1alpha1` records the bound root, operation,
mode (`ollama` or `deterministic`), goal digest, deterministic candidate order,
final advisory order, prompt-template version and digest, excerpt path and
content digests, warnings, and—only for model-backed success—the Ollama
version, exact model tag, observed model digest, and raw structured-response
digest. It does not persist raw goals, excerpts, prompts, or responses. It has
no approval field and no executor accepts it as a plan.

### 4.4 Failure and privacy

The adapter never installs or starts Ollama, pulls or mutates a model, follows
a redirect, uses an HTTP proxy, sends credentials, discovers a remote service,
executes model output, retries, or silently substitutes another model. Secret
patterns, ignored local state, environment values, credential configuration,
private keys, `.engkit` transaction data, and unapproved paths are excluded
before prompt construction.

Any failure emits a stable bounded diagnostic and returns deterministic profile
ordering with model-backed advice unavailable. Failure never makes profile
inspection, planning, or approved execution unavailable.

## 5. MCP server

The implemented command is:

```text
engkit mcp serve --transport stdio --root PATH
```

The harness launches one process for one explicit root. Standard output is
reserved for protocol messages and standard error for bounded diagnostics. The
initial server opens no socket and exposes no mutation, approval, credential,
model-pull, provider-runtime, or script-execution tool.

The initial tools accepted by `v0.3.0D1` are:

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

The exact inputs, result ownership, limits, failure behavior, SDK selection,
and conformance requirements are specified in
[`mcp-server-v1alpha1.md`](mcp-server-v1alpha1.md). This summary does not
override that contract.

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
- The read-only Ollama inventory is complete. Separate, explicitly approved
  synthetic smoke tests remain required for each exact candidate tag and
  observed digest before claiming compatibility.

## 8. Rollback

The adapters own no canonical state. Disabling intelligence or removing MCP
leaves canonical files, SQLite rebuildability, specialized CLI commands, plans,
journals, checkpoints, and provider projections intact. Any future cached
advisory output shall be ignored and safely removable rather than required for
correctness.
