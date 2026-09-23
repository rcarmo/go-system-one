# Benchmarks

The five charts use **[`774c5da`](https://github.com/rcarmo/go-system-one/commit/774c5da5c8cf8c833102137d21062b79505c4929)**: automatic packed tree scoring, staged Q6 and Q5 with 512-column chunks. Hardware is an RTX 3060 12 GB with driver 580.173.02 and the [pinned Gemma 4 12B artifacts](../artifacts.md).

Chart times are `/v1/decision` HTTP handler `timings.total_ms`, excluding artifact verification, model loading and device upload. All chart inputs used binary SHA-256 `b73f6923fb7ff5d1f67ea600abacb198711596d736108234c68bd2ae4ce4497b`. [TypeSafe workloads](#typesafe-question-types) were measured separately at `594ba47`, which adds `/v1/systemone`; their client HTTP timings are not mixed into these charts. [History](history.md) retains earlier results.

## JevBench public accuracy

The complete [JevBench v1.4.0 public run](jevbench-v140-public.md) on `b18ee0d` scored **196/231 (84.85%)**: easy 48/48, original 71/72 and hard 77/111. All 231 requests returned valid responses under the normal stop rules. The report includes latency, calibration and source pins; no official sealed-set score or rank is claimed.

The [earlier evaluation](jevbench-public-20260923.md) and [long-context failure-subset recheck](../performance/long-context.md) remain historical evidence. They are not used to fill cells in the new single-revision run.

## Multi-field batches

One boolean plus a three-choice multi-token enum; contexts cycle the [frozen cohort](multifield-cohort.json) with unique ticket numbers. Automatic execution uses a 512-token-row budget and single-request admission.

| Entries | Median | Min–max | Entries/s |
|---:|---:|---:|---:|
| 1 | 131.71 ms | 131.68–137.68 ms | 7.59 |
| 10 | 825.73 ms | 823.01–841.98 ms | 12.11 |
| 25 | 2,114.29 ms | 2,107.56–2,117.68 ms | 11.82 |
| 50 | 4,180.65 ms | 4,178.84–4,181.02 ms | 11.96 |
| 100 | 8,329.54 ms | 8,323.82–8,332.29 ms | 12.01 |

![Current multi-field batch latency](automatic-batches.svg)

[Requests, results and samples](data/q5-chunk512-batches.json): one same-size warm-up and three measured requests per size, cooling to at most 55°C before each. Temperature reached 73°C; sampled device memory peaked at 9,582 MiB. Three samples do not establish production tail latency.

Collection used bounded invocations. The initial automatic run stopped during size 25; sizes 25/50/100 were then collected together. Source chunk hashes, start times and interrupted observations are retained. Only complete cells enter the tables and charts.

## Serial versus packed

The serial override (`-packed-token-rows=0`) and automatic mode used **identical requests and the same binary**. Only sizes 1 and 10 were measured serially.

| Entries | Serial median | Automatic median | Speedup |
|---:|---:|---:|---:|
| 1 | 1,033.58 ms | 131.71 ms | 7.85× |
| 10 | 10,454.30 ms | 825.73 ms | 12.66× |

![Current serial versus packed comparison](multifield-comparison.svg)

[Paired results](data/q5-chunk512-paired.json) retain both modes. No field winners, candidate ranks or diagnostic 95% confidence crossings changed across the 22 field comparisons; maximum candidate-probability movement was 1.898 percentage points. The [serial run](data/q5-chunk512-serial.json) reached 75°C and sampled 9,509 MiB. Its interrupted size-10 cell was re-collected in full using the saved binary. Both modes use the same three-sample cooling protocol. Run order was not randomised; no larger serial batches are extrapolated.

The separate [32-context precision study](../performance/multifield-precision.md) found no winner changes in 80 fields, four losing-rank swaps and one diagnostic 95% crossing. Maximum probability movement there was 16.54 percentage points. Packing uses Q8 activations where tiny serial branch batches use F32. Those differences matter for confidence thresholds even when rankings agree; none of these synthetic comparisons establishes labelled accuracy or calibrated confidence.

## Single-boolean latency

This smaller [request](request.json) asks one boolean question about a short outage context. It differs from the multi-field batches above. All 100 measured responses selected `urgent=true`.

| Statistic | Handler latency |
|---|---:|
| Minimum | 72.43 ms |
| Median | 72.80 ms |
| p95 | 73.49 ms |
| p99 | 73.65 ms |
| Maximum | 73.65 ms |

![Current warm single-boolean latency](warm-latency.svg)

[Complete distribution](data/q5-chunk512-warm.json): one warm-up followed by 100 uninterrupted sequential requests after cooling to at most 55°C. Temperature reached 67°C and sampled memory peaked at 9,372 MiB. These percentiles describe this run; they are not multi-user service guarantees.

## Workload costs

Five warm samples per case, with one warm-up and cooling before each measured request. [Exact requests and responses](data/q5-chunk512-workloads.json) are recorded. These are the same requests as the preceding Q6-staged matrix; older historical matrices used some different prompts.

| Case | Median | Min–max |
|---|---:|---:|
| Boolean cache hit | 74.07 ms | 73.91–83.57 ms |
| Cache disabled | 216.68 ms | 215.62–217.73 ms |
| Cache miss | 212.84 ms | 212.47–213.55 ms |
| Short context | 72.46 ms | 72.36–72.97 ms |
| Long context | 947.26 ms | 945.81–948.36 ms |
| Multi-token enum tree | 128.96 ms | 128.70–129.59 ms |
| Multi-token enum greedy | 198.17 ms | 197.80–218.40 ms |
| Four distinct contexts | 279.65 ms | 277.78–281.17 ms |
| Four boolean fields | 135.00 ms | 134.87–137.99 ms |
| Boolean + enum | 107.22 ms | 106.85–109.51 ms |

![Current workload matrix](workload-matrix.svg)

Temperature reached 65°C and sampled memory peaked at 9,590 MiB. Cache-miss samples replace the schema before each timed request. The long-context request contains the repeated status text recorded in the data. Its token count is not the historical matrix's 371-token case. Sampling can miss brief memory peaks.

## TypeSafe question types

[`594ba47`](https://github.com/rcarmo/go-system-one/commit/594ba476bb33d7a38b8be482687b075ecdb2238d) adds [Noul, Choice and Score](../systemone-api.md). These measurements use one structured outage state and the default packed scorer. Each row has one warm-up and five measured requests, cooled to at most 55°C before every call.

| Question set | Median client HTTP latency | Min–max |
|---|---:|---:|
| Noul | 95.11 ms | 95.09–95.20 ms |
| Choice | 96.87 ms | 95.49–99.45 ms |
| Score | 97.57 ms | 97.43–100.06 ms |
| Noul + choice + score | 130.57 ms | 130.35–134.24 ms |

[Raw requests, responses and samples](data/typesafe-594ba47.json) record binary SHA-256 `7b058c7add081a6ca3438b3cdad78cd75001891914025c56822ba982c9e52347`. Temperature reached 61°C and sampled memory peaked at 9,553 MiB. The HTTP interval runs from `fetch` through JSON decoding; it excludes telemetry queries and loading. `/v1/systemone` exposes no handler timing field, so these numbers have a different timing boundary from the five charts.

This small synthetic workload checks route cost and typed outputs. It is not a TypeSafe accuracy comparison, a serial/packed speedup test or a batch benchmark. Confidence formulas are local approximations; no TypeSafe calibration parity is established.

## Implementation history

![Historical stages and current single-boolean result](latency-comparison.svg)

The chart preserves the Gemma / llama.cpp prototype / native Go / tuned PTX sequence, including the earlier 77.91 ms Q6 result and the refreshed 72.80 ms Q5 result. Entries came from different revisions and runs. llama.cpp's 96.0 ms is worker prefill plus suffix scoring; Go entries are handler timings. [History](history.md) links the earlier datasets. The controlled [Q5 chunk comparison](../performance/q5-chunk512.md) measured its incremental 2.6% batch-latency gain separately.

## Reproduce

All five SVGs are generated from checked-in JSON. `make check` verifies that they match their inputs.

```sh
make benchmark-charts
make benchmark-check
```

For a batch sweep on an idle GPU with verified artifacts:

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

Use `--mode serial --sizes 1,10` for the paired reference. The collector starts and stops a loopback server, rejects a busy GPU and saves each observation. Allow enough command time for cooling; split `--sizes` across runs if needed. Never present incomplete cells as measured medians. Before combining runs, verify revision, binary, model and cohort hashes, exact request bodies, and sample counts.

For the single-boolean distribution and workload matrix, start the verified service with default NVIDIA settings on `127.0.0.1:18085`, then run:

```sh
bun scripts/workload-benchmark.ts \
  --binary bin/go-system-one \
  --revision "$(git rev-parse HEAD)" \
  --url http://127.0.0.1:18085 \
  --out dist/benchmarks/workloads
```

For the separate TypeSafe workload matrix:

```sh
bun scripts/systemone-benchmark.ts \
  --binary bin/go-system-one \
  --model /path/to/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  --tokenizer-dir /path/to/tokenizer \
  --revision "$(git rev-parse HEAD)" \
  --out dist/benchmarks/typesafe.json
```

The collectors abort at 83°C. The workload collector uses an already-running server; ensure it is the specified binary. Other collectors manage their own server. Use a clean source build and record its actual revision, including any local runtime changes. Default chart input paths are in `scripts/benchmarks/main.go`; the derived summary is [current.json](data/current.json). Collection never overwrites committed evidence automatically.
