# Milestone 09E: First-party pilot content

## Approved boundary

- Package `project-guide@0.1.0` as the first provider-neutral skill template.
- Package `go-service@0.1.0` as the first project profile.
- Bind only the required `project_name` variable.
- Generate `AGENTS.md`, project context, Go conventions, a verification
  workflow, one canonical skill, the four `.agents` surfaces, and `src/`.
- Keep application source, `go.mod`, scripts, provider output, Spec Kit
  mutation, dependency installation, and network access outside the pilot.

## Content decisions

- The pilot is an engineering-workspace overlay, not a Go application
  generator.
- Template and profile versions are immutable after publication; revisions use
  new semantic versions and new directory digests.
- Generated workflow commands are documentation and never execution authority.
- The canonical skill requests no permissions and declares no provider parity.

## Validation boundary

- Strict catalog loading verifies each source digest and complete directory
  digest from embedded bytes.
- CLI tests plan and execute a new project through exact plan approval, then
  validate the installed canonical skill.

## Observed validation

- `go test ./...` passed, including loopback TUF transport tests.
- `go vet ./...` passed.
- Catalog, template, profile, planning, creation, and installed-skill validation
  passed through the public CLI test boundary.
- The CLI compiled with `CGO_ENABLED=0` for Windows, Linux, and macOS on AMD64
  and ARM64 without retaining release artifacts.
- Changed files contain no pending placeholders, conflict markers, or trailing
  whitespace.
