# Milestone 09B: Strict readers and authoring scaffold

## Implemented scope

- Added strict YAML, portable-path, and regular-file inventory primitives.
- Added an immutable embedded catalog with no first-party templates or profiles.
- Added strict catalog, skill-template, and project-profile readers.
- Added deterministic catalog, template, and profile list/inspection commands.
- Added bounded template scaffolding through sibling staging and atomic rename.
- Added successful scoped help for command families and subcommands.

## Safety boundary

- Readers reject aliases, anchors, duplicate or unknown fields, multiple YAML
  documents, unsafe paths, links, special files, undeclared files, invalid
  identities, invalid versions, and digest mismatches.
- The scaffold requires an explicit absent output path, refuses linked
  ancestors, never merges or overwrites, and removes staging after failure.
- No template rendering, profile recommendation, project composition, provider
  rendering, network access, credential access, or resource execution exists.

## Validation evidence

- Focused package and CLI tests pass.
- `go test -p 1 ./...` passes with an authorized repository-local temporary
  base and loopback access for the existing TUF HTTPS tests.
- `go vet ./...` passes.
- The CLI cross-builds for Windows, Linux, and macOS on AMD64 and ARM64.

The Windows application sandbox denies directory `os.Rename` inside its
virtualized system temporary directory; the same restriction affects unchanged
workspace-initialization tests. Atomic-publication validation therefore uses a
repository-local temporary base. Native CI remains required before release.

## Next approval gate

`v0.2.1C` may implement deterministic text rendering, evidence-based profile
recommendations, and non-executable project/workspace composition plans.
