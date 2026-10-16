# GitHub Pages TUF Staging Runbook

## Purpose

This runbook publishes the pre-production TUF repository to GitHub Pages. The
staging root uses provisional software keys held on one workstation. It must
never be embedded in a public Engineering Toolkit release or represented as a
production trust anchor.

## Safety properties

- The workflow is manual-only.
- GitHub receives public metadata and authenticated target bytes, never private
  keys.
- The workflow rebuilds all six executables and requires their lengths and
  SHA-256 digests to match signed targets metadata.
- Expired root, targets, snapshot, or timestamp metadata stops deployment.
- Pages deployment uses GitHub's OIDC flow with job-scoped `pages: write` and
  `id-token: write` permissions.
- Every third-party action is pinned to a reviewed commit SHA.

## Repository preparation

1. Review `tuf/staging/metadata` and confirm every file is public metadata.
2. Confirm `.engkit-private/` is ignored and absent from Git's index.
3. Confirm the staging metadata is not expired.
4. Commit and push the reviewed project changes only after separate approval.
5. In GitHub, open **Settings > Pages** and select **GitHub Actions** as the
   source.
6. Optionally add required reviewers to the `github-pages` environment before
   the first deployment.

## Deployment

1. Open **Actions > TUF staging Pages**.
2. Select **Run workflow** on the reviewed `main` branch.
3. Enter the exact stable version, initially `v0.1.0`.
4. Confirm the assemble job validates tests, key exclusion, metadata expiry,
   role thresholds, target paths, lengths, and SHA-256 digests.
5. Approve the `github-pages` environment if protection rules require it.
6. Record the resulting Pages URL and compare its `root.json` SHA-256 with the
   reviewed staging manifest.

## Expected paths

```text
/metadata/root.json
/metadata/1.root.json
/metadata/1.targets.json
/metadata/1.snapshot.json
/metadata/timestamp.json
/targets/releases/stable/v0.1.0/<os>-<arch>/<sha256>.engkit[.exe]
```

## Failure and rollback

Do not bypass an expiration, signature threshold, artifact hash, target path,
or private-key exclusion failure. Generate fresh staging metadata through an
approved local ceremony and review its public output instead.

To stop staging publication, disable GitHub Pages or pause the workflow. A
successful staging deployment does not authorize production activation,
release creation, root embedding, installation, or self-update.
