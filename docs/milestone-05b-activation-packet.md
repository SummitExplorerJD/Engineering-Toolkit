# Milestone 5B Production Activation Packet

## 1. Status and decision boundary

**Status:** Prepared; blocked pending production trust and repository inputs.

This packet defines the review contract for exposing the Milestone 5A
verified-download core through the public CLI. It does not authorize a
production network request, key generation, signing ceremony, metadata
publication, CI change, installer, executable replacement, release, commit, or
push.

Implementation may begin only after the production inputs and approvals in
Section 12 are complete. Installation and self-replacement remain separate
milestones even after this activation is approved.

## 2. Activation outcome

The proposed activation adds two public commands:

```text
engkit update check [--channel stable]
engkit update download [--channel stable] [--version vMAJOR.MINOR.PATCH]
```

`check` refreshes and verifies TUF metadata, selects the exact supported
platform target, and reports whether a newer stable version exists. It does not
download a release target. `download` refreshes metadata, verifies the selected
target, and publishes a non-executable content-addressed cache entry. Neither
command installs or runs the downloaded file.

Successful output uses the previously reviewed
`engkit.update-check/v1alpha1` and `engkit.update-download/v1alpha1` JSON
contracts in the Milestone 5A packet. Usage errors return exit code 2. Network,
trust, policy, and filesystem failures return exit code 1. A new cross-command
JSON error envelope is not part of this slice.

## 3. Required production inputs

The following values must be supplied as one reviewed activation bundle. No
placeholder may reach a release build.

| Input | Required evidence | Acceptance condition |
| --- | --- | --- |
| Bootstrap `root.json` | Exact canonical bytes, root version, SHA-256 digest, expiration, role key IDs, thresholds, and ceremony record reference | Valid go-tuf root; expected 2-of-3 root and targets thresholds; no private material |
| Metadata base URL | Exact HTTPS origin and path prefix | Dedicated approved host; no credentials; final URL policy passes transport tests |
| Targets base URL | Exact HTTPS origin and path prefix | Same approved repository origin unless a separately reviewed host allow-list is provided |
| Repository ownership | Legal/operator owner, service owner, escalation contact, and change approvers | Ownership and incident authority are explicit |
| Release identity | Initial released CLI version and stable-channel target set | Canonical semantic version and all supported platform targets agree with signed custom metadata |
| Hosting controls | CDN/object-store behavior, TLS ownership, cache rules, redirect behavior, retention, access logs, and deletion controls | Immutable versioned objects; bounded redirects; stale timestamp risk addressed |
| Availability policy | Expected service level and planned outage behavior | Client fails closed while retaining last-known-good state |
| Time source assumption | Supported clock-skew policy and operator recovery procedure | No expiration bypass; recovery uses clock correction or fresh signed metadata |

The activation bundle must also record the SHA-256 digest of the reviewed
bootstrap root independently of the repository hosting account. Private keys,
recovery phrases, tokens, cookies, and signing credentials must never be added
to this repository, a plan, a journal, a diagnostic bundle, or chat.

## 4. Repository descriptor and trust bootstrap

Production builds should compile an immutable descriptor that contains only:

```go
type RepositoryDescriptor struct {
    MetadataBaseURL string
    TargetsBaseURL  string
    BootstrapRoot   []byte
    BootstrapSHA256 string
}
```

The concrete representation may use an internal embedded file for
`root.json`, but the bytes and expected digest must be reviewable in the source
diff. Build startup or command construction must verify the embedded digest
before creating a TUF client.

Trust and repository locations must not be overridden by workspace YAML,
`.engkit`, environment variables, command-line flags, the writable metadata
cache, DNS-discovered configuration, or redirect responses. Test-only
constructors may inject local repositories from `_test.go` files. Development
builds without an approved descriptor must return `update_not_configured`
without making a network request.

## 5. Planned implementation file set

This is a proposal for a later, separately approved implementation diff:

```text
internal/update/repository.go
internal/update/repository_test.go
internal/update/service.go
internal/update/service_test.go
internal/update/embedded/root.json
internal/cli/run.go
internal/cli/run_test.go
README.md
docs/roadmap.md
.agents/context/milestone-05b-production-activation.md
```

The final approval must replace this list if inspection proves that fewer or
different files are required. No new Go dependency is expected. Any dependency
or public-schema change reopens its own approval gate.

The embedded root is public signed metadata, not a secret. It is nevertheless
security-critical and must be changed only through the root-rotation procedure.
No generated private key, signing helper, CI workflow, installer, or release
artifact belongs in this file set.

## 6. CLI composition rules

The CLI boundary supplies the current build version and platform to the update
service. The update service owns orchestration; policy remains in
`internal/update`, and TUF wire behavior remains in `internal/update/tufclient`.
The CLI must not implement signature, version, target-path, or cache policy.

Only the `stable` channel is accepted. An omitted `--version` on `download`
means the newest verified compatible stable target. A supplied version must be
canonical, present in verified metadata, newer than or equal to the currently
running release according to the approved no-downgrade policy, and compatible
with the current OS and architecture.

The output cache path is evidence, not an instruction to execute. Help text and
documentation must state that installation is unavailable. A development
version that cannot be compared under the release policy must fail with a
stable policy error rather than silently treating itself as an old release.

## 7. Network and hosting approval boundary

Approving implementation of this packet would authorize network access only
for the two update commands and only to the compiled repository descriptor.
It would not authorize telemetry, background checks, startup checks, automatic
downloads, credential use, mirrors, arbitrary proxies, or any network access
from other commands.

The existing bounded transport remains mandatory: HTTPS only, explicit host
validation, bounded redirects, retries, timeouts, metadata sizes, and target
sizes. Redirects must remain within the approved host policy. System proxy
behavior must be tested, but proxy credentials must not be read from workspace
files or written to diagnostics.

Hosting must preserve immutable versioned target objects and versioned TUF
metadata. Publication order is targets, snapshot, then timestamp. A CDN must
not cache timestamp metadata beyond its signed validity or serve mixed
repository generations. Destructive object replacement is prohibited; recovery
publishes a new signed metadata version.

## 8. Validation plan

Before production activation, a staging repository with disposable keys must
exercise the exact production topology and descriptor shape. Tests must cover:

- successful `check` with both `current` and `update_available` results;
- successful `download` with exact JSON fields and a non-executable cached file;
- development/unconfigured builds making zero network requests;
- expired metadata, rollback, freeze, mix-and-match, invalid signatures,
  threshold failure, skipped root rotation, and incompatible target metadata;
- target length/hash failure, interrupted downloads, cache collisions, and
  last-known-good preservation;
- supported CDN redirects and rejection of cross-origin, downgrade, traversal,
  and redirect-loop cases;
- proxy/no-proxy operation without credential leakage;
- DNS failure, TLS failure, timeout, HTTP rate limiting, partial outage, and
  repository unavailability;
- root rotation signed by both old and new thresholds;
- Windows, Linux, and macOS state/cache permissions and native smoke tests on
  `amd64` and `arm64`; and
- concurrent processes, race instrumentation in a working toolchain, and crash
  recovery around metadata/cache publication.

Required repository validation includes formatting, focused and full tests,
`go vet`, `govulncheck`, `git diff --check`, link validation, embedded-root
digest verification, and six-platform compilation. Production traffic remains
disabled until the staging evidence is reviewed.

## 9. Operational readiness

Before activation, operators must have:

- a documented release runbook with two-person targets approval;
- independent root and targets custody records;
- snapshot/timestamp credential scope and rotation records;
- monitoring for metadata age, signature/threshold failures, origin and CDN
  errors, unexpected redirects, publication lag, and target mismatch reports;
- alerts scheduled before half of every role validity window elapses;
- retained, access-controlled ceremony and release evidence;
- an incident owner and tested contact path; and
- a rehearsal for origin compromise, online-key compromise, accidental bad
  metadata, lost key, expired metadata, and CDN rollback.

Logs must not contain private keys, tokens, proxy credentials, full environment
dumps, or user workspace paths. Client diagnostics should identify only the
stable error category, repository role/version needed for support, and a
correlation identifier when one is explicitly designed.

## 10. Incident response and rollback

The client always fails closed. Invalid or expired new metadata must not replace
last-known-good metadata. Suspected target or targets-key compromise requires
publication to stop, affected objects to be retained for investigation, and a
root-authorized targets-key rotation before service resumes.

Before public release, code rollback is the removal of the CLI composition and
production descriptor while retaining the already validated internal 5A core.
After public release, an already distributed root cannot be revoked by deleting
it from source or hosting. Recovery must publish sequential, dual-threshold
root rotations and a newly built client when required by the TUF trust model.

An emergency unactivation build may omit the repository descriptor so update
commands return `update_not_configured`; it must never substitute unsigned
metadata, disable expiration, accept HTTP, or reset trusted versions. Removing
server objects is not a safe rollback because it can strand clients and erase
incident evidence.

## 11. Explicit exclusions

Milestone 5B does not include:

- key generation or any private-key custody operation;
- CI/CD, signing automation, publication, installer, package manager, or
  executable replacement;
- background, startup, or scheduled update checks;
- beta/nightly channels, mirrors, delegated targets, TAP 4, delta updates, or
  automatic execution;
- telemetry or a remote configuration service; or
- commit, push, tag, GitHub release, or deployment.

A later release-operations slice may implement signing and publication only
after the key ceremony and hosting controls are independently approved. A later
installation slice must define immutable plans, user approval, privilege
boundaries, platform signing/notarization, rollback, and Windows replacement.

## 12. Approval checklist

Planning is complete when this packet is reviewed. Production activation may
begin only when every implementation input below is checked:

- [ ] Exact canonical production `root.json` bytes are supplied.
- [ ] Root SHA-256 digest, version, expiration, role keys, and thresholds are
      independently verified against an approved ceremony record.
- [ ] Exact metadata and target HTTPS base URLs are supplied and owned by the
      approved operator.
- [ ] Hosting, CDN, TLS, redirects, caching, retention, logging, and deletion
      controls are reviewed.
- [ ] Initial stable release version and six-platform target set are fixed.
- [ ] Release, security, and incident owners accept their responsibilities.
- [ ] Staging uses the production topology with disposable keys and passes the
      validation plan.
- [ ] The final exact file diff, CLI contract, network scope, risk, and rollback
      plan receive explicit implementation approval.
- [ ] CI, signing, publication, installation, self-replacement, commit, and
      push remain separately unapproved unless named in a later approval.

Until these items are complete, the correct state is a documented but
unconfigured updater that performs no production network request.
