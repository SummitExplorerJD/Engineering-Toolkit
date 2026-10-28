# Milestone 13D1 Local Real-Project Pilot

## Approved scope

The separately approved D1 pilot selected the current Engineering Toolkit
working tree as its only real project. Execution used a disposable local copy so
the original repository required no initialization, configuration, index, or
canonical-file mutation. The snapshot included the current uncommitted A through
C2 work and therefore represents local development state rather than a release
or immutable Git revision.

The copy excluded `.git`, `.engkit`, `.engkit-private`, `.engkit-test-temp`,
`.pages-tuf-staging`, and `dist` before configuration or indexing. The pilot
then applied the same explicit exclusions inside its temporary context
configuration as defense in depth. A preflight check failed closed if any
excluded top-level directory appeared in the copy.

## Local execution boundary

A temporary CGO-disabled `engkit` binary rebuilt SQLite v2 inside the disposable
copy. A temporary client using the pinned official Go MCP SDK launched that
binary through command-backed standard input/output. It verified the exact
twelve-tool inventory and exercised content-free status, paged source metadata,
the indexed workspace tree, three compact searches, and one bounded indexed
read per search.

The client held returned paths, snippets, and excerpts only in local process
memory. It emitted one sanitized JSON record containing metrics, digests,
labels, and pass/fail outcomes. It failed closed rather than printing a raw MCP
payload on parse, schema, bound, duplication, excluded-path, or protocol error.
No source path, source text, snippet, excerpt, query text, prompt, environment
value, credential, or unrelated absolute path was retained.

## Sanitized observed evidence

- Evidence schema: `engkit.real-context-pilot/v1alpha1`
- Project label: `engineering-toolkit-working-tree-copy`
- Transport: `official-go-sdk-command-stdio`
- Projection fingerprint:
  `sha256:4f0e3df213c677396dd3c5e6cc103e99b8a905cb3f9e6cef6ff7dd52f73a2984`
- Advertised tools: 12
- Indexed sources: 264
- Indexed source bytes: 894,798
- Stored chunks: 474
- Aggregate exclusions: 33
- Source inventory pages: 2
- Workspace-tree entries: 343, not truncated
- Rebuild duration: 7,180 ms
- MCP validation duration: 1,687 ms

| Query label | Results | Snippet bytes | Read bytes | Staged bytes | Full-file baseline | Reduction | Outcome |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `q1` | 5 | 2,560 | 1,242 | 6,688 | 40,748 | 83.58% | pass |
| `q2` | 5 | 2,560 | 2,048 | 7,659 | 83,556 | 90.83% | pass |
| `q3` | 5 | 2,560 | 2,048 | 7,552 | 58,555 | 87.10% | pass |

The full-file baseline for each label is the sum of indexed source sizes for
the unique returned paths. Staged bytes are the complete serialized compact
search result plus the complete serialized bounded read result. These are byte
reductions, not provider-token claims.

## Safety outcome

- The projection was fresh and matched the rebuild fingerprint.
- All expected tools were present and no additional capability was advertised.
- Source and tree pagination remained on the same projection.
- No explicitly excluded prefix appeared in source or tree results.
- Compact summaries contained none of the returned snippet or excerpt bytes.
- Every query returned evidence, remained below its full-file baseline, and
  passed the declared reduction check.
- No network, model, Ollama daemon, provider harness, authentication,
  dependency operation, Git mutation, workflow, installation, or release was
  used.

The temporary workspace, configuration, SQLite index, product binary, pilot
client, and temporary build root were removed after the sanitized result was
emitted. The temporary harness sources in the ignored local scratch directory
were also removed.

## Remaining gate

D2 remains optional and unopened. It requires another approval naming the
project, provider, exact model-backed host version, authentication and network
behavior, disclosure boundary, queries, retained evidence, cleanup, and stop
conditions. D1 establishes local protocol and byte-efficiency behavior only; it
does not authorize sending real project context to any model.

## Rollback

Revert this record and the D1 roadmap status. All executable pilot state was
temporary and has already been removed, so rollback requires no configuration,
database, installation, provider, model, or canonical-project cleanup.
