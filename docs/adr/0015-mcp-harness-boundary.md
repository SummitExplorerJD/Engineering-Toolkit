# ADR 0015: MCP is a read-only presentation adapter over application ports

## Status

Accepted for the `v0.3.0A` design slice on 2026-10-23. Protocol dependency
selection, implementation, harness configuration, execution, mutation tools,
Git operations, and publication require separate approval.

## Context

Codex, Claude Code, OpenCode, and other compatible agent hosts can communicate
with tools through Model Context Protocol. Directly exposing workspace files,
SQLite, or transaction storage would bypass Engineering Toolkit validation and
couple a public protocol to infrastructure details. Allowing mutations in the
first server would also combine protocol correctness, agent authority,
approval, and recovery risks in one step.

## Decision

Implement MCP as a presentation adapter beside the CLI. Protocol handlers call
the same application-owned read-only interfaces and return the same versioned
results. They do not import storage implementations or access SQLite directly.

Use standard input/output as the only initial transport. Reserve standard output
for protocol messages, send diagnostics to standard error, bind one launched
process to one explicit validated workspace root, support cancellation, and
open no listening socket.

Expose only bounded read-only tools for diagnostics, workspace inspection,
context search, catalog inspection, profile recommendation, canonical skill
listing and validation, and plan inspection. MCP session presence grants no
approval. Mutations, recovery, elicitation, remote transports, resources,
prompts, subscriptions, model access, and provider-runtime invocation require
later accepted contracts.

## Consequences

- Agent harnesses gain a structured interface without a second policy engine.
- CLI and MCP behavior can be compared at the application boundary.
- Standard-input/output operation avoids an initial service lifecycle, port,
  authentication, and remote-exposure surface.
- The first MCP release cannot initialize, synchronize, recover, or otherwise
  mutate a project through protocol tools.
- Provider-specific installation and discovery configuration remains separate
  from the MCP protocol implementation.

## Alternatives considered

- **Expose SQLite as an MCP database tool:** rejected because it leaks an
  optional projection and bypasses canonical-source and freshness rules.
- **Wrap CLI subprocesses inside the server:** rejected because parsing its own
  presentation output duplicates orchestration and weakens cancellation and
  typed errors.
- **Ship mutation tools immediately:** rejected because approval semantics and
  hostile-host behavior need a separate threat model and conformance gate.
- **Open an HTTP server initially:** rejected because it adds lifecycle,
  authentication, origin, port, and remote-access decisions unnecessarily.
- **Create provider-specific MCP cores:** rejected because provider discovery
  differences belong in adapters, not the domain or protocol boundary.
