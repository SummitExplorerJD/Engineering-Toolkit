# Doctor and Unified Initialization Design `v1alpha1`

## 1. Status and scope

The `v0.3.0B` doctor contract and `v0.3.0C2` unified-init facade are implemented.
The facade follows the accepted `v0.3.0C1` contract and reuses existing planners
and executors without adding a persisted schema or transaction mechanism.
`v0.5.0A` additionally fixes the documentation-only contract for an optional
human doctor summary. This document does not authorize implementation,
dependency changes, Git operations, or publication.

## 2. Doctor command

The implemented command is:

```text
engkit doctor [--root PATH] [--scope core|workspace|integrations|all]
```

The planned additive presentation form is:

```text
engkit doctor [--root PATH] [--scope core|workspace|integrations|all] [--summary|-s]
```

JSON remains the canonical default output. `--summary` and `-s` select a
human-readable renderer that consumes the same completed result and does not
own policy. The result schema remains `engkit.doctor-result/v1alpha1`; no second
schema or alternate probe path is introduced.

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

Doctor returns `0` when required checks contain no errors, `1` when one or more
required checks fail or an operational error prevents inspection, and `2` for
invalid command input or scope. Optional `unavailable` findings do not alone
produce exit `1`.

### 2.1 Human-summary presentation

The summary renderer emits one bounded line per existing finding in the same
deterministic order as the canonical result. Its status mapping is fixed:

| Canonical status | Human label |
| --- | --- |
| `pass` | `[OK]` |
| `warning` | `[WARN]` |
| `error` | `[ERROR]` |
| `unavailable` | `[N/A]` |

Each line includes the stable finding code and existing summary. Raw evidence
and paths are omitted from the concise default. When a non-pass finding already
contains remediation, the renderer may add one indented `Next:` line containing
that bounded remediation verbatim. It never synthesizes or executes a command.

The footer renders `Overall: OK` when `result.summary.healthy` is true and
`Overall: NEEDS ATTENTION` otherwise, plus the canonical pass, warning, error,
and unavailable counts. The renderer cannot recalculate severity, treat an
optional unavailable integration as a failure, or change the canonical exit
code. Output uses understandable ASCII text without depending on color or
Unicode symbols. Human-facing labels remain isolated for later localization.

For example:

```text
Engineering Toolkit doctor

[OK]    core.runtime - Engineering Toolkit runtime information is available.
[WARN]  workspace.initialized - The directory is not initialized as an Engineering Toolkit workspace.
        Next: Review engkit workspace plan-init before approving initialization.
[N/A]   integrations.ollama - Ollama was not found in PATH.

Overall: OK
Passed: 1  Warnings: 1  Errors: 0  Unavailable: 1
```

The example is illustrative presentation text rather than a golden finding
inventory. Tests must construct synthetic results so they do not depend on the
host's installed integrations or workspace state.

## 3. Unified initialization commands

The accepted facade is:

```text
engkit init plan --target PATH [--profile ID@VERSION] [--set NAME=VALUE]
engkit init apply --target PATH [--profile ID@VERSION] [--set NAME=VALUE] --approve PLAN_ID
```

`--target` is always required; the facade never defaults to the current
directory. `--profile` must be an exact `ID@VERSION`. `--set` is repeatable and
is valid only when `--profile` is present. `--approve` is valid and required
only for `apply`, and it must be the exact selected plan ID.

The commands are non-interactive. They do not detect, recommend, or select a
profile automatically. Local intelligence may explain choices elsewhere, but
the user must supply the exact profile and variables to this deterministic
facade.

## 4. Target classification and lifecycle selection

Classification uses the explicit target after the same path-safety and
ancestor checks required by the selected specialized planner. An empty
directory has zero entries; hidden or system entries are not ignored. A valid
initialized workspace has a supported, internally consistent `.engkit`
configuration. A partial, invalid, linked, or unreadable `.engkit` is an
ambiguous state, not an uninitialized workspace.

The facade selects exactly one lifecycle:

| Observed target | Profile | Result and specialized equivalent |
| --- | --- | --- |
| Absent | Present | New-project composition: `project plan-create` or `project create` |
| Existing empty directory | Present | New-project composition: `project plan-create` or `project create` |
| Existing empty directory, uninitialized | Absent | Workspace initialization: `workspace plan-init` or `workspace init` |
| Existing non-empty directory, uninitialized | Absent | Workspace initialization: `workspace plan-init` or `workspace init` |
| Existing initialized workspace | Present | Existing-workspace composition: `project plan-compose` or `project compose` |
| Existing initialized workspace | Absent | Refuse as already initialized; direct the user to inspection commands |
| Existing non-empty directory, uninitialized | Present | Refuse the compound request; initialization and composition require separate plans and approvals |
| Absent | Absent | Refuse; neither an existing workspace nor an exact project profile was supplied |
| Regular file, linked target or ancestor, junction, special file, unreadable path, or invalid path | Any | Refuse without selecting a lifecycle |
| Partial, invalid, linked, or unreadable `.engkit` state | Any | Refuse and preserve evidence; direct the user to doctor or the applicable recovery command |

The facade performs no cleanup, repair, directory creation, profile inference,
or fallback while classifying. A target that changes classification between
planning and applying invalidates the prior approval.

## 5. Plan and apply equivalence

`init plan` calls the selected existing planner and emits that planner's result
without a facade envelope. The schema, operation, canonical JSON fields,
verification steps, risk data, and `sha256:` plan ID are exactly those of the
specialized command for the same resolved target and inputs. The facade adds no
persisted plan type or compatibility translation.

`init apply` repeats path validation and classification, invokes the matching
existing executor, rebuilds the same plan from the supplied flags and current
state, and requires exact approval of that rebuilt plan ID. It does not accept
a plan file or trust a previously printed classification. Changed target state,
profile bytes, variables, catalog content, or other plan inputs fail the normal
approval check.

The mapping is fixed:

| Facade selection | Planning delegate | Apply delegate | Recovery authority |
| --- | --- | --- | --- |
| Workspace initialization | `workspace plan-init --root TARGET` | `workspace init --root TARGET --approve PLAN_ID` | Existing workspace initialization guarantees; no facade recovery state |
| New-project composition | `project plan-create --destination TARGET ...` | `project create --destination TARGET ... --approve PLAN_ID` | `project recover --root TARGET --approve PLAN_ID` |
| Existing-workspace composition | `project plan-compose --root TARGET ...` | `project compose --root TARGET ... --approve PLAN_ID` | `project recover --root TARGET --approve PLAN_ID` |

The facade owns no journal, checkpoint, lock, staging directory, backup,
ownership record, or recovery command. It cannot combine workspace
initialization and composition into one transaction. Existing specialized
commands remain supported and authoritative.

## 6. Output, errors, and help

Successful `plan` and `apply` write only the selected command's JSON result to
standard output. Diagnostics and errors go to standard error. Exit codes are:

- `0` for a successful selected operation;
- `1` for classification, policy, environment, approval, or execution failure;
- `2` for invalid CLI syntax or flags.

Refusals identify the observed target class and the specialized next command
without executing it. They do not synthesize a plan ID.

`engkit init --help`, `engkit init plan --help`, and
`engkit init apply --help` succeed without filesystem, catalog, SQLite,
Ollama, MCP, provider-runtime, credential, or network access. Help documents
the exact selection table and explains that specialized recovery remains
authoritative.

`engkit doctor -h` and `engkit doctor --help` remain help aliases and never run
diagnostic probes. They are not aliases for the human summary. Doctor JSON and
summary modes both write their successful diagnostic result only to standard
output, keep operational diagnostics on standard error, and select the same
exit code from the canonical result.

## 7. Security and side-effect boundary

Planning is read-only. Applying has exactly the side effects already authorized
by the selected specialized executor. The facade adds no dependency, network
access, credential access, provider synchronization, Spec Kit mutation, model
call, SQLite requirement, Git operation, or release action.

Filesystem artifacts remain canonical. SQLite, MCP, Ollama, and provider
adapters are neither consulted nor required for lifecycle selection. Symlink,
junction, special-file, containment, ownership, and atomic-publication policy
remain owned by the selected existing planner and executor.

## 8. `v0.3.0C2` validation coverage

- Table-driven classification tests cover every row in the selection table,
  including partial `.engkit`, linked ancestors, and state changes.
- Equivalence tests prove facade output and plan IDs equal the selected specialized
  planner for identical inputs.
- Tests prove planning writes no filesystem artifact and help performs no
  discovery.
- Tests prove apply rebuilds the plan, rejects stale approval, and cannot
  chain lifecycles.
- Specialized recovery remains authoritative because the facade invokes the
  existing executors and creates no facade-owned state.
- Cross-platform CI and release qualification remain required before
  publication.

## 9. Planned `v0.5.0B` doctor-summary validation

- Prove the default JSON bytes and `engkit.doctor-result/v1alpha1` decoding are
  unchanged when no presentation flag is supplied.
- Cover `--summary` and `-s` equivalence, all four status labels, deterministic
  ordering, count rendering, optional remediation, and omission of evidence
  and paths.
- Prove healthy, optional-unavailable, warning, required-error, invalid-scope,
  `-h`, and `--help` exit behavior without host-dependent probes.
- Prove help performs no filesystem, catalog, SQLite, Ollama, MCP, provider,
  credential, or network discovery.
- Run focused CLI and doctor tests before formatting, static analysis, the full
  Go test suite, cross-builds, and release qualification appropriate to the
  later implementation approval.

## 10. Rollback

Doctor adds no persisted state. Removing its command restores the prior surface.
Unified init owns no new schema or transaction artifact, so removing the facade
leaves every specialized command and existing workspace usable. No data
migration or rollback utility is required.

The planned human summary owns no persisted state or schema. Implementation
rollback removes only the presentation flag, renderer, and related help/tests;
the canonical JSON command and all diagnostic probes remain available.
