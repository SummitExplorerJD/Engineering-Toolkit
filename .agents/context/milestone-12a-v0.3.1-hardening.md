# Milestone 12A `v0.3.1` Post-Release Hardening Context

## Approved planning scope

`v0.3.1A` defines a validation-first maintenance boundary for published
`v0.3.0`. It changes documentation only. It does not authorize validation
execution, installation, network access, product changes, Git operations,
workflow dispatch, tagging, or publication.

The maintenance line exists to reproduce and correct compatibility-preserving
defects. It is not a vehicle for features deferred from `v0.3.0`. If all
approved validation scenarios pass, no `v0.3.1` release is required.

## Authoritative baseline

- Immutable tag: `v0.3.0`.
- Release commit: `2bd28273e072b2a33647fb5d7ba09968e381f27a`.
- Public release:
  `https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/tag/v0.3.0`.
- Uploaded asset inventory: the 12 names recorded in
  `docs/release-publication-v0.3.0.md`.
- Published package digest:
  `sha256:550d92230c1562d8d9f12ac5fcd5bed0284bfb1e2d06e0703529c1a07cac80fe`.
- Windows AMD64 executable digest:
  `sha256:8775446dce378fe29ca8d19895684208706a169996d738873fd7d3b0174e86ce`.

The post-publication user-scoped upgrade to `v0.3.0` is evidence that the
verified local asset set installs transactionally. It does not substitute for
the planned fresh public-network installation in an isolated directory.

## Planned validation boundary

The proposed `v0.3.1B` execution packet must bind an exact disposable root and
record its creation, expected contents, and cleanup policy before execution.
The existing installation at
`C:\Users\julit\AppData\Local\Programs\engkit\engkit.exe` remains out of
scope and must not be replaced, removed, or used as the test target.

The initial execution packet should cover:

1. public metadata, asset inventory, size, and digest comparison;
2. fresh Windows AMD64 installation without administrator rights or PATH
   mutation;
3. exact `v0.3.0` version and `engineering-toolkit@0.3.0` catalog inspection;
4. read-only doctor, catalog, and profile commands on synthetic roots;
5. unified-init planning and refusal without applying a real workspace change;
6. disabled, missing-context, and unavailable local-advice fallback without an
   Ollama request;
7. bounded MCP standard-input/output tool discovery and representative
   read-only calls; and
8. installer upgrade, injected failure, byte-exact rollback, and transaction
   cleanup inside a separately approved disposable installation root.

All workspace names, file contents, goals, and MCP inputs must be synthetic.
No real repository content may enter a prompt, protocol request, fixture, or
retained evidence record.

## Explicit exclusions

- No Ollama invocation or model installation, pull, update, replacement, or
  deletion.
- No compatibility change for `phi4-mini:latest`, `qwen3.5:0.8b`,
  `granite4:3b`, or exploratory `qwen3.5:4b`.
- No allowlist, prompt, deadline, reasoning, retry, or structured-response
  change.
- No new CLI command, MCP tool, profile, template, schema, persistence format,
  dependency, provider integration, network endpoint, or trust root.
- No production TUF activation, self-update, signing, notarization,
  marketplace, package-manager, or background-update behavior.

## Triage and stop conditions

A scenario failure is evidence, not implementation authority. The failure must
be reproduced, reduced to a bounded product assertion, and classified as a
product defect, environment limitation, external-service condition, or test
defect. Only a reproduced product defect may open `v0.3.1C`.

`v0.3.1C` must present exact source files, regression tests, compatibility
impact, risks, and rollback before any change. If all `v0.3.1B` scenarios pass,
the milestone records that result and stops without a candidate or release.
