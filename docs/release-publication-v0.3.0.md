# `v0.3.0` Stable Publication Packet

## Status and authority

This packet prepares the stable `v0.3.0` release. It does not authorize a
commit, push, workflow run, tag, draft release, asset upload, immutable release,
or publication. Each external or Git operation requires its own explicit
approval.

Candidate `v0.3.0-rc.1` qualified the implementation at commit
`08dac40b4b7f7d2549ce318dcf5e446036347fda`. The stable release commit is not
yet selected. It must include the reviewed release documentation, and no source
change may be introduced without repeating candidate qualification under a new
`-rc.N` identifier.

## Stable build gate

After the release-documentation commit and push are separately approved:

1. record the exact full commit SHA and confirm `main` equals `origin/main`;
2. dispatch `.github/workflows/release-candidate.yml` from that commit with
   input `version=v0.3.0`;
3. require the validation/build job, all six native smoke jobs, and aggregate
   provider-conformance job to pass;
4. download the stable workflow artifact and independently verify every
   `SHA256SUMS` entry, manifest size and hash, binary format, installer line
   ending, embedded version, and six qualification records; and
5. assemble `engkit-v0.3.0.zip` from the exact 11 verified packaged files
   without changing their bytes.

The stable run must produce version `v0.3.0`, not reuse or rename any
`v0.3.0-rc.1` binary. A failed stable run blocks publication and requires a
reviewed correction and a new candidate when source or harness behavior
changes.

## Expected release assets

The draft GitHub Release must contain exactly these 12 uploaded assets:

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

## Tag and release gate

After stable artifacts pass independent review, request separate approval for:

- annotated tag `v0.3.0` at the exact stable workflow commit;
- pushing that tag without moving or replacing an existing ref;
- creating a draft, non-prerelease GitHub Release targeting the exact tag;
- uploading the 12 verified assets and confirming their remote hashes;
- confirming the repository's immutable-release protection applies to the
  final release; and
- final publication and latest-release designation.

No approval for one item implies approval for the next.

## Post-publication verification

After publication:

- fetch the public release metadata and confirm tag, target, immutability,
  non-draft, non-prerelease, latest status, and exact asset inventory;
- redownload public `SHA256SUMS`, installers, manifest, and representative
  binaries and compare them with the approved stable files;
- complete a fresh Windows network installation without administrator rights,
  confirm `engkit version --json` reports exactly `v0.3.0`, and inspect catalog
  `engineering-toolkit@0.3.0`; and
- record the immutable release URL, stable commit, asset digests, and observed
  verification in milestone context and release notes.

## Risk and rollback

Before publication, discard a failed stable artifact or draft and correct the
source under a new approval. Never rename candidate binaries, reuse a failed
build, move an existing tag, or replace bytes in an immutable release.

After publication, do not edit or replace immutable assets. Address a material
defect with a new patch release. TUF recovery, if later activated, must preserve
signature, threshold, expiration, hash, and rollback protections.
