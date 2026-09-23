# Benchmark history

The [main benchmark page](README.md) and all five charts use the Q5 chunk512 runtime at `774c5da`. This page retains earlier measurements and their original timing boundaries. The separate TypeSafe workload matrix uses `594ba47`; it measures client HTTP latency.

## Single-boolean implementation sequence

| Stage | Recorded latency | Basis |
|---|---:|---|
| llama.cpp prototype | 96.0 ms | 53.8 ms worker prefill + 42.2 ms suffix scoring |
| Early native Go | 520.8–521.9 ms | Earlier warm request range |
| Initial native tuning | 135.3–136.7 ms | Later warm request range |
| Earlier hand-tuned Go/PTX | 81.13 ms median | 100 warm handlers at `0dd65d4` |
| Earlier staged-Q6 Go/PTX | 77.91 ms median | 100 warm handlers at `aca5e4c` |
| Current Q5 chunk512 Go/PTX | 72.80 ms median | 100 warm handlers at `774c5da` |

The model and boolean fixture were pinned, but the measurements came from different executable revisions and runs. Worker and HTTP-handler intervals differ. These figures are not a controlled simultaneous A/B throughput test.

The earlier 100-request run had p95 81.59 ms and p99 82.15 ms. [Its raw distribution](data/nvidia-http-100-0dd65d4-20260922.json) and [old comparison/workload summary](data/go-system-one-v1.json) are unchanged. An even earlier standalone sample is in [nvidia-http-100-f65652f6-20260922.json](../validation/data/nvidia-http-100-f65652f6-20260922.json). Detailed model/kernel provenance is in the [v1 validation record](../validation/go-system-one-v1-20260921.md).

The Q6 refresh is preserved in its [summary](data/q6-staged-summary.json), [100-request distribution](data/q6-staged-warm.json), [workload matrix](data/q6-staged-workloads.json), [automatic sweep](data/q6-staged-batches.json), [serial run](data/q6-staged-serial.json) and [paired comparison](data/q6-staged-paired.json). Those files retain their original revision and binary hashes.

## Packed execution stages

| Stage | Evidence |
|---|---|
| Initial cross-context packing | [Packed versus serial samples](data/packed-contexts-initial.json) |
| Segmented attention and selected batch readout | [Segmented samples](data/packed-contexts-segmented.json) |
| Scratch reuse and projection dispatch | [Dispatch samples](data/packed-contexts-dispatch.json) |
| Initial multi-field tree packing | [Paired serial/packed samples](data/packed-multifield.json) |
| Automatic default before Q6 staging | [Completed 1/10/25/50/100 sweep](data/automatic-batch-sweep.json) |
| Staged Q6 | [Full sweep](data/q6-staged-batches.json) |
| Staged Q5 metadata | [Pre-commit candidate sweep](data/q5q6-staged-candidate.json), [arithmetic/resource notes](../performance/q5-staged.md) |
| Q5 512-column chunks | [Alternating comparison](data/q5-chunk512.json), [resource trade-off](../performance/q5-chunk512.md) |

The pre-Q6 automatic sweep recorded 162.04 ms for one entry, 1.149 s for 10 and 11.662 s for 100. Its requests match the current automatic sweep; Q5 chunk512 medians are 131.71 ms, 0.826 s and 8.330 s respectively. Run order and host conditions were not randomised, so keep the per-run samples and limits beside that comparison.

The initial paired multi-field experiment recorded 1.017 s versus 0.142 s for one entry and 10.427 s versus 0.968 s for 10. Those inputs differ from the current paired comparison. Do not mix their bars or derive a cross-workload speedup.

The [precision study](../performance/multifield-precision.md) and [execution notes](../performance/packed-decisions.md) record the Q8 activation trade-off, attention race fix, tested kernels and rejected experiments. The later [Q/K activation-reuse experiment](../performance/activation-reuse.md) is recorded separately in [six raw runs](data/activation-reuse-evaluation.json). Baseline drift exceeded its initial gain; no runtime change was retained and the current charts are unchanged. Historical benchmark data establishes neither task accuracy nor probability calibration.
