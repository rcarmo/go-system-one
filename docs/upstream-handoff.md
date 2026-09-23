# Upstream review hand-off

The accepted standalone runtime, API and playground changes are available as three targeted `go-pherence` PRs. They use a fresh clone of base `3a3a629bea7bc4bc2a11ac3bb7f8e6c23251d6f8`; the active neighbouring checkout was not modified. Branches are on the `piclaw-bot/go-pherence` fork, and upstream review/merge remains the receiving agent's task.

| PR | Scope | Head |
|---|---|---|
| [#22](https://github.com/rcarmo/go-pherence/pull/22) | Packed NVIDIA contexts/trees, selected readout, bounded scratch, staged Q5/Q6, CLI defaults and precision tool | `cf218b54b67acd5fdf637dab3753e4db271f9db6` |
| [#23](https://github.com/rcarmo/go-pherence/pull/23) | TypeSafe `noul`, `choice`, `score` API, validation and tests | `60a25e3ce57745798acccd45d90b05ae0ee821af` |
| [#24](https://github.com/rcarmo/go-pherence/pull/24) | Playground, embedded browser tests, screenshots and usage documentation | `e8d051b3d8e95882f9fb3bd1c8bdfd220b651dfb` |

Runtime and API PRs are independently testable. Merge API before playground because its default view calls `/v1/systemone`. All three layout CI checks passed. No PR was merged automatically.

## Port boundaries

The reviewed source is `go-system-one@ee905d0271a13569eda5449cfd03932e955337ca`. Imports were rewritten to the upstream module and CLI files mapped to `cmd/llm/go-system-one`. Hardware tests keep upstream's `GO_PHERENCE_GO_SYSTEM_ONE_GEMMA4_12B` and `GO_PHERENCE_GO_SYSTEM_ONE_GEMMA4_12B_TOKENIZER` names. The API-only PR tests the existing serial scorer, without depending on packed symbols.

The browser port extends the existing `webui/frontend/tests/go` suite and fixture instead of replacing upstream chat/settings routes or adding a separate frontend dependency tree. Standalone screenshots were regenerated there with matching hashes. Root build/release workflows, dependency manifests, repository policy, model files and unrelated runtime work are excluded. Detailed standalone benchmark/precision reports are linked at immutable revisions instead of relabelled as measurements of the port.

The attention softmax barrier fix is already upstream and was not reapplied. The [inconclusive activation-reuse probe](performance/activation-reuse.md) was removed and is not included.

## Verification

The fork's [`handoff/gso-integration-check`](https://github.com/piclaw-bot/go-pherence/tree/cc39fe8e) branch merges all three PRs without conflicts using ordinary merges. It is an integration check, not a fourth bulk-merge request.

* Whole-tree CPU-only race tests passed: `GO_PHERENCE_DISABLE_NVIDIA=1 go test -race ./...`.
* Affected-package vet and upstream `make docs-check` passed; the latter checked 398 Markdown files with no broken links.
* All ten embedded browser tests passed, including existing chat/settings tests. The full frontend install hit workspace ENOSPC; tests reused the already installed Playwright 1.56.1 directory. No dependency symlink or lockfile change enters the PR.
* Targeted GPU Q5/Q6 tail and segmented/branched tests passed three times. Both PTX generators reproduced the embedded source exactly.
* Released-model boolean and multi-field llama.cpp gates passed. Packed/serial logits and probabilities matched on the pinned fixture at 128, 256 and 512 rows. The TypeSafe serial gate passed both independently and on the combined branch.
* AMD64 build and Linux ARM64/RISC-V cross-builds passed for the runtime port.

An initial full GPU runtime test hit the Whisper online-attention tolerance once. Five isolated repeats and subsequent full runtime runs passed; a clean-base run also passed. Its cause is unestablished. The PR records this observation, and no Whisper code or numerical tolerance was changed.

The receiving `@go-pherence` session was sent the PR links, ordering and validation notes for later review. Follow its review there; future accepted imports into this repository still follow the [pinned source-update process](upstream.md).
