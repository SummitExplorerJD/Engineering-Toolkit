# Distribution Threat Model

## 1. Scope

This threat model covers release discovery, TUF metadata refresh, target
selection, verified download, local trust state, and recovery for Engineering
Toolkit. It applies to the planned `engkit` distribution client and the
operator process that will eventually publish release metadata.

Milestone 5 is planning-only. No key, signed metadata, network client,
installer, CI workflow, or publishing path is created by this document.

## 2. Protected assets

- The bootstrap root embedded in a released `engkit` binary.
- The highest trusted metadata versions and last-known-good metadata cache.
- Release target identity: version, channel, operating system, architecture,
  byte length, and digest.
- The downloaded executable bytes before they are exposed for installation.
- Offline root keys and release-signing keys outside the application runtime.
- Operator approvals and evidence for metadata creation and publication.

## 3. Trust boundaries

| Boundary | Trusted for | Not trusted for |
| --- | --- | --- |
| Embedded bootstrap root | Initial TUF trust anchor | Freshness after release |
| TUF metadata | Authorized target identity and repository state | Build correctness or absence of malicious source code |
| HTTPS transport and mirror | Availability and confidentiality in transit | Artifact authenticity |
| Local metadata state | Rollback comparison and recovery | Surviving a malicious local administrator |
| Release CI | Reproducible build and evidence production | Root-key custody |
| Human release operators | Threshold authorization | Unreviewed unilateral publication |

TUF does not prove that source code or a build system is benign. A compromised
build that receives valid target authorization remains a valid TUF target.
Reproducible builds, provenance, source review, and CI hardening therefore
remain complementary Milestone 6 controls.

## 4. Adversaries and assumptions

The design assumes an attacker may control a network path, mirror, CDN,
release-hosting account, or a subset of signing keys. It also assumes metadata
and downloaded files may be truncated, replayed, reordered, mixed between
releases, or replaced. A local unprivileged user may tamper with files they can
write, and a process may be interrupted at any point.

The following are outside the protection claim:

- A malicious local administrator, kernel, or process with equivalent access.
- Compromise of enough independent root keys to satisfy the root threshold.
- Compromise of the build and the full threshold authorized to publish its
  malicious output.
- Guaranteed availability during repository, network, or clock outages.

## 5. Threats and required controls

| Threat | Required control | Failure behavior |
| --- | --- | --- |
| Arbitrary target substitution | Verify authorized target path, non-empty SHA-256 hash, and exact length through TUF | Reject and retain no installable output |
| Rollback | Persist trusted metadata versions; reject lower metadata and release versions | Fail closed with rollback diagnostics |
| Freeze or replay | Enforce metadata expiration with a trustworthy local clock | Fail closed; do not offer an override in normal commands |
| Mix-and-match metadata | Verify the complete timestamp, snapshot, targets chain and use consistent snapshots | Reject the inconsistent chain |
| Partial key compromise | Separate roles and use thresholds for root and targets | Accept only when the configured threshold is met |
| Root-key replacement | Process sequential root versions signed by both old and new thresholds | Stop at the last trusted root |
| Endless data or resource exhaustion | Set metadata and target byte limits, timeouts, redirect limits, and bounded retries | Cancel the operation and discard the partial file |
| Wrong platform or channel | Select only an exact allow-listed OS, architecture, format, and channel tuple | Report that no compatible target exists |
| Path traversal | Treat target paths as repository-relative data; reject absolute paths, traversal, separators outside the TUF path grammar, and ambiguous encodings | Reject before filesystem access |
| Cache tampering | Reverify cached metadata and targets; keep the bootstrap root outside the writable cache | Rebuild the cache from trusted state |
| Time-of-check/time-of-use race | Verify a closed temporary file, synchronize it, and publish it once without reopening by an attacker-controlled path | Delete staging output and retry from inspection |
| Interrupted download | Write to a unique temporary file and publish only after complete verification | Leave the prior cache entry untouched |
| Interrupted installation | Keep installation outside the first implementation slice; later require a platform-specific checkpoint and recovery design | Never report success for an incomplete replacement |
| Compromised multi-repository map | Do not enable TAP 4 multi-repository mode in the initial client | Refuse unsupported repository configuration |

## 6. Local state policy

The bootstrap root is compiled into the application and is never loaded from a
workspace or writable metadata cache. Trusted metadata and downloaded targets
are application state, not project state, and must not be stored under a
workspace `.engkit` directory.

Proposed per-user state roots are:

| Platform | Trusted metadata state | Download cache |
| --- | --- | --- |
| Windows | `%LOCALAPPDATA%\EngineeringToolkit\update\metadata` | `%LOCALAPPDATA%\EngineeringToolkit\cache\updates` |
| macOS | `~/Library/Application Support/EngineeringToolkit/update/metadata` | `~/Library/Caches/EngineeringToolkit/updates` |
| Linux | `$XDG_STATE_HOME/engkit/update/metadata`, falling back to `~/.local/state/engkit/update/metadata` | `$XDG_CACHE_HOME/engkit/updates`, falling back to `~/.cache/engkit/updates` |

Directories must be created with the most restrictive practical user-only
permissions. A future implementation must test permission and atomic-rename
semantics independently on all supported platforms.

## 7. Security validation required before implementation completion

- Official TUF client conformance fixtures for valid and invalid metadata.
- Expiration, rollback, freeze, mix-and-match, threshold, and root-rotation
  tests with a controlled clock.
- Target length, hash, maximum-size, redirect, timeout, and interrupted-write
  tests.
- Traversal and platform-confusion test vectors, including Windows reserved
  names and mixed path separators.
- Recovery tests proving that a failed refresh preserves last-known-good state.
- Dependency vulnerability review and a recorded minimum safe go-tuf version.
- Independent review of the embedded-root extraction and update boundary.

## 8. References

- [The Update Framework specification](https://theupdateframework.github.io/specification/)
- [go-tuf v2](https://github.com/theupdateframework/go-tuf)
- [go-tuf TAP 4 path-traversal advisory](https://github.com/theupdateframework/go-tuf/security/advisories/GHSA-jqc5-w2xx-5vq4)
- [python-tuf application deployment guidance](https://github.com/theupdateframework/python-tuf/blob/develop/docs/INSTALLATION.rst#application-deployment)

