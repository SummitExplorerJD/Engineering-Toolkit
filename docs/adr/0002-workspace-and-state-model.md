# ADR-0002: Use Files as Canonical State with an Optional SQLite Projection

- **Status:** Accepted
- **Date:** 2026-10-15

## Context

Workspaces must be portable, reviewable in Git, recoverable after interruption, and usable without a database. Context queries and relationship discovery may benefit from indexing.

## Decision

Use declarative YAML for human-authored configuration, immutable JSON for approved plans, append-only JSONL for execution journals, and atomically replaced JSON for active checkpoints. Treat these files as canonical. Permit SQLite only as an optional, local, rebuildable projection for context discovery and performance.

Use conventional Go repository directories. Apply `src/` within managed project workspaces only after ecosystem compatibility checks; do not wrap this repository's Go packages in `src/`.

## Consequences

- State is inspectable with standard tools and portable between machines.
- Journals preserve history while checkpoints provide efficient resume state.
- SQLite loss or corruption does not destroy canonical project state.
- Schema versioning, atomic writes, compaction, privacy, and retention require explicit design.
- Derived index freshness must be observable.

## Alternatives considered

- **SQLite as the sole store:** rejected because it weakens portability, diffability, and recovery transparency.
- **Only YAML files:** rejected because append-only events and machine-stable checkpoints have different integrity needs.
- **Only JSON files:** rejected because primary configuration should remain comfortable for human editing.

