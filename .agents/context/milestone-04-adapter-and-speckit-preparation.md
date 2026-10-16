# Milestone 04: Adapter and Spec Kit Preparation

## Delivered scope

- A versioned capability catalog for Codex, Claude Code, and OpenCode.
- Read-only inspection of provider-relevant workspace paths.
- Deterministic Spec Kit preparation plans pinned to an explicit release tag.
- Conservative ownership boundaries and mandatory isolated inventory review.

## Provider findings

- Codex discovers `AGENTS.md` hierarchically and supports repository-local
  skills under `.agents/skills`.
- Claude Code uses repository-root `CLAUDE.md` project memory and project skills
  under `.claude/skills`.
- OpenCode V2 recognizes `AGENTS.md`, uses `.opencode/commands` for project
  commands, and exposes runtime tool permissions separately.
- Official Spec Kit declares the integration keys `codex`, `claude`, and
  `opencode`. Codex and Claude use skills-based Spec Kit layouts; the exact
  OpenCode inventory is release-sensitive and must be captured from the pinned
  release before repository initialization.

## Safety boundary

No provider file or Spec Kit artifact is generated in this milestone. The
`speckit` package intentionally exposes planning only and has no apply path.
Every plan requires a pinned semantic release tag, preserves `AGENTS.md`, and
requires a separately approved exact generated-file diff.

## Milestone 4B isolated inventory

The approved isolated rehearsal used official `specify-cli==1.2.0`, Python
3.11.15, the `codex` integration, and PowerShell scripts. It created 30 files,
modified none of the 44 pre-existing files, deleted none, and created no
symlinks. The real repository remained unchanged by Spec Kit.

The full inventory and hashes are recorded in
`docs/spec-kit-v1.2.0-codex-inventory.md`. The deprecated
`speckit-taskstoissues` skill is an external-mutation risk and remains subject
to separate approval. Real initialization remains gated.

## Milestone 4C real installation

The approved 30-file set was installed in the real repository. No pre-existing
file was modified or deleted, and no symlink was created. Twenty-seven stable
files matched the rehearsal byte-for-byte; the two integration manifests and
workflow registry differed only in installation timestamps and passed
structure, version, ownership-map, and referenced-hash validation.

`specify check` completed successfully and found Codex CLI available. No
generated workflow or script was executed. `speckit-taskstoissues` remains
installed for upstream manifest integrity but requires separate explicit
approval before any GitHub mutation. Upgrades and removal remain gated.
