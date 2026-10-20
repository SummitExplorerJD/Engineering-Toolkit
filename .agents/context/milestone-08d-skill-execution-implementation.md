# Milestone 08D: Skill execution implementation

## Approved boundary

This slice implements the accepted transactional contract for local canonical
skill creation, update, removal, and recovery. It does not execute skill
resources or synchronize provider files.

## Material facts

- Plans are rebuilt before execution and now include `executable: true`, so
  plans generated before this implementation require regeneration.
- `internal/skillexec` owns staging, per-skill locks, same-filesystem package
  publication, retained backups, journals, checkpoints, verification, and
  explicit recovery.
- Existing initialized workspaces receive only the missing `/recovery/*`
  ignore entry; new workspaces include the recovery directory and rule.
- Persisted plans, checkpoints, journals, lock records, and paths are validated
  before recovery. Truncated journals and ambiguous package content fail
  closed.
- Failure injection verifies recovery after the original package has moved and
  after a new package has been published.
- No dependency, network access, provider synchronization, remote install,
  script execution, credential access, Git operation, or release publication
  was added.

## Remaining gate

The local diff, full test suite, static analysis, and six-target build must be
reviewed before a separate Git commit or release decision.
