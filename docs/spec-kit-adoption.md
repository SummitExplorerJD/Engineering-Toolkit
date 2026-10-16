# Spec Kit Adoption Plan

## Objective

Adopt the official GitHub Spec Kit workflow without overwriting repository-owned instructions, duplicating canonical requirements, or coupling Engineering Toolkit's runtime to Spec Kit internals.

## Current decision

Official Spec Kit `v1.2.0` is initialized with the `codex` integration and
PowerShell script variant. Engineering Toolkit continues to generate
deterministic, non-executable preparation plans with `engkit speckit plan`.
Initialization does not authorize running a generated workflow.

The isolated Codex rehearsal for `v1.2.0` is recorded in
[Spec Kit v1.2.0 Codex inventory](spec-kit-v1.2.0-codex-inventory.md). It found
30 new files and no modification or deletion of existing files. The same set
was subsequently approved, installed, and verified in the real repository.

## Supported targets

| Provider | Integration key | Provider artifact root | Existing instruction strategy |
| --- | --- | --- | --- |
| Codex | `codex` | `.agents/skills` | Reuse repository-owned `AGENTS.md` |
| Claude Code | `claude` | `.claude/skills` | A future adapter may add a minimal `CLAUDE.md` pointer only after approval |
| OpenCode | `opencode` | `.opencode/commands` | Reuse repository-owned `AGENTS.md` |

The provider root is a review boundary, not an assertion that every path under
it is owned by Spec Kit. Exact ownership comes from the pinned release's
generated-file inventory.

## Ownership boundary

- Engineering Toolkit owns `AGENTS.md`, `docs/`, `.agents/context`,
  `.agents/conventions`, `.agents/workflows`, `.agents/skills/README.md`,
  application code, and `.engkit` runtime contracts.
- Spec Kit owns only the generated files recorded by its two manifests and the
  approved inventory under `.specify` and `.agents/skills/speckit-*`.
- Product requirements remain canonical in the approved Engineering Toolkit documents until a later decision explicitly assigns feature-level specification ownership.
- Provider adapters may point to Spec Kit processes but must not rewrite upstream templates silently.

## Adoption and upgrade procedure

1. Pin and report the intended `specify` version and provenance.
2. Verify prerequisites with read-only commands.
3. Inspect all existing repository paths and applicable instructions.
4. Determine the exact integration key and PowerShell script mode for Windows.
5. Confirm Python 3.11 or newer and generate an `engkit speckit plan` using an
   explicit release tag.
6. Run the displayed command only in an isolated temporary copy to capture the
   exact paths the selected Spec Kit version would create or replace.
7. Compare that inventory with repository-owned paths.
8. Back up or snapshot through normal Git history only after Git operations are separately approved.
9. Present the expected diff, conflicts, removability, tests, and upgrade policy.
10. Obtain explicit approval for initialization.
11. Initialize, inspect every change, and stop before feature implementation.

## Safety constraints

- Do not run `specify init --here --force` merely because the repository is non-empty.
- Do not re-run initialization or upgrade the installed integration without a
  new version-pinned inventory and approval.
- Do not accept a non-interactive default integration implicitly.
- Do not run generated scripts before inspecting them.
- Do not overwrite `AGENTS.md` or canonical project documentation.
- Do not add community extensions, presets, bundles, or catalogs without separate provenance review and approval.
- Do not initialize Git or change branches as a side effect of Spec Kit adoption.

## Verification policy

- `specify check`
- Exact generated-file inventory
- Diff review against the approved path list
- Validation of PowerShell scripts on Windows
- Confirmation that existing files are byte-for-byte preserved unless explicitly approved
- Confirmation that removal restores the pre-initialization repository state

## Open decisions

- Whether Spec Kit feature specifications become canonical or are referenced from Engineering Toolkit plans.
- Which Spec Kit version and upgrade cadence to support.
- Whether to remove the deprecated `speckit-taskstoissues` skill through an
  official, separately approved integration update.

