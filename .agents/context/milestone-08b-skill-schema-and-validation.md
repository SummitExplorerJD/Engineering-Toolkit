# Milestone 08B: Skill schema and read-only validation

## Approved boundary

This milestone implements the complete experimental `engkit.skill/v1alpha1`
manifest and read-only list, inspect, and validate commands. It does not create,
update, remove, import, export, install, render, synchronize, or execute skills.

## Material facts

- `gopkg.in/yaml.v3 v3.0.1` is a direct dependency for strict YAML decoding; it
  was already present transitively in `go.sum` before this milestone.
- A valid `skill.yaml` is the only canonical ownership marker.
- Existing official Spec Kit directories contain `SKILL.md` but no canonical
  manifest, so they are classified as external and remain unchanged.
- Every canonical package file except `skill.yaml` is declared and
  integrity-protected by the manifest.
- YAML aliases, unknown fields, multiple documents, oversized manifests,
  escaping paths, symlinks, special files, and integrity mismatches fail closed.
- External packages do not make workspace validation fail.
- Validation does not execute resources, resolve dependencies, use credentials,
  contact networks, or mutate the workspace.

## Verification constraint

Tests that atomically rename directories or open local sockets can fail in the
Codex Windows sandbox with access-denied errors. The new skill-domain tests do
neither and can be run independently. Full CI remains the authoritative
cross-platform integration gate.

## Remaining gates

1. Define and approve immutable plans for skill creation, update, and removal.
2. Implement those plan-only commands without applying mutations.
3. Separately approve guarded local mutation, journaling, checkpoints, and
   recovery.
4. Keep provider synchronization deferred to `v0.2.1`.
