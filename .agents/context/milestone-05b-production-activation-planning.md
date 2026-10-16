# Milestone 5B Production Activation Planning Context

## Current state

The Milestone 5A internal verified-download core is implemented and validated,
but it is intentionally unreachable from the public CLI. The application has
no production bootstrap root or repository endpoint and therefore makes no
production update request.

Milestone 5B planning produced
`docs/milestone-05b-activation-packet.md`. Implementation is blocked pending a
reviewed production activation bundle.

## Required activation bundle

- Canonical signed production `root.json` bytes.
- Independently verified SHA-256 digest, root version, expiration, public role
  keys, thresholds, and ceremony evidence reference.
- Exact metadata and target HTTPS base URLs.
- Repository, release, security, and incident ownership.
- Initial stable release version and supported target inventory.
- Origin/CDN behavior for TLS, redirects, caching, retention, logging, and
  deletion controls.
- Staging evidence against the production topology using disposable keys.

No private key, recovery phrase, token, cookie, or signing credential belongs
in the repository or this context record.

## Frozen boundaries

- Trust root and endpoints are compile-time release inputs and cannot be
  overridden by workspace configuration, `.engkit`, YAML, environment
  variables, command-line flags, or writable state.
- `engkit update check` may verify metadata but not download a target.
- `engkit update download` may publish only a verified, non-executable,
  content-addressed cache entry.
- Installation, self-replacement, background checks, CI, signing, publication,
  commit, and push require later independent approvals.
- An unconfigured or development build fails with `update_not_configured` and
  makes no network request.

## Next decision

After the production activation bundle exists, inspect every input, update the
packet's checklist, present the final exact code/file diff, network scope,
validation commands, risks, and rollback plan, and wait for explicit
implementation approval before changing CLI or production trust state.
