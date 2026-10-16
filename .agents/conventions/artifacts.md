# Artifact Conventions

## YAML configuration

- Human-authored, declarative, versioned, and schema-validated.
- Unknown fields fail unless a compatibility rule explicitly permits them.
- YAML content never authorizes arbitrary command execution.

## JSON plans

- Deterministic and immutable after approval.
- Include schema version, plan identifier, creation facts, inputs, preconditions, operations, affected paths, risks, approvals, and verification.
- Approval binds to the canonical plan digest and expires when the plan changes.

## JSONL journals

- One complete JSON object per line.
- Append-only; corrections are new events, not rewrites.
- Events include schema version, event identifier, workflow identifier, sequence, timestamp, type, and non-secret payload.
- A truncated final line must be detectable and recoverable without accepting fabricated success.

## JSON checkpoints

- Represent current resumable state, not full history.
- Written to a sibling temporary file, flushed as appropriate, and atomically replaced when supported.
- Reference the corresponding plan and latest journal sequence.

## SQLite context

- Derived and rebuildable from canonical artifacts.
- Never contains the only copy of approvals, plans, evidence, or recovery state.
- Schema migrations and privacy behavior require explicit approval.

