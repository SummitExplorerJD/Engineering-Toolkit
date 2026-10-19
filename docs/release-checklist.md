# First Release Checklist

## Status

This checklist prepares `v0.1.0`; it does not authorize a commit, push, tag,
GitHub Actions run, GitHub Release, GitHub Pages deployment, signing, or
publication.

The first public release should include the production TUF bootstrap root if
`v0.1.0` is expected to discover or download future updates. Publishing without
that root is acceptable only when the release is explicitly manual-update-only.

## Repository gate

- [ ] Review every tracked and untracked file intended for the first commit.
- [ ] Confirm generated caches, binaries, credentials, and private keys are
      absent and ignored.
- [ ] Run `go mod tidy -diff` and review the complete module graph.
- [ ] Run formatting, tests, vet, vulnerability analysis, link validation, and
      `git diff --check`.
- [ ] Approve the exact commit contents and commit message.
- [ ] Approve push to `origin/main` separately.

## GitHub repository settings

- [ ] Enable immutable releases before publishing `v0.1.0`.
- [ ] Require two-factor authentication for release operators.
- [ ] Protect `main` and require successful validation before merging.
- [ ] Limit Actions permissions to read by default.
- [ ] Review the pinned SHAs in `.github/workflows/release-candidate.yml`.
- [ ] Decide whether GitHub Pages will host the initial TUF repository.
- [ ] Configure GitHub Pages only after its exact source branch/workflow and
      custom-domain policy are approved.

## Candidate build

- [ ] Name prerelease candidates `vMAJOR.MINOR.PATCH-rc.N`, starting with
      `rc.1` and incrementing `N` for every replacement candidate. Rebuild
      approved final artifacts with the stable `vMAJOR.MINOR.PATCH` version.
- [ ] Run the manual `Release candidate` workflow from an approved commit.
- [ ] Confirm tests, vet, `govulncheck`, and the TUF rehearsal pass.
- [ ] Download the candidate artifact and verify `SHA256SUMS` independently.
- [ ] Confirm all six raw executables exist.
- [ ] Confirm every executable reports exactly `v0.1.0`.
- [ ] Run native smoke tests on Windows, Linux, and macOS for supported
      architectures available to the project.
- [ ] Review `release-manifest.json` and module/license evidence.

## Trust and platform controls

- [ ] Decide whether `v0.1.0` is manual-update-only or TUF-enabled.
- [ ] If TUF-enabled, complete every open item in the
      [Milestone 5B activation packet](milestone-05b-activation-packet.md).
- [ ] Complete or explicitly defer Windows Authenticode signing.
- [ ] Complete or explicitly defer macOS signing and notarization.
- [ ] Record the Linux executable-mode installation instructions.
- [ ] Document every accepted limitation in the release notes.

## Publication gate

- [ ] Approve the exact annotated tag `v0.1.0` and target commit.
- [ ] Create a draft release and attach the six executables, `SHA256SUMS`,
      `release-manifest.json`, `LICENSE`, and final release notes.
- [ ] Verify asset hashes after upload.
- [ ] Confirm the release is marked immutable.
- [ ] Obtain final human publication approval.
- [ ] Publish the release and preserve the approval/evidence record.

## Rollback

Before publication, discard the candidate artifact and fix the source; do not
reuse a failed candidate. After an immutable release is public, publish a new
patch version rather than replacing an asset or moving the tag. TUF metadata
recovery, if enabled, follows the secure distribution plan and never disables
expiration, signatures, hashes, or rollback protection.

## `v0.1.1` bootstrap installer gate

This additive gate applies to `v0.1.1`; it does not retroactively change the
published `v0.1.0` asset set.

- [ ] Confirm `install.ps1` and `install.sh` are included in the candidate,
      `release-manifest.json`, and `SHA256SUMS`.
- [ ] Confirm the packaged `install.sh` uses LF line endings and both scripts
      accept the exact candidate version.
- [ ] Pass fresh-install, checksum-failure, replacement-refusal, rollback,
      explicit-upgrade, PATH-idempotence, and version checks on all six runners.
- [ ] Review the fixed GitHub Releases URL and confirm no network-source override
      was introduced.
- [ ] Test installation from the published `v0.1.1-rc.1` assets on physical or
      representative Windows, Linux, and macOS systems.
- [ ] Confirm the scripts install without administrator or root privileges.
- [ ] Record that HTTPS plus GitHub repository controls, not production TUF or
      platform signing, authenticate this bootstrap release.
