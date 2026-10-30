# Milestone 14RC Release Qualification Packet

## Status

This packet defined the exact `v0.5.0RC` qualification boundary and was
implemented after separate explicit approval. Local package validation,
candidate and stable workflow qualification, independent artifact review,
tagging, and immutable publication are complete. Installation outside the
isolated installer test remains separately gated.

## Outcome

Qualify the additive doctor human summary without dropping the native context
and MCP regression surface accepted for `v0.4.0`. Every scenario uses synthetic
content and runs against the packaged native executable on Windows, Linux, and
macOS for AMD64 and ARM64. The workflow continues its existing tests, vet,
vulnerability analysis, TUF staging rehearsal, installer transaction, provider
conformance, and aggregate provider comparison.

## Candidate identity

The first candidate name will be `v0.5.0-rc.1`. A failed or discarded attempt
is never reused; the next attempt increments the candidate number. The candidate
must be built from one separately approved exact commit. A passing candidate
does not authorize the stable rebuild, tag, or GitHub Release.

## Exact implementation inventory

Only these files may change in the RC preparation gate:

1. `.github/workflows/release-candidate.yml`
   - Add one conditional `v0.5.0` qualification step in each native smoke job.
   - Upload the bounded `v0.5.0` and nested `v0.4.0` regression records as one
     per-tuple artifact with the existing 14-day retention.
   - Preserve pinned action SHAs, permissions, runners, matrix, timeouts,
     candidate package, installer, and provider-conformance jobs.
2. `scripts/test-v040-release-binary.ps1`
   - Change only its accepted version pattern so the existing ten context/MCP
     scenarios can run against `v0.5.0` candidates as regression coverage.
   - Preserve its schema, scenario identifiers, logic, evidence bounds, and
     `v0.4.0` behavior.
3. `scripts/test-v050-release-binary.ps1` (new)
   - Validate only `v0.5.0` and `v0.5.0-rc.N` identifiers.
   - Invoke the existing `v0.4.0` harness in a dedicated nested output
     directory before running doctor-summary scenarios.
   - Create and clean only an exact synthetic workspace inside the supplied
     empty output directory.
   - Write one bounded `engkit.v050-release-qualification/v1alpha1` record.
4. `docs/release-notes/v0.5.0.md` (new)
   - Describe the human summary, unchanged JSON and exit behavior, retained
     safety boundary, qualification plan, upgrade compatibility, and accepted
     release limitations.

No other source, test, module, schema, configuration, catalog, installer,
provider, workflow, or release file is in the RC preparation inventory. If the
implementation needs another file, stop and prepare a replacement packet.

## Native qualification scenarios

The new harness first requires all ten existing `v0.4.0` scenarios to pass:

1. context v2 rebuild and freshness;
2. mandatory and user exclusions;
3. bounded search and indexed read;
4. CLI and MCP result equivalence;
5. twelve-tool stdio inventory;
6. indexed workspace tree;
7. compact result non-duplication;
8. text compatibility with one representation;
9. strict input and path refusal; and
10. stale projection refusal.

It then runs these six `v0.5.0` doctor scenarios:

1. **JSON compatibility:** unflagged `doctor --scope core` returns exit `0`,
   schema `engkit.doctor-result/v1alpha1`, canonical counts, and healthy state.
2. **Summary alias equivalence:** `--summary` and `-s` return byte-identical
   UTF-8 output with exit `0`, `[OK]`, `Overall: OK`, and canonical counts.
3. **Help isolation:** `doctor -h` and `doctor --help` return byte-identical
   help with exit `0` and no diagnostic heading, status label, or workspace
   access.
4. **Optional-state health:** an existing empty synthetic directory returns
   exit `0`, at least one `[WARN]` and `[N/A]`, and `Overall: OK` in workspace
   scope because no required error exists.
5. **Required-error health:** a synthetic absent root returns exit `1`, at
   least one `[ERROR]`, and `Overall: NEEDS ATTENTION`.
6. **Disclosure boundary:** summary output contains no synthetic absolute-root
   marker, evidence field name, evidence value, JSON object, carriage return,
   tab, or line longer than 1,040 UTF-8 bytes. The line ceiling accommodates
   two independently bounded 512-byte fields plus fixed presentation text.

The harness does not invoke Ollama, a model, an MCP host, a provider CLI, a
credential service, or a non-loopback product endpoint. It does not use real
repository content in a workspace or prompt.

## Evidence contract

Each platform tuple writes one
`engkit.v050-release-qualification/v1alpha1` JSON record containing only:

- exact candidate version and packaged binary filename;
- SHA-256 digests of the canonical JSON, long summary, short summary, short
  help, and long help outputs;
- byte counts and line counts for bounded presentation outputs;
- canonical JSON schema and health;
- observed healthy, optional-state, and required-error exit codes;
- the six fixed scenario identifiers and pass outcomes; and
- the relative path and digest of the nested v0.4.0 regression record.

Raw doctor JSON, summary text, help text, finding evidence, paths, synthetic
workspace content, credentials, environment values, and prompts are not
retained. Per-command temporary stdout and stderr files are removed in `finally`
blocks. The artifact contains only the two qualification records.

## Validation before workflow execution

Run locally from a clean approved source tree:

```text
gofmt -w internal/cli/run.go internal/cli/doctor_summary.go internal/cli/doctor_summary_test.go internal/cli/run_test.go
go test ./internal/cli ./internal/doctor
go test ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

Build a disposable `v0.5.0-rc.1` Windows AMD64 package, run the generic native
smoke, the existing v0.4.0 regression harness through the new wrapper, the
installer transaction, and provider conformance. Inspect both evidence records,
package manifest, checksums, executable version, and exact eleven-file package
inventory. Finish with `git diff --check` and an exact changed-file review.

Workflow execution later requires all six native jobs, provider comparison,
tests, vet, vulnerability analysis, TUF rehearsal, package upload, installer
tests, sixteen qualification scenarios per tuple, and bounded evidence uploads
to pass. Download and independently verify every workflow artifact before any
candidate is accepted.

## Risks and controls

- **Regression coverage loss:** a new minor release could test only the new
  flag. The wrapper first requires the unchanged ten-scenario v0.4.0 harness.
- **Evidence disclosure:** doctor JSON and paths may contain local facts. Store
  only digests, counts, fixed identifiers, relative evidence paths, and
  outcomes; remove temporary command output.
- **Exit-status masking:** PowerShell pipeline behavior could hide an unhealthy
  exit. Capture `$LASTEXITCODE` immediately after every native invocation and
  assert the exact allowed value before reading output.
- **Platform encoding drift:** PowerShell redirection can vary by host. Read
  bytes explicitly, normalize neither expected summary nor help, and compare
  digests only between aliases produced on the same tuple. Require valid UTF-8
  and LF-only renderer output.
- **Harness coupling:** expanding the accepted version range could change old
  qualification behavior. Modify only the v0.4.0 script's version validation
  and run its original `v0.4.0` path locally before workflow approval.
- **Candidate reuse:** a failed RC can leave ambiguous artifacts. Discard the
  entire run and increment the RC number; never replace immutable candidate or
  stable assets.

## Rollback

Before workflow execution, revert the four RC preparation files. No product
schema, workspace, SQLite projection, dependency, installation, or external
state requires migration. After a failed candidate run, discard its artifacts
and prepare a higher RC number from a separately reviewed commit. After an
immutable stable release, corrections require a new patch release; never move
the tag or replace assets.

## Approval boundaries

Implementation, local package execution, destructive temporary cleanup, Git
commit, push, workflow dispatch, artifact download, independent review, stable
rebuild, annotated tag, GitHub Release creation, asset upload, publication, and
local installation are separate approvals. This packet grants none of them.

## Observed implementation and local validation

The implemented RC preparation inventory is exactly:

- `.github/workflows/release-candidate.yml`;
- `scripts/test-v040-release-binary.ps1`;
- `scripts/test-v050-release-binary.ps1`; and
- `docs/release-notes/v0.5.0.md`.

The workflow retains its existing six-platform matrix, pinned actions,
permissions, installer validation, provider conformance, and aggregate
comparison. `v0.5.0` inputs additionally run the new wrapper and upload one
bounded artifact per tuple. The existing v0.4.0 harness changed only its
accepted version pattern and passed again against the installed stable
`v0.4.0` binary.

The first disposable Windows AMD64 wrapper run completed all sixteen scenarios
but exposed an isolated harness-status defect: the accepted required-error
scenario left native exit `1` in global `$LASTEXITCODE`, so its caller
misclassified the successful script as failed. The wrapper now resets global
`$LASTEXITCODE` only after asserting an allowed native status. The entire
package validation was then rebuilt and repeated from an empty temporary root.

Observed successful rerun:

- exact version `v0.5.0-rc.1` and generic native smoke;
- 11 package files, 10 independently verified checksum entries, six manifest
  executables, and two manifest installers;
- all ten nested v0.4.0 context/MCP scenarios and all six v0.5.0 doctor-summary
  scenarios;
- exactly two retained qualification records and no raw command output or
  synthetic workspace content;
- Windows AMD64 installer transaction and provider conformance;
- the unchanged ten-scenario v0.4.0 harness against the installed stable
  `v0.4.0` binary;
- `go test ./...` outside the sandbox, `go vet ./...`, and
  `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`;
- zero reachable vulnerabilities and zero affected imported packages; five
  known vulnerabilities exist only in required modules whose affected symbols
  are not called;
- PowerShell syntax parsing and `git diff --check`.

One combined test-and-vet attempt encountered a Windows access denial while an
unrelated CLI test tried to rename its atomic temporary scaffold concurrently.
The isolated complete `go test ./...` rerun passed. All validation-specific
system-temporary directories were removed. Pre-existing ignored
`.engkit-test-temp` entries were inspected but not altered. `actionlint` was not
available locally, so workflow YAML execution remains a GitHub Actions gate.

No user installation or model was changed by local validation.

## Observed remote qualification and publication

Candidate `v0.5.0-rc.1` passed GitHub Actions run
[`40945166094`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40945166094)
from exact commit `16384272abae8e7006805191d5f269c64330d2ff`. All six native
jobs passed the generic smoke, installer transaction, provider conformance, the
ten retained v0.4.0 context/MCP scenarios, and the six v0.5.0 doctor-summary
scenarios. The aggregate portable provider-evidence comparison also passed.
Independent review accepted all 13 workflow artifact digests, the 11-file
package, all 10 package checksums, six bounded qualification records, and six
provider records.

Stable workflow run
[`41065556447`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/41065556447)
rebuilt exact version `v0.5.0` from the same commit. All six native jobs and the
aggregate provider comparison passed. Independent review verified the stable
package, executable identities, installer transaction, 96 qualification
scenarios, six provider records, and portable-evidence equivalence. The stable
ZIP digest is
`0fc676ac76a806f17d39ad1616adb0277af07bc23b1167c3e86d5a3309a34fae`.

Annotated tag `v0.5.0` has tag-object SHA
`99ed1f74032a3eece4f6caa73185d182ec81b814` and resolves to the exact release
commit. The
[`v0.5.0` GitHub Release](https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.5.0)
was published on 2026-10-30 as non-draft, non-prerelease, immutable, and latest.
Its 12 uploaded assets match the independently reviewed names, sizes, and
SHA-256 digests. GitHub's generated source archives and release attestation are
outside that uploaded-asset inventory.
