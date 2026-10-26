# ADR 0016: Unified initialization is a transparent facade over one existing lifecycle

## Status

Accepted for the `v0.3.0C1` design slice on 2026-10-26. Implementation, Git
operations, and publication require separate approval.

## Context

Engineering Toolkit exposes safe but specialized entry points for workspace
initialization, new-project creation, and composition into an initialized
workspace. A user starting a project should not need to choose among those
names before the toolkit has classified the target. A convenience command,
however, could weaken the system if it invents a second plan format, selects a
profile implicitly, or combines initialization and composition under one
approval.

The existing planners and executors already own filesystem safety, immutable
plans, exact approval, durable evidence, verification, and recovery. Unified
initialization must improve discoverability without becoming a competing
transaction coordinator.

## Decision

Add `engkit init plan` and `engkit init apply` as a non-interactive facade that
requires one explicit target. It classifies the target and selects exactly one
existing lifecycle:

- new-project composition for an absent or empty destination with an exact
  profile;
- workspace initialization for an existing uninitialized directory without a
  profile; or
- existing-workspace composition for an initialized workspace with an exact
  profile.

Refuse compound, incomplete, already-complete, unsafe, and ambiguous requests.
In particular, do not initialize and compose a non-empty existing directory in
one operation, do not choose a profile automatically, and do not reinterpret a
partial or invalid `.engkit` directory as uninitialized.

Return the selected specialized plan without wrapping it or changing its plan
ID. On apply, repeat classification, rebuild that plan from current state, and
delegate to the matching executor with exact approval. The facade owns no
persisted schema, transaction evidence, lock, staging state, backup, or recovery
command. Existing specialized commands remain supported and authoritative.

## Consequences

- Users gain one discoverable starting command while plan review remains
  truthful about the operation that will execute.
- A facade plan is interchangeable with the corresponding specialized plan;
  there is no compatibility layer or migration.
- Target or input changes between plan and apply invalidate the approval through
  the existing rebuild-and-compare rule.
- A non-empty uninitialized target that also needs composition requires two
  user-visible plans and approvals.
- Recovery documentation and tooling remain lifecycle-specific.
- Removing the facade cannot strand or invalidate an initialized project.

## Alternatives considered

- **Create one compound initialization transaction:** rejected because it would
  merge independent approval, publication, and recovery scopes.
- **Wrap specialized plans in a new facade schema:** rejected because it would
  create a second plan identity and compatibility burden without adding policy.
- **Infer a profile from repository contents or a local model:** rejected
  because recommendation is not authorization and must not change deterministic
  planning inputs.
- **Provide an interactive wizard first:** rejected because prompts complicate
  automation, reproducibility, help behavior, and exact approval.
- **Replace the specialized commands:** rejected because their explicit forms
  remain useful for automation, diagnosis, and recovery.
