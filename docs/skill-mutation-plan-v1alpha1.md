# Skill mutation plan `v1alpha1`

## 1. Purpose and boundary

`engkit.skill-mutation-plan/v1alpha1` is the immutable preview contract for a
future local skill mutation. A plan describes exact observed inputs and proposed
filesystem operations. It does not authorize or execute them.

The `v0.2.0C` commands are:

```text
engkit skills plan-create [--root PATH] --source PACKAGE_PATH
engkit skills plan-update [--root PATH] --id SKILL_ID --source PACKAGE_PATH
engkit skills plan-remove [--root PATH] --id SKILL_ID
```

`--source` names a complete local canonical package. Planning validates and
hashes it without copying or modifying it. The absolute normalized source path
is included in the JSON plan because later execution must re-read and revalidate
the same explicit input.

## 2. Plan fields

| Field | Contract |
| --- | --- |
| `schema_version` | Exactly `engkit.skill-mutation-plan/v1alpha1`. |
| `kind` | `skill.create`, `skill.update`, or `skill.remove`. |
| `plan_id` | SHA-256 of the complete plan body excluding `plan_id`. |
| `root` | Absolute normalized workspace root. |
| `skill_id` | Canonical target identity. |
| `source_path` | Absolute source package path for create and update. |
| `source_sha256` | Digest of the source directory and file inventory. |
| `target_version` | Current version for update and remove. |
| `proposed_version` | Source version for create and update. |
| `requires_approval` | Always `true`. |
| `executable` | Always `false` in this milestone. |
| `preconditions` | States and digests that must still match before execution. |
| `operations` | Ordered proposed filesystem changes. |
| `verification` | Required post-execution checks for a future executor. |
| `recovery` | Required rollback constraints for a future executor. |

Plans contain no timestamps or random identifiers. Identical inputs and
filesystem state produce the same `plan_id`.

## 3. Preconditions and operations

A precondition contains `path`, `expected_state`, and an optional
`expected_sha256`. Package digests cover the sorted directory inventory and the
sorted path-and-digest file inventory, including `skill.yaml`.

Supported operation descriptions are:

- `create_directory`
- `write_file`
- `replace_file`
- `remove_file`
- `remove_directory`

File operations carry the applicable `before_sha256`, `after_sha256`, and
absolute `source_path`. Operations are ordered so that file-to-directory and
directory-to-file transitions remove blockers before creating replacements.

## 4. Planning policies

### Create

- The source must be a valid canonical `engkit.skill/v1alpha1` package.
- Its directory name, manifest identity, and destination identity must match.
- No destination path may already exist, including a case-insensitive collision.
- Existing `.agents` and `.agents/skills` paths must be real directories, not
  files or symbolic links.

### Update

- The target and source must both be valid canonical packages with the same ID.
- The proposed Semantic Version must be greater than the current version.
- The plan describes exact additions, replacements, removals, and path-type
  transitions using before and after hashes.

### Remove

- The target must be a valid canonical package.
- External, invalid, conflicting, or missing packages are refused.
- The plan removes only the exact package inventory and then its empty package
  directories; shared parent directories are preserved.

## 5. Security and future execution

Planning performs no writes, script execution, network access, credential
access, dependency resolution, imports, or provider rendering. Passing a plan
ID to any current command cannot execute it because no skill-plan execution API
exists in `v0.2.0C`.

A future executor must rebuild the plan immediately before execution, require
the same complete `plan_id`, keep every mutation inside the canonical package
root, journal each operation, maintain an atomic checkpoint, and apply the
documented recovery rules. That executor requires a separate approval and
implementation milestone.
