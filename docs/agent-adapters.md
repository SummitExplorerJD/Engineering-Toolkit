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
| Project skills | Native: `.agents/skills` | Native: `.claude/skills` | Native: `.opencode/skills`; also discovers compatible `.agents/skills` and `.claude/skills` |
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

The current adapter code remains read-only. The accepted `v0.2.2A` design adds
no renderer or mutation entry point; it defines the contract a later
implementation must satisfy. Official Spec Kit `v1.2.0` owns the approved Codex
`speckit-*` skills installed under the shared `.agents/skills` root. Future
Engineering Toolkit rendering must provide, for each generated file:

- a deterministic preview and plan digest;
- the owning adapter and upstream source;
- create, merge, or preserve semantics;
- an exact approval bound to current workspace state;
- verification and removal behavior; and
- tests proving that unrelated files remain byte-for-byte unchanged.

For Claude Code, a future `CLAUDE.md` should be a minimal navigation layer and
must not duplicate canonical requirements or architecture. Codex and OpenCode
already consume the repository-owned `AGENTS.md`, so an adapter must not create
parallel instruction files for them. Skill synchronization is separate from
project-instruction synchronization and does not create or edit `AGENTS.md` or
`CLAUDE.md`.

The initial derived skill targets are `.agents/skills/<id>` for Codex,
`.claude/skills/<id>` for Claude Code, and `.opencode/skills/<id>` for OpenCode.
Using OpenCode's isolated native root avoids making its compatibility discovery
of `.agents/skills` an ownership claim over the canonical package. These paths
were checked against the providers' official documentation on 2026-10-22:

- [OpenAI Skills](https://developers.openai.com/api/docs/guides/tools-skills)
- [Claude Code Skills](https://code.claude.com/docs/en/skills)
- [OpenCode Agent Skills](https://opencode.ai/docs/skills/)

Provider documentation remains release-sensitive. The adapter contract version
and the evidence date are therefore plan inputs; implementation must update its
conformance fixtures when a provider contract changes.

## Spec Kit boundary

Spec Kit is upstream workflow infrastructure, not an interchangeable provider.
`engkit speckit plan` therefore emits a non-executable preparation artifact. It
requires an explicit release tag, reports potential path conflicts, preserves
`AGENTS.md`, and requires an isolated exact inventory before repository
initialization or an upgrade can be approved. The current Codex installation is
`v1.2.0`; installing it did not execute its workflows.
