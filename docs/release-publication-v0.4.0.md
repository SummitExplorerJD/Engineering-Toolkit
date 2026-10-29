# `v0.4.0` Stable Publication Packet

## Status and authority

This packet records the approved stable `v0.4.0` publication completed on
2026-10-29. Each commit, push, workflow run, tag, draft, upload, and publication
was separately approved before execution. No local installation was performed
as part of publication.

Candidate `v0.4.0-rc.2` qualified the implementation at commit
`1de90e29242ecb6ec5171a8463203b3a20b3d657` in GitHub Actions run
[`40770854564`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40770854564).
Stable release commit `a7bb270569a520628b2b7db3385ebf9b4a007b97`
added only the reviewed publication documentation after candidate acceptance.

## Stable build gate

The approved stable build gate completed as follows:

1. `main` and `origin/main` both identified exact commit
   `a7bb270569a520628b2b7db3385ebf9b4a007b97`;
2. `.github/workflows/release-candidate.yml` ran as
   [`40773213540`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40773213540)
   with input `version=v0.4.0`;
3. the validation/build job, all six native smoke jobs, all ten v0.4.0
   scenarios per tuple, every installer and provider-conformance check, and the
   aggregate provider comparison passed;
4. independent review verified all 13 workflow artifact digests, every
   `SHA256SUMS` entry, manifest size and hash, package inventory, executable
   format, installer line ending, embedded version, six v0.4.0 qualification
   records, and six provider records; and
5. the exact workflow package was retained as `engkit-v0.4.0.zip`, and its 11
   entries match the 11 individual release assets byte for byte.

The stable run produced version `v0.4.0`; no `v0.4.0-rc.2` binary was renamed
or reused. The stable package digest is
`sha256:1999101eaeed1222f0cc7463a59f5e88d4a950f97cadd79dab22bd027a830c18`.

## Expected release assets

The published GitHub Release contains exactly these 12 uploaded assets:

1. `engkit-v0.4.0-windows-amd64.exe`
2. `engkit-v0.4.0-windows-arm64.exe`
3. `engkit-v0.4.0-linux-amd64`
4. `engkit-v0.4.0-linux-arm64`
5. `engkit-v0.4.0-darwin-amd64`
6. `engkit-v0.4.0-darwin-arm64`
7. `engkit-v0.4.0.zip`
8. `install.ps1`
9. `install.sh`
10. `LICENSE`
11. `release-manifest.json`
12. `SHA256SUMS`

GitHub-generated source archives are platform-generated links and are not part
of this uploaded-asset inventory. The release body must use the final reviewed
contents of `docs/release-notes/v0.4.0.md`.

## Trust and compatibility disposition

- Publish as manual-update-only; production TUF activation remains unavailable.
- Defer Windows Authenticode signing explicitly.
- Defer macOS signing and notarization explicitly.
- Preserve GitHub Actions read-only repository permissions and reviewed pinned
  action SHAs.
- Keep canonical workspace files authoritative; SQLite remains a local,
  replaceable projection.
- Keep MCP read-only and stdio-only; do not claim remote transport or mutation.
- Do not attach workspace content, raw MCP frames, prompts, model responses,
  local indexes, credentials, provider configuration, or temporary evidence.
- Make no new model-compatibility claim; the optional D2 model-backed harness
  was skipped.

## Publication result

After separate approvals, annotated tag `v0.4.0` was created at exact stable
commit `a7bb270569a520628b2b7db3385ebf9b4a007b97` and pushed without replacing an
existing ref. The
[`v0.4.0` GitHub Release](https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.4.0)
was published as non-draft, non-prerelease, immutable, and latest. Its 12
uploaded assets have the exact approved names, sizes, and SHA-256 digests.
GitHub additionally exposes its two generated source archives, which are not
part of the uploaded-asset inventory.

## Post-publication verification

Public metadata confirms the exact tag, release commit, immutable status,
non-draft and non-prerelease state, latest designation, and 12 uploaded assets.
GitHub reports every uploaded asset in `uploaded` state, with sizes and SHA-256
digests matching the independently reviewed stable files.

The distinct public redownload and fresh Windows network-installation check
remain pending separate approval. No public installer execution, installed
version claim, context rebuild, or installed-binary MCP exercise is recorded by
this publication update.

## Risk and rollback

Before publication, discard a failed stable artifact or draft and correct the
source under a new approval. Never rename candidate binaries, reuse a failed
build, move an existing tag, or replace bytes in an immutable release.

After publication, do not edit or replace immutable assets. Address a material
defect with a new patch release. TUF recovery, if later activated, must preserve
signature, threshold, expiration, hash, and rollback protections.
