# Q5 shared-memory staging

This first Q5 staging revision stores activation scales and sums alongside quantised activation bytes. A [subsequent 512-column chunk revision](q5-chunk512.md) reduces barriers and adds a measured 2.6% batch latency improvement.

The Q5 projection kernel stages activation scales and sums alongside quantised activation bytes. Two tiles handle 24 or 32 input rows and 64 output rows per block. This reuses metadata across outputs and reduces repeated global loads. Production still loads embedded PTX without a CUDA toolkit.

## Measured service effect

Automatic multi-field requests use the same frozen contexts and collector as the [Q6-only benchmark](../benchmarks/README.md). One warm-up preceded three cooled samples per size on the RTX 3060, driver 580.173.02.

| Entries | Q6-only median | Q5 + Q6 median | Speedup |
|---:|---:|---:|---:|
| 1 | 131.31 ms | 121.39 ms | 1.08× |
| 10 | 935.39 ms | 832.06 ms | 1.12× |
| 25 | 2,409.13 ms | 2,171.49 ms | 1.11× |
| 50 | 4,760.64 ms | 4,298.79 ms | 1.11× |
| 100 | 9,490.67 ms | 8,563.13 ms | 1.11× |

[Raw Q5/Q6 candidate sweep](../benchmarks/data/q5q6-staged-candidate.json) records the binary hash, exact requests, results and measurements. Its source qualifier is `d624b88+q5-natural-staging`, because it was collected before this kernel commit. The [Q6-only baseline](../benchmarks/data/q6-staged-batches.json) is unchanged. Runs were sequential rather than randomised; three samples do not establish tail latency. Maximum sampled temperature was 73°C and sampled memory use 9,580 MiB.

All 372 field winners and candidate probability vectors across the five sizes matched the saved Q6-only responses exactly. This demonstrates no observed difference for this cohort; it does not establish universal bitwise equivalence or task accuracy. The original boolean and multi-field llama.cpp gates, and the separate packed-versus-serial logit test, pass without tolerance changes.

## Arithmetic and resource checks

The first Q5 prototype forced separate rounding of scale multiplication and minimum subtraction. It improved kernel throughput but moved released-model logits more than expected. The accepted version permits the same contraction as the existing MMQ kernel. Projection probes then produced identical outputs on the tested shapes, followed by unchanged released-model logits and probabilities.

CUDA 12.8 `sm_86` compilation reports:

| Kernel | Registers/thread | Shared memory | Stack/spills |
|---|---:|---:|---:|
| Q5, 24 × 64 | 88 | 7,680 bytes | 0 |
| Q5, 32 × 64 | 90 | 10,240 bytes | 0 |
| Existing staged Q6, 24 × 64 | 69 | 7,680 bytes | 0 |

Dispatch uses the 32-row Q5 tile at 128 or more input rows, and the 24-row tile above 16 input rows when there are more than 2,048 output rows. Other shapes retain their previous kernels. Synthetic tests cover partial output tiles and row tails against the original batch-4 arithmetic. Kernel microbenchmarks guided selection; the HTTP sweep above determined retention.

```sh
./scripts/generate-qk-staged-ptx.sh
go test ./backends/nvidia/runtime -run 'TestQ[56]PackedMMQ' -count=10
make check
make cross-build
```

Generated PTX is reproducible with the recorded compiler. Hardware acceptance also ran `make hardware-check` and `TestGoSystemOnePackedReleasedModelMatchesSerial` against the pinned artifacts. Main benchmark charts remain labelled with their Q6-only revision until a full post-commit refresh; the new sweep is kept separate rather than mixing revisions silently.
