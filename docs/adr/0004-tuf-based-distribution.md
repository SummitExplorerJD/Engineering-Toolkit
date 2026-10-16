# ADR-0004: Base Distribution Trust on TUF

- **Status:** Accepted
- **Date:** 2026-10-15

## Context

Engineering Toolkit will eventually distribute executable artifacts and may support updates. Checksums or a single online signing key do not adequately address repository compromise, freeze attacks, rollback attacks, or key rotation.

## Decision

Design release verification around The Update Framework. Separate root, targets, snapshot, and timestamp responsibilities; pin an initial trusted root; enforce signature thresholds, metadata versions, expiration, target length and hashes, and rollback protection. Keep publication and key custody outside the CLI's normal runtime authority.

## Consequences

- Update trust and compromise recovery are explicit from the beginning.
- Release operations require disciplined key custody and metadata publication.
- Expired or inconsistent metadata must fail closed with useful diagnostics.
- Installation rollback behavior is separate from metadata rollback protection and requires its own design.
- A library and repository layout must be approved before implementation.

## Alternatives considered

- **Unsigned downloads:** rejected as unauthenticated.
- **Checksums published beside binaries:** rejected because compromise of the same channel can replace both.
- **Single signing key:** rejected because rotation, delegation, threshold trust, and compromise recovery are insufficient.

