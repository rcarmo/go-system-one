# AGENTS.md — go-system-one

This repository owns the Go System One decision service, its public contract, playground, pinned evaluations and release packaging.

## Source ownership

- `go-system-one` is an independent repository. Commit and push changes here.
- This repository owns its complete compiled source tree. `go-pherence` is an immutable source/update origin, not a Go module dependency.
- Never edit, commit, push, reset or clean a neighbouring `go-pherence` checkout while working here.
- Pull product-owned files through `scripts/sync-upstream.sh`. The sync direction is one way: `go-pherence` to `go-system-one`.
- Pin every upstream source update to an immutable full commit. Do not track an upstream branch or use an uncommitted checkout as release evidence.
- Do not add a `go-pherence` module requirement or local `replace` directive.

## Repository layout

```text
cmd/go-system-one/       HTTP service entry point
model/gosystemone/       decision contract, validation, scorer adapters and tests
half/                    FP16/BF16 conversion kernel and exhaustive tests
internal/httpinput/      bounded JSON request decoding
webui/                   embedded standalone playground and status endpoint
docs/validation/         model pins, oracle results and benchmark evidence
scripts/                 one-way upstream sync and dependency helpers
vendor/                  pinned offline build closure and dependency licences
```

## Required tooling

- Go 1.26.2 or the version declared in `go.mod`.
- `gofmt`, `go test`, `go vet` and `go build` from the same Go toolchain.
- An NVIDIA driver is required only for the opt-in NVIDIA released-model gate. Production PTX loads through the Driver API without CGo or a CUDA toolkit.

## Before editing

1. Read the relevant files and tests.
2. Search all callers before changing a public function, schema field or route.
3. Treat `POST /v1/decision`, `/go-system-one`, artifact hashes, candidate order and validation limits as public contracts.
4. Check `scripts/upstream-files.tsv` when adding, moving or removing a first-party file that still originates in `go-pherence`.
5. Keep package-path mappings in `scripts/local-packages.tsv`; sync uses them to rewrite imported source to this module path.
6. Keep ordinary tests offline and deterministic. Released-model and hardware tests must remain opt-in.

## Correctness rules

- The portable CPU/SIMD scorer is the numerical oracle.
- Preserve deterministic candidate ordering, request validation, cancellation, single admission and returned-output ownership.
- Keep existing pinned-reference tolerances unchanged. Floating-point differences are expected across kernels, reduction orders and precision formats; bitwise equality is a diagnostic, not a general acceptance requirement. Evaluate precision changes under the performance rules below.
- Pin external oracle code by repository and full revision. Record model and tokenizer filenames, byte counts and SHA-256 digests.
- Compare semantic output separately from timings. Never assert wall-clock latency in a correctness test.
- Use synthetic repository-owned fixtures for the default suite. An absent released model may skip only with an exact setup instruction.
- Validate delimiter forgery before tokenisation. User text must not be able to inject model control markers.

## Optimisation priorities

- Maximise batch throughput and general inference performance. Prioritise the requested workloads and measured bottlenecks; do not substitute an easier optimisation merely because it passes existing tests.
- Multi-field classification is required work, including multi-token enum trees, mixed context lengths and multiple independent entries. A faster single-boolean path does not complete the objective.
- Pursue parallelism inside model execution: shared context/prefix work, packed projections, isolated branch attention and batched candidate-logit extraction. More HTTP workers or goroutines alone do not establish GPU parallelism.
- Review relevant techniques in `go-pherence` read-only. Distinguish runtime improvements from prompt changes and approaches requiring trained heads or different weights.
- Investigate projection layouts, activation precision, fusion, scratch reuse, transfers, synchronisation and graph replay according to likely impact and measured cost. Record rejected experiments and why they failed; do not dismiss a class of optimisation without assessing its trade-offs.

## Precision and performance evidence

- Optimise measured whole requests. Microbenchmarks guide investigation but cannot establish service improvement. Report batch latency, entries/s and single-entry latency so throughput gains do not hide latency regressions.
- Compare the same inputs under comparable warm/cold, device-load and thermal conditions. Test batches of 1, 10, 25, 50 and 100 entries, varied positive/negative and ambiguous contexts, multi-field schemas and multi-token enums. Repeated copies of one sentence are insufficient evidence.
- Assess lower-precision paths explicitly. Ordinary rounding differences and activation quantisation are different changes; neither is an automatic reason to accept or reject an optimisation. Weigh speed and memory gains against measured numerical and semantic effects.
- Evaluate error for finite-choice scoring, not unrestricted text generation. Tree mode scores fixed candidate paths without feeding sampled answers into a growing continuation. Floating-point error still propagates through transformer layers and accumulates in multi-token path scores; greedy fallback can also change subsequent nodes within a field.
- Prioritise per-field candidate ranking, conditional probability movement, top-two margins and crossings of declared decision/confidence thresholds. Report decision changes and their margins, including near ties; do not impose long-generation divergence assumptions on independent classifications.
- Candidate rank shifts are expected when scores or input option order change. Record lower-rank swaps separately from winner changes and do not automatically reject either without assessing margins and task impact. Distinguish input/prompt ordering changes from same-input precision comparisons and from independent batch-entry reordering. The latter must preserve isolation; small numerical variation alone is not evidence of sibling leakage.
- Record maximum/RMS raw and centred logit error as diagnostics alongside absolute candidate-probability changes. A common additive shift across the final candidate logits at one node cancels in its softmax; unequal shifts and scale changes do not. For multi-token enums, compare complete path scores and final field distributions as well as node logits.
- Report marginal errors even when decisions agree. Saturated probabilities can hide substantial logit changes, while raw-logit differences alone can overstate decision impact. Distinguish same-input numerical variation from genuine dependence on sibling contexts; cross-context interference is a correctness defect.
- Reference disagreement is not a measured accuracy loss without labelled data. Use separate labelled evaluation for prompt, head or model changes; do not infer task quality from parity fixtures alone.
- Preserve existing acceptance gates. If an experiment exceeds a tolerance, record the failure and keep it opt-in while assessing the trade-off. Define and review any new acceptance contract separately; never widen an existing tolerance just to make a result pass.
- Record hardware, driver, source revision, model hash, precision settings, sample count, min/median/p95/max latency and device memory. Distinguish sampled memory peaks from allocation bounds and report thermal interference. Preserve reproducible requests and results.

## Execution constraints

- Keep weights immutable and scratch/caches owned, bounded and released correctly. Size batches by token rows and measured memory headroom, not entry count alone.
- Preserve context and sibling-branch isolation, candidate order, cancellation and result ownership. Test reordered, unequal-length and contradictory siblings, prefix reuse, rejection and recovery.
- Keep single-request admission until measured memory and concurrency tests justify changing it; aggressively parallelise work within that request.
- Keep accepted fallbacks while qualifying new paths. Promote defaults only after whole-request gains and correctness/quality trade-offs are verified for the intended workloads.
- Do not add unbounded caches, CGo, a llama.cpp runtime wrapper or a production CUDA-toolkit dependency.

## Upstream synchronisation

Run the complete update against an explicit upstream commit:

```sh
./scripts/update-upstream.sh <full-go-pherence-commit>
```

Use `scripts/sync-upstream.sh` only for a copy-only review before accepting a source pin. Set `GO_PHERENCE_SOURCE=/path/to/go-pherence` to use an existing clean checkout with that lower-level script. Both scripts refuse dirty source trees. After syncing:

1. Review every changed file.
2. Update `scripts/upstream.env` to the reviewed full commit and UTC commit time.
3. Confirm `go.mod`, `go.sum` and `vendor/` contain no `go-pherence` module or source.
4. Run `go mod tidy`, `./scripts/vendor.sh` and `make check`.
5. Run the opt-in released-model gate on authorised hardware when scorer, tokenizer, model or backend code changed.
6. Commit the source pin, copied changes, third-party dependency snapshot and validation evidence together.

The script must never write to the upstream checkout or create commits there.

## Validation before commit

Run:

```sh
make check
```

This includes formatting checks, tests, vet and a full build. Also run:

```sh
git diff --check -- . ':(exclude)vendor/**'
go test -race ./model/gosystemone ./internal/httpinput ./webui
```

For NVIDIA or released-model changes, set the artifact paths documented in `docs/validation/go-system-one-v1-20260921.md` and run the named opt-in parity test. Record skipped hardware gates as skipped, not passed.

## Git workflow

- Never use `git rebase`; use merge or `git pull --no-rebase`.
- Commit as `Rui Carmo <rui.carmo@gmail.com>`. Configure local and global Git identity before committing.
- Keep commits focused and use `scope: concise change` subjects.
- Do not commit model weights, tokenizers, GGUF artifacts, profiles, generated benchmark databases, secrets or temporary probe code. `scripts/check-no-gguf-artifacts.sh` rejects GGUF-like names and file magic.
- Do not reformat vendored dependencies. Regenerate them with `scripts/vendor.sh`; apply whitespace checks to first-party paths.
- Push only this repository. Verify the remote branch and CI after pushing.
