# Provider native discovery D2 approval packet

## 1. Status and authority

**Status:** D2-I inventory completed on one Windows x64 host. A separately
approved OpenCode `1.17.13` D2-S run passed on 2026-10-23. Codex remains
blocked and Claude Code remains uninstalled and untested.

This packet defines the evidence required to prove that an exact installed
version of Codex, Claude Code, or OpenCode discovers the synchronized
`project-guide@0.1.0` skill. It does not authorize provider installation,
upgrade, authentication, network access, credential access, configuration
changes, skill execution, or repository mutation.

D2 is split into two separately approved stages:

1. **D2-I — local inventory:** resolve the installed executable, capture its
   exact version and bounded help output, and determine whether a local
   diagnostic can list skills without authentication or network access.
2. **D2-S — native discovery smoke:** create an isolated disposable workspace,
   synchronize the pilot through Engineering Toolkit, and run one exact
   reviewed discovery command for one provider version.

Approval of D2-I does not authorize D2-S. Approval for one provider does not
authorize either other provider.

## 2. Reviewed discovery baseline

The provider contracts remain those recorded in
[`provider-conformance-v1alpha1.md`](provider-conformance-v1alpha1.md). The
following upstream references are planning evidence, not permission to access a
provider:

- [OpenAI Skills](https://developers.openai.com/api/docs/guides/tools-skills)
  documents metadata-first discovery from a `SKILL.md` package.
- [Claude Code Skills](https://code.claude.com/docs/en/skills) documents
  project skills beneath `.claude/skills/` and on-demand loading.
- [OpenCode Agent Skills](https://docs.opencode.ai/docs/skills/) documents
  project discovery and the model-visible skill list.
- [OpenCode CLI](https://docs.opencode.ai/docs/cli/) documents its version,
  diagnostic, and non-interactive command surfaces.

Upstream documentation is mutable. Before each D2-S approval, record the exact
review date and recheck the documentation relevant to the installed version.
Documentation changes do not silently change Engineering Toolkit output.

## 3. D2-I inventory evidence

D2-I must run from an ordinary shell outside any disposable provider session.
It captures text only and writes no provider configuration.

| Provider | Path-only host probe | Version probe | Help probe | D2-I result |
| --- | --- | --- | --- | --- |
| Codex | `Get-Command codex -All` | `codex --version` | `codex --help`; `codex debug --help` was rejected before TUI startup | `blocked` for D2-S pending a non-mutating discovery surface |
| Claude Code | `Get-Command claude -All` | Not run | Not run | `not installed`; installation is not authorized |
| OpenCode | `Get-Command opencode -All` | `opencode --version` | `opencode --help`; `opencode debug --help`; `opencode debug skill --help` | `eligible` for a separately approved D2-S design |

Before running a version or help probe, review whether that exact executable
can auto-update, contact the network, initialize state, or read credentials.
If this cannot be established, do not run it. Record `not installed` rather
than installing a missing provider.

Required D2-I evidence for each provider:

- resolved executable path and file SHA-256 when it is a regular file;
- exact version output and host operating system/architecture;
- exact help text for the proposed discovery surface;
- observed exit code, duration, stdout, and stderr;
- before/after inventory of any provider state paths touched by the probe;
- authentication, subscription, credential, and network observations; and
- a conclusion of `eligible`, `blocked`, or `not installed` for D2-S.

Secrets, tokens, account identifiers, prompts from unrelated sessions, and
provider configuration contents must not be captured in evidence.

The inventory ran on Microsoft Windows `10.0.26220`, x64, on 2026-10-23. Every
probe had a ten-second timeout and ran in an environment where external network
access was restricted. No model session, authentication flow, provider
installation, skill, or workspace mutation was requested.

Observed results:

- Codex resolved to
  `C:\Users\julit\AppData\Local\OpenAI\Codex\bin\896a0166b774037a\codex.exe`
  with SHA-256
  `f6dbf48cb70f62649c645f26c87dc9bf84cc36986202dfad8e12fb5b8ca70312`
  and reported `codex-cli 0.163.0-alpha.10.2`. Version and top-level help exited
  successfully, but both attempted stale temporary-alias cleanup under
  `.codex/tmp` and were denied by the sandbox. `codex debug --help` then
  attempted interactive TUI startup and exited with code 1. The active Codex
  session also made two concurrent metadata changes under the observed state
  roots, so the run cannot prove a clean Codex before/after state boundary.
- Claude Code did not resolve through `Get-Command claude -All`. No Claude
  command ran and no installation was attempted.
- OpenCode resolved through Node launchers to
  `C:\Program Files\nodejs\node_modules\opencode-ai\bin\opencode.exe`, whose
  SHA-256 is
  `1d312c76504369c14deadb2676746bdb55779c6e396b0e8d43e2858ad4186d03`,
  and reported version `1.17.13`. Its update, pruning, model-fetch, default
  plugin, and LSP-download behaviors were disabled for the probes. Top-level
  help advertised `opencode debug skill` as the local skill-listing surface,
  and the bounded `debug --help` probe produced no OpenCode state-file metadata
  changes. The diagnostic itself was not run.

## 4. D2-S provider records

No row is executable until every `TBD` field is replaced with observed D2-I
evidence and that exact row receives separate approval.

| Field | Codex | Claude Code | OpenCode |
| --- | --- | --- | --- |
| Executable path | `C:\Users\julit\AppData\Local\OpenAI\Codex\bin\896a0166b774037a\codex.exe` | Not installed | `C:\Program Files\nodejs\node_modules\opencode-ai\bin\opencode.exe` |
| Executable SHA-256 | `f6dbf48cb70f62649c645f26c87dc9bf84cc36986202dfad8e12fb5b8ca70312` | Not available | `1d312c76504369c14deadb2676746bdb55779c6e396b0e8d43e2858ad4186d03` |
| Exact version | `0.163.0-alpha.10.2` | Not installed | `1.17.13` |
| Supported host | Windows `10.0.26220`, x64 | Not evaluated | Windows `10.0.26220`, x64 |
| Discovery command | Blocked: no bounded local diagnostic identified | Not available | `opencode --pure debug skill`; executed successfully |
| Authentication required | TBD | TBD | No authentication requested or observed |
| Network required | TBD | TBD | No provider network access was required to complete the observed run; no OS firewall rule was installed |
| Credential sources read | TBD | TBD | No global provider-state metadata changed; credential contents were not inspected |
| Automatic update disabled | Not established | Not applicable | Yes; update, pruning, model fetch, default plugins, Claude compatibility, external skills, and LSP download were disabled |
| Tools/scripts executable by command | TBD | TBD | The local diagnostic listed skill metadata and content; it did not start a model session or execute skill instructions |
| Timeout | TBD | Not applicable | 10 seconds |
| Output capture and redaction | TBD | Not applicable | Bounded stdout/stderr; raw stdout retained only in memory and recorded by length and SHA-256 |
| Cleanup and rollback | TBD | TBD | Transactional unsync completed; target, ownership record, and permitted generated `.gitignore` are absent |
| D2-S approval | Blocked | Not available | Passed on Windows x64 with OpenCode `1.17.13` on 2026-10-23 |

### 4.1 First OpenCode D2-S attempt

The approved OpenCode-only attempt ran on 2026-10-23 with released
`engkit v0.2.2` and OpenCode `1.17.13`. It used the exact executable and digest
recorded above, a ten-second provider timeout, restricted network access, and
the reviewed environment switches that disable updates, pruning, model fetch,
default plugins, Claude compatibility, and LSP downloads.

The disposable workspace was
`.engkit-test-temp/d2-opencode-634e0559f8df433c9625330da1a92121/workspace`.
Project creation, canonical `project-guide@0.1.0` validation, and deterministic
OpenCode synchronization planning succeeded. Synchronization then failed while
testing the required atomic no-replace hard-link primitive: the host sandbox
returned `Access is denied`. No provider output or ownership record was
published, and `opencode debug skill` was never invoked.

The exact synchronization plan
`sha256:55a9e8d1646437149c51d5f48f9d635e601a897d887be67ee338e8acbdc984f1`
was passed to `engkit skills recover-sync`. Recovery completed with status
`rolled_back`; the target, ownership record, and locks are absent. Transaction
recovery evidence remains in the disposable workspace. The private D2 evidence
file has SHA-256
`768cdeccf49af9af3658456c12465270be53a0bc18505cf8b7bb5841a81da545`.

This is an environment-blocked attempt, not an OpenCode conformance result. A
later attempt must use a reviewed host where Engineering Toolkit's atomic
no-replace capability succeeds, while preserving the same OpenCode executable,
environment, diagnostic, timeout, evidence, and rollback boundaries unless a
new approval explicitly changes them.

### 4.2 Successful OpenCode D2-S result

A later, separately approved OpenCode-only run completed on 2026-10-23 with
released `engkit v0.2.2`, OpenCode `1.17.13`, and the executable SHA-256
recorded in the provider table. The host was Microsoft Windows
`10.0.26220`, AMD64. The run used `opencode --pure debug skill`, a ten-second
timeout, an in-memory OpenCode database, disposable XDG roots, and the reviewed
switches that disable background and external-skill behavior. It did not start
a model session or request authentication.

The synchronization plan
`sha256:7245155c87c151c767c742489228dfe077d632fc47da77e8f65deb4d14bd4953`
completed and published one derived `project-guide` target with SHA-256
`12d9b2f7107108b8323ea648cf049a5377d6270a6742f95f157b8d691eef3bd1`.
The bounded provider diagnostic exited with code 0 after 2,903 milliseconds,
produced no stderr, and reported exactly one `project-guide`. Its name,
description, resolved location, instruction marker, content length, and content
SHA-256 matched the synchronized package. Captured stdout was not persisted;
its 18,794-character projection has SHA-256
`88f7dd99a550a41779dce34a3d1c1d6ff98480cbddf899528a6f40adba6b60eb`.

OpenCode deterministically created `.opencode/.gitignore` while initializing
the project root. The harness accepted only that exact 63-byte file with
SHA-256
`663a068e76d264d0bc6740f5450b6c4193c7b41ecf5e0dc222485b8a17404d95`;
any other workspace change remained a failure. Global OpenCode state metadata
was unchanged, while isolated runtime state was recorded beneath the disposable
run root.

Transactional unsync used plan
`sha256:fb9e2cfe822174096bf9fc7001e99c15324065f3cdc80ce60576ba8e8085671f`
and completed successfully. The provider target, ownership record, and
permitted generated `.gitignore` were absent after cleanup. The ignored private
evidence file has SHA-256
`89ea90482d4527dd2855fd9137c7c95c56e8d534a4ab344cc8af4173d0983e1a`.
It is not a repository artifact because it contains host-specific absolute
paths and state metadata.

This result proves native discovery only for the exact OpenCode version,
executable, host, command, and boundaries recorded above. It grants no standing
authority for another OpenCode run and no authority for Codex, Claude Code,
provider installation, authentication, or network access.

Prefer a documented local diagnostic that lists discovered skill metadata
without starting a model session. A prompt-based fallback is admissible only
when no diagnostic exists and its exact model, authentication, network,
permission, persistence, and cost behavior are explicitly approved.

## 5. Disposable workspace contract

Each approved D2-S run uses a new provider-specific directory beneath the
repository-owned ignored `.engkit-test-temp` root. The run must:

1. create a unique empty workspace and record its physical resolved path;
2. initialize only the minimum local structure required by the reviewed
   provider version;
3. use the released `engkit v0.2.2` binary to compose the embedded
   `go-service@0.1.0` profile and synchronize `project-guide@0.1.0` to exactly
   one provider;
4. record canonical and provider output digests before provider invocation;
5. capture a recursive inventory and digests of the disposable workspace;
6. invoke only the exact approved discovery command with a fixed timeout;
7. capture bounded stdout and stderr after secret redaction;
8. compare the complete workspace inventory and file digests after invocation;
9. unsynchronize through the exact approved Engineering Toolkit plan; and
10. remove only the unique disposable directory after evidence review.

Provider home, cache, state, and configuration paths must either be redirected
to provider-supported disposable locations or inventoried before and after.
Undocumented redirection through `HOME`, `USERPROFILE`, or equivalent process
state is prohibited because it can change provider behavior unpredictably.

## 6. Fixed discovery request

When a provider lacks a local metadata diagnostic and a model-backed command is
separately approved, its request must be semantically equivalent to:

> Report whether a skill named `project-guide` is available in this workspace
> and whether its instructions can be loaded. Do not execute the skill, call
> tools, modify files, change configuration, access external systems, or infer
> availability from paths supplied in this request. Return only the discovered
> skill name, description, instruction-load status, and any refusal.

The provider must discover the skill from its own reviewed mechanism. Passing
the generated file path or contents in the request invalidates the result.

## 7. Pass, block, and failure rules

A provider/version pair passes only when all of the following are observed:

- the executable and version match the independently approved D2-S row;
- the provider reports `project-guide` from the expected project-local root;
- the reported name and description equal the reviewed synchronized metadata;
- the provider can load the instructions without executing their workflow;
- no canonical, synchronized, unrelated, or provider-native workspace byte
  changes during discovery;
- no unapproved tool, script, network destination, credential source, config,
  or persistent session is used; and
- cleanup preserves the captured evidence while removing only disposable data.

The result is blocked rather than skipped when the provider is absent, its
version or command differs, authentication or network behavior is unknown,
permissions cannot be bounded, evidence could expose secrets, or cleanup cannot
be proven. A timeout, unexpected prompt, file mutation, auto-update, tool call,
or discovery mismatch is a failure and stops that provider's run.

Passing one version is evidence only for that exact provider/version/host tuple.
It does not establish model quality, automatic selection accuracy, future
compatibility, or permission to repeat the command.

## 8. Rollback and incident response

Before execution, preserve the disposable workspace manifest and any approved
provider-state baseline. On failure:

- terminate only the process started by the smoke test;
- retain redacted logs and before/after manifests;
- do not delete changed or ambiguous provider state;
- do not run provider uninstall, logout, cache cleanup, or repair commands;
- unsynchronize only when Engineering Toolkit ownership evidence still matches;
  otherwise preserve the workspace for inspection; and
- require a new review before retrying with any changed command or version.

Documentation rollback removes this packet and restores its links from the
roadmap, conformance contract, and milestone context. It does not change
provider installations, credentials, workspaces, Git state, or release assets.

## 9. Approval checklist

Before D2-I:

- [ ] Approve the exact providers and host on which inventory may run.
- [ ] Approve each exact version/help command after auto-update and network
      behavior are reviewed.
- [ ] Approve evidence locations and redaction rules.

Before each D2-S row:

- [ ] Replace every `TBD` field with observed D2-I evidence.
- [ ] Review current authoritative provider documentation.
- [ ] Approve the exact executable digest, version, command, environment,
      timeout, authentication, network, credential, and cost boundary.
- [ ] Approve disposable workspace creation, Engineering Toolkit sync/unsync,
      evidence capture, cleanup, and rollback.
- [ ] Confirm that the run grants no Git, release, Spec Kit, skill-execution,
      provider-installation, or configuration authority.

