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

## D2-I evidence

D2-I completed on Windows `10.0.26220`, x64, without a model session,
authentication, installation, or requested network access:

- Codex `0.163.0-alpha.10.2` was identified by exact path and executable
  digest. Its version and help probes attempted denied `.codex/tmp` alias
  maintenance, and `debug --help` attempted TUI startup. Codex D2-S is blocked
  until a bounded non-mutating discovery surface is accepted.
- Claude Code was not installed and no Claude command ran.
- OpenCode `1.17.13` was identified by exact path and executable digest. With
  update and other background behaviors disabled, its help advertised
  `opencode debug skill` as a local skill-listing diagnostic. The diagnostic
  was not run, and OpenCode D2-S remains unapproved.

## Next approval gate

Review and explicitly approve an OpenCode-only D2-S execution packet containing
the exact disposable workspace setup, `opencode debug skill` invocation,
environment, timeout, evidence, cleanup, and rollback. Codex and Claude Code
remain outside that approval.

