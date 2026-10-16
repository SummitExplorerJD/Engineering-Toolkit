# Milestone 06B: GitHub TUF Staging Preparation

## Scope

Prepare a manual GitHub Pages deployment for public, pre-production TUF
metadata and authenticated `v0.1.0` targets. Keep all private keys local and
ignored. Do not commit, push, enable Pages, run the workflow, create a release,
or embed the provisional root as part of this milestone.

## Decisions

- GitHub Pages is the proposed staging origin.
- Deployment is manual-only and uses the protected `github-pages` environment.
- CI rebuilds targets from the reviewed commit and refuses bytes that do not
  match the signed TUF length and SHA-256 values.
- Provisional root and targets metadata may be reviewed in Git because they
  contain public keys and signatures only.
- Snapshot and timestamp expiry remains fail-closed; staging metadata must be
  refreshed rather than bypassed.
- GitHub Actions are pinned to full commit SHAs.

## Validation gate

- Confirm public metadata contains no private key field or private PEM file.
- Assemble the Pages tree locally and verify all six signed targets.
- Run the complete Go test suite and static analysis.
- Validate workflow syntax with `actionlint`.
- Confirm `.engkit-private/` remains ignored and untracked.

## Next approval

Review the complete local diff, then separately approve commit and push. After
the reviewed commit reaches GitHub, separately approve enabling Pages and
running the staging workflow.
