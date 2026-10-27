# `v0.3.0` Stable Publication Packet

## Status and authority

This packet records the approved stable `v0.3.0` publication completed on
2026-10-27. Each commit, push, workflow run, tag, draft, upload, publication,
and local installation was separately approved before execution.

Candidate `v0.3.0-rc.1` qualified the implementation at commit
`08dac40b4b7f7d2549ce318dcf5e446036347fda`. Stable release commit
`2bd28273e072b2a33647fb5d7ba09968e381f27a` added only the reviewed release
documentation after candidate acceptance.

## Stable build gate

The approved stable build gate completed as follows:

1. `main` and `origin/main` both identified exact commit
   `2bd28273e072b2a33647fb5d7ba09968e381f27a`;
2. `.github/workflows/release-candidate.yml` ran as
   [`40568038154`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40568038154)
   with input `version=v0.3.0`;
3. the validation/build job, all six native smoke jobs, and aggregate
   provider-conformance job passed;
4. independent review verified every `SHA256SUMS` entry, manifest size and
   hash, executable format, installer line ending, embedded version, and six
   qualification records; and
5. the exact workflow ZIP was retained as `engkit-v0.3.0.zip`, and its 11
   entries matched the 11 uploaded individual assets byte for byte.

The stable run produced version `v0.3.0`; no `v0.3.0-rc.1` binary was renamed
or reused.

## Expected release assets

The published GitHub Release contains exactly these 12 uploaded assets:

1. `engkit-v0.3.0-windows-amd64.exe`
2. `engkit-v0.3.0-windows-arm64.exe`
3. `engkit-v0.3.0-linux-amd64`
4. `engkit-v0.3.0-linux-arm64`
5. `engkit-v0.3.0-darwin-amd64`
6. `engkit-v0.3.0-darwin-arm64`
7. `engkit-v0.3.0.zip`
8. `install.ps1`
9. `install.sh`
10. `LICENSE`
11. `release-manifest.json`
12. `SHA256SUMS`

GitHub-generated source archives are platform-generated links and are not part
of this uploaded-asset inventory. The release body must use the final reviewed
contents of `docs/release-notes/v0.3.0.md`.

## Trust and compatibility disposition

- Publish as manual-update-only; production TUF activation remains unavailable.
- Defer Windows Authenticode signing explicitly.
- Defer macOS signing and notarization explicitly.
- Preserve GitHub Actions read-only repository permissions and reviewed pinned
  action SHAs.
- Do not attach prompts, model responses, local evidence directories,
  credentials, or provider configuration.
- Describe Ollama observations only for the exact accepted version, tag,
  digest, host, prompt, and limits. Do not claim general model compatibility.

## Publication result

After separate approvals, annotated tag `v0.3.0` was created at exact stable
commit `2bd28273e072b2a33647fb5d7ba09968e381f27a` and pushed without replacing an
existing ref. The
[`v0.3.0` GitHub Release](https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.3.0)
was published as non-draft, non-prerelease, immutable, and latest. Its 12
uploaded assets have the exact approved names, sizes, and SHA-256 digests. The
published package digest is
`sha256:550d92230c1562d8d9f12ac5fcd5bed0284bfb1e2d06e0703529c1a07cac80fe`.

## Post-publication verification

Public metadata confirms the exact tag, immutable status, non-draft and
non-prerelease state, latest designation, and 12-asset inventory. GitHub's
recorded asset sizes and SHA-256 digests match the approved stable files.

An existing Windows AMD64 installation was transactionally upgraded from
`v0.2.2` to `v0.3.0` using the verified stable asset directory without
administrator rights. The installed executable matched approved digest
`sha256:8775446dce378fe29ca8d19895684208706a169996d738873fd7d3b0174e86ce`,
`engkit version --json` reported exactly `v0.3.0`, catalog inspection returned
`engineering-toolkit@0.3.0`, and no transaction directory remained.

This local-asset upgrade does not claim completion of the distinct fresh
network-installation check. A later network check requires its own approval and
must not alter the immutable release assets.

## Risk and rollback

Before publication, discard a failed stable artifact or draft and correct the
source under a new approval. Never rename candidate binaries, reuse a failed
build, move an existing tag, or replace bytes in an immutable release.

After publication, do not edit or replace immutable assets. Address a material
defect with a new patch release. TUF recovery, if later activated, must preserve
signature, threshold, expiration, hash, and rollback protections.
