# Provider native discovery D2 approval packet

## 1. Status and authority

**Status:** Draft planning packet. No provider command or native discovery smoke
is authorized by this document.

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

## 3. D2-I inventory proposal

D2-I must run from an ordinary shell outside any disposable provider session.
It captures text only and writes no provider configuration.

| Provider | Path-only host probe | Proposed version probe | Proposed help probe | Current state |
| --- | --- | --- | --- | --- |
| Codex | `Get-Command codex -All` | `codex --version` | `codex --help` | Version and exact discovery surface unresolved |
| Claude Code | `Get-Command claude -All` | `claude --version` | `claude --help` | Version and exact discovery surface unresolved |
| OpenCode | `Get-Command opencode -All` | `opencode --version` | `opencode --help` and, only if advertised, `opencode debug --help` | Version and exact discovery surface unresolved |

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

## 4. D2-S provider records

No row is executable until every `TBD` field is replaced with observed D2-I
evidence and that exact row receives separate approval.

| Field | Codex | Claude Code | OpenCode |
| --- | --- | --- | --- |
| Executable path | TBD | TBD | TBD |
| Executable SHA-256 | TBD | TBD | TBD |
| Exact version | TBD | TBD | TBD |
| Supported host | TBD | TBD | TBD |
| Discovery command | TBD | TBD | TBD |
| Authentication required | TBD | TBD | TBD |
| Network required | TBD | TBD | TBD |
| Credential sources read | TBD | TBD | TBD |
| Automatic update disabled | TBD | TBD | TBD |
| Tools/scripts executable by command | TBD | TBD | TBD |
| Timeout | TBD | TBD | TBD |
| Output capture and redaction | TBD | TBD | TBD |
| Cleanup and rollback | TBD | TBD | TBD |
| D2-S approval | Not approved | Not approved | Not approved |

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

