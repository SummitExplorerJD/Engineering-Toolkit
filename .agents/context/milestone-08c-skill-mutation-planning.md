# Milestone 08C: Skill mutation planning

## Approved boundary

This milestone implements deterministic JSON plans for local canonical skill
creation, update, and removal. It has no apply path and performs no workspace or
source-package mutation.

## Material facts

- The plan schema is `engkit.skill-mutation-plan/v1alpha1`.
- Source packages are complete local canonical packages supplied explicitly by
  path; create and update validate and snapshot them twice to detect changes
  during inspection.
- Package digests bind sorted directory inventories and every file, including
  `skill.yaml`, to the plan.
- Update requires a strictly higher Semantic Version and safely orders file and
  directory type transitions.
- Create refuses existing destination paths and unsafe shared parents.
- Remove refuses provider-native, external, invalid, and conflicting packages.
- Every plan requires approval but is marked non-executable.
- No dependency, provider, network, credential, script, journal, checkpoint, or
  mutation behavior is introduced by this milestone.

## Remaining gate

`v0.2.0D` must separately define and approve execution-time locking, stable-plan
revalidation, backups, atomic publication, journals, checkpoints, interruption
recovery, and the exact CLI approval interface before any skill mutation can be
applied.
