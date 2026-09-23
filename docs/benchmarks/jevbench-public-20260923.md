# JevBench public evaluation

Go System One answered **48/48 easy** and **71/72 original** public JevBench items correctly. The hard diagnostic scored **50/111** with **36 input rejections counted as wrong**. Every rejected input exceeded the NVIDIA runtime's 2,048-token limit. This is a public-subset evaluation, not an official JevBench ranking or full composite score.

## Measured results

| Public tier | Correct / attempted | Accuracy | Valid responses | HTTP p50 | HTTP p95 |
|---|---:|---:|---:|---:|---:|
| Easy | 48/48 | 100.00% | 48/48 | 126.21 ms | 389.60 ms |
| Original (standard) | 71/72 | 98.61% | 72/72 | 131.47 ms | 361.00 ms |
| Hard, diagnostic continuation | 50/111 | 45.05% | 75/111 | 1,044.40 ms | 4,996.16 ms |

The hard tier contains 50 correct answers, 25 wrong answers and 36 HTTP 400 rejections. Among its 75 valid responses, accuracy is 66.67%; that conditional number excludes failures and is not the headline score. Pooled across the public diagnostic, accuracy is 169/231 (73.16%). Pooling weights these tiers by their public item counts and is not JevBench's Intelligence formula.

All 195 successful responses passed JevBench's strict distribution validation without renormalisation. Score-question accuracy uses the modal label, as implemented in the pinned evaluator; expected-level error is reported separately. JevBench breaks exact probability ties lexicographically. Its calibration calculation uses the returned probability vectors, not the adapter's separate `confidence` field.

| Public tier | Calibration responses | Multiclass Brier | Top-label ECE | Ordinal MAE |
|---|---:|---:|---:|---:|
| Easy | 48 | 0.000507 | 0.003051 | not applicable |
| Original | 72 | 0.028068 | 0.015251 | 0.00000506 |
| Hard diagnostic | 75 | 0.585837 | 0.296991 | 1.271519 |

Calibration excludes failed responses rather than inventing distributions for them. All ten public hard gold-distribution items returned valid distributions; mean total-variation distance was **0.398105**. The original tier's 36 paraphrase pairs had 35 both-correct pairs and 35 agreeing pairs. These synthetic public results do not establish general accuracy or calibrated correctness probabilities.

## Hard-run stop and diagnostic continuation

The original hard run stopped after its first three HTTP 400 responses, following JevBench's rule for consecutive infrastructure errors. That run has **3/111 attempts, zero valid answers and 108 unattempted items**. The saved records and manifest are unchanged.

A separately labelled diagnostic then attempted each remaining public hard item once. It kept the same binary, model, request construction and scoring; only HTTP 400 input rejections were allowed to continue rather than trigger the three-error stop. Those responses still count as failed and wrong. Access/rate-limit errors and other infrastructure failures retain the stop rule. No failed item was retried, skipped from the denominator or replaced with a shortened request.

The original stopped run and the diagnostic must not be represented as a standard completed JevBench run. Their separate aggregates are in [summary.json](data/jevbench-public-20260923/summary.json).

## Context limit

All 36 failures returned:

```text
HTTP 400: NVIDIA shared prefill: invalid NVIDIA Gemma4 prefill
```

A post-run, tokenizer-only audit compiled the exact captured requests through the unchanged production schema and tokenizer. Its temporary test source was removed after use; the frozen `.go.txt` copy is included solely as evidence. Rejected requests required **2,149–3,937 visible tokens**; all 195 accepted requests required **105–2,011**. The count includes the shared question prompt, state and longest scored branch. It matched the 2,048-token NVIDIA prefill guard for every request.

The service's general 32,768-token schema limit does not increase the NVIDIA backend's capacity. The backend currently reports a generic HTTP 400, which JevBench interprets as an infrastructure error; it does not use JevBench's recognised HTTP 422 input-refusal classification. Neither capacity nor status handling was changed during this evaluation. [Per-item token counts](data/jevbench-public-20260923/token-lengths.jsonl) retain the diagnosis.

## Comparison on identical public IDs

These peer counts are extracted from JevBench's committed per-task outcomes at the pinned revision. They are not new runs, and their latency/hardware are not compared here.

| System | Easy, 48 | Original, 72 | Hard, 111 |
|---|---:|---:|---:|
| Go System One, this run | 48 | 71 | 50, including 36 rejections |
| Jev 1.13.0 | 48 | 71 | 81 |
| SemIf Qwen3.5-4B | 48 | 71 | 68 |
| Winnow-12B Q8 | 48 | 69 | 81 |
| kev 0.6B preview | 48 | 58 | 48 |

The same easy/original counts as Jev and SemIf do not imply equivalent capability: our hard result is substantially lower, with both capacity failures and wrong answers. The private 109 hard items, 48 other held-out items and 146 imported judge items were unavailable. The public suite covers 231 of 534 decisions. No full JevBench Score, leaderboard rank or Cost axis is assigned; local compute cost is unknown, not zero.

## Pins and timing method

* Service: [`a4e49833c85c96b775f2f2240f129279001f3167`](https://github.com/rcarmo/go-system-one/commit/a4e49833c85c96b775f2f2240f129279001f3167), built with `go build -mod=vendor -trimpath`. Binary SHA-256: `d455d5d3ce1dc269bc730e3c62d28711d532061b1c95866bd57e710e30db117b`.
* JevBench: [`f79a1cab94ab9a5879383b7ef9ee1805b9dc2d84`](https://github.com/fstandhartinger/jevbench/tree/f79a1cab94ab9a5879383b7ef9ee1805b9dc2d84), native `typesafe` adapter, Python 3.13.14. Its tests passed: 97 tests and five subtests.
* Model: pinned Gemma 4 12B IT `UD-Q4_K_XL` GGUF, SHA-256 `90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e`. [Tokenizer pins and licences](../artifacts.md) are unchanged. RTX 3060 12 GB, driver 580.173.02, default automatic 512-row packing with normal per-entry serial fallback.
* Transport: loopback `POST /v1/systemone`, one question and one request at a time. No auth token, warm-up request, retries, truncation, task-order changes, prompt tuning or model changes. JevBench's request wall time includes HTTP and response processing; loading and cooling are excluded.

Each request waited for a verified start temperature of at most 55°C. A separate 500 ms monitor would abort at 83°C; observed temperature peaked at 74°C and sampled device memory at 10,955 MiB. Cooling totalled about 62.5 minutes, excluded from latency. Sampling can miss brief memory peaks.

Collection used bounded invocations and fresh run directories. The initial easy invocation stopped after 46 durable responses; the remaining two were collected without repeating them. The fleet restart interrupted hard diagnostic chunk 6 after 15 durable responses; the final 13 unattempted hard items were collected after restart. No raw response was left without its matching result. First requests after each server load remain in the latency statistics; the summary also supplies first-request records and percentiles excluding them. No production-load adjustment such as JevBench's assumed ×2 + 0.15 s was applied.

## Evidence and reproduction

[The evidence directory](data/jevbench-public-20260923/) contains per-item records, exact request/response evidence, token counts, thermal starts, sampled GPU telemetry, chunk manifests inside the summary and the collection/aggregation source. `audit.py --suite /path/to/pinned/jevbench` verifies every raw hash, exact request, score and tier aggregate offline. The original runner is unchanged in the pinned JevBench checkout. The local wrapper adds cooling, durable chunk resumption and the explicitly labelled diagnostic stop-rule exception. Its exact scripts retain the original workspace paths for auditing.

For a normal run on an already-running local service, use upstream's CLI with a fresh output directory outside its checkout:

```sh
python -m jevbench.cli run \
  --tasks datasets/public/easy.jsonl \
  --adapter typesafe --endpoint http://127.0.0.1:18087 \
  --model gemma-4-12b-it-go-system-one --key-env '' \
  --cost-basis local_gpu_compute_cost_unmeasured --reserve-usd 0 \
  --results /path/to/new-run/results.jsonl \
  --raw-dir /path/to/new-run/raw --ledger /path/to/new-run/ledger.jsonl \
  --manifest /path/to/new-run/manifest.json
```

Repeat with the original or hard task file and new output paths. This plain CLI command does not add the cooling hook or diagnostic continuation; the saved protocol scripts show those differences. The hard run is expected to stop on this binary's long-input responses. Public task text is MIT-licensed by JevBench; its notice is included with the evidence. No model files or private benchmark items are included.
