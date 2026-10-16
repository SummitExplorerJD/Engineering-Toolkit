# ADR-0001: Adopt a Modular Monolith

- **Status:** Accepted
- **Date:** 2026-10-15

## Context

Engineering Toolkit needs clear subsystem boundaries for workspace management, durable execution, provider adapters, indexing, and distribution. It is initially a local CLI with one release lifecycle and no demonstrated need for independently deployed services.

## Decision

Build one Go CLI executable as a modular monolith. Core domain packages define interfaces; infrastructure adapters implement filesystem, SQLite, provider, Spec Kit, operating-system, and distribution concerns. Module boundaries are enforced through package dependencies and tests.

## Consequences

- Installation, local operation, testing, and releases remain simple.
- Transactions and recovery can be coordinated in one process.
- Internal boundaries must be actively maintained to prevent package coupling.
- A future service split remains possible but requires evidence and a new ADR.

## Alternatives considered

- **Microservices:** rejected because deployment and operational complexity do not match the initial local-CLI problem.
- **Single undifferentiated package:** rejected because adapters and persistence would couple directly to policy.
- **Runtime plugin system:** deferred because executing third-party code expands the security model substantially.

