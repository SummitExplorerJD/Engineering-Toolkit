# Milestone 11E1: local-intelligence contract and inventory

## Observed inventory

The approved read-only inventory ran on Windows on 2026-10-26. It contacted
only `http://127.0.0.1:11434/api/version` and `/api/tags`, with proxy use
disabled, a three-second timeout, a one-MiB response limit, and no redirects.
No prompt was sent and no model was loaded, started, stopped, or downloaded.

- Ollama client and daemon version: `0.40.4`.
- Executable SHA-256:
  `96d1fd3377ab61340e18a7be12860baace02df965906461dfb9cdb66b3458824`.
- `phi4-mini:latest` is installed; observed digest
  `78fad5d182a7c33065e153a5f8ba210754207ba9d91973f57dffa7f487363753`,
  size `2491876774` bytes.
- `qwen3.5:0.8b` is installed; observed digest
  `de63045f29755d4721c09d54c9b45c2dadb644f4fd7d604d4cac90d96049971c`,
  size `1322069277` bytes.
- `granite4:3b` is installed; observed digest
  `89962fcc75239ac434cdebceb6b7e0669397f92eaef9c487774b718bc36a3e5f`,
  size `2099521385` bytes.

## Constraints

- Installation and tag presence are inventory facts, not compatibility claims.
- The inventory intentionally omitted unrelated local models and user-specific
  installation paths.
- The official registry listed all three candidate tags when the contract was
  prepared, but registry state and mutable tags are not runtime trust roots.
- Model-backed smoke tests and implementation remain separate approval gates.

## Next evidence gate

An exact smoke packet must bind Ollama `0.40.4`, one candidate tag and observed
digest at a time, one synthetic secret-free prompt, the accepted response
schema, a deadline, resource cleanup, and retained bounded evidence. Passing
one candidate grants no authority to invoke another candidate or workspace
content.
