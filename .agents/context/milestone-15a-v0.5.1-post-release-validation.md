# Milestone 15A `v0.5.1` Post-Release Validation Context

## Approved planning scope

`v0.5.1A` defines a validation-first maintenance boundary for published
`v0.5.0`. It changes documentation only. It does not authorize validation
execution, installation, network access, product changes, SQLite or MCP
execution, Git operations, workflow dispatch, tagging, or publication.

The maintenance line exists to reproduce and correct compatibility-preserving
defects. It is not a vehicle for features deferred from `v0.5.0`. If all
approved validation scenarios pass, no `v0.5.1` release is required.

## Authoritative baseline

- Immutable tag: `v0.5.0`.
- Release commit: `16384272abae8e7006805191d5f269c64330d2ff`.
- Public release:
  `https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.5.0`.
- Published package digest:
  `sha256:0fc676ac76a806f17d39ad1616adb0277af07bc23b1167c3e86d5a3309a34fae`.
- Windows AMD64 executable digest:
  `sha256:6ed6e32b6060b0a673d3ee66b4f4560b5a50ef6e5139e8a35d48123c07710071`.
- PowerShell installer digest:
  `sha256:ef7bd3c1d425d4514680fdc2ae2ccc39acf448640b7555a24718f96dc091d120`.
- Checksum-file digest:
  `sha256:b794388e672c7d9413a4eb6ac8f7aa095785cbcf9403d7f3fbab897a5519c454`.

The 12 uploaded release assets are:

1. `engkit-v0.5.0-darwin-amd64`;
2. `engkit-v0.5.0-darwin-arm64`;
3. `engkit-v0.5.0-linux-amd64`;
4. `engkit-v0.5.0-linux-arm64`;
5. `engkit-v0.5.0-windows-amd64.exe`;
6. `engkit-v0.5.0-windows-arm64.exe`;
7. `engkit-v0.5.0.zip`;
8. `install.ps1`;
9. `install.sh`;
10. `LICENSE`;
11. `release-manifest.json`; and
12. `SHA256SUMS`.

GitHub's generated source archives and release attestation are outside the
uploaded-asset inventory.

The post-publication user-scoped upgrade to `v0.5.0` verified the reviewed
local asset set, exact Windows AMD64 executable digest, canonical version JSON,
and healthy core doctor summary. It does not substitute for the planned fresh
public-network installation in an isolated directory.

## Planned validation boundary

The proposed `v0.5.1B1` execution packet must bind an exact disposable root and
record its creation, expected contents, and cleanup policy before execution.
The existing installation at
`C:\Users\julit\AppData\Local\Programs\engkit\engkit.exe` remains out of
scope and must not be replaced, removed, or used as the test target.

The B1 execution packet should cover:

1. public metadata, the 12-asset inventory, sizes, and digest comparison;
2. fresh Windows AMD64 installation without administrator rights or PATH
   mutation;
3. exact `v0.5.0` version and `engineering-toolkit@0.3.0` catalog inspection;
4. canonical doctor JSON, `--summary` and `-s` equivalence, isolated help, and
   required-error behavior;
5. explicit synthetic context sources, mandatory and user exclusions, SQLite
   v2 rebuild, freshness, inventory, bounded search, exact indexed read,
   synthesized tree, stale refusal, and projection cleanup;
6. exact twelve-tool MCP discovery plus representative application-equivalent
   calls in `compact` and `text` modes, non-duplication, strict inputs, result
   limits, safe errors, and stdio output isolation;
7. unified-init planning and ambiguous-state refusal without applying a real
   workspace mutation; and
8. disabled, missing-context, and unavailable local-advice fallback without an
   Ollama request.

The proposed `v0.5.1B2` execution packet must use a separate disposable
installation root and a known prior executable. It should cover explicit
upgrade, bounded failure injection after backup creation, byte-exact rollback,
successful replay to the approved `v0.5.0` digest, and removal of transaction
directories.

All workspace names, file contents, goals, and MCP inputs must be synthetic.
No real repository content may enter a prompt, protocol request, fixture, or
retained evidence record.

## `v0.5.1B1` exact execution packet

This section prepares B1 only. It does not authorize any command below. Network
access, root creation, installation, harness execution, and later cleanup each
remain separately approved actions.

### Bound paths and identities

The future execution root is exactly:

```text
C:\Users\julit\AppData\Local\Temp\engkit-v0.5.1-b1-public-install-20261030
```

It was observed absent while this packet was prepared. B1 must fail closed if
the root exists at execution time; it must not inspect, reuse, merge, empty, or
delete a pre-existing path. The approved layout is:

```text
engkit-v0.5.1-b1-public-install-20261030/
  download/
    install.ps1
  install/
    engkit.exe
  qualification/
    v050-release-qualification.json
    v040-regression/
      v040-release-qualification.json
  supplemental/
  evidence/
    v051-b1-validation.json
```

The existing user installation remains outside the root at
`C:\Users\julit\AppData\Local\Programs\engkit\engkit.exe`. B1 must record its
preflight version and SHA-256 and confirm both remain unchanged after every B1
step. It must not use that binary as the test target.

The only approved local harness files and prepared SHA-256 digests are:

| Harness | Bytes | SHA-256 |
| --- | ---: | --- |
| `scripts/test-release-binary.ps1` | 1,384 | `45a88dd8581051e5652ffdd12769dd7e392d5e400bab919aa9d72952f8511a83` |
| `scripts/test-v050-release-binary.ps1` | 12,288 | `4ec70062dc53b43c3065e86cabe8855cb8fd4f86a997b4f55c797adc8a03b1ef` |
| `scripts/test-v040-release-binary.ps1` | 17,888 | `e727b921dd2932c35c5447ae9d075f24cce13ac0644bbee00918d2c1624c4406` |

B1 must recompute all three hashes before creating the execution root. A size
or digest mismatch stops the run and requires packet review; it does not
authorize updating these values during execution. The v0.5.0 harness invokes
the v0.4.0 harness from the same `scripts` directory.

### Exact network boundary

Only these public GitHub resources are approved for the later B1 network step:

1. release metadata at
   `https://api.github.com/repos/SummitExplorerJD/Engineering-Toolkit/releases/tags/v0.5.0`;
2. `https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/download/v0.5.0/install.ps1`;
3. `https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/download/v0.5.0/engkit-v0.5.0-windows-amd64.exe`; and
4. `https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/download/v0.5.0/SHA256SUMS`.

Normal HTTPS redirects from those exact GitHub URLs are allowed. No credential,
cookie, API token, alternate mirror, branch archive, source archive, workflow
artifact, latest-release alias, package manager, or other endpoint is in scope.

Public metadata must report tag `v0.5.0`, non-draft, non-prerelease, immutable,
12 uploaded assets, and the exact release digests recorded in the authoritative
baseline. Any mismatch stops before installation.

### Ordered execution and assertions

After separate approval, B1 must execute in this order:

1. Reconfirm the root is absent, the three harness identities match, Git has no
   unexpected change, and the existing user installation identity is recorded.
2. Fetch and validate public release metadata without retaining response bodies
   in evidence.
3. Create only the exact root and its `download`, `install`, `qualification`,
   `supplemental`, and `evidence` subdirectories.
4. Download `install.ps1` to `download/install.ps1` and require SHA-256
   `ef7bd3c1d425d4514680fdc2ae2ccc39acf448640b7555a24718f96dc091d120`.
5. Run that exact script with `-Version v0.5.0 -InstallDir` bound to the isolated
   `install` directory. Do not pass `-Upgrade`, `-AddToPath`, or
   `-AssetDirectory`. The installer may fetch only the exact executable and
   checksum URLs above.
6. Require exit `0`, no `.engkit-install-*` directory, version JSON
   `engkit.version/v1alpha1` with version `v0.5.0`, and installed executable
   SHA-256
   `6ed6e32b6060b0a673d3ee66b4f4560b5a50ef6e5139e8a35d48123c07710071`.
7. Run `scripts/test-release-binary.ps1` against only the isolated executable
   with version `v0.5.0`.
8. Run `scripts/test-v050-release-binary.ps1` against only the isolated
   executable with version `v0.5.0` and the empty `qualification` directory.
   Require its six doctor scenarios, the nested ten v0.4.0 context/MCP
   scenarios, and exactly the two bounded qualification JSON records.
9. Use only `supplemental` to create bounded synthetic targets and assert:
   catalog schema `engkit.catalog/v1alpha1`, catalog version `0.3.0`, and exact
   profiles `go-cli@0.1.0`, `go-library@0.1.0`, and `go-service@0.1.0`;
   workspace-init and project-composition plan schemas without apply; refusal
   of a non-empty uninitialized target; read-only profile recommendation; and
   deterministic disabled and missing-context advice fallback. No configuration
   may contain an enabled fresh context projection, so no Ollama request path
   is reachable.
10. Temporarily restrict the child process `PATH` to the isolated root and
    require integrations doctor to report the Ollama executable unavailable
    while the MCP server finding remains available. Restore the parent `PATH`
    in a `finally` block and do not persist a PATH change.
11. Compare the existing user installation version and digest with the
    preflight values, write the bounded evidence record, inspect all retained
    paths, and stop. B1 does not delete the execution root.

No step may run `init apply`, composition apply, recovery, provider sync,
provider unsync, model execution, or an MCP mutation. The MCP server runs only
as a child of the existing v0.5.0 synthetic harness and receives only its fixed
synthetic JSON-RPC requests.

### Bounded B1 evidence record

`evidence/v051-b1-validation.json` uses planned schema
`engkit.v051-b1-validation/v1alpha1`. It may contain only:

- version, release commit, uploaded-asset count, and approved public digests;
- installer, installed executable, and three harness digests;
- catalog version and the three exact profile identities;
- the two nested qualification-record relative paths and SHA-256 digests;
- scenario identifiers, expected and observed exit status, bounded item and
  byte counts, sanitized classifications, and pass or fail outcomes;
- booleans confirming no PATH mutation, no Ollama request, no user-installation
  change, no transaction residue, and Git-state equivalence; and
- the execution-root basename, never its absolute parent path.

The record contains no raw release metadata, raw command output, MCP frames,
synthetic content, plan bodies, finding evidence, environment values, prompt,
model response, credential, username, or unrelated path. A failure record may
add only the failed scenario identifier, process exit status, and one sanitized
diagnostic limited to 512 UTF-8 bytes.

### B1 stop and rollback behavior

Any preflight mismatch stops before root creation. Any later failure stops the
sequence, writes bounded failure evidence when possible, preserves the exact
root for review, and does not retry, clean up, open B2, or propose a product
change. Because B1 never targets the existing installation, product rollback is
not expected. The installer must perform its own transactional rollback inside
the isolated install directory if publication fails.

After successful evidence review, removal of the exact B1 root is a separate
destructive approval. Passing B1 does not authorize B2, C, RC, a tag, or a
release.

## Evidence boundary

Retained evidence may record only exact commands, expected and observed exit
status, schema and scenario identifiers, version, catalog identity, asset and
executable digests, bounded counts and byte metrics, sanitized classifications,
and pass or fail outcomes. It must not retain credentials, prompts, model
responses, synthetic file contents, environment values, unrelated absolute
paths, raw MCP frames, or raw command output.

Temporary binaries, installers, workspaces, configurations, SQLite databases,
MCP requests and responses, transaction directories, and client helpers remain
inside the exact disposable roots. Their removal is destructive cleanup and
requires separate approval after evidence review.

## Explicit exclusions

- No Ollama invocation or model installation, pull, update, replacement, or
  deletion.
- No compatibility change for `phi4-mini:latest`, `qwen3.5:0.8b`,
  `granite4:3b`, or exploratory `qwen3.5:4b`.
- No allowlist, prompt, deadline, reasoning, retry, or structured-response
  change.
- No new CLI command, MCP tool, profile, template, schema, SQLite format,
  dependency, provider integration, network endpoint, or trust root.
- No remote MCP transport, automatic context rebuild, production TUF
  activation, self-update, signing, notarization, marketplace,
  package-manager, or background-update behavior.

## Triage and stop conditions

A scenario failure is evidence, not implementation authority. The failure must
be reproduced, reduced to a bounded product assertion, and classified as a
product defect, environment limitation, external-service condition, or test
defect. Only a reproduced product defect may open `v0.5.1C`.

`v0.5.1C` must present exact source files, regression tests, compatibility
impact, risks, and rollback before any change. If all approved B1 and B2
scenarios pass, the milestone records that result and stops without a
candidate, tag, or release.

## Validation of this planning gate

Planning review must confirm the exact three-file documentation inventory,
check links and cross-document consistency, search for statements that imply
execution authority, run `git diff --check`, and inspect the final changed-file
list. No product test is required because this gate changes no executable code.

## Rollback

Before any commit, revert only this context file and the associated
`docs/requirements.md` and `docs/roadmap.md` additions. No executable, schema,
configuration, SQLite projection, dependency, installation, or external state
requires rollback.
