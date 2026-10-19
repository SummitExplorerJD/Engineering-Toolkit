# Milestone 6C v0.1.0 Post-Release Context

## Published release

- Release: `v0.1.0`
- URL: <https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.1.0>
- Tag type: annotated
- Tagged commit: `c97b1d420f7033acd4d8cdd7668641f5b8d0dad3`
- Publication date: 2026-10-16
- Release state: stable and immutable
- Assets: six platform executables, `SHA256SUMS`,
  `release-manifest.json`, `LICENSE`, and a convenience ZIP
- Convenience ZIP SHA-256:
  `c01760af5ed84a620398a17d461c7c608e641010ecca2a178bc1ad5f6937d064`

After publication, the GitHub-calculated digests for all eight entries covered
by `SHA256SUMS` matched the published checksum file. The GitHub digest for the
convenience ZIP matched the independently verified pre-publication digest.

## Manual smoke evidence

On 2026-10-19, the release operator reported that a manual Windows installation
successfully ran both:

```text
engkit version
engkit --help
```

This evidence is operator-reported rather than captured by an automated test.
The release workflow separately completed native smoke jobs for Windows,
Linux, and macOS on `amd64` and `arm64` before publication.

## Accepted limitations

- `v0.1.0` is manual-update-only.
- Production TUF trust and repository endpoints are not activated.
- Windows Authenticode signing is deferred.
- macOS signing and notarization are deferred.
- Installers, package-manager distribution, automatic installation, and
  self-replacement are not included.

Any correction to the immutable release requires a new semantic version. The
published `v0.1.0` tag and assets must not be replaced or moved.
