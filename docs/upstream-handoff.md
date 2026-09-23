# Upstream review hand-off

The standalone runtime, API and playground hand-offs (#25–27) are merged in `rcarmo/go-pherence`. The subsequent long-context fix is submitted as [#29](https://github.com/rcarmo/go-pherence/pull/29), on an owner-repository branch. Review and merging of #29 are the receiving maintainer's task; the active neighbouring checkout was not modified.

The original three PRs originated from base `3a3a629bea7bc4bc2a11ac3bb7f8e6c23251d6f8`; reconciliation also verified integration with `6f74c75ea7874c1e7c740c47d9532d2aa442b1ea`.

| PR | Scope | Head |
|---|---|---|
| [#25](https://github.com/rcarmo/go-pherence/pull/25) | Packed NVIDIA contexts/trees, selected readout, bounded scratch, staged Q5/Q6, CLI defaults and precision tool | `cf218b54b67acd5fdf637dab3753e4db271f9db6` |
| [#26](https://github.com/rcarmo/go-pherence/pull/26) | TypeSafe `noul`, `choice`, `score` API, validation and tests | `d22c206b0c99044331bef42435c3dde45d0f65c6` |
| [#27](https://github.com/rcarmo/go-pherence/pull/27) | Playground, embedded browser tests, screenshots and usage documentation | `4d5c5819e04df57e6dacd9445312adf07b01a74e` |

The runtime and API PRs were independently testable; the playground depended on the API because its default view calls `/v1/systemone`. All three layout CI checks passed. They were merged by the receiving side.

## Long-context follow-up

[#29](https://github.com/rcarmo/go-pherence/pull/29) ports standalone `b18ee0d4748bac436999aa72c000e06406c3cce6` onto upstream `08ac3fbae8b6afe35c64416cb5e347c4813937db`. Its head is `b703e47a15c49241e11aa03fc2fadbc9a094ea7b`, on `handoff/gso-long-context-20260923`. The pre-fix runtime files matched exactly, so the patch applied without logic changes. Imports and hardware-test environment names follow the upstream repository. The missing selected-batch Q5 regression test is included.

The port removes the fixed 2,048-token attention limit, bounds scratch to 8 MiB, compacts sliding-window KV and returns explicit capacity refusals as HTTP 422. The logical ceiling is min(model context, 32,768); full-model validation reaches 3,937 tokens on an RTX 3060 12 GB. [Implementation and standalone measurements](performance/long-context.md) and [upstream validation](https://github.com/rcarmo/go-pherence/blob/b703e47a15c49241e11aa03fc2fadbc9a094ea7b/docs/go-system-one-long-context.md) retain the limits and exact gates.

Whole-tree CPU-only race tests, vet, build and documentation checks passed in the temporary clone. Targeted attention and selected-batch tests passed three times, along with synthetic sliding-KV parity, released-model recovery and unchanged llama.cpp boolean/multi-field gates. Packed/serial probabilities and logits matched at 128/256/512 rows. PTX regenerated exactly; ARM64/RISC-V compilation passed. No new browser run, native foreign-architecture execution or universal GPU-suite pass is claimed for this port. A read-only agent review found no blocking code issue and prompted a correction to the validation wording.

Remaining mapped differences are repository-specific test configuration/documentation, formatting and unrelated upstream MiniCPM work. They were not copied over. Dependency files, source pins and release workflows remain unchanged.

The [complete JevBench v1.4.0 public report](benchmarks/jevbench-v140-public.md) is published at `a73f987be3abf0d84fb51b489ccb007f2d84fa07`: 196/231 correct (84.85%), all 231 responses strict-valid, zero failures. The [issue #57 update](https://github.com/fstandhartinger/jevbench/issues/57#issuecomment-5802130498) links immutable evidence and asks about sealed evaluation and Speed/Cost treatment. No benchmark request was repeated for the upstream hand-off.

## Original port boundaries

The reviewed source is `go-system-one@ee905d0271a13569eda5449cfd03932e955337ca`. Imports were rewritten to the upstream module and CLI files mapped to `cmd/llm/go-system-one`. Hardware tests keep upstream's `GO_PHERENCE_GO_SYSTEM_ONE_GEMMA4_12B` and `GO_PHERENCE_GO_SYSTEM_ONE_GEMMA4_12B_TOKENIZER` names. The API-only PR tests the existing serial scorer, without depending on packed symbols.

The browser port extends the existing `webui/frontend/tests/go` suite and fixture instead of replacing upstream chat/settings routes or adding a separate frontend dependency tree. Standalone screenshots were regenerated there with matching hashes. Root build/release workflows, dependency manifests, repository policy, model files and unrelated runtime work are excluded. Detailed standalone benchmark/precision reports are linked at immutable revisions instead of relabelled as measurements of the port.

The attention softmax barrier fix is already upstream and was not reapplied. The [inconclusive activation-reuse probe](performance/activation-reuse.md) was removed and is not included.

## Verification

The owner-repository [`handoff/gso-integration-check`](https://github.com/rcarmo/go-pherence/tree/670fb2dbde10b6a82ff4603a2d36289ec379f65c) branch merges the checked current main and all three PRs without conflicts using ordinary merges. The original integration commit `cc39fe8ef25de690778a39530c179955a1c5a7a7` remains in its history. It is an integration check, not a fourth bulk-merge request. The CPU-only race suite, docs checks and all ten browser tests passed again after reconciliation; the original hardware results below apply to unchanged runtime code.

* Whole-tree CPU-only race tests passed: `GO_PHERENCE_DISABLE_NVIDIA=1 go test -race ./...`.
* Affected-package vet and upstream `make docs-check` passed; the latter checked 398 Markdown files with no broken links.
* All ten embedded browser tests passed, including existing chat/settings tests. The full frontend install hit workspace ENOSPC; tests reused the already installed Playwright 1.56.1 directory. No dependency symlink or lockfile change enters the PR.
* Targeted GPU Q5/Q6 tail and segmented/branched tests passed three times. Both PTX generators reproduced the embedded source exactly.
* Released-model boolean and multi-field llama.cpp gates passed. Packed/serial logits and probabilities matched on the pinned fixture at 128, 256 and 512 rows. The TypeSafe serial gate passed both independently and on the combined branch.
* AMD64 build and Linux ARM64/RISC-V cross-builds passed for the runtime port.

An initial full GPU runtime test hit the Whisper online-attention tolerance once. Five isolated repeats and subsequent full runtime runs passed; a clean-base run also passed. Its cause is unestablished. The PR records this observation, and no Whisper code or numerical tolerance was changed.

## Fork reconciliation

The unnecessary `piclaw-bot/go-pherence` fork was used for the initial hand-off without a technical requirement. At the user's request, all four hand-off branches and their original commits were preserved directly in `rcarmo/go-pherence`. Old PRs #22–24 were closed with links to replacements #25–27. The replacements preserve the source changes and validation notes; the API/UI branches only add documentation corrections to the PR numbers. All replacement checks passed. Reconciliation itself made no merge into `main`; the receiving side subsequently merged #25–27.

Deleting the fork was attempted after reconciliation. GitHub returned HTTP 403 because the available bot token lacks the `delete_repo` scope, despite having repository admin rights. The fork still exists until deletion is performed with an authorised credential; no active hand-off depends on it.

The receiving `@go-pherence` session was sent the replacement PR links, ordering and validation notes for later review. Follow its review there; future accepted imports into this repository still follow the [pinned source-update process](upstream.md).
