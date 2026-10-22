# Milestone 10A: selective provider-synchronization design

## Approved scope

This slice defines the `v0.2.2` provider-synchronization contract. It changes
documentation only. It does not add a renderer, CLI command, ownership record,
journal, checkpoint, provider file, dependency, network access, Spec Kit
mutation, Git operation, or release behavior.

## Findings

- Codex, Claude Code, and OpenCode all support `SKILL.md`-based Agent Skills,
  but discovery paths and optional metadata remain provider-specific.
- OpenCode now documents native `.opencode/skills` plus compatibility discovery
  for `.agents/skills` and `.claude/skills`; the existing adapter catalog's
  command-only boundary is therefore stale for future synchronization work.
- Codex output shares `.agents/skills` with canonical Engineering Toolkit
  packages and official Spec Kit packages. Only per-file ownership is safe.
- A common `SKILL.md` shape does not imply semantic parity for permissions,
  scripts, dependencies, inputs, outputs, or provider runtime policy.

## Accepted decisions

1. Keep canonical skills as the sole authored source.
2. Select exactly one provider per plan.
3. Use `.agents/skills`, `.claude/skills`, and `.opencode/skills` as the initial
   Codex, Claude Code, and OpenCode output roots respectively.
4. Track generated files individually under `.engkit/providers` with exact
   source and output digests.
5. Make lossy or unsupported plans non-executable.
6. Keep scripts, non-empty permission requests, unresolved dependencies,
   runtime policy, and Spec Kit mutation outside the initial executable scope.
7. Split read-only planning, transactional execution, conformance testing, Git,
   and release publication into separate approval gates.

## Next gate

Prepare the exact `v0.2.2B` read-only implementation packet: CLI contract, Go
module boundaries, plan structs, renderer interfaces, fixtures, tests, file
list, risks, validation commands, and rollback plan. Do not implement it until
that packet is explicitly approved.
