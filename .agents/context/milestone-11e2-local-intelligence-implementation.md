# Milestone 11E2: local-intelligence implementation

## Implemented boundary

The approved E2 slice was implemented locally on 2026-10-27. It adds the
explicit `engkit advise profiles --root PATH --goal TEXT [--model MODEL_TAG]`
command, optional backward-compatible `intelligence` configuration, bounded
application policy, fresh SQLite context retrieval, and a fixed-loopback Ollama
adapter. Local intelligence remains disabled by default.

The deterministic profile recommender owns eligibility. The adviser receives
only eligible candidate identifiers and bounded evidence, cannot introduce a
candidate, and cannot select, approve, plan, or execute anything. Missing or
stale context, an absent tag, timeout, cancellation, transport failure, or an
invalid structured response returns the deterministic order with a bounded
warning.

Configuration parsing is centralized so doctor, unified-init classification,
and advice share one strict `engkit.config/v1alpha1` reader. Existing
configuration remains valid when `intelligence` is absent. No SQLite schema or
dependency changed.

## Privacy and transport

The context adapter requires a fresh projection and reads only `README.md`,
`AGENTS.md`, Markdown below `docs/`, and Markdown below
`.agents/conventions/`. Application policy sorts excerpts, limits count and
bytes, rejects unapproved paths, and excludes private-key markers and the
accepted secret-key patterns before prompt construction.

The production client fixes the endpoint at `http://127.0.0.1:11434`, disables
proxy use and compression, rejects redirects, uses a three-second connection
limit and a 60-second complete-operation deadline, performs no retry, and
sends no credentials, tools, or images. Requests use `stream: false`,
`think: false`, `keep_alive: 0`, temperature zero, and the complete accepted
response schema.

## Synthetic model evidence

All model prompts were synthetic and contained no repository content.

- `phi4-mini:latest`, digest
  `78fad5d182a7c33065e153a5f8ba210754207ba9d91973f57dffa7f487363753`:
  passed; retained private evidence SHA-256
  `0e8e2787d7fc3693adbb418672c252af51f3158fa647a66d36eedc7508d8145f`.
- `granite4:3b`, digest
  `89962fcc75239ac434cdebceb6b7e0669397f92eaef9c487774b718bc36a3e5f`:
  passed; retained private evidence SHA-256
  `351bdf93a638393d38247a04ebabc7bc89008221a83f11539c4ac47f5aaaca04`.
- `qwen3.5:0.8b`, digest
  `de63045f29755d4721c09d54c9b45c2dadb644f4fd7d604d4cac90d96049971c`:
  failed the exact-candidate cardinality check twice with thinking disabled.
  The complete second-attempt evidence SHA-256 is
  `497f05d80f4f311386c9a602a5ead7f51a93e38765bd3a4b0840b870356005e8`.
  A thinking-enabled diagnostic produced reasoning but no final structured
  response; evidence SHA-256
  `b7975a499666b2a798c798c3bb0b43fe351e6a75f099e44cdde1a2f2d054c6cb`.
- Exploratory `qwen3.5:4b`, digest
  `d8b0f5e9760cd1682034f292d7ef72ec46f432149be0df7574bf2d6e92e38c04`:
  exceeded the 60-second deadline and remains outside the allowlist; evidence
  SHA-256
  `69e4366e326ab23a68204db2042dabdde9a72318dfb81e8a2b764de23213b020`.

Passing evidence applies only to the exact Ollama version, tag, digest, host,
request, and limits observed. The two passing results do not make another tag
compatible. The failed Qwen result does not block deterministic behavior.

## Validation

- Focused configuration, policy, privacy, SQLite, HTTP, and CLI tests passed.
- `go test ./...` passed.
- `go vet ./...` passed.
- CGO-disabled `go build -trimpath ./cmd/engkit` passed for Windows, Linux, and
  macOS on AMD64 and ARM64.
- `govulncheck` was not installed and was not downloaded.

## Preserved exclusions

E2 adds no dependency, SQLite migration, MCP model access, canonical advice
state, approval or execution authority, model installation or pull, endpoint
override, remote network access, retry, Git operation, CI change, release, or
publication. The next catalog and release gates remain separately approved.
