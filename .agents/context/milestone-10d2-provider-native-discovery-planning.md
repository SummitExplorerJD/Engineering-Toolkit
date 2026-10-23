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
  was approved for one D2-S attempt.

## First D2-S attempt

The OpenCode-only attempt created and validated the canonical pilot in a
disposable workspace, but provider synchronization stopped at the required
atomic no-replace capability probe because the host sandbox denied hard-link
creation. OpenCode was not invoked, and no provider output or ownership record
was published.

Recovery with the exact approved synchronization plan completed as
`rolled_back`; provider target, ownership, and lock paths are absent. The
ignored workspace and private evidence remain available for inspection. This
is an environment block rather than a provider pass or failure.

## Next approval gate

Select and review a host where Engineering Toolkit's atomic no-replace
capability is available, then separately approve repeating the OpenCode-only
D2-S attempt there. Codex and Claude Code remain outside that approval.

