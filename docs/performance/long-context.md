# NVIDIA long-context decisions

The NVIDIA backend no longer rejects every request above 2,048 tokens. All 36 JevBench requests rejected by the previous runtime now return HTTP 200 with valid distributions, without truncation or prompt changes. Their visible lengths range from 2,149 to 3,937 tokens.

## Execution and memory

The old limit came from three implementation choices: fixed 2,048-element attention score arrays, rotary-position tables capped at 2,048 positions, and full-length KV allocation for every attention layer. It was not the Gemma checkpoint's context limit.

The validated short-context kernels remain unchanged. For attention spans above 2,048, a new kernel puts scores in bounded global scratch and launches query rows in chunks. It keeps the same grouped-query, causal, sliding-window and independent-branch addressing. Scratch is at most 8 MiB for the supported long-attention head counts (up to 64); no allocation scales with all query rows times all visible tokens. CUDA 12.8 `sm_86` compilation reports 72 registers/thread, 1,024 bytes of shared memory and no spills. The driver loads embedded PTX; production still needs no CUDA toolkit.

Sliding-window layers keep only the window plus one 512-row prefill chunk. Compaction copies through temporary device storage to avoid overlapping CUDA copies. Global-attention layers retain the complete history. Absolute rotary positions are preserved; attention offsets are relative to each layer's retained KV base. An arena that discarded its original prefix cannot be reused for a different context, so the scorer rebuilds it instead.

The logical NVIDIA limit is the smaller of the model's declared context and **32,768 visible tokens**, including the shared prompt, state and scored suffix. Actual admission also depends on KV size and available device memory. The arena checks overflow-safe trunk/private-suffix sizes before allocating and leaves a 512 MiB device reserve for work buffers. This is a conservative preflight check, not protection against another process consuming VRAM afterwards.

Capacity refusals return HTTP **422** with a token-limit or required/free-memory explanation. Requests are never silently shortened. Kernel-level tests at 32,768 tokens do not establish that a full 12B request of that length fits a 12 GB GPU; released-model validation here reaches 3,937 tokens.

## Preserved numerical checks

* Causal attention matches the CPU oracle at lengths 2,048, 2,049, 4,096, 8,193 and 32,768, with full and sliding windows and the existing `2e-5` tolerance. Separate tests cover wide heads, scratch-chunk boundaries, parent branches and reordered private suffixes.
* A synthetic Gemma model compares compacted sliding KV against full KV under the same attention mask, including absolute RoPE, packed prefixes and branch-local suffixes. The test uses `1e-6` absolute/relative comparison.
* The released-model boolean and multi-field llama.cpp gates pass unchanged. The pinned packed/serial fixture still reports zero logit/probability difference at 128, 256 and 512 rows.
* A 2,458-token released-model request returns identical answers on repetition after cache compaction. Short-request recovery, cancellation and explicit capacity refusal are tested, including release of the HTTP admission gate.

The long kernel generator reproduces its embedded PTX exactly. Full repository checks, race checks, seven browser scenarios and Linux ARM64/RISC-V cross-builds pass.

## Previously rejected JevBench requests

[Raw recheck evidence](../benchmarks/data/long-context-20260923/recheck.json) records the exact 36 previously failed requests, responses, timing, thermal observations and source hashes. All returned strict-valid candidate distributions; **27/36 answers were correct** using the pinned JevBench scorer. This verifies removal of the observed capacity failures, not correctness of every long answer.

Each request ran once against the fix candidate after cooling to at most 55°C, with an 83°C abort guard. Temperature reached 82°C and sampled device memory 10,891 MiB. Client HTTP p50/p95 were **14.02/21.63 seconds** for this long-input subset. The measurements include actual long-input processing; they are not comparable to the short boolean benchmark's workload.

The binary SHA-256 is `c5792bb836624ccb4f554f946bc4085a1ae19b2d9d4f5856b8addcb58f90dad1`. It was built before commit; the evidence records exact runtime source hashes. A fresh build after validation produced the same binary hash. The [original JevBench report](../benchmarks/jevbench-public-20260923.md) and its stopped/diagnostic runs remain unchanged. This failure-subset recheck is not a new full-suite result; combining its 27 successes with old answers would mix revisions.

## Short-request regression check

[An A/B/B/A comparison](../benchmarks/data/long-context-20260923/short-check.json) used the same short boolean request, one warm-up and 20 measured handlers per block. Each block started at at most 55°C.

| Run | Runtime | Median |
|---:|---|---:|
| 1 | Previous | 72.46 ms |
| 2 | Long-context candidate | 72.41 ms |
| 3 | Long-context candidate | 72.67 ms |
| 4 | Previous | 72.60 ms |

Complete results matched across all 80 measured requests. No material short-request latency change was observed. The existing five benchmark charts remain pinned to their complete earlier sweep; this small regression check does not replace them.

## Reproduce the gates

```sh
./scripts/generate-long-attention-ptx.sh
go test ./backends/nvidia/runtime -run 'Test(Long|CausalBatch|Segmented|Independent)' -count=3
go test ./model -run 'TestGemma4SlidingPrefillMatchesFullKV|TestSlidingKVTailAndClone' -count=1
GO_SYSTEM_ONE_MODEL=/path/to/gemma-4-12b-it-UD-Q4_K_XL.gguf \
GO_SYSTEM_ONE_TOKENIZER_DIR=/path/to/tokenizer \
go test ./model/gosystemone -run '^TestLongContextReleasedModelRecovery$' -v -count=1
```

The evidence directory includes the exact collection scripts and original parity/resource logs. GPU tests skip when CUDA is unavailable; the released-model gate requires the explicit artifact variables above.
