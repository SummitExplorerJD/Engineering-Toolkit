# Spec Kit v1.2.0 Codex Inventory

## Scope

This report records the isolated Milestone 4B rehearsal for the official Spec
Kit `v1.2.0` Codex integration on Windows. No Spec Kit file was written to the
real Engineering Toolkit repository during that rehearsal. Milestone 4C later
used this inventory as the approved real-installation contract.

## Provenance and execution

- Requested upstream release: `github/spec-kit@v1.2.0`.
- Resolved upstream commit: `4ca46a0706a662b1edc07086ef16f021820c2db0`.
- Executed package: `specify-cli==1.2.0` from the official PyPI distribution.
- Reported runtime: Specify CLI `1.2.0`, Python `3.11.15`, Windows AMD64.
- Integration: `codex`.
- Script variant: `ps`.
- Symlink policy: denied (`allow_symlinks: false`).

The GitHub-source `uvx` route resolved the correct tag and commit but Git for
Windows could not create its internal MSYS named object inside the application
sandbox. The official version-matched PyPI route was therefore used to obtain
the inventory. The CLI version was checked before initialization.

The successful isolated command was:

```text
uvx --python 3.11 --from specify-cli==1.2.0 specify init --here --force --non-interactive --ignore-agent-tools --integration codex --script ps
```

`--ignore-agent-tools` skipped only the Codex CLI presence check because the
agent runtime is outside the temporary initialization environment.

## Result

- Pre-existing files: 44.
- Files after initialization: 74.
- New files: 30.
- Modified pre-existing files: 0.
- Deleted pre-existing files: 0.
- Generated symbolic links: 0.
- Real repository pre-existing files with hash mismatches: 0.
- `.specify` present in the real repository: no.

Those counts describe the isolated rehearsal before approval of the real
installation.

## Real installation result

Milestone 4C installed the same 30 paths in the real repository with the same
version, integration, script variant, Python requirement, and agent-tool-check
policy used by the rehearsal.

- Pre-existing files before installation: 45, including this inventory report.
- New files: 30.
- Modified pre-existing files: 0.
- Deleted pre-existing files: 0.
- Generated symbolic links: 0.
- Deterministic hashes verified: 27 of 27.
- Timestamp-bearing structures verified: 3 of 3.
- `AGENTS.md` SHA-256 remained
  `e8972678f8c39159639511876c5667c737a25b3140aa4076aff85582e27c363b`.
- `specify check` completed successfully and found Codex CLI available.
- Generated workflows or scripts executed after installation: 0.

The two integration manifests and `workflow-registry.json` embed installation
timestamps, so their complete file hashes differ from the rehearsal. Their
schemas, versions, ownership maps, and referenced content hashes matched. The
other 27 files matched the inventory byte-for-byte.

## Exact generated inventory

| Path | Bytes | SHA-256 |
| --- | ---: | --- |
| `.agents/skills/speckit-analyze/SKILL.md` | 12035 | `fad1c37a451e017995b15fc3eefe243274642fbb3ff7c8a5ab021908df71aa93` |
| `.agents/skills/speckit-checklist/SKILL.md` | 22657 | `5c1dce9333a2b733422e2dc75de7fc4d63ba91a460833d14ef3966a672328b9d` |
| `.agents/skills/speckit-clarify/SKILL.md` | 19707 | `1e9de2b2a673c923f7ca650fc40e4d5f24654163c577e0f1ceb70edfaa08114b` |
| `.agents/skills/speckit-constitution/SKILL.md` | 10520 | `ff6604c8424d4f4edf80f19ccfda1ede1a1dc5091a397cc1f0078a0bc3b0a538` |
| `.agents/skills/speckit-converge/SKILL.md` | 13455 | `369e168cbcc9f903d1b8f3f360a32d24a7ebf3d07066b8676032cffcafbebd9e` |
| `.agents/skills/speckit-implement/SKILL.md` | 13083 | `4d58a80ede8d837798d13a838b83a88fd4335104d0806d27f8018664a1ee0b85` |
| `.agents/skills/speckit-plan/SKILL.md` | 8239 | `0834b2298c078b0a89ee64a3e47e140cc9b31c4049492bd504491199a0fe3e73` |
| `.agents/skills/speckit-specify/SKILL.md` | 18638 | `a505028de7f6489878696d6656c7d057f3a66119ebb8c0cc82d69fabb7b344ff` |
| `.agents/skills/speckit-tasks/SKILL.md` | 11582 | `1e1d18b34e39d62d6388297f035505905741b07973102e0d44aa8647d85ab093` |
| `.agents/skills/speckit-taskstoissues/SKILL.md` | 8637 | `47d5a13f2a39f2e40d8915feef7ff08cd36e08b176c7f28d107758e9e983b4d2` |
| `.specify/.gitignore` | 586 | `0a61c5378b0bb2e0f85babff99c8b166f2170a2c12378cd17d929aa19296ec88` |
| `.specify/init-options.json` | 205 | `95418456a148302d6874325e1e280e0e81eba67db270b367f6811f8922216689` |
| `.specify/integration.json` | 286 | `3a63723d0570f4d29f075a9253ee9270897f6e383c09f9b4bebf9811a255bd40` |
| `.specify/integrations/codex.manifest.json` | 1300 | `49d20f616e2c3b2f2784f382ec11a1e9ec3e9c7ffcf81815fdf234bdbf8c98ef` |
| `.specify/integrations/speckit.manifest.json` | 1535 | `17b0ae669b9c4f6606b1702bdd619be0165810dbd68362aec5d939f6b411cd0a` |
| `.specify/memory/.constitution-template.json` | 103 | `8fc4b39294bae0c10d59547ad7d415b6f0e890d656cbf7171dd567f4e9054711` |
| `.specify/memory/constitution.md` | 2346 | `ce7549540fa45543cca797a150201d868e64495fdff39dc38246fb17bd4024b3` |
| `.specify/scripts/powershell/check-prerequisites.ps1` | 6631 | `ef7fe740091441e5aeadccb8dc36b08d840f524513d79e2fcf258133ee6568b4` |
| `.specify/scripts/powershell/common.ps1` | 35296 | `8b188a9c3a2e979cfcb874acc4452175c00e9095cb0caef18374ea0cfbc384c0` |
| `.specify/scripts/powershell/create-new-feature.ps1` | 14876 | `481c7368541a816b1679e5ed65fa9e5aea84b09f8be3a02ff8a340e3b48dd58a` |
| `.specify/scripts/powershell/resolve-template.ps1` | 889 | `e49c565a09902e4ebd4b5a51c4e014d5067fdee5b1592f15cb44ef31f430d745` |
| `.specify/scripts/powershell/setup-plan.ps1` | 3058 | `3b8173f1553bbf67ae021af949a29e2d3e3bb35805be95e4ba8d0ab83173c66f` |
| `.specify/scripts/powershell/setup-tasks.ps1` | 3892 | `578c19c7fa196327c1c6791e3179accffab049fe62851961139d67bd699f336e` |
| `.specify/templates/checklist-template.md` | 2033 | `70bbc48e884fab372ffb09e83e78ff983e28b69caebf8cd9964a02b6836683ac` |
| `.specify/templates/constitution-template.md` | 2346 | `ce7549540fa45543cca797a150201d868e64495fdff39dc38246fb17bd4024b3` |
| `.specify/templates/plan-template.md` | 3682 | `a828ad6206526574ab3fc99be6d0154588cb1c9774c5919d71b81ab5b2dbffa0` |
| `.specify/templates/spec-template.md` | 4556 | `3945437fc35cd30a5b2bf7beea680337c3516826d3efa5a6b92c4a7eca1ba28e` |
| `.specify/templates/tasks-template.md` | 9171 | `b0f527d41e546af0c31903e04f9b20965b6b497b573792198f663a64ce128863` |
| `.specify/workflows/speckit/workflow.yml` | 2118 | `177761c5e7df28126beea043b5cb45aeaf7958090c022dc11ac71399422fcfbc` |
| `.specify/workflows/workflow-registry.json` | 381 | `92b4e34089e2196c1253b60703cbfcb0fcc1bc34f3c6d506f318dc4b94f384ba` |

## Review findings

All six generated PowerShell scripts parse without syntax errors. Static command
inspection found no network download command, `Invoke-Expression`, child-process
launcher, Git mutation, or broad recursive deletion. The scripts create feature
directories and files under `.specify` or `specs`; `common.ps1` removes only its
own temporary Python stderr file.

The generated skills carry `github-spec-kit` provenance metadata and point to
the generated PowerShell helpers where applicable. They are instructions, so
invocation remains subject to repository approval policy.

`speckit-taskstoissues` is deprecated and can create GitHub issues through an
external connector. Installation does not execute it. It must not be invoked
without separate explicit approval for the external mutation. Any future
removal must use an official, separately approved integration operation so the
upstream manifest remains coherent.

`speckit-implement` performs code changes by design. Its presence does not grant
permission to execute tasks or bypass Engineering Toolkit plan approvals.

## Ownership and removal

- Spec Kit owns the 20 files under `.specify` listed above.
- Spec Kit owns only the ten `speckit-*` skill files listed above, not the shared
  `.agents/skills` directory or its existing `README.md`.
- Engineering Toolkit continues to own `AGENTS.md`, `.agents/conventions`,
  `.agents/context`, `.agents/workflows`, source code, and canonical documents.
- Removal must use the two generated manifests and verified hashes, deleting
  only unchanged files recorded for this installation.
- Shared directories must remain when they contain non-Spec-Kit content.

## Decision gate

The exact 30-file set was approved and installed in Milestone 4C. This approval
does not extend to executing generated workflows, changing the constitution,
creating feature artifacts, invoking `speckit-taskstoissues`, upgrading Spec
Kit, removing files, committing, or pushing.
