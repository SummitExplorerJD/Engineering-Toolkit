# Milestone 09D1: Composition execution design

## Observed facts

- New-project creation can publish a complete project tree, while an existing
  workspace cannot be replaced without taking ownership of unrelated content.
- The current planner treats mismatching existing outputs as blocking and
  already matching files as informational; it has no safe ownership record for
  replacement or removal.
- Successful new projects need durable evidence inside their own `.engkit`
  state, including when evidence was first created in a sibling staging tree.
- Go's portable filesystem surface does not offer a replacing-safe rename API
  for arbitrary files, so implementation must capability-check an atomic
  no-replace publication primitive rather than rely on a preflight race window.

## Accepted boundary

- `v0.2.1D1` is design only; `v0.2.1D2` remains a separate implementation gate.
- The first executor is additive and cannot adopt, replace, or remove existing
  workspace content.
- New-project creation stages an initialized project beside the destination and
  publishes the complete tree by rename. An approved empty destination receives
  a retained exact backup during the transition.
- Existing-workspace composition requires initialization, uses a workspace-wide
  composition lock, publishes canonical skills as complete directories, and
  uses capability-checked no-replace publication for other files.
- Recovery completes exact published output, rolls back only exact known output,
  and refuses changed or ambiguous state without deleting evidence.

## Deferred decisions

- The executor, platform no-replace adapter, failure-injection suite, and CLI
  wiring remain `v0.2.1D2`.
- Replacement, removal, owned upgrades, and transaction cleanup require future
  contracts.
- First-party templates and the pilot profile remain `v0.2.1E`.
- Provider synchronization and Spec Kit mutation remain outside `v0.2.1`.
