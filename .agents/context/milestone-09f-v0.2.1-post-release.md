# Milestone 09F: v0.2.1 post-release context

## Published release

- Release: `v0.2.1`
- URL: <https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.2.1>
- Tag type: lightweight
- Tagged commit: `38b7aab7b24253fee3e55ff1dbfae000dced4a4e`
- Publication date: 2026-10-22
- Release state: stable and immutable
- Assets: six platform executables, `install.ps1`, `install.sh`, `LICENSE`,
  `release-manifest.json`, `SHA256SUMS`, and a convenience ZIP
- Convenience ZIP SHA-256:
  `8f35c8fb5fee9fc369cf0a08d1fed0a26a7b3010513b05c6f637ae7e452bbe36`

GitHub reported all 12 assets in the uploaded state. Every published asset
digest matched the independently reviewed final package. The final package
manifest records `v0.2.1`, Go 1.27.3, CGO disabled, and the six supported
operating-system and architecture targets.

## Release workflow evidence

- Candidate `v0.2.1-rc.1` completed successfully in workflow run
  `39804867776`.
- Stable `v0.2.1` completed successfully in workflow run `39807628575`.
- Both successful runs used source commit
  `38b7aab7b24253fee3e55ff1dbfae000dced4a4e`.
- The stable package's outer artifact digest and all entries in `SHA256SUMS`
  matched before publication.
- Native smoke and installation-transaction jobs passed on Windows, Linux, and
  macOS for AMD64 and ARM64.

## Manual acceptance evidence

The final Windows AMD64 binary and local-asset installer were independently
checked before publication. Fresh installation and explicit upgrade completed
without modifying PATH, preserved the expected executable digest, and reported
`engkit v0.2.1`. The installed binary successfully inspected catalog
`engineering-toolkit@0.2.1`, template `project-guide@0.1.0`, and profile
`go-service@0.1.0` with their declared directory digests.

## Accepted limitations

- Windows Authenticode signing is deferred.
- macOS signing and notarization are deferred.
- Production TUF trust and repository endpoints are not activated.
- Provider synchronization, package-manager distribution, automatic updates,
  and self-replacement require separate future approval.

Any correction to the immutable release requires a new semantic version. The
published `v0.2.1` tag and assets must not be replaced or moved.
