# Engineering Toolkit

Engineering Toolkit is a planned cross-platform command-line application for creating and operating durable engineering workspaces. It is designed as a modular monolith in Go, with explicit approval gates, inspectable artifacts, optional contextual indexing, selective coding-agent adapters, Spec Kit interoperability, and a secure distribution path based on The Update Framework (TUF).

## Status

Milestones 1 through 4 and the internal Milestone 5A verified-download core are
implemented. Milestone 5B production activation planning is prepared but
blocked pending an approved bootstrap root, repository endpoints, and hosting
controls. CI and the Milestone 6A release workflow validate six targets across
Windows, Linux, and macOS on `amd64` and `arm64`. The CLI can inspect and
initialize workspace state, rebuild the optional context index, describe
supported agent capabilities, and prepare non-executable Spec Kit plans.
Official Spec Kit `v1.2.0` is installed for Codex; its generated workflows have
not been executed. The update core is not exposed through the CLI and has no
production trust root or endpoint, so `v0.1.0` is a manual-update-only release.
Production TUF activation, Windows Authenticode signing, macOS signing and
notarization, installers, and self-replacement are deferred.

## Design principles

- Files remain the durable, portable source of truth.
- Human-authored configuration is declarative YAML.
- Execution plans and checkpoints are JSON.
- Append-only execution journals are JSONL.
- SQLite, when enabled, is a rebuildable context index rather than authoritative storage.
- Provider-specific behavior lives behind capability-based adapters.
- Material changes require human approval before execution.
- Updates are verified, rollback-aware, and designed around TUF.

## Documentation map

- [Requirements](docs/requirements.md)
- [Architecture](docs/architecture.md)
- [Roadmap](docs/roadmap.md)
- [Agent adapter matrix](docs/agent-adapters.md)
- [Spec Kit adoption](docs/spec-kit-adoption.md)
- [Spec Kit v1.2.0 Codex inventory](docs/spec-kit-v1.2.0-codex-inventory.md)
- [Secure distribution plan](docs/secure-distribution-plan.md)
- [Milestone 5A implementation packet](docs/milestone-05a-implementation-packet.md)
- [Milestone 5B production activation packet](docs/milestone-05b-activation-packet.md)
- [Distribution threat model](docs/security/distribution-threat-model.md)
- [Installation](docs/installation.md)
- [First release checklist](docs/release-checklist.md)
- [v0.1.0 release notes](docs/release-notes/v0.1.0.md)
- [Security policy](SECURITY.md)
- [Architecture decisions](docs/adr/)
- [Agent conventions](.agents/conventions/)

## Repository layout

Repository Go code uses conventional `cmd/` and `internal/` boundaries. The
Project Workspace Standard may use `src/` inside workspaces managed by the CLI
when the target ecosystem supports it.

## Current CLI surface

```text
engkit version [--json]
engkit workspace inspect [--root PATH]
engkit workspace plan-init [--root PATH]
engkit workspace init [--root PATH] --approve PLAN_ID
engkit context rebuild [--root PATH]
engkit context status [--root PATH]
engkit context search [--root PATH] QUERY
engkit adapters list
engkit adapters inspect [--root PATH] --provider PROVIDER
engkit speckit plan [--root PATH] --integration PROVIDER --version RELEASE_TAG
```

The context database is derived, local, and ignored by generated `.engkit/.gitignore` rules. Deleting it does not remove canonical project artifacts; run `context rebuild` to recreate it.

Adapter inspection and Spec Kit planning are read-only. A Spec Kit plan pins an
official release source and reports potential ownership conflicts, but it is
deliberately marked non-executable and does not invoke `uvx` or `specify`.
The installed Spec Kit skills remain subject to `AGENTS.md` approval boundaries;
their presence does not authorize workflow execution or external mutations.

## Contributing

Read `AGENTS.md`, the applicable conventions, requirements, architecture, and accepted ADRs before proposing changes. Planning, implementation, Git operations, releases, and external mutations have separate approval gates.

