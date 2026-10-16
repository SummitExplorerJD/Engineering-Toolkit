# Security Policy

## Supported versions

Engineering Toolkit has not published its first release. Security support will
begin with `v0.1.0`; before then, only the current `main` development state is
reviewed.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Use the repository's
private vulnerability reporting or draft security-advisory feature in GitHub.
Include the affected version or commit, operating system, reproduction steps,
impact, and any suggested mitigation. Do not include credentials, private keys,
tokens, or unrelated personal data.

Maintainers should acknowledge a complete report within five business days.
No fix or disclosure date is promised until severity and reproducibility are
confirmed. Coordinated disclosure details will be agreed with the reporter.

## Distribution trust

Release assets, hashes, signatures, and TUF metadata are security-sensitive.
Never bypass signature, expiration, length, hash, or rollback verification to
work around an outage. Suspected release compromise follows the incident and
rotation procedures in `docs/secure-distribution-plan.md`.
