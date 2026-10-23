# Milestone 10D: provider-conformance planning

## Outcome

The `v0.2.2D` planning contract separates offline adapter and transaction
conformance from native provider discovery. It validates the immutable
`project-guide@0.1.0` pilot for Codex, Claude Code, and OpenCode without making
provider-native or Spec Kit content canonical.

## Approved D1 boundary

- Create fresh disposable workspaces from `go-service@0.1.0`.
- Exercise public composition and provider-sync CLI surfaces.
- Validate exact `SKILL.md` bytes, ownership, sync, unchanged replay, unsync,
  conflicts, refusal behavior, recovery boundaries, and preservation.
- Run the same native scenario on Windows, Linux, and macOS for AMD64 and ARM64.
- Require plan IDs to repeat within one unchanged workspace, and compare
  rendered bytes, relative paths, findings, and generated-file digests across
  all six targets.
- Invoke no provider runtime, network, credential, skill resource, Spec Kit,
  Git, or release operation.

## Separately gated D2 boundary

Native provider discovery requires an independent packet for each exact Codex,
Claude Code, and OpenCode version. That packet must define installation state,
host support, command, authentication and network behavior, evidence, timeout,
cleanup, and rollback. D1 success grants none of that authority.

## Deferred work

- D1 fixture, test, orchestration, and CI implementation.
- D2 provider-runtime smoke execution.
- Adapter-contract changes caused by later provider documentation.
- Git operations and `v0.2.2` release preparation or publication.

The complete planning contract is
[`docs/provider-conformance-v1alpha1.md`](../../docs/provider-conformance-v1alpha1.md).
