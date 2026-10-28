# `v0.4.0` Stable Publication Packet

## Status and authority

This packet prepares the stable `v0.4.0` release. It does not authorize a
commit, push, workflow run, tag, draft release, asset upload, immutable release,
publication, or local installation. Each external or Git operation requires
its own explicit approval.

Candidate `v0.4.0-rc.2` qualified the implementation at commit
`1de90e29242ecb6ec5171a8463203b3a20b3d657` in GitHub Actions run
[`40770854564`](https://github.com/SummitExplorerJD/Engineering-Toolkit/actions/runs/40770854564).
The stable release commit is not yet selected. It must include the reviewed
release documentation, and no source or harness change may be introduced
without repeating candidate qualification under a new `-rc.N` identifier.

## Stable build gate

After the release-documentation commit and push are separately approved:

1. record the exact full commit SHA and confirm `main` equals `origin/main`;
2. dispatch `.github/workflows/release-candidate.yml` from that commit with
   input `version=v0.4.0`;
3. require the validation/build job, all six native smoke jobs, all ten v0.4.0
   scenarios per tuple, every installer and provider-conformance check, and the
   aggregate provider comparison to pass;
4. download all stable workflow artifacts and independently verify artifact
   digests, every `SHA256SUMS` entry, manifest size and hash, package inventory,
   executable format, installer line ending, embedded version, six v0.4.0
   qualification records, and six provider records; and
5. retain the exact workflow package as `engkit-v0.4.0.zip` and verify its 11
   entries match the 11 individual release assets byte for byte.

The stable run must produce version `v0.4.0`; it must not reuse or rename any
`v0.4.0-rc.2` binary. A failed stable run blocks publication. A source or
harness correction requires review and a new candidate identifier before
another stable attempt.

## Expected release assets

The draft GitHub Release must contain exactly these 12 uploaded assets:

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

## Tag and release gate

After stable artifacts pass independent review, request separate approval for:

- annotated tag `v0.4.0` at the exact stable workflow commit;
- pushing that tag without moving or replacing an existing ref;
- creating a draft, non-prerelease GitHub Release targeting the exact tag;
- uploading the 12 verified assets and confirming their remote hashes;
- confirming the repository's immutable-release protection applies to the
  final release; and
- final publication and latest-release designation.

No approval for one item implies approval for the next.

## Post-publication verification

After publication:

- fetch public release metadata and confirm the tag, target, immutability,
  non-draft, non-prerelease, latest status, and exact asset inventory;
- redownload public `SHA256SUMS`, installers, manifest, ZIP, and representative
  binaries and compare them with the approved stable files;
- complete a fresh Windows network installation without administrator rights,
  confirm `engkit version --json` reports exactly `v0.4.0`, rebuild a synthetic
  context projection, and exercise bounded CLI and MCP retrieval; and
- record the immutable release URL, stable commit, asset digests, and observed
  verification in roadmap, milestone context, publication packet, and release
  notes.

## Risk and rollback

Before publication, discard a failed stable artifact or draft and correct the
source under a new approval. Never rename candidate binaries, reuse a failed
build, move an existing tag, or replace bytes in an immutable release.

After publication, do not edit or replace immutable assets. Address a material
defect with a new patch release. TUF recovery, if later activated, must preserve
signature, threshold, expiration, hash, and rollback protections.
