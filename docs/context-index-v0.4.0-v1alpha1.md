# Context Index Contract `v1alpha1`

## 1. Status and authority

This document defines the approved documentation-only `v0.4.0B1` packet for a
replaceable SQLite v2 projection and its application boundary. It refines the
direction accepted in ADR 0019.

B1 does not authorize implementation, configuration edits, database creation or
replacement, dependency changes, real-project indexing, MCP execution, model
use, network access, Git operations, or publication. B2 requires a separate
approval against the exact file and test inventory in this document.

## 2. Compatibility boundary

The human-authored configuration keeps schema identifier
`engkit.config/v1alpha1`. A new optional `context` mapping is additive; existing
files that omit it remain valid and retain the current documentation-only source
inventory.

The derived database advances from SQLite `user_version = 1` to
`user_version = 2`. It is not migrated in place. A v1 database remains intact
until a complete v2 staging database has been built, verified, closed, and
atomically published. Read operations report v1 as `unsupported_schema` and
require an explicit `engkit context rebuild`; they never silently rebuild.

The existing `engkit.context-index/v1alpha1` public result retains its current
required fields. B2 may add optional fields but may not remove or reinterpret
existing fields. New source-inventory and excerpt-read results use independent
schemas so they do not overload the legacy search contract.

## 3. Configuration contract

The optional configuration is:

```yaml
schema_version: engkit.config/v1alpha1
context:
  source_roots:
    - .
    - internal
  exclude_paths:
    - internal/testdata/private
```

`context` rules are:

- When omitted, source-code expansion is disabled and current canonical
  documentation, configuration metadata, plans, journals, and checkpoints are
  inventoried as before.
- When present, `source_roots` is required and contains 1 through 32 unique
  normalized workspace-relative directory paths.
- `.` is an explicit root value. Empty, absolute, drive-qualified, UNC,
  backslash-separated, dot-segment, escaping, NUL-containing, or linked roots
  are invalid. Each encoded path is at most 512 UTF-8 bytes.
- `exclude_paths` is optional and contains at most 128 unique normalized
  workspace-relative file or directory prefixes. `.` is not an exclusion.
  Excluding `a/b` excludes that entry and descendants of that directory; it
  does not exclude sibling prefixes such as `a/beta`.
- User exclusions can only narrow discovery. They cannot re-enable a mandatory
  exclusion, add an extension, follow a link, or raise a limit.
- Configuration paths are slash-separated and case-sensitive canonical values.
  A case-insensitive filesystem collision or two configured paths resolving to
  the same entry fails validation rather than selecting one implicitly.
- Glob, regular-expression, environment-variable, home-directory, and Git
  ignore expansion are not supported. This keeps source selection portable and
  inspectable without invoking Git. Users must list additional ignored or local
  trees explicitly.

## 4. Deterministic source eligibility

The existing canonical inventory remains independently classified. Source-root
discovery adds only files that pass every rule below, then deduplicates them by
normalized path.

An eligible source is a regular file reached without a symbolic link, junction,
or other reparse-point ancestor. It is at most 1 MiB, is valid UTF-8 after an
optional leading UTF-8 byte-order mark, and contains no NUL byte. A file that
changes identity, size, or modification time while being read fails the entire
rebuild. An unreadable candidate also fails the rebuild; it is not silently
omitted.

Extensions are matched case-insensitively against this fixed allowlist:

```text
.c .cc .cpp .cxx .cs .css .dart .go .h .hh .hpp .hxx .html .java
.js .jsx .json .kt .kts .lua .md .markdown .php .proto .py .rb .rs
.scss .sh .sql .swift .toml .ts .tsx .txt .xml .yaml .yml .zig
```

These exact basenames are also eligible case-sensitively:

```text
CMakeLists.txt Dockerfile Gemfile Gemfile.lock Makefile
Cargo.lock Cargo.toml go.mod go.sum go.work go.work.sum
package.json package-lock.json pnpm-lock.yaml pyproject.toml
requirements.txt yarn.lock
```

The following directory segments are mandatory case-insensitive exclusions at
any depth:

```text
.git .hg .svn .engkit .agents .cache .next .nuxt .terraform .venv
DerivedData Pods __pycache__ bin build coverage dist node_modules obj
out target third_party vendor venv
```

Existing canonical collectors remain responsible for the approved `.engkit`
and `.agents` metadata or documentation inventory. Source expansion never
re-enters those trees.

The following path forms are mandatory case-insensitive secret exclusions:

- basename `.env` or any basename beginning `.env.`;
- basename `.netrc`, `_netrc`, `.npmrc`, or `.pypirc`;
- basename `credentials.json`, `credentials.yaml`, `credentials.yml`,
  `secrets.json`, `secrets.yaml`, or `secrets.yml`;
- basename `id_rsa`, `id_dsa`, `id_ecdsa`, or `id_ed25519`; and
- suffix `.key`, `.pem`, `.p12`, `.pfx`, `.jks`, `.keystore`, `.kdbx`, or
  `.ovpn`.

This path policy is defense in depth, not secret detection. B2 performs no
content-classification or entropy scan and must document that users remain
responsible for explicit exclusions of project-specific sensitive paths.

## 5. Rebuild limits and exclusion behavior

The v2 hard limits are:

| Boundary | Limit |
| --- | ---: |
| Configured source roots | 32 |
| Configured exclusion prefixes | 128 |
| Normalized relative path | 512 UTF-8 bytes |
| Candidate files after mandatory path exclusions | 20,000 |
| One eligible textual file | 1 MiB |
| Total eligible source bytes | 128 MiB |
| Stored chunks | 50,000 |
| One normalized chunk | 8 KiB |
| Lines per chunk | 120 |
| Published database file | 512 MiB |

A linked, non-regular, disallowed-extension, mandatory-secret, user-excluded,
binary, or oversized individual file is omitted and counted under one stable
reason. The database stores only aggregate counts for:

```text
binary_or_invalid_utf8 disallowed_extension linked_or_non_regular
mandatory_path mandatory_secret oversized_file user_excluded
```

It stores no excluded path or content. Exceeding the candidate-file, aggregate
source-byte, stored-chunk, or database-file ceiling fails the rebuild before
publication. This prevents a successful but silently partial project index.

## 6. Normalization and chunk identity

The source SHA-256 is computed over exact file bytes. A leading UTF-8
byte-order mark is removed from searchable text, and CRLF or lone CR line
endings are normalized to LF. The normalized file content receives a separate
SHA-256 digest.

Chunks are produced in source order without overlap:

1. append complete normalized lines until the next line would exceed either
   8 KiB or 120 lines;
2. publish the non-empty chunk and continue with the next line;
3. split a single line longer than 8 KiB at the largest valid UTF-8 boundary;
   all pieces retain that source line number; and
4. omit content chunks for an empty file while retaining its source metadata.

Each chunk records a zero-based ordinal, inclusive one-based start and end line,
normalized content, and `sha256:` content digest. Its stable identifier is the
SHA-256 of this length-delimited sequence:

```text
schema identifier, normalized source path, source SHA-256,
ordinal, start line, end line, chunk content SHA-256
```

No overlapping text is stored or returned merely to provide context. A caller
requests adjacent chunks explicitly when needed.

## 7. SQLite v2 schema

The staging database uses `PRAGMA user_version = 2`, foreign keys, full
synchronous writes, one open connection, and the existing delete-journal
publication discipline.

`metadata` remains a strict key/value table and contains at least:

```text
schema_version = engkit.context-index/v1alpha1
policy_version = engkit.context-policy/v1alpha1
configuration_sha256
source_fingerprint
rebuilt_at
source_count
source_bytes
chunk_count
excluded_count
```

`sources` is a strict table with:

```text
path TEXT PRIMARY KEY
kind TEXT NOT NULL
language TEXT NOT NULL
size_bytes INTEGER NOT NULL
modified_ns INTEGER NOT NULL
source_sha256 TEXT NOT NULL
normalized_sha256 TEXT
chunk_count INTEGER NOT NULL
```

`chunks` is a strict table with:

```text
rowid INTEGER PRIMARY KEY
chunk_id TEXT NOT NULL UNIQUE
source_path TEXT NOT NULL REFERENCES sources(path) ON DELETE CASCADE
ordinal INTEGER NOT NULL
start_line INTEGER NOT NULL
end_line INTEGER NOT NULL
content_sha256 TEXT NOT NULL
content TEXT NOT NULL
UNIQUE(source_path, ordinal)
```

`exclusion_summary` is a strict table containing `reason TEXT PRIMARY KEY` and
`count INTEGER NOT NULL`. It contains no paths.

Indexes cover `sources(kind, language, path)`, `chunks(source_path, ordinal)`,
and exact chunk identity. `chunk_search` is an FTS5 external-content virtual
table over `chunks.content` using the built-in `unicode61` tokenizer. B2 may
import the FTS5 extension from the already pinned
`github.com/ncruces/go-sqlite3 v0.35.4`; it adds no module and changes neither
`go.mod` nor `go.sum`. Database creation must prove FTS5 availability before
reading any workspace source and fail without replacing the current index when
the extension is unavailable.

## 8. Freshness and publication

The configuration digest covers the canonical decoded `context` selection and
policy version, not YAML presentation. The source fingerprint covers, in sorted
path order, the configuration digest plus every accepted source path, kind,
language, exact source SHA-256, size, and modification time. The legacy
canonical inventory participates under the same deterministic ordering.

Inspection recollects eligible metadata and exact hashes. Any configuration,
policy, path, content, size, or modification-time change marks the projection
stale. Missing configured roots, unreadable entries, limit overflow, or an
unsupported database schema produces a stable unavailable status rather than a
fresh partial view.

Rebuild writes only to a random staging file beneath `.engkit/context`, creates
all rows and FTS data in one transaction, runs `PRAGMA integrity_check`, checks
metadata counts and the 512 MiB file ceiling, closes the database, and then uses
the existing backup-and-rename publication. Publication failure restores the
previous exact database. Staging, journal, WAL, shared-memory, and backup files
are removed after success or recoverable failure.

## 9. Application result contracts

`engkit.context-status/v1alpha1` is content-free. It reports existence,
supported schema, freshness, policy and projection digests, configured-root
count, source/chunk/exclusion counts, source/index bytes, rebuild timestamp, and
one stable reason when unavailable. It does not expose the absolute database
path.

`engkit.context-sources/v1alpha1` returns sources ordered by normalized path.
It supports exact path-prefix, kind, and language filters; a default page of 50
and hard limit of 200; and an optional exclusive `after` path. A continuation
requires the projection fingerprint returned by the prior page and fails if the
projection changed. Items contain metadata and digests but no content.

The existing search result becomes additively richer. Search accepts 1 through
512 UTF-8 bytes, trims outer whitespace, treats the input as literal text rather
than FTS syntax, and converts whitespace-separated terms into escaped FTS
phrases joined with AND. Filters cover path prefix, kinds, and languages.
Defaults are 5 results, 512 snippet bytes, and 4 KiB combined snippet content;
hard limits are 20 results, 1,024 bytes per snippet, and 16 KiB combined.
Ordering uses FTS rank, then path and chunk ordinal. Raw floating-point rank is
not a public field. Each result includes chunk identity, path, kind, language,
line range, content digest, snippet, and truncation state.

`engkit.context-read/v1alpha1` requires one exact indexed path and either one
chunk identity or an inclusive line range, never both. A line range spans at
most 400 lines. The default response budget is 8 KiB and the hard limit is
32 KiB. It returns stored normalized text only, plus source and content digests,
line range, applied budget, truncation, and the next line when continuation is
available. It never opens a canonical file during retrieval.

All application operations require the same fresh projection. They apply
budgets before serialization and return no partial structured result after a
policy, freshness, cancellation, or encoding failure.

## 10. CLI boundary proposed for B2

The existing commands remain and gain optional filters and budgets:

```text
engkit context rebuild --root PATH
engkit context status --root PATH
engkit context search --root PATH [--path-prefix PATH] [--kind KIND]
  [--language LANGUAGE] [--limit N] [--max-content-bytes N] QUERY
```

B2 also proposes read-only commands:

```text
engkit context sources --root PATH [--path-prefix PATH] [--kind KIND]
  [--language LANGUAGE] [--limit N] [--after PATH]
  [--projection SHA256]
engkit context read --root PATH --path FILE
  (--chunk SHA256 | --start-line N --end-line N) [--max-bytes N]
```

Repeated `--kind` and `--language` flags are not accepted in B2; C1 may propose
MCP arrays independently. Commands emit one versioned JSON document and no
source content to standard error.

## 11. Exact proposed B2 file inventory

No file below is authorized to change by B1. The proposed B2 implementation is
limited to:

- `internal/workspaceconfig/config.go`
- `internal/workspaceconfig/config_test.go`
- `internal/contextsearch/search.go`
- `internal/contextsearch/search_test.go`
- new `internal/contextview/context.go`
- new `internal/contextview/context_test.go`
- `internal/index/sqlite/index.go`
- `internal/index/sqlite/index_test.go`
- `internal/index/sqlite/sources.go`
- new `internal/index/sqlite/chunks.go`
- `internal/index/sqlite/advice.go`
- `internal/index/sqlite/advice_test.go`
- `internal/cli/run.go`
- `internal/cli/run_test.go`
- `internal/doctor/doctor.go`
- `internal/doctor/doctor_test.go`
- `.agents/context/milestone-13b2-context-index-implementation.md`

`go.mod`, `go.sum`, MCP code, catalog content, workspace initialization output,
public schemas outside the additive context contracts, and release files remain
unchanged. Newly initialized workspaces continue to omit `context` and retain
documentation-only indexing until a later explicit configuration edit.

## 12. B2 verification requirements

B2 must run focused configuration, path, eligibility, exclusion, chunk,
schema, FTS, search, read, freshness, replacement, failure-injection, CLI,
doctor, and local-intelligence regression tests first. It must then run format,
`go vet ./...`, `go test ./...`, `git diff --check`, and six-target CGO-disabled
cross-compilation.

Synthetic scale fixtures must exercise 20,000 candidates, 128 MiB aggregate
refusal, 50,000-chunk refusal, one-MiB file handling, 512-MiB database refusal
through an injected size probe, cancellation, and publication rollback. Tests
must not allocate those maxima simultaneously when a deterministic injected
counter or boundary seam proves the same condition.

Token-efficiency fixtures declare a bounded full-file baseline and a staged
status/search/read sequence. They pass only when the staged sequence retains the
required answer evidence, returns no excluded content, and records both byte
counts and their reduction ratio. No real repository content, model prompt, or
provider runtime participates in B2.

## 13. Risks and rollback

The primary risks are secret-bearing paths not covered by fixed exclusions,
resource amplification during rebuild or FTS creation, relevance loss from
small non-overlapping chunks, platform path differences, v1 compatibility, and
stale advice retrieval.

B2 rollback removes the optional configuration fields and new application
operations, restores the v1 index implementation and CLI surface, and deletes
only the derived `.engkit/context/index.db` when explicitly authorized. Canonical
files and configuration authored before B2 remain valid. A v2 index is never
downgraded in place; the restored implementation requires an explicit rebuild.
