# TUF Staging Metadata

This directory contains public, signed metadata for the Engineering Toolkit
staging repository. It contains no private keys.

The trust root is provisional because its software keys are currently held on
one workstation. Do not embed this root in a public binary, use this repository
for production updates, or treat a successful Pages deployment as release
approval.

Snapshot and timestamp metadata intentionally expire quickly. Regenerate and
review the public metadata instead of extending or bypassing expiration checks.
