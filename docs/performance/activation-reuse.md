# Q/K activation reuse evaluation

Sharing one quantised activation buffer between query and key projections passed the existing parity checks but did not establish a reliable whole-request improvement. The prototype was removed; runtime code remains at the accepted Q5 chunk512 implementation.

## Candidate

The probe starts from `ee905d0271a13569eda5449cfd03932e955337ca`. It changes only prefill query/key projection dispatch: matching Q4_K matrices share the existing prepared Q8 buffer, and matching Q5_K matrices share quantised bytes, scales and sums. Batches smaller than four keep the existing path. Matrix kernels, reduction arithmetic, prompts and precision choices are unchanged.

The pinned model has 43 layers with Q4 query/key matrices and five with Q5 query/key matrices. Value projection formats vary: Q4/Q4/Q4 in five layers, Q4/Q4/Q5 in five, Q4/Q4/Q6 in 33, Q5/Q5/Q5 in three and Q5/Q5/Q6 in two. Q6 uses 16-value activation groups and cannot consume the Q5 32-value groups unchanged. The existing compatible Q4 gate/up weights already use a combined projection followed by GELU; this experiment does not duplicate that optimisation.

## Measurements

The same frozen two-field HTTP workload, idle GPU checks and cooling protocol as the [main benchmarks](../benchmarks/README.md) were used. Each cell has one warm-up and three measured handler samples. The collector waits for at most 55°C with an 83°C abort guard; two baseline-run-4 calls recorded 56°C in the immediate pre-request query after that wait (one each at sizes 10 and 100). Those observations are retained as protocol deviations, not claimed to meet the strict 55°C start bound. Run order was baseline/candidate/candidate/baseline, followed by another baseline/candidate check. The RTX 3060 used driver 580.173.02; observed temperature reached 73°C and sampled device memory 9,732 MiB. Sampling can miss brief peaks.

| Run | Binary | 1 entry | 10 entries | 100 entries |
|---:|---|---:|---:|---:|
| 1 | Baseline | 119.24 ms | 814.02 ms | 8,331.24 ms |
| 2 | Reuse | 121.74 ms | 809.58 ms | 8,318.29 ms |
| 3 | Reuse | 121.54 ms | 811.74 ms | 8,313.61 ms |
| 4 | Baseline | 128.16 ms | 835.46 ms | 8,644.37 ms |
| 5 | Baseline | not measured | 842.05 ms | 8,627.18 ms |
| 6 | Reuse | not measured | 811.45 ms | 8,307.53 ms |

The first 100-entry comparison improved by only 0.16–0.21%. Later baseline runs were 3.6–3.8% slower than the first baseline, despite using the same binary. Sparse clock and temperature observations do not establish the cause. That drift exceeds the initial gain, so these runs cannot support a reliable percentage improvement. More fusion or graph-capture work needs a new controlled experiment; no runtime feature is retained solely because the micro-level operation count fell.

[Raw requests, responses, samples, hashes and telemetry](../benchmarks/data/activation-reuse-evaluation.json) retain all six runs. All 1,106 cross-run field comparisons matched winners and candidate probabilities exactly. Both the pinned multi-field llama.cpp fixture and packed-versus-serial released-model logit tests passed; the latter reported zero logit/probability difference at 128, 256 and 512 rows. No tolerance changed.

The investigation-only source patch is kept outside the source tree for the development session (SHA-256 `bb23b3d9141a94293776d7c5ba6d1c9a783f12e453e496da1dd3105afa69ba4e`). It is not part of the upstream hand-off or production build. This closes the bounded activation-reuse pass with an inconclusive result, leaving the accepted execution path unchanged.
