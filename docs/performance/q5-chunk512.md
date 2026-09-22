# Q5 512-column chunks

Staging 512 input columns per Q5 tile reduces block barriers by half. The 24-row and 32-row tiles still compute 64 output rows. Each four-lane subgroup visits quantisation groups in the same order, including the final 256-column tail. Q6 retains its 256-column implementation.

## Whole-request comparison

The reference is `543f0be2fcd9e43a7387fecf63656babca089192`. The candidate was built from that source plus this Q5 change. Both use automatic packing, the same pinned Gemma artifacts and the frozen two-field benchmark requests.

| Run order | Kernel | 10 entries, median | 100 entries, median |
|---:|---|---:|---:|
| 1 | Reference, 256 columns | 832.43 ms | 8,552.16 ms |
| 2 | Candidate, 512 columns | 813.30 ms | 8,326.74 ms |
| 3 | Candidate, 512 columns | 812.56 ms | 8,329.36 ms |
| 4 | Reference, 256 columns | 836.04 ms | 8,555.15 ms |

Each row has one warm-up and three measured requests per size, cooled to at most 55°C before each call. The A/B/B/A order reduces run-order bias but is not a randomised trial. The 100-entry latency reduction is about 2.6%; throughput rises from about 11.69 to 12.01 entries/s. Sampled temperature reached 73°C and device memory 9,580 MiB. Sampling can miss brief peaks.

The preceding full candidate sweep measured 121.98 ms, 812.96 ms, 2,114.85 ms, 4,182.48 ms and 8,333.02 ms for 1, 10, 25, 50 and 100 entries. Single-entry timing is effectively unchanged from the prior 121.39 ms sample; the retained gain is in batch throughput.

[Raw full sweep, alternating runs and kernel measurements](../benchmarks/data/q5-chunk512.json) include requests, results, binary hashes and device observations. The alternating runs contain 220 field decisions per run. All three later runs match the first reference run's winners and candidate probabilities exactly (660 cross-run field comparisons). The existing packed-versus-serial released-model logit gate also reports zero difference at 128, 256 and 512 rows. No tolerance changed.

## Kernel selection and resource cost

Eight variants tested chunk sizes 128, 256, 512 and 1,024; row tiles 24, 32, 48 and 64; and output tiles 64 and 128. Three repeated measurements rotated variant order. The 512-column variants gave useful projection improvements of roughly 7–21% over the current dispatch candidates on the tested matrix shapes. Larger row tiles and the 1,024-column chunk did not win consistently.

| Accepted kernel | Registers/thread | Shared memory | Stack / spill stores / spill loads |
|---|---:|---:|---:|
| Q5 24 × 64, 512 columns | 80 | 15,360 bytes | 8 / 4 / 4 bytes |
| Q5 32 × 64, 512 columns | 80 | 20,480 bytes | 0 / 0 / 0 bytes |
| Unchanged Q6 24 × 64 | 69 | 7,680 bytes | 0 / 0 / 0 bytes |

A looped group variant removed the 24-row tile's small spill but was generally slower. The unrolled implementation is retained because its whole-request benefit survived the alternating test. These are CUDA 12.8 `sm_86` resource reports; the runtime uses embedded PTX and the NVIDIA driver only.

Synthetic width/tail tests cover 256, 512, 768 and 3,840 input columns, partial output tiles and partial row batches. They pass against the existing Q5 reference, as do both pinned llama.cpp fixtures, batch isolation/logit tests, race checks, `make check` and Linux ARM64/RISC-V builds.

The [main benchmark tables and all five charts](../benchmarks/README.md) now use the committed `774c5da` runtime: 72.80 ms single-boolean median and 8,329.54 ms for 100 two-field entries. The pre-commit alternating comparison above remains separate, with its original binary hashes and run order.
