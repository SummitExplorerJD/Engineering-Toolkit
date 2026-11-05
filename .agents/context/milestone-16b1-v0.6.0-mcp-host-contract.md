# Milestone 16B1 context: Codex project MCP host contract

## Decision

`v0.6.0B1` retains explicit CLI context rebuild and selects one create-only,
project-scoped Codex MCP projection. The packet is documentation-only and does
not authorize its implementation or execution.

The supported destination is `<root>/.codex/config.toml` for a Codex trusted
project. Engineering Toolkit may create the complete file only when it is
absent. It does not parse, merge, append, replace, or remove an existing
unowned configuration. This sacrifices automatic adoption in already
configured projects to preserve user settings without adding a TOML parser or
unknown-field round-trip risk.

## Evidence and boundary

Official Codex MCP documentation reviewed on 2026-11-05 confirms project scope
through `.codex/config.toml` and local STDIO configuration through `command`,
`args`, and `cwd`. It also documents tool allowlists and write-sensitive
approval policy. The evidence does not authorize starting Codex, trusting a
project, authenticating, invoking a model, or transmitting project content.

Claude Code, OpenCode, ChatGPT web, plugins, user-global configuration, and
remote executors have no accepted v0.6.0B1 projection.

## Architecture decisions

- New `mcphost` and `mcphostexec` modules remain separate from read-only MCP
  protocol handling and skill-specific provider synchronization.
- Plans bind the explicit normalized root, current executable path and digest,
  host contract, exact output digest, and complete approval identity.
- The rendered Codex entry uses a literal argument array, explicit working
  directory, the exact twelve-tool allowlist, and write-sensitive host approval
  mode. It captures no environment or credential.
- Publication is transactional and create-only. Ownership and digest equality
  are required for later removal or recovery.
- `.codex` becomes mandatory excluded contextual input and the policy identifier
  advances from `engkit.context-policy/v1alpha1` to
  `engkit.context-policy/v1alpha2`. This makes prior indexes stale without
  changing SQLite schema version; rebuilding remains explicit.
- MCP remains read-only and cannot plan, configure, remove, recover, rebuild,
  or start a host.

## Closed B2 inventory

The exact implementation and test files are listed in
`docs/mcp-host-adoption-v0.6.0-v1alpha1.md`. B2 may not add a dependency,
schema version, MCP tool, transport, host, destination, merge behavior,
automatic rebuild, network access, model call, or real-project execution.
Any need to leave that inventory stops implementation and requires a revised
proposal.

## Validation expectation

Synthetic tests must cover deterministic cross-platform TOML rendering,
executable and root binding, exact tool allowlisting, linked and existing path
refusal, approval mismatch, transaction interruption, recovery, ownership
drift, removal, `.codex` exclusion and staleness, plan inspection, CLI output,
and unchanged MCP conformance. Full Go tests, vet, vulnerability analysis,
six-platform builds, `git diff --check`, and changed-file inspection follow the
focused suite.

No B2 test may invoke Codex, access the network, use a real project, or retain
source content. C1 and C2 remain separate execution and disclosure gates.

## Risks and rollback

The most important risks are overwriting user configuration, unsafe path
escaping, executable drift, interrupted multi-artifact publication, ownership
confusion, future tool exposure, and indexing local host settings. The contract
mitigates them with create-only output, strict paths, digest-bound approval,
transactions, exact ownership, an explicit tool allowlist, and `.codex`
exclusion.

Before any commit, B1 rollback removes this context file and the new contract
document and reverts only the B1 additions to requirements, architecture, and
roadmap. No Go, configuration, SQLite, MCP, provider, credential, installation,
or external state requires rollback.

## Approval boundary

The user approved the exact five-file B1 documentation scope after receiving
the proposed commands, Codex-only destination, create-only conflict behavior,
`.codex` exclusion, validation, risks, and rollback. That approval does not
extend to B2 implementation or any execution or Git operation.
