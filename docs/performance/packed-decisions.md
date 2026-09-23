# Packed Gemma decision scoring

Process several independent contexts in the same transformer matrix operations, then extract only the required candidate logits. Keep the pinned Gemma weights, prompt, token paths and probability calculation unchanged.

The NVIDIA service uses packed execution by default with a 512-token-row budget. It supports multi-field tree decisions and multi-token enums, sharing each context among its independent branches. Greedy/mixed-mode requests and groups exceeding the branch budget retain the serial path. Oversized contexts fall back individually without disabling packing for their peers. Use `-packed-token-rows=0` for the serial diagnostic reference; `-1` selects the automatic backend default. CPU/SIMD behaviour is unchanged. The source review used `go-system-one@63e49ff0a49de4eca17e5e88aea9154d8be41754` and `go-pherence@32bfb937ca1c8608c66411c63f9b1eb39d152cb3`.

## First measurements

Both modes used the same binary with the attention barrier fix from `e53e9770a756f981a3ac3bacc414e9181fa834f5`. One warm-up and three measured requests per case used varied positive and negative contexts with unique ticket numbers. Before each request, the GPU cooled to at most 60°C. Packed mode ran first.

| Entries | Serial median | Packed median | Speedup |
|---:|---:|---:|---:|
| 10 | 1,133.16 ms | 827.51 ms | 1.37× |
| 100 | 11,587.71 ms | 8,517.50 ms | 1.36× |

All candidate probabilities were identical between modes in these HTTP samples. The released-model raw-logit gate also produced identical logits at 128, 256 and 512 token-row budgets. These contexts are longer than the earlier repeated 20-token context; their timings are not directly comparable to its 8.08-second 100-entry result.

The GPU reached at most 75°C. One-second sampling observed up to 9,658 MiB used in packed mode and 9,424 MiB serially; sampling can miss brief peaks. [Raw requests, responses, timings and device readings](../benchmarks/data/packed-contexts-initial.json) are committed. Three samples do not establish tail latency. An earlier uninterrupted sweep used the faulty attention kernel and reached 87°C; its numbers are not acceptance evidence.

Packing exposed an existing shared-memory softmax race: a warp could overwrite the maximum before another warp read it. The new short-softmax regression failed ten consecutive runs with the old kernel and passes with the barrier. Saturated probabilities had hidden raw-logit changes. The original llama.cpp parity fixtures still pass.

The first executor shared projections but launched attention separately per context and copied KV into a reusable suffix region. The next version uses one segmented attention launch per layer, reads prefix and suffix KV directly, and batches terminal normalisation and selected-logit projection into one download. It preserves the same raw logits in the released-model comparison.

Using the same requests and cooling protocol, three warm samples gave 809.33 ms for 10 entries and 8,340.94 ms for 100 entries. This is about 2% below the first packed executor and 1.39–1.40× faster than serial. Sampled memory use peaked at 9,562 MiB; temperature reached 74°C. [Segmented executor samples](../benchmarks/data/packed-contexts-segmented.json) contain identical complete HTTP results to the serial comparison.

Development source for the segmented attention, rotary-position and batched Q5 selected-head kernels is in `scripts/kernels/packed_decision.cu`; `scripts/generate-packed-ptx.sh` embeds CUDA 12.8 PTX. The runtime still requires only the NVIDIA driver. The next dispatch/scratch revision selects existing Q5/Q6 tiles for packed workloads and retains at most one 512-row prefill workspace under the model mutex. Using the same cooled protocol, medians were 794.03 ms for 10 entries and 8,054.34 ms for 100. Complete HTTP results remained identical; sampled device use peaked at 9,562 MiB and temperature at 73°C. [Dispatch/scratch samples](../benchmarks/data/packed-contexts-dispatch.json) retain the measurements. Scratch reuse, growth and freeing have explicit tests.

Two INT8 tensor-core prototypes were rejected: they were slower than the existing dp4a kernels, and Q5 accumulation also differed slightly. No prototype kernel enters production. Later Q6 PTX work stages quantised activations and scales in shared memory across a 24-row, 64-output tile. It uses 69 registers, 7,680 bytes of shared memory and no spills. Shape-tail differentials, original pinned parity and the stricter batch-logit comparison passed. The first Q5 staged tiles were held back because their larger released-model logit movement needed separate analysis. The [accepted Q5 revision](q5-staged.md) restores the existing arithmetic contraction, passes the original gates and improves the measured batch sweep by another 8–12%.

The [current benchmark tables and all five charts](../benchmarks/README.md) use `774c5da` measurements, including staged Q5/Q6 and the [512-column Q5 chunks](q5-chunk512.md), the full batch sweep and paired comparison. Earlier numbers in this note are development history. A subsequent [Q/K activation-reuse evaluation](activation-reuse.md) passed parity but produced inconclusive whole-request timing; the prototype was removed. Accepted runtime, API and UI work has been [submitted upstream as targeted PRs](../upstream-handoff.md). Nsight attempts did not yield a usable kernel trace, so no new kernel timing breakdown is claimed.

## Multi-field experiment

The tree layout shares each context once, then packs field/node suffixes with explicit parent references. A branch sees the schema prefix, its own context and its own causal suffix. Selected-logit extraction replaces full-vocabulary projection and hidden-state CPU transfers.

For one boolean and a three-choice multi-token enum, the same binary produced these medians on the RTX 3060:

| Entries | Serial | Packed tree | Speedup |
|---:|---:|---:|---:|
| 1 | 1,017.42 ms | 141.57 ms | 7.19× |
| 10 | 10,427.21 ms | 967.59 ms | 10.78× |

One warm-up preceded three measured requests per case. Each request started at no more than 60°C; GPU temperature reached 80°C in serial mode. Sampled device memory peaked at 9,619 MiB packed and 9,472 MiB serially. [Requests, complete responses and samples](../benchmarks/data/packed-multifield.json) retain the comparison. Three samples do not establish tail latency.

All 22 compared field decisions agreed. Maximum candidate-probability movement was **0.0000251846**, or **0.00252 percentage points**. This exceeds the `1e-6` probability tolerance used by the pinned multi-field fixture, although that fixture itself passes unchanged at all tested budgets. This broader comparison is experimental evidence, not a new accuracy or numerical acceptance claim.

The old scorer uses F32 activation kernels for fewer than four active branch rows. Packed execution uses Q8 activations. Thus the speed comparison includes both parallel execution and a precision change. Diagnostic independent causal-prefill scoring agreed exactly with packed logits on the sampled inputs. The [broader precision evaluation](multifield-precision.md) completed 80 fields: no winner changes, four losing-rank shifts and one diagnostic 95% crossing at each tested packed budget. Maximum probability movement was 16.54 percentage points. The throughput gain and winner agreement support promoting this execution path, with the serial override retained. This does not establish calibrated confidence, labelled accuracy or near-tie invariance; consumers using confidence thresholds must account for the documented precision change.

## Serial-path cost

Without packing, [`Engine.Decide`](../../model/gosystemone/engine.go) scores contexts serially. The NVIDIA single-branch path combines the context and forced suffix into one causal prefill. It already projects only candidate vocabulary rows through `finishSelectedDevice` in [`gemma4_nvidia.go`](../../model/gemma4_nvidia.go).

The short boolean fixture processes 24 token rows per context. Ten entries therefore cause ten separate transformer passes over 24 rows. Packing processes up to 240 rows through each layer's projections, with attention isolated by context. This can improve matrix utilisation and amortise launches and weight reads. It does not remove the transformer arithmetic for each token.

The multi-branch path has another cost: `runDepth` downloads hidden rows, and `finishRow` uploads each row, computes the full vocabulary vector and downloads it. The scorer then selects the candidate entries. That path needs a device-resident, selected-row readout too.

The earlier [kernel profile](../validation/go-system-one-v1-20260921.md#warp-parallel-causal-attention) attributed 71.64 ms to Q4/Q5/Q6 projections out of 80.4 ms of GPU kernels. This favours packing projection work before tuning launch overhead alone. It is historical profiling, not a profile of the current packed path.

## Techniques reviewed

All upstream links below are pinned to the reviewed commit.

| Implementation | Technique | Use with Gemma |
|---|---|---|
| [Frozen Qwen prefix scorer](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/model/frozen_gpu_prefix.go) | `ScoreSuffixes` packs variable-length projection rows, shares immutable prefix KV and isolates attention. | Adapt the layout and isolation tests to Gemma's quantised runtime. Its attention loop and CPU selected-head dot products need not be copied. |
| [Decider](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/model/decider/runtime.go) | Reads the final hidden row and projects one-token answer labels. | Existing boolean scoring already uses selected logits. A label-code prompt could reduce multi-token enum work, but changes the scoring contract and needs a separate quality evaluation. |
| [OpenJEV](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/model/openjev/runtime.go) | Applies a trained three-class NLI head to the last hidden row. | Requires matching trained weights; not an inference-only replacement for Gemma's vocabulary head. |
| [Kev assessment](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/docs/models/kev-porting-roadmap.md) | Trained pointer projections over option-end and decision rows, with LoRA adaptation. | Reuse the isolation discipline. No native Kev scorer or Gemma-trained pointer head is available here. |
| [Jevlike experiment](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/docs/experiments/jevlike-qwen3/final-report-20260921.md) | Scores options from frozen hidden features with a trained attention head. | Do not adopt the failed head recipe: mean accuracy was 22.20%, against 23.52% random expectation. Direct instruction scoring reached 81.25%, with an earlier option-order failure. |

## Implementation sequence

Steps 1–4 and bounded scratch reuse in step 5 are implemented. Segmented attention is also implemented; CUDA graph replay has not been qualified. This sequence records the original design order.

1. Add an optional batch scorer to the engine. Keep the scalar scorer as a reference and fallback. Start with single-node boolean and enum fields under the existing prompt contract.
2. Pack real context-plus-suffix token rows into a bounded workspace. Run Q/K/V, attention-output and feed-forward projections across the whole group. Use context offsets, lengths and absolute positions to preserve each entry's causal and sliding-window attention. An entry sees the shared prefix and its own tokens only.
3. Gather each last real hidden row on-device. Apply Gemma's final normalisation, project selected head rows, preserve output transforms and return only the small logit matrix. Existing constrained softmax produces the probabilities; no text generation is needed.
4. Extend this to multi-field and multi-token candidate trees. Share each context's KV among its field branches. Preserve candidate order and tree path probabilities while batching independent nodes.
5. Reuse bounded activation scratch. Consider segmented attention kernels and CUDA graph replay after profiling the packed path. More HTTP workers would still contend on the same GPU and scorer lock.

Choose group size by token rows and measured memory, not entry count alone. The current weights occupy 9,173,277,696 device bytes on a 12 GB card. Sweep 128, 256 and 512 token-row budgets, including all KV, activation and projection scratch. Split larger requests into groups while retaining the public 256-context limit.

## Acceptance

- Preserve pinned-reference tolerances. For broader precision experiments, report raw/centred logit errors, probability movement, candidate margins and decision/threshold crossings separately. Bitwise equality is a diagnostic, not a general acceptance requirement.
- Test unequal lengths, reversed context order, contradictory siblings, repeated calls, cancellation and prefix immutability. Verify that changing one context cannot change another result.
- Preserve Gemma normalisation, rotary positions, sliding attention and applicable logit softcapping or token suppression.
- Benchmark distinct positive and negative contexts at 1, 10, 25, 50 and 100 entries, including longer contexts and multi-token enums. Repeated copies of one sentence alone are insufficient evidence.
- Report whole-request latency, entries/s, peak device memory, phase timings and numerical differences. Do not predict a speedup from another model's benchmark.
