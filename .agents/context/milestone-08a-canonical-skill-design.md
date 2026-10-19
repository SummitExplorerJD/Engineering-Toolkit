# Milestone 08A: Canonical skill design

## Approved boundary

This milestone defines the provider-neutral skill package, ownership classes,
security boundary, and staged `v0.2.0` delivery plan. It does not add schemas,
CLI commands, dependencies, provider synchronization, imports, network access,
or script execution.

## Decisions

- Canonical packages live at `.agents/skills/SKILL_ID/`.
- `skill.yaml` with schema `engkit.skill/v1alpha1` is the explicit canonical
  ownership marker.
- `instructions.md` contains provider-neutral instructional prose.
- Optional resources are limited to declared content under `references/`,
  `scripts/`, `templates/`, and `assets/`.
- Provider-native and Spec Kit directories without the canonical manifest are
  discoverable external state, not implicitly managed state.
- Requested permissions are metadata, not authority to execute a resource.
- Read-only list, inspect, and validate behavior precedes every mutation.
- Creation, update, and removal must use plans, approvals, journals, and
  checkpoints.

## Remaining gates

1. Specify and approve the complete `v1alpha1` schema and validation rules.
2. Approve implementation of read-only discovery, inspection, and validation.
3. Approve mutation-plan contracts separately from mutation execution.
4. Approve local create, update, and removal execution with recovery evidence.
5. Defer provider synchronization to `v0.2.1` and project/workflow composition
   to `v0.2.2`.
