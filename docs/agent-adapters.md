# Agent Adapter Capability Matrix

## Purpose

Engineering Toolkit models integrations by capability instead of assuming that
all coding agents expose equivalent instruction, skill, command, or permission
systems. The machine-readable catalog is available through `engkit adapters
list`; this document records the human-reviewable boundary.

## Capability matrix

| Capability | Codex | Claude Code | OpenCode |
| --- | --- | --- | --- |
| Project instructions | Native: `AGENTS.md` | Native: `CLAUDE.md` | Native: `AGENTS.md` |
| Project skills | Native: `.agents/skills` | Native: `.claude/skills` | Not claimed by this adapter |
| Project commands | Not claimed by this adapter | Not claimed separately from skills | Native: `.opencode/commands` |
| Spec Kit integration | `codex`, skills under `.agents/skills` | `claude`, skills under `.claude/skills` | `opencode`, release-derived inventory required |
| Approval or tool policy | Runtime-owned | Runtime-owned | Runtime-owned; repository configuration is not generated |

“Not claimed” does not mean that a provider can never expose the capability.
It means Engineering Toolkit has not established a stable, tested contract for
it and will not emulate or generate it silently.

## Inspection behavior

`engkit adapters inspect --provider <provider>` classifies relevant paths as
missing, file, directory, symlink, or other. It does not read provider content,
create directories, invoke a provider CLI, or modify permissions.

An existing path means “configured or shared”, not “owned by Engineering
Toolkit”. Provider roots can contain user-created assets and must be reviewed at
file granularity before any generated integration is applied.

## Rendering boundary

Engineering Toolkit's adapter code does not render provider files. Official
Spec Kit `v1.2.0` now owns the approved Codex `speckit-*` skills installed under
the shared `.agents/skills` root. Any future Engineering Toolkit rendering must
provide, for each generated file:

- a deterministic preview and plan digest;
- the owning adapter and upstream source;
- create, merge, or preserve semantics;
- an exact approval bound to current workspace state;
- verification and removal behavior; and
- tests proving that unrelated files remain byte-for-byte unchanged.

For Claude Code, a future `CLAUDE.md` should be a minimal navigation layer and
must not duplicate canonical requirements or architecture. Codex and OpenCode
already consume the repository-owned `AGENTS.md`, so an adapter must not create
parallel instruction files for them.

## Spec Kit boundary

Spec Kit is upstream workflow infrastructure, not an interchangeable provider.
`engkit speckit plan` therefore emits a non-executable preparation artifact. It
requires an explicit release tag, reports potential path conflicts, preserves
`AGENTS.md`, and requires an isolated exact inventory before repository
initialization or an upgrade can be approved. The current Codex installation is
`v1.2.0`; installing it did not execute its workflows.
