# ADR 0018: Use a bounded loopback Ollama adviser

## Status

Accepted for the `v0.3.0E1` contract and inventory slice on 2026-10-26.
Implementation, model invocation, configuration-schema changes, SQLite
migrations, Git operations, and publication require separate approval.

## Context

ADR 0014 established that local intelligence is optional, advisory, and unable
to grant execution authority. The implementation boundary still needed an
exact endpoint, candidate set, request shape, resource limits, and deterministic
failure behavior. Ollama model tags are mutable names, local services are not
implicitly trustworthy, and repository excerpts can contain prompt-injection
content even when they contain no recognized secret.

## Decision

The first adviser uses Ollama's native HTTP API at the fixed endpoint
`http://127.0.0.1:11434`. It uses the Go standard library, disables proxy use
and redirects, sends no credentials, and accepts no endpoint override. The
initial allowlist contains only `phi4-mini:latest`, `qwen3.5:0.8b`, and
`granite4:3b`. No candidate is an implicit default.

The first operation ranks profile candidates that were already declared
eligible by deterministic application policy. The request uses `/api/chat`
with streaming and thinking disabled, temperature zero, no tools, immediate
model unload, and a strict JSON Schema response. The prompt identifies every
workspace excerpt as untrusted data and permits only candidate identifiers
provided by the application.

Before a request, the adapter resolves the configured exact tag through
`/api/tags` and records the observed model digest. It never pulls, creates,
copies, deletes, starts, or substitutes a model. A missing tag, stale or absent
context index, malformed response, unknown candidate, exceeded limit,
cancellation, or timeout returns a stable unavailable result and preserves the
deterministic ordering.

The application result is `engkit.advice-result/v1alpha1`. It records only
digests for the goal, input excerpts, prompt template, and raw structured
response. It is not a plan, has no approval field, and is not accepted by an
executor.

## Consequences

- Correctness and mutation remain available without Ollama, SQLite, or a
  compatible model.
- Loopback traffic and prompt construction are bounded and independently
  testable without loading a real model.
- A documented tag does not prove compatibility; every exact
  model/version/platform tuple needs an approved smoke test.
- `phi4-mini:latest` can resolve to different content over time, so the
  observed digest is evidence and part of every model-backed result.
- Immediate unload reduces persistent resource use but can increase latency.

## Alternatives considered

- **Use an OpenAI-compatible endpoint:** rejected for the first slice because
  it broadens credentials, remote-network, and disclosure policy.
- **Permit configurable loopback ports or hostnames:** deferred until the
  fixed endpoint is proven; hostnames also add name-resolution ambiguity.
- **Allow arbitrary locally installed models:** rejected because presence does
  not establish structured-output behavior or reviewed licensing.
- **Retry or silently switch models:** rejected because it hides resource use
  and changes the observed decision input.
- **Persist raw prompts and responses:** rejected because they can contain
  workspace content and are not required for reproducible evidence.

## References

- [Ollama list-models API](https://docs.ollama.com/api/tags)
- [Ollama chat API](https://docs.ollama.com/api/chat)
- [Ollama structured outputs](https://docs.ollama.com/capabilities/structured-outputs)
- [Phi-4 Mini registry entry](https://ollama.com/library/phi4-mini)
- [Qwen 3.5 tags](https://ollama.com/library/qwen3.5/tags)
- [Granite 4 tags](https://ollama.com/library/granite4/tags)

## Rollback

The E1 slice changes documentation only. Reverting this ADR and its linked
contract removes the decision without changing Ollama, models, SQLite,
canonical files, plans, journals, checkpoints, or provider projections.
