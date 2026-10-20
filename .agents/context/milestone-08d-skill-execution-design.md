# Milestone 08D: Skill execution design

## Approved boundary

This slice defines execution-time approval, locking, durable artifacts,
directory publication, backup, verification, and interruption recovery for
local canonical skill mutations. It changes documentation only; executable
implementation remains a separate approval gate.

## Material facts

- Execution requires an initialized `.engkit` workspace and recomputes the
  applicable plan immediately before exact full-ID approval.
- Publication uses complete same-filesystem directory renames, never a live
  file-by-file merge.
- Recovery state has a dedicated ignored `.engkit/recovery/` owner and is not
  ordinary cache.
- Update and removal backups remain available after successful completion;
  automatic cleanup is outside `v0.2.0D`.
- Recovery is explicit, bound to the original approved plan, and refuses
  unknown content rather than deleting or overwriting it.
- Existing Spec Kit and provider-native skill directories remain external and
  cannot be mutated or adopted by this lifecycle.
- No dependency, network, provider synchronization, remote installation,
  script execution, Git operation, or publication is introduced.

## Remaining gate

Implementation requires a separate exact file and public-CLI approval. It must
include failure-injection tests across every rename boundary and cross-platform
CI evidence before the lifecycle can be called complete.
