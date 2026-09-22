# Benchmark history

These measurements predate the current staged-Q6 runtime. The [main benchmark page](README.md) and all five chart files now use `aca5e4c` measurements; only the implementation-history chart intentionally combines earlier stages with the current single-boolean run.

## Single-boolean implementation sequence

| Stage | Recorded latency | Basis |
|---|---:|---|
| llama.cpp prototype | 96.0 ms | 53.8 ms worker prefill + 42.2 ms suffix scoring |
| Early native Go | 520.8–521.9 ms | Earlier warm request range |
| Initial native tuning | 135.3–136.7 ms | Later warm request range |
| Earlier hand-tuned Go/PTX | 81.13 ms median | 100 warm handlers at `0dd65d4` |
| Current staged-Q6 Go/PTX | 77.91 ms median | 100 warm handlers at `aca5e4c` |

The model and boolean fixture were pinned, but the measurements came from different executable revisions and runs. Worker and HTTP-handler intervals differ. These figures are not a controlled simultaneous A/B throughput test.

The earlier 100-request run had p95 81.59 ms and p99 82.15 ms. [Its raw distribution](data/nvidia-http-100-0dd65d4-20260922.json) and [old comparison/workload summary](data/go-system-one-v1.json) are unchanged. An even earlier standalone sample is in [nvidia-http-100-f65652f6-20260922.json](../validation/data/nvidia-http-100-f65652f6-20260922.json). Detailed model/kernel provenance is in the [v1 validation record](../validation/go-system-one-v1-20260921.md).

## Packed execution stages

| Stage | Evidence |
|---|---|
| Initial cross-context packing | [Packed versus serial samples](data/packed-contexts-initial.json) |
| Segmented attention and selected batch readout | [Segmented samples](data/packed-contexts-segmented.json) |
| Scratch reuse and projection dispatch | [Dispatch samples](data/packed-contexts-dispatch.json) |
| Initial multi-field tree packing | [Paired serial/packed samples](data/packed-multifield.json) |
| Automatic default before Q6 staging | [Completed 1/10/25/50/100 sweep](data/automatic-batch-sweep.json) |

The pre-Q6 automatic sweep recorded 162.04 ms for one entry, 1.149 s for 10 and 11.662 s for 100. Its requests match the current automatic sweep; current medians are 131.31 ms, 0.935 s and 9.491 s respectively. Run order and host conditions were not randomised, so keep the per-run samples and limits beside that comparison.

The initial paired multi-field experiment recorded 1.017 s versus 0.142 s for one entry and 10.427 s versus 0.968 s for 10. Those inputs differ from the current paired comparison. Do not mix their bars or derive a cross-workload speedup.

The [precision study](../performance/multifield-precision.md) and [execution notes](../performance/packed-decisions.md) record the Q8 activation trade-off, attention race fix, tested kernels and rejected experiments. Historical benchmark data establishes neither task accuracy nor probability calibration.
