# Doctor and Unified Initialization Design `v1alpha1`

## 1. Status and scope

The `v0.3.0B` doctor contract is implemented locally. The `v0.3.0C`
unified-init contract remains proposed and requires separate implementation
approval. This document does not authorize dependency changes, workspace
mutation, Git operations, or publication.

## 2. Doctor command

The implemented command is:

```text
engkit doctor [--root PATH] [--scope core|workspace|integrations|all]
```

JSON is the canonical initial output. A later human-readable renderer may consume
the same result but shall not own policy. The result schema is
`engkit.doctor-result/v1alpha1`.

Each finding contains:

- a stable code and scope;
- `pass`, `warning`, `error`, or `unavailable` status;
- whether the checked capability is required for the selected scope;
- bounded, secret-safe evidence;
- an actionable remediation description; and
- references to affected paths using workspace-relative paths where possible.

Implemented probes cover:

| Scope | Checks |
| --- | --- |
| Core | CLI version, operating system, architecture, path primitives, catalog integrity |
| Workspace | root classification, config schema, canonical skill validity, plan/journal/checkpoint consistency, locks and recovery state |
| Context | SQLite presence, schema support, source inventory, freshness, rebuildability |
| Integrations | provider ownership health, Ollama executable presence in `PATH`, and current MCP feature availability |

Doctor is read-only. It does not rebuild context, remove locks, recover a
transaction, rewrite configuration, invoke provider runtimes, start or install
Ollama, pull a model, or contact a non-loopback endpoint. A remediation may name
an existing command, but the user runs and approves that command separately.

The initial implementation does not contact the Ollama endpoint and therefore
does not claim daemon or model availability. Those checks remain in the
separately approved local-intelligence inventory gate.

Exit behavior is proposed as `0` when required checks contain no errors, `1`
when one or more required checks fail, and `2` for invalid command input or an
internal inability to complete the requested diagnostic scope. Optional
`unavailable` findings do not alone produce exit `1`.

## 3. Unified initialization commands

The proposed facade is:

```text
engkit init plan --target PATH [--profile ID@VERSION] [--set NAME=VALUE]
engkit init apply --target PATH [--profile ID@VERSION] [--set NAME=VALUE] --approve PLAN_ID
```

The facade classifies the target and selects exactly one existing lifecycle:

| Observed target | Profile | Selected lifecycle |
| --- | --- | --- |
| Absent or empty | Required | New-project composition |
| Existing, non-empty, uninitialized | Absent | Workspace initialization |
| Existing, initialized | Required | Existing-workspace composition |
| Existing, initialized | Absent | Refuse as already initialized and report inspection commands |
| Existing, non-empty, uninitialized | Present | Refuse the compound request and require two separately approved operations |

The facade emits the selected existing plan without wrapping or changing its
plan ID. Apply repeats classification and delegates to the existing executor,
which rebuilds the plan and validates the exact approval. The facade does not
own a second journal, checkpoint, lock, or recovery mechanism.

Profile selection remains exact and explicit. Detection may recommend profiles
but cannot feed a recommendation into apply without the user supplying that
exact selection. `--set` retains existing typed-variable validation.

## 4. Help and compatibility

`engkit init --help`, `engkit init plan --help`, `engkit init apply --help`, and
`engkit doctor --help` shall succeed without filesystem or integration access.
Existing `workspace` and `project` commands remain supported. The facade is an
additional entry point, not a migration of persisted plans or a replacement for
specialized recovery commands.

## 5. Validation plan

- Table-driven classification tests for absent, empty, non-empty,
  initialized, linked, and invalid targets.
- Golden doctor findings and stable JSON ordering.
- Tests proving every probe is read-only and secret-safe.
- Tests proving facade plans and plan IDs equal the selected existing planner.
- Tests proving apply revalidates changed state and cannot chain lifecycles.
- Windows, Linux, and macOS path and optional-capability tests.

## 6. Rollback

Doctor adds no persisted state. Removing its command restores the prior surface.
Unified init owns no new transaction artifacts, so removing the facade leaves
all specialized commands and existing workspaces usable.
