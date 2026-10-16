# ADR-0003: Use Selective Capability-Based Agent Adapters

- **Status:** Accepted
- **Date:** 2026-10-15

## Context

Codex, Claude Code, OpenCode, and Spec Kit use different instruction, command, skill, and workflow conventions. A lowest-common-denominator abstraction would hide useful features, while direct provider logic in core workflows would create coupling.

## Decision

Define provider-neutral workflow capabilities in the core and implement selective adapters at the boundary. Each adapter declares the capabilities and artifact types it supports. Unsupported capabilities are reported explicitly rather than emulated silently. Spec Kit is treated as an upstream workflow integration, not as another interchangeable model provider.

## Consequences

- Core plans and approval semantics remain stable across providers.
- Each integration can use native strengths without claiming full parity.
- Capability negotiation and adapter conformance tests are required.
- Provider-specific generated files need ownership, upgrade, and removal metadata.

## Alternatives considered

- **One universal generated format:** rejected because provider semantics and discovery paths differ.
- **Provider logic in command handlers:** rejected because it couples user interface, policy, and integration details.
- **Full parity requirement:** rejected because it would encourage unsafe or misleading emulation.

