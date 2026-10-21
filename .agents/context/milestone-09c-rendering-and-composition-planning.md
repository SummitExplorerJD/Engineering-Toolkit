# Milestone 09C: Rendering and composition planning

## Observed facts

- The existing canonical skill validator originally required a physical
  package directory, which would have forced planning to create temporary
  files.
- The packaged catalog remains intentionally empty, so public planning commands
  can inspect projects now but cannot select first-party content until the
  separately approved pilot catalog is added.
- Windows sandbox virtual temporary directories can deny directory rename even
  when repository-local paths work; final validation therefore uses a
  repository-local temporary base.

## Implemented boundary

- Template variables resolve by declared type and constraints, and text
  substitution remains one-pass and non-recursive.
- Rendered resources, derived provenance, `skill.yaml`, and package digests are
  produced and validated entirely in memory.
- Profile recommendations report evidence and compatibility without implicit
  selection.
- Project creation and workspace composition produce deterministic
  `engkit.composition-plan/v1alpha1` values with `executable: false`.
- Target inventory rejects links and special files. Existing unowned output
  collisions are blocking; unrelated paths are never removed.

## Deferred decisions

- Composition execution, locks, staging, journals, checkpoints, verification,
  and recovery remain `v0.2.1D`.
- First-party templates and the pilot profile remain `v0.2.1E`.
- Provider-native output and Spec Kit mutation remain outside this milestone.
