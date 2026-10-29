# Milestone 14A Doctor Human-Summary Contract

## Approved outcome

`v0.5.0A` defines a documentation-only additive presentation mode for the
existing read-only doctor. `engkit doctor --summary` and the exact short alias
`engkit doctor -s` will present the canonical findings as a concise list for a
person. `engkit doctor` remains canonical JSON, and `-h`/`--help` remain help.

## Contract

- The renderer consumes one completed `engkit.doctor-result/v1alpha1` result.
- Existing probe selection, stable codes, deterministic ordering, summaries,
  remediation, counts, health, and exit codes remain authoritative.
- Statuses map exactly to `[OK]`, `[WARN]`, `[ERROR]`, and `[N/A]`.
- Non-pass findings may show their existing bounded remediation as `Next:`.
- Raw evidence and paths are not rendered in the concise default.
- The footer maps canonical health to `Overall: OK` or
  `Overall: NEEDS ATTENTION` and prints canonical status counts.
- Output does not require color or Unicode and keeps human strings inside the
  CLI presentation boundary for later localization.

## Compatibility and safety

Default JSON output, schema, health calculation, and exit status do not change.
The renderer cannot install, repair, rebuild, recover, download, invoke an
integration, access credentials, contact a network service, approve a plan, or
execute remediation. Help performs no diagnostic discovery.

## Next approval packet

`v0.5.0B` must name the exact CLI source and test files, provide representative
golden or structural assertions for all labels and footer states, prove default
JSON byte compatibility, prove `--summary`/`-s` equivalence and help isolation,
and list focused, full, static-analysis, and cross-platform validation. It must
also explain line-width bounds, localization readiness, write-error behavior,
and rollback before any Go change is approved.

## Exclusions

This gate adds no code, dependency, public schema, configuration, persistence,
SQLite behavior, MCP surface, model or provider behavior, network access, Git
operation, workflow, installation, tag, or release action. Other `v0.5.0`
features require their own requirements and approval gates.

## Rollback

Revert the five `v0.5.0A` documentation changes. No runtime, workspace,
installation, database, Git, or external state was changed by this gate.
