# Packed Gemma decision scoring

Process several independent contexts in the same transformer matrix operations, then extract only the required candidate logits. Keep the pinned Gemma weights, prompt, token paths and probability calculation unchanged.

The first implementation is opt-in with `-packed-token-rows=512`. It packs single-field, single-node tree decisions; multi-field, deeper trees and oversized contexts still use the serial path. Default behaviour is unchanged. The source review used `go-system-one@63e49ff0a49de4eca17e5e88aea9154d8be41754` and `go-pherence@32bfb937ca1c8608c66411c63f9b1eb39d152cb3`.

## First measurements

Both modes used the same binary with the attention barrier fix from `e53e9770a756f981a3ac3bacc414e9181fa834f5`. One warm-up and three measured requests per case used varied positive and negative contexts with unique ticket numbers. Before each request, the GPU cooled to at most 60°C. Packed mode ran first.

| Entries | Serial median | Packed median | Speedup |
|---:|---:|---:|---:|
| 10 | 1,133.16 ms | 827.51 ms | 1.37× |
| 100 | 11,587.71 ms | 8,517.50 ms | 1.36× |

All candidate probabilities were identical between modes in these HTTP samples. The released-model raw-logit gate also produced identical logits at 128, 256 and 512 token-row budgets. These contexts are longer than the earlier repeated 20-token context; their timings are not directly comparable to its 8.08-second 100-entry result.

The GPU reached at most 75°C. One-second sampling observed up to 9,658 MiB used in packed mode and 9,424 MiB serially; sampling can miss brief peaks. [Raw requests, responses, timings and device readings](../benchmarks/data/packed-contexts-initial.json) are committed. Three samples do not establish tail latency. An earlier uninterrupted sweep used the faulty attention kernel and reached 87°C; its numbers are not acceptance evidence.

Packing exposed an existing shared-memory softmax race: a warp could overwrite the maximum before another warp read it. The new short-softmax regression failed ten consecutive runs with the old kernel and passes with the barrier. Saturated probabilities had hidden raw-logit changes. The original llama.cpp parity fixtures still pass.

The first executor shares projections but launches attention separately per context and uses a reusable suffix region in a private arena. Selected hidden rows remain on-device; selected projection and download still run once per result. Next steps are segmented attention, batched selected readout, persistent bounded scratch and larger-batch projection tuning. Nsight attempts did not yield a usable kernel trace, so no new kernel timing breakdown is claimed.

## Current cost

[`Engine.Decide`](../../model/gosystemone/engine.go) scores contexts serially. The NVIDIA single-branch path combines the context and forced suffix into one causal prefill. It already projects only candidate vocabulary rows through `finishSelectedDevice` in [`gemma4_nvidia.go`](../../model/gemma4_nvidia.go).

The short boolean fixture processes 24 token rows per context. Ten entries therefore cause ten separate transformer passes over 24 rows. Packing would instead process 240 rows through each layer's projections, with attention isolated by context. This can improve matrix utilisation and amortise launches and weight reads. It does not remove the transformer arithmetic for each token.

The multi-branch path has another cost: `runDepth` downloads hidden rows, and `finishRow` uploads each row, computes the full vocabulary vector and downloads it. The scorer then selects the candidate entries. That path needs a device-resident, selected-row readout too.

The earlier [kernel profile](../validation/go-system-one-v1-20260921.md#warp-parallel-causal-attention) attributed 71.64 ms to Q4/Q5/Q6 projections out of 80.4 ms of GPU kernels. This favours packing projection work before tuning launch overhead alone. It is historical profiling, not a profile of the proposed batch path.

## Techniques reviewed

All upstream links below are pinned to the reviewed commit.

| Implementation | Technique | Use with Gemma |
|---|---|---|
| [Frozen Qwen prefix scorer](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/model/frozen_gpu_prefix.go) | `ScoreSuffixes` packs variable-length projection rows, shares immutable prefix KV and isolates attention. | Adapt the layout and isolation tests to Gemma's quantised runtime. Its attention loop and CPU selected-head dot products need not be copied. |
| [Decider](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/model/decider/runtime.go) | Reads the final hidden row and projects one-token answer labels. | Existing boolean scoring already uses selected logits. A label-code prompt could reduce multi-token enum work, but changes the scoring contract and needs a separate quality evaluation. |
| [OpenJEV](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/model/openjev/runtime.go) | Applies a trained three-class NLI head to the last hidden row. | Requires matching trained weights; not an inference-only replacement for Gemma's vocabulary head. |
| [Kev assessment](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/docs/models/kev-porting-roadmap.md) | Trained pointer projections over option-end and decision rows, with LoRA adaptation. | Reuse the isolation discipline. No native Kev scorer or Gemma-trained pointer head is available here. |
| [Jevlike experiment](https://github.com/rcarmo/go-pherence/blob/32bfb937ca1c8608c66411c63f9b1eb39d152cb3/docs/experiments/jevlike-qwen3/final-report-20260921.md) | Scores options from frozen hidden features with a trained attention head. | Do not adopt the failed head recipe: mean accuracy was 22.20%, against 23.52% random expectation. Direct instruction scoring reached 81.25%, with an earlier option-order failure. |

## Implementation order

1. Add an optional batch scorer to the engine. Keep the scalar scorer as a reference and fallback. Start with single-node boolean and enum fields under the existing prompt contract.
2. Pack real context-plus-suffix token rows into a bounded workspace. Run Q/K/V, attention-output and feed-forward projections across the whole group. Use context offsets, lengths and absolute positions to preserve each entry's causal and sliding-window attention. An entry sees the shared prefix and its own tokens only.
3. Gather each last real hidden row on-device. Apply Gemma's final normalisation, project selected head rows, preserve output transforms and return only the small logit matrix. Existing constrained softmax produces the probabilities; no text generation is needed.
4. Extend this to multi-field and multi-token candidate trees. Share each context's KV among its field branches. Preserve candidate order and tree path probabilities while batching independent nodes.
5. Reuse bounded activation scratch. Consider segmented attention kernels and CUDA graph replay after profiling the packed path. More HTTP workers would still contend on the same GPU and scorer lock.

Choose group size by token rows and measured memory, not entry count alone. The current weights occupy 9,173,277,696 device bytes on a 12 GB card. Sweep 128, 256 and 512 token-row budgets, including all KV, activation and projection scratch. Split larger requests into groups while retaining the public 256-context limit.

## Acceptance

- Compare packed versus independent candidate logits and probabilities without weakening existing numerical tolerances.
- Test unequal lengths, reversed context order, contradictory siblings, repeated calls, cancellation and prefix immutability. Verify that changing one context cannot change another result.
- Preserve Gemma normalisation, rotary positions, sliding attention and applicable logit softcapping or token suppression.
- Benchmark distinct positive and negative contexts at 1, 10, 25, 50 and 100 entries, including longer contexts and multi-token enums. Repeated copies of one sentence alone are insufficient evidence.
- Report whole-request latency, entries/s, peak device memory, phase timings and numerical differences. Do not predict a speedup from another model's benchmark.
