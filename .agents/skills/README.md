# Repository Skills

This directory contains official Spec Kit Codex-facing skills. They are
provider-native external assets, not Engineering Toolkit canonical packages,
because they do not contain an `engkit.skill/v1alpha1` `skill.yaml` manifest.
Engineering Toolkit may discover and report them but must not adopt, edit, or
remove them implicitly.

Future canonical packages use `.agents/skills/SKILL_ID/skill.yaml` plus
`instructions.md`. Provider-facing generated files remain derived output and
require separate ownership and conflict controls.

Add a skill only when a repeatable workflow has demonstrated value beyond repository instructions or a short prompt. Each skill must have a distinct trigger, narrow scope, explicit non-goals, required references, validation, and ownership. It must not silently authorize application changes, Git operations, external writes, credential access, or progression to another milestone.

Before adding or updating a skill:

1. Present its destination, trigger description, files, scope, non-goals, tests, and acceptance criteria.
2. Obtain explicit approval.
3. Use the target agent's official skill format and validator.
4. Confirm it does not duplicate `AGENTS.md`, Spec Kit, or another adapter's assets.
5. Inspect the final diff and report forward-testing limitations.

