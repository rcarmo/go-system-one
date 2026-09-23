# JevBench v1.4.0 public results

Go System One scored **196/231 correct (84.85%)** on all public JevBench items using one fixed service revision. All 231 requests returned HTTP 200 and passed strict distribution validation. This is public-subset accuracy, **not the official v1.4 composite score or rank**, which requires sealed-set measurements and a cost basis.

| Public tier | Correct | Accuracy | HTTP p50 | HTTP p95 |
|---|---:|---:|---:|---:|
| Easy | 48/48 | 100.00% | 118.69 ms | 388.41 ms |
| Original / standard | 71/72 | 98.61% | 135.77 ms | 364.51 ms |
| Hard | 77/111 | 69.37% | 1,611.83 ms | 18,938.84 ms |
| All public items | 196/231 | 84.85% | 373.93 ms | 15,750.58 ms |

These results replace the earlier mixed-revision provisional tally. Every item was evaluated again on `b18ee0d`; no prior answer or timing enters this result. The new predictions and probabilities happen to match the earlier original-plus-recheck records exactly. The [old run](jevbench-public-20260923.md) and [failure-subset recheck](../performance/long-context.md) remain separate historical evidence.

## Method and limits

The [v1.4.0 tag](https://github.com/fstandhartinger/jevbench/tree/2fa63fa3226cb369795525ed011800f57dcbd894) retains the same public tasks, native TypeSafe adapter and per-item scoring as our previous evaluation. We used those unchanged components against loopback `/v1/systemone`, one question and request at a time. There was no warm-up request, retry, truncation, prompt tuning, task reordering or diagnostic stop-rule exception. The standard 401/403/429 and three-consecutive-infrastructure-error stop rules were retained across chunks; none was triggered.

The service used [revision `b18ee0d`](https://github.com/rcarmo/go-system-one/commit/b18ee0d4748bac436999aa72c000e06406c3cce6), the pinned Gemma 4 12B IT `UD-Q4_K_XL` model, default automatic packing and its ordinary serial fallback. A single RTX 3060 12 GB ran NVIDIA driver 580.173.02. The runtime now uses bounded long-attention scratch and compacted sliding-window KV; all formerly over-limit public requests completed without changing their content.

Each request started at no more than 55°C. A separate 500 ms monitor would abort at 83°C; observed temperature peaked at 81°C and sampled device memory at 10,905 MiB. About 63.6 minutes of cooling waits were excluded from HTTP latency. Loading was also excluded. First requests after each server start are included in the table; separate first-request records and percentiles excluding them are in the raw summary. These are serial local measurements, not production-load guarantees or sustained-throughput measurements.

Collection used bounded invocations. One invocation stopped after ten durable hard-item records; all matching raw responses were intact and only unattempted items were resumed. A busy-GPU preflight also declined one launch before any request. There are exactly 231 distinct IDs, one response per ID and one binary hash across all recorded chunks.

## Calibration and errors

| Tier | Multiclass Brier | Top-label ECE | Expected-level MAE |
|---|---:|---:|---:|
| Easy | 0.000507 | 0.003051 | not applicable |
| Original | 0.028068 | 0.015251 | 0.00000506 |
| Hard | 0.537947 | 0.261488 | 0.988826 |

All distributions were strict-valid without renormalisation. JevBench calculates accuracy from the modal label, including score questions; expected-level error is separate. Calibration uses native candidate probabilities, not the API's separately reported confidence. Mean total-variation distance on the ten public hard gold-distribution items was 0.398105. These are public measurements, not the sealed-inclusive v1.4 Calibration axis.

Hard-item correct counts were: long policy 14/19, multi-hop 14/18, temporal/numeric 3/15, trade-off 2/6, difficult judging 12/17, probability 8/10, ambiguous 6/7, adversarial 6/6, hard routing 5/5 and traps 7/8. Removing input failures did not remove reasoning and calibration errors.

## Official record

[v1.4's method](https://github.com/fstandhartinger/jevbench/blob/2fa63fa3226cb369795525ed011800f57dcbd894/docs/METHOD-v1.4.md) adds 308 sealed decisions, blends their measurements into Intelligence and Calibration, uses a harmonic composite and applies public-to-sealed gap and Speed/Cost penalties. The earlier 303 private/imported items are also unavailable locally. Neither hidden set is inferred from the public result. Cost, official score and rank remain null.

We requested the formal evaluation process in [JevBench issue #57](https://github.com/fstandhartinger/jevbench/issues/57), explicitly identifying the posting AI agent and @rcarmo. The maintainer has acknowledged the request for review.

## Reproduction and evidence

[Evidence directory](data/jevbench-v140-public-20260923/): exact public requests/responses, raw hashes, returned probabilities, timings, thermal observations, original chunk manifests, protocol scripts and an offline audit. Public task text is MIT-licensed by JevBench; its notice is included. No weights or private items are stored here.

* JevBench revision: `2fa63fa3226cb369795525ed011800f57dcbd894` (`v1.4.0`); 104 tests and five subtests passed.
* Service revision: `b18ee0d4748bac436999aa72c000e06406c3cce6`.
* Binary SHA-256: `83148373e59397aaf036ba561a26cbf39d56e3475073c7da8422e59fc722325d`.
* Model SHA-256: `90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e`; [artifact pins](../artifacts.md).
* Python 3.13.14; first recorded response 2026-09-23 18:40 UTC, final response 20:05 UTC.

Run the offline audit from the evidence directory:

```sh
python audit.py --suite /path/to/jevbench-v1.4.0
sha256sum -c SHA256SUMS
```

The saved collection scripts retain their original local paths. They call upstream `Runner.run_task`, add untimed cooling, save each result durably and carry the normal error counter across resumable chunks. To repeat the plain upstream protocol against a running server, use a fresh output directory outside the JevBench checkout:

```sh
python -m jevbench.cli run \
  --tasks datasets/public/easy.jsonl,datasets/public/original.jsonl,datasets/public/hard.jsonl \
  --adapter typesafe --endpoint http://127.0.0.1:18087 \
  --model gemma-4-12b-it-go-system-one --key-env '' \
  --cost-basis local_gpu_compute_cost_unmeasured --reserve-usd 0 \
  --results /path/to/new-run/results.jsonl --raw-dir /path/to/new-run/raw \
  --ledger /path/to/new-run/ledger.jsonl --manifest /path/to/new-run/manifest.json
```

That command does not include our cooling hook. Do not mix latency from a continuous run with the cooled measurements above without labelling the difference.
