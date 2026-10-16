# Release Workflow

This workflow describes future release governance; it does not authorize a release.

## Preconditions

- The release scope, version, supported platforms, and compatibility statement are approved.
- Tests and security checks have observed passing evidence.
- The source revision is identified and the working tree is reviewed.
- TUF roles, thresholds, trusted root, expiration policy, and key custodians are approved.
- Signing and publication environments are separated from normal development authority.

## Stages

1. Build reproducible target artifacts for the approved platforms.
2. Generate provenance, checksums, and a software bill of materials where required.
3. Verify artifacts independently before signing metadata.
4. Update targets, snapshot, and timestamp metadata according to role separation.
5. Test installation, update, expiration, rollback-attack rejection, and recovery in an isolated environment.
6. Obtain separate approval to publish artifacts and metadata.
7. Publish in the safe order defined by the adopted TUF repository procedure.
8. Observe release health and retain a documented compromise-recovery path.

## Stop conditions

Stop on signature failure, threshold failure, expired or inconsistent metadata, unexpected artifact changes, incomplete provenance, failed platform tests, unavailable recovery keys, or any scope difference from the approved release plan.

