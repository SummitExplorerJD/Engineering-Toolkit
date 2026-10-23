# Milestone 10D2: provider native-discovery planning

## Outcome

The D2 approval packet separates local inventory from native provider smoke
execution. No Codex, Claude Code, or OpenCode runtime was invoked while
preparing it, and no provider installation, authentication, credential,
network, configuration, or workspace mutation was authorized.

## Planning boundary

- D2-I resolves exact installed executables, versions, help surfaces, and side
  effects before any native discovery command is selected.
- D2-S is approved independently for each exact provider/version/host tuple.
- A local metadata diagnostic is preferred over a model-backed request.
- Prompt-based discovery remains blocked until its model, network,
  authentication, permission, persistence, and cost behavior are explicit.
- Every run uses one disposable workspace, one canonical pilot, and one
  provider, then verifies byte preservation and bounded cleanup.

## Current state

Codex, Claude Code, and OpenCode executable paths, digests, versions, and exact
discovery commands remain unresolved because the approved planning slice did
not invoke provider runtimes. Every D2-S record is therefore `Not approved`.

## Next approval gate

Review and explicitly approve the D2-I host and exact version/help probes in
[`docs/provider-native-discovery-d2.md`](../../docs/provider-native-discovery-d2.md).
D2-I results must be recorded and reviewed before any D2-S execution packet can
be approved.

