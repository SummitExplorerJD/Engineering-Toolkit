# ADR 0006: User-scoped bootstrap installers

## Status

Accepted for `v0.1.1` implementation.

## Context

`v0.1.0` publishes six raw executables and requires users to download, verify,
rename, install, and optionally add the destination to `PATH` manually. The
internal TUF client is not production-activated, and the CLI exposes no update
or installation command. A safer convenience path is needed without implying
automatic updates, self-replacement, package-manager integration, or production
TUF trust.

Bootstrap installation is different from application-managed installation. A
standalone script is invoked directly by the user before `engkit` is necessarily
available. A future `engkit update install` operation remains subject to an
immutable JSON plan, explicit approval, TUF verification, journaling, and
recovery requirements.

## Decision

Publish two reviewed bootstrap helpers with each release:

- `install.ps1` for Windows.
- `install.sh` for Linux and macOS.

Both helpers require an exact stable or release-candidate version. They derive
the artifact name from the detected operating system and architecture, download
only from the fixed official GitHub Releases location unless an explicit local
asset directory is supplied, require exactly one matching `SHA256SUMS` entry,
verify SHA-256, execute the staged binary only after verification to confirm its
version, and install into a user-scoped directory.

An existing installation is never replaced unless the user supplies
`-Upgrade` or `--upgrade`. Replacement uses same-filesystem staging and restores
the previous executable if publication fails. If automatic restoration itself
fails, the transaction preserves the backup and reports its path instead of
deleting it. PATH modification is opt-in and idempotent. The scripts do not use
elevation, mutate project workspaces, invoke package managers, or access
application TUF state.

Local asset-directory mode exists for downloaded release bundles, air-gapped
transfer, and release-candidate testing. A local checksum file detects damage
but does not establish provenance; the operator remains responsible for the
source of local assets.

The release manifest retains schema `engkit.release-manifest/v1alpha1` and gains
an additive `installers` collection containing filename, size, and SHA-256. Both
scripts are also covered by `SHA256SUMS`.

## Security boundary

For `v0.1.1`, network bootstrap authenticity depends on HTTPS transport and the
GitHub repository boundary. A checksum downloaded from the same release detects
corruption but is not an independent signature. The scripts must not claim TUF,
code-signing, or package-manager assurance.

The fixed network base cannot be overridden. Local mode is explicit and never
silently selected. Test-only failure and PATH-state hooks are accepted only when
local asset mode is active and are not documented as user interfaces.

## Consequences

- Installation is shorter, repeatable, and safer than manual file replacement.
- Release CI can exercise installation and recovery against locally built
  candidate assets before publication.
- The scripts add a small public command-line surface that must remain backward
  compatible throughout the `v0.1.x` line.
- Production TUF activation, signed executables, and application-managed
  installation remain independent future decisions.

## Rejected alternatives

- **Use `latest`:** rejected because it makes installation non-deterministic and
  can cross stable/prerelease boundaries.
- **Overwrite automatically:** rejected because replacement must express user
  intent and preserve a working installation.
- **Use administrator or root destinations:** rejected because elevation
  expands risk and is unnecessary for the initial CLI.
- **Activate the internal updater:** rejected because production roots,
  endpoints, and operational controls are not approved.
- **Publish package-manager definitions now:** rejected because each ecosystem
  adds signing, ownership, moderation, and update-policy commitments.

