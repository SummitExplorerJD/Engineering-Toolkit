# Engineering Toolkit Repository Instructions

## Purpose

Engineering Toolkit is a modular-monolith CLI written in Go. It manages portable engineering workspaces, durable execution artifacts, optional contextual indexing, and selective integrations with coding agents and Spec Kit.

## Sources of truth

- `docs/requirements.md` owns product requirements and acceptance criteria.
- `docs/architecture.md` owns the current system structure and module boundaries.
- `docs/adr/` owns accepted architectural decisions and their rationale.
- `docs/roadmap.md` owns milestones and approval gates.
- `.agents/conventions/` owns detailed working conventions.
- `.agents/context/` records material, phase-specific discoveries after a phase starts.

Do not duplicate detailed requirements or architecture here. Resolve conflicts in favor of the most specific applicable repository instruction, then the accepted ADRs, architecture, and requirements.

## Working agreement

1. Inspect the repository and applicable instructions before changing files.
2. Preserve existing files and user changes. Never overwrite an existing artifact without inspecting it.
3. Treat planning and implementation as separately approved activities.
4. Before a material change, report scope, affected files, validation, risks, and rollback approach; wait for explicit approval.
5. Stop at the end of the approved milestone. Do not begin the next milestone automatically.
6. Keep technical documentation, ADRs, code identifiers, schemas, and commit messages in English.
7. Keep user-facing CLI text ready for localization; do not embed workflow policy in presentation strings.
8. Do not commit, push, publish, deploy, access credentials, or mutate external systems without separate explicit approval.

## Engineering rules

- Prefer a Go modular monolith: one deployable CLI with explicit internal module boundaries.
- Follow standard Go layout for this repository (`cmd/`, `internal/`, and optional `pkg/`). Use `src/` inside managed project workspaces only when compatible with the target ecosystem.
- Keep YAML human-authored and declarative. Keep plans and checkpoints in JSON, and append-only journals in JSONL.
- Treat filesystem artifacts as canonical. SQLite is an optional, rebuildable contextual index and must not become the sole source of truth.
- Make operations deterministic, idempotent where practical, and recoverable after interruption.
- Keep core domain logic independent of Codex, Claude Code, OpenCode, Spec Kit, terminal UI, SQLite, and distribution mechanisms.
- Integrations are capability-based adapters. Do not claim unsupported parity across providers.
- Distribution and self-update design must use TUF principles and fail closed on invalid metadata or signatures.

## Verification

- Add tests with implementation changes and run the smallest relevant suite first.
- Run formatting, static analysis, unit tests, and integration tests appropriate to the approved scope.
- Record observed commands and results; never claim validation that was not run.
- Run `git diff --check` and inspect the final changed-file list before handoff.

## Approval boundaries

Explicit approval is required before:

- Adding or upgrading dependencies.
- Changing public schemas, CLI contracts, module boundaries, or persistence formats.
- Initializing or upgrading Spec Kit in the repository.
- Creating or changing agent integrations.
- Introducing SQLite migrations or network access.
- Adding release signing, TUF roots, CI/CD, installation, or self-update behavior.
- Changing Git state beyond reading status and diffs.

