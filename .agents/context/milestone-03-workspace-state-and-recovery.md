# Milestone 03 Context: Workspace State and Recovery

## Approved scope

- Bind execution approval to the complete deterministic plan ID.
- Reinspect the workspace immediately before mutation.
- Build the complete `.engkit` tree in a same-filesystem staging directory.
- Publish through a directory rename only after all files are written and synchronized.
- Persist YAML configuration, the approved JSON plan, an append-only JSONL journal, and a JSON checkpoint.
- Exclude SQLite, Spec Kit, provider adapters, dependencies, and Git operations.

## Material decisions

- `workspace init` requires `--approve` with the exact ID returned by `workspace plan-init`.
- Initialization uses a fixed canonical plan path because only one initial workspace creation is valid.
- The journal records approval, start, and completion events with monotonic sequence numbers.
- The completion checkpoint references the plan and final journal sequence.
- Existing `.engkit` state causes a refusal rather than a merge or overwrite.
- Filesystem probe errors other than a missing path fail closed.

## Verification evidence

- `go test ./...` passed for CLI, approval, execution, plan, and workspace packages.
- `go vet ./...` passed.
- `gofmt -d cmd internal` produced no output.
- CLI tests confirmed that plan preview does not write and approved initialization creates the expected artifacts.

## Environment observation

The Windows AppContainer default temporary directory allowed file creation but denied directory rename to `.engkit`. Tests passed when `TEMP` and `TMP` were set to the task's authorized working directory. The production operation stages inside the selected workspace, so source and destination remain on the same filesystem.

## Remaining work

- Recovery for future multi-step mutations after initialization needs failure-injection design.
- Checkpoint replacement rules for resumed operations need a cross-platform implementation.
- SQLite remains a separately approved milestone because it requires a dependency and derived-schema decisions.

