# Milestone 14B Doctor Human-Summary Implementation Packet

## Status

This packet was prepared from the accepted `v0.5.0A` contract and implemented
after separate explicit approval. The implementation stayed within the exact
source and test inventory below. Git operations, workflows, installation, and
release publication remain unauthorized.

## Outcome

Add `engkit doctor --summary` and the exact alias `engkit doctor -s` as a
concise human presentation of the existing canonical
`engkit.doctor-result/v1alpha1` result. Keep unflagged `engkit doctor` JSON,
`-h`/`--help`, diagnostic probes, health policy, deterministic ordering, and
exit codes unchanged.

## Exact implementation inventory

Only these Go files may change in the implementation gate:

1. `internal/cli/run.go`
   - Register `--summary` and `-s` against the same Boolean value.
   - Select the summary renderer only after one normal `doctor.Inspect` call.
   - Preserve JSON as the unflagged default and preserve the canonical health
     exit decision after either renderer succeeds.
   - Update the top-level doctor usage line without changing global help
     routing; `-h` and `--help` continue to return before probe execution.
2. `internal/cli/doctor_summary.go` (new)
   - Own all human labels and rendering logic inside the CLI adapter.
   - Consume `doctor.Result`; do not import filesystem, SQLite, MCP, Ollama,
     provider, network, or execution packages.
   - Preserve input finding order and map only the four accepted statuses.
   - Omit evidence and paths, render existing non-pass remediation as `Next:`,
     and derive the footer exclusively from the canonical summary.
   - Normalize carriage return, line feed, and tab characters to spaces before
     rendering. Limit each rendered code, summary, and remediation field to 512
     UTF-8 bytes without splitting a code point; append `...` when truncated.
   - Assemble the complete output in memory and perform one final write so a
     formatting failure cannot publish a partial result. Propagate the final
     write error to the existing CLI error path.
3. `internal/cli/doctor_summary_test.go` (new)
   - Test all status labels, stable input order, footer states, counts,
     remediation inclusion, evidence/path omission, whitespace normalization,
     UTF-8-safe truncation, unknown-status refusal, and final write failure.
4. `internal/cli/run_test.go`
   - Prove unflagged JSON still decodes as
     `engkit.doctor-result/v1alpha1` and retains its existing result and exit
     behavior.
   - Prove `--summary` and `-s` produce identical output for controlled healthy
     and unhealthy cases without depending on host integrations.
   - Prove `doctor -h` and `doctor --help` remain help-only, and invalid flags
     and positional arguments retain exit `2` behavior.
   - Verify the updated global usage documents the optional summary flag.

No other source, test, generated, module, configuration, schema, catalog,
workflow, or release file is in the implementation inventory. If implementation
reveals a need outside this list, stop and prepare a replacement packet.

## Presentation contract

The renderer emits:

```text
Engineering Toolkit doctor

[LABEL] code - summary
        Next: remediation

Overall: OK|NEEDS ATTENTION
Passed: N  Warnings: N  Errors: N  Unavailable: N
```

`Next:` is emitted only for a non-pass finding with non-empty canonical
remediation. Alignment is presentation-only and cannot depend on terminal
width, color, locale, environment variables, or TTY detection. The renderer
does not expose evidence or paths. English labels are grouped in the new
presentation file so a later localization adapter can replace them without
changing the domain result.

## Validation

Run the smallest suites first, then the complete approved checks:

```text
gofmt -w internal/cli/run.go internal/cli/doctor_summary.go internal/cli/doctor_summary_test.go internal/cli/run_test.go
go test ./internal/cli ./internal/doctor
go test ./...
go vet ./...
govulncheck ./...
```

Build the CLI with CGO disabled for Windows, Linux, and macOS on AMD64 and
ARM64 using disposable output paths. Exercise unflagged JSON, `--summary`,
`-s`, `-h`, `--help`, healthy output, required-error output, and optional
unavailable output with synthetic inputs. Finish with `git diff --check` and an
exact changed-file review. Record commands and observed results; do not claim a
check that was not run.

## Risks and controls

- **Automation regression:** default JSON could change accidentally. Preserve
  the existing branch and assert schema decoding and output behavior before
  summary tests.
- **Policy duplication:** a renderer could recalculate health or ordering.
  Read only the canonical findings and summary; do not sort or inspect state.
- **Unsafe or unreadable text:** future controlled finding text could contain
  control characters or grow unexpectedly. Normalize line controls and apply
  UTF-8-safe 512-byte field ceilings.
- **Partial output:** a writer could fail after some lines. Build the bounded
  result first and use one final write whose error is propagated.
- **Help collision:** `-h` could be confused with the requested human view.
  Preserve existing pre-dispatch help routing and test it without discovery.
- **Localization coupling:** domain code could acquire presentation strings.
  Keep every human label in `internal/cli/doctor_summary.go`.

## Rollback

Remove `internal/cli/doctor_summary.go` and its test, revert the doctor flag and
usage changes in `internal/cli/run.go`, and revert the added integration tests
in `internal/cli/run_test.go`. The unflagged JSON command, doctor package,
schemas, workspaces, SQLite projections, and installed releases require no
migration or cleanup.

## Exclusions

The gate adds no new diagnostic probe, finding code, health rule, schema,
configuration, persistence, SQLite behavior, MCP tool, model or provider
behavior, dependency, network access, automatic repair, approval, execution,
Git operation, workflow, installation, tag, or release publication.

## Observed implementation and validation

The implemented source inventory is exactly:

- `internal/cli/run.go`;
- `internal/cli/doctor_summary.go`;
- `internal/cli/doctor_summary_test.go`; and
- `internal/cli/run_test.go`.

The implementation adds only the accepted `--summary` and `-s` flag wiring,
bounded renderer, usage text, and tests. `internal/doctor`, public schemas,
configuration, SQLite, MCP, models, providers, dependencies, workflows, and
release behavior were unchanged.

Observed validation:

- `gofmt` completed for the four approved Go files.
- `go test ./internal/cli ./internal/doctor` passed. The first sandboxed run
  exposed Windows access denial for unrelated atomic temporary-directory
  renames; the same focused suite passed outside the sandbox.
- `go test ./...` passed outside the sandbox.
- `go vet ./...` passed.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` reported no
  reachable vulnerabilities and no affected imported packages. It observed
  five vulnerabilities in required modules whose affected symbols are not
  called by Engineering Toolkit.
- CGO-disabled builds passed for Windows, Linux, and macOS on AMD64 and ARM64.
- Native Windows AMD64 smokes confirmed unchanged
  `engkit.doctor-result/v1alpha1` JSON, byte-identical `--summary` and `-s`
  output, byte-identical `-h` and `--help` output, and exit `1` with
  `Overall: NEEDS ATTENTION` for a synthetic missing workspace.
- `git diff --check` passed. `go.mod` and `go.sum` remained unchanged.

No installation, Git mutation, workflow, tag, or release action occurred.
