# Benchmarks

NVIDIA tree scoring now packs contexts and field branches automatically. The measurements below use the pinned Gemma 4 12B model on an RTX 3060. Times are handler `timings.total_ms`, excluding model loading, artifact hashing and device upload.

## Automatic multi-field batches

Measured at [`321e6ce`](https://github.com/rcarmo/go-system-one/commit/321e6ce9d7859f30a6b31949b933ee4d1312c456), with the default 512-row budget, one boolean and a three-choice multi-token enum:

| Entries | Median | Min–max | Entries/s |
|---:|---:|---:|---:|
| 1 | 162.04 ms | 157.97–162.10 ms | 6.17 |
| 10 | 1,149.19 ms | 1,149.05–1,155.27 ms | 8.70 |
| 25 | 2,941.18 ms | 2,938.23–2,943.15 ms | 8.50 |
| 50 | Incomplete: two measured samples before collector timeout | — | — |
| 100 | Not yet measured in this sweep | — | — |

![Automatic multi-field batch latency](automatic-batches.svg)

Each completed row has one same-size warm-up and three measured requests, cooled to at most 55°C before each request. Contexts cycle the [frozen cohort](multifield-cohort.json) with unique ticket numbers. The [raw sweep](data/automatic-batch-sweep.json) includes request bodies, results, device readings, binary/model hashes and the incomplete 50-entry observations. The 600-second tool timeout interrupted collection; unfinished cells are excluded from the chart. These three-sample rows do not establish tail latency.

## Paired serial versus packed comparison

The earlier paired run used the same binary and requests in both modes, with one boolean and a three-choice multi-token enum:

| Entries | Serial median | Packed median | Speedup |
|---:|---:|---:|---:|
| 1 | 1,017.42 ms | 141.57 ms | 7.19× |
| 10 | 10,427.21 ms | 967.59 ms | 10.78× |

![Paired multi-field comparison](multifield-comparison.svg)

[Paired requests and samples](data/packed-multifield.json) use different context text from the automatic sweep. Do not calculate a cross-table speedup. Three warm measurements followed one warm-up for each case. The packed path avoids repeated context work and full-vocabulary projection; it also uses Q8 activations where tiny serial branch batches use F32.

The [broader precision comparison](../performance/multifield-precision.md) recorded no winner changes in 80 fields, four losing-rank changes and one diagnostic 95% threshold crossing. Maximum probability movement was 16.54 percentage points. These are execution-path comparisons on unlabelled synthetic inputs, not accuracy or calibration results. Use `-packed-token-rows=0` to retain serial scoring when comparing confidence-sensitive workloads.

## Historical single-boolean distribution

Before the batch work, 100 sequential warm requests produced an 81.13 ms median, 81.59 ms p95 and 82.15 ms p99. Every response selected `urgent=true`. This is a different workload from the multi-field batches above.

![Sorted warm HTTP latency distribution](warm-latency.svg)

The chart uses the complete committed [100-request sample](data/nvidia-http-100-0dd65d4-20260922.json), SHA-256 `fb8326c2f6920c088fd31fc75bdbc37195decf267fcb8074358e75d11df502a9`. One request warmed the service; the next 100 ran sequentially. The collector rejected response drift and recorded the common `urgent=true` decision. The fixture contains one 77-token prepared prompt, a 20-token context, one boolean tree node and a four-token suffix.

| Statistic | Latency |
|---|---:|
| Minimum | 80.31 ms |
| Median | 81.13 ms |
| p95 | 81.59 ms |
| p99 | 82.15 ms |
| Maximum | 82.39 ms |

## Historical implementation sequence

![Latency from the llama.cpp prototype to hand-tuned Go/PTX](latency-comparison.svg)

The chart follows the development order. The llama.cpp prototype established a 96.0 ms reference on the fixed Gemma 4 12B request. The first native Go/NVIDIA runtime took 520.8–521.9 ms. Initial tuning reduced it to 135.3–136.7 ms. The recorded hand-tuned Go/PTX run had an 81.13 ms median.

That median is 15.5% lower than the llama.cpp prototype, 40.3% lower than the midpoint of the initially tuned native range, and 6.43 times faster than the midpoint of the early native range.

The native entries measure complete warm HTTP requests from different executable revisions. The llama.cpp entry is its reported 53.8 ms prefill plus 42.2 ms suffix scoring. All measurements use the same frozen fixture and host. They do not measure throughput or performance on other hardware.

## Historical workload scaling

![Warm workload matrix](workload-matrix.svg)

Each point is the median of five warm HTTP samples. Lines span the observed minimum and maximum. The horizontal axis is logarithmic because the long-context and four-field cases are much slower than the baseline.

| Case | Median | Range |
|---|---:|---:|
| Baseline cache hit | 81.00 ms | 80.44–81.94 ms |
| Short context | 81.14 ms | 80.90–81.63 ms |
| Auto greedy | 97.66 ms | 97.57–98.86 ms |
| Auto tree | 97.84 ms | 97.79–98.41 ms |
| Deep enum | 99.25 ms | 99.05–101.38 ms |
| Cache disabled | 252.06 ms | 251.38–252.70 ms |
| Cache miss | 255.03 ms | 252.70–257.63 ms |
| Four contexts | 327.51 ms | 326.50–328.28 ms |
| Four fields | 742.57 ms | 739.86–750.00 ms |
| Long context | 1,224.59 ms | 1,221.76–1,231.35 ms |

The matrix shows the value of schema-prefix caching and the cost of multiplying contexts or fields. It does not justify concurrent admission: the qualified process already uses 9,364 MiB of device memory on a 12 GB card.

## Reproduce the charts

The SVGs are generated from committed JSON with a dependency-free Go tool:

```sh
make benchmark-charts
make benchmark-check
```

`make check` includes `benchmark-check`; CI fails when a chart no longer matches its source data. SVGs have transparent backgrounds and select the documented light/dark palette through `prefers-color-scheme`.

## Collect a fresh run

For the automatic multi-field sweep, use an idle GPU and the pinned artifacts:

```sh
make build
bun scripts/batch-benchmark.ts \
  --binary bin/go-system-one \
  --model /path/to/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  --tokenizer-dir /path/to/tokenizer \
  --revision "$(git rev-parse HEAD)" \
  --mode automatic --sizes 1,10,25,50,100 \
  --out dist/benchmarks/automatic-batch-sweep.json
```

Use `--mode serial` for an explicit serial comparison, and `--sizes 50,100` to collect the missing sizes separately. The collector rejects a busy GPU, verifies artifacts through the server, cools before requests, and aborts at 83°C. It saves each observation; only complete three-sample cells produce medians. Long sweeps need a command timeout that accommodates cooling.

For the single-boolean fixture:


```sh
make benchmark \
  MODEL=/path/to/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  TOKENIZER_DIR=/path/to/tokenizer \
  BENCHMARK_REQUESTS=100
```

The command starts a loopback-only server, verifies all artifact hashes, performs one warm-up by default and writes sorted handler samples to `dist/benchmarks/nvidia-http.json`. Override `BENCHMARK_LISTEN`, `BENCHMARK_WARMUP` or `BENCHMARK_OUT` when needed. It does not modify committed benchmark data or charts.

The full model, driver, numerical, kernel-profile and robustness record is in the [v1 validation report](../validation/go-system-one-v1-20260921.md). The standalone collection command and request hash are in the [standalone NVIDIA report](../validation/standalone-nvidia-f65652f6-20260922.md).
