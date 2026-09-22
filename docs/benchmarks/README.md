# Benchmarks

Current charts use **[`aca5e4c`](https://github.com/rcarmo/go-system-one/commit/aca5e4c8952219d1d90bd4c45ff9b8fa80cc4d4d)**: automatic packed tree scoring and staged Q6 PTX. Hardware is an RTX 3060 12 GB with driver 580.173.02 and the [pinned Gemma 4 12B artifacts](../artifacts.md).

Times are HTTP handler `timings.total_ms`, excluding artifact verification, model loading and device upload. Every current run used the same binary, SHA-256 `364cc7f2924f0bb53e91ef2840765eb6fc07ebc84e11ffebc4d753775942c31a`. The old raw measurements are retained in [History](history.md).

A subsequent [Q5 staging run](../performance/q5-staged.md) measured a further 8–12% throughput gain. Its pre-commit sweep is reported separately; charts on this page retain the explicit `aca5e4c` source until the next full refresh.

## Multi-field batches

One boolean plus a three-choice multi-token enum; contexts cycle the [frozen cohort](multifield-cohort.json) with unique ticket numbers. Automatic execution uses a 512-token-row budget and single-request admission.

| Entries | Median | Min–max | Entries/s |
|---:|---:|---:|---:|
| 1 | 131.31 ms | 131.05–131.35 ms | 7.62 |
| 10 | 935.39 ms | 933.57–935.39 ms | 10.69 |
| 25 | 2,409.13 ms | 2,404.46–2,411.10 ms | 10.38 |
| 50 | 4,760.64 ms | 4,756.93–4,763.14 ms | 10.50 |
| 100 | 9,490.67 ms | 9,482.38–9,491.01 ms | 10.54 |

![Current multi-field batch latency](automatic-batches.svg)

[Requests, results and samples](data/q6-staged-batches.json): one same-size warm-up and three measured requests per size, cooling to at most 55°C before each. Temperature reached 73°C; sampled device memory peaked at 9,580 MiB. Three samples do not establish production tail latency.

## Serial versus packed

The serial override (`-packed-token-rows=0`) and automatic mode used **identical requests and the same binary**. Only sizes 1 and 10 were measured serially.

| Entries | Serial median | Automatic median | Speedup |
|---:|---:|---:|---:|
| 1 | 1,039.08 ms | 131.31 ms | 7.91× |
| 10 | 10,638.51 ms | 935.39 ms | 11.37× |

![Current serial versus packed comparison](multifield-comparison.svg)

[Paired results](data/q6-staged-paired.json) retain both modes. No field winners changed across the 22 comparisons; maximum candidate-probability movement was 1.898 percentage points. The serial run reached 75°C and sampled 9,587 MiB. It uses the same three-sample cooling protocol as the automatic sweep. Speedups are measured only at those sizes, with no extrapolation to larger serial batches.

The separate [32-context precision study](../performance/multifield-precision.md) found no winner changes in 80 fields, four losing-rank swaps and one diagnostic 95% crossing. Maximum probability movement there was 16.54 percentage points. Packing uses Q8 activations where tiny serial branch batches use F32. Those differences matter for confidence thresholds even when rankings agree; none of these synthetic comparisons establishes labelled accuracy or calibrated confidence.

## Single-boolean latency

This smaller [request](request.json) asks one boolean question about a short outage context. It is a different workload from the multi-field batches above. All 100 measured responses selected `urgent=true`.

| Statistic | Handler latency |
|---|---:|
| Minimum | 77.55 ms |
| Median | 77.91 ms |
| p95 | 78.71 ms |
| p99 | 78.79 ms |
| Maximum | 78.80 ms |

![Current warm single-boolean latency](warm-latency.svg)

[Complete distribution](data/q6-staged-warm.json): one warm-up followed by 100 uninterrupted sequential requests after cooling to at most 55°C. Temperature reached 67°C and sampled memory peaked at 9,372 MiB. The percentile estimates describe this run; they are not multi-user service guarantees.

## Workload costs

Five warm samples per case, with one warm-up and cooling before each measured request. [Exact requests and responses](data/q6-staged-workloads.json) are recorded; these cases replace the old matrix rather than claiming to reproduce every historical prompt.

| Case | Median | Min–max |
|---|---:|---:|
| Boolean cache hit | 79.06 ms | 79.00–88.56 ms |
| Cache disabled | 234.82 ms | 232.65–236.59 ms |
| Cache miss | 231.09 ms | 230.76–231.24 ms |
| Short context | 75.52 ms | 75.44–75.86 ms |
| Long context | 1,047.54 ms | 1,043.84–1,057.73 ms |
| Multi-token enum tree | 137.92 ms | 137.26–139.19 ms |
| Multi-token enum greedy | 203.49 ms | 202.63–203.64 ms |
| Four distinct contexts | 307.09 ms | 306.11–308.21 ms |
| Four boolean fields | 144.09 ms | 143.55–144.41 ms |
| Boolean + enum | 111.07 ms | 110.93–111.37 ms |

![Current workload matrix](workload-matrix.svg)

Temperature reached 65°C and sampled memory peaked at 9,590 MiB. Cache-miss samples replace the schema before each timed request. The long-context request contains the repeated status text recorded in the data. Its token count is not the historical matrix's 371-token case. Sampling can miss brief memory peaks.

## Implementation history

![Historical stages and current single-boolean result](latency-comparison.svg)

The chart preserves the Gemma → llama.cpp prototype → native Go → tuned PTX sequence and adds the current 77.91 ms result. Earlier entries came from different revisions and runs. llama.cpp's 96.0 ms is worker prefill plus suffix scoring; the Go entries are handler timings. [History](history.md) records those distinctions and links every earlier dataset.

## Reproduce

All five SVGs are generated from checked-in JSON. `make check` verifies that they match their inputs.

```sh
make benchmark-charts
make benchmark-check
```

For a batch sweep on an idle GPU with the verified artifacts:

```sh
make build
bun scripts/batch-benchmark.ts \
  --binary bin/go-system-one \
  --model /path/to/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  --tokenizer-dir /path/to/tokenizer \
  --revision "$(git rev-parse HEAD)" \
  --mode automatic --sizes 1,10,25,50,100 \
  --out dist/benchmarks/automatic.json
```

Use `--mode serial --sizes 1,10` for the paired reference. The collector starts and stops a loopback server, rejects a busy GPU and saves each observation. Allow enough command time for cooling; split `--sizes` across runs if needed. Never present incomplete cells as measured medians.

For the single-boolean distribution and workload matrix, start the verified service with default NVIDIA settings on `127.0.0.1:18085`, then run:

```sh
bun scripts/workload-benchmark.ts \
  --binary bin/go-system-one \
  --revision "$(git rev-parse HEAD)" \
  --url http://127.0.0.1:18085 \
  --out dist/benchmarks/workloads
```

The collectors abort at 83°C. The workload collector uses the already-running server; the operator must ensure it is the specified binary. Default filenames for chart generation are in `scripts/benchmarks/main.go`; the single-request/workload chart summary is [current.json](data/current.json). Collection never overwrites committed evidence automatically.
