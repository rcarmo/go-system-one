# Multi-field precision comparison

Packed execution selected the same winners as serial execution for all 80 fields in the frozen 32-context cohort. This held at 128, 256 and 512 packed-token-row budgets. Four fields changed the order of losing candidates. One candidate crossed the diagnostic 95% threshold.

This compares execution paths on synthetic inputs without ground-truth labels. It measures numerical and decision agreement, not classification accuracy. Prompts, context text, schema and option order were unchanged between paths.

## Results

All three packed budgets produced the same aggregate measurements:

| Measurement | Result |
|---|---:|
| Contexts / fields per budget | 32 / 80 |
| Changed field winners | 0 |
| Changed candidate rankings | 4 (losing candidates only) |
| Candidate threshold crossings at 0.5 / 0.9 / 0.95 | 0 / 0 / 1 |
| Maximum absolute probability movement | 0.16542037 (16.542 percentage points) |
| RMS probability movement | 0.02411891 (2.412 percentage points) |
| Maximum / RMS raw-logit difference | 1.84056544 / 0.52296632 |
| Maximum / RMS per-node mean-centred logit difference | 0.75180340 / 0.23446346 |
| Smallest serial top-two probability margin | 0.10754121 (10.754 percentage points) |

The largest probability change was for `urgent` in the context about unusual traffic that could be a successful campaign or a denial-of-service attack. The `true` probability moved from 0.66065959 to 0.82607996; its rank stayed first. The threshold crossing was `customer follow-up`, from 0.94596079 to 0.96159617, for a customer requesting a status update during planned maintenance.

The lower-rank swaps occurred in four enum fields. A rank change alone is not a failure: its impact depends on whether the caller uses only the winner, a ranked list, or a confidence threshold. These probabilities remain uncalibrated. This cohort contained ambiguous wording, but its smallest measured winner margin was still 10.75 percentage points; it does not establish near-tie stability.

## Why scores differ

The old depth-by-depth scorer uses F32 activation projections when fewer than four branches are active. Packing those branches into larger matrix operations uses Q8 activations. Their marginal errors propagate through the transformer and candidate path scores. Fields do not feed sampled answers into a long generated continuation.

A common additive shift within one node's candidate logits cancels in softmax. Mean-centred differences help distinguish that shift from changes in relative scores. The report also retains each field's normalised complete-path log-score error. Winner agreement, losing-rank swaps and confidence changes are separate observations.

The original pinned llama.cpp fixture still passes its unchanged tolerance. That fixture's saturated probabilities do not bound confidence movement on these broader inputs. The new comparison does not change any existing test tolerance or declare a new general acceptance threshold.

## Method and reproduction

The [cohort](../benchmarks/multifield-cohort.json) was committed before execution as `8102ced5f2e6015827a10bb91dcf74b690cf3ae8`; its SHA-256 is `021d1419c38bec3070818c350c0e5bb734afd93fcbd0a885c798e59881842b3e`. It contains two schemas, including multi-token enum choices with shared prefixes. The runtime remained at that revision throughout collection; local changes only added cooled chunks and resume support to the collector. Reports record the source qualifier `8102ced5f2e6015827a10bb91dcf74b690cf3ae8+cooled-chunks`.

Hardware was the NVIDIA RTX 3060 12 GB, driver 580.173.02, with the pinned Gemma artifact. A full serial request hit the 83°C guard and produced no complete report. The revised collector used four-context calls, cooled to at most 55°C before each warm-up and measurement, and retained the 83°C abort. The largest recorded post-call temperature was 68°C; post-call readings are not a peak-temperature measurement.

One measured call per mode/chunk captured every node logit and field result. The runner saved seven complete chunks before a tool timeout; identity-checked resume completed the final chunk. Timings are instrumented descriptive observations, not a latency distribution. Splitting requests reduces simultaneous batch coverage; the separate [throughput measurements](packed-decisions.md#multi-field-experiment) remain the performance evidence.

```sh
go run ./scripts/decisioncompare \
  -model /path/to/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  -tokenizer-dir /path/to/tokenizer \
  -source-revision "$(git rev-parse HEAD)" \
  -contexts-per-call 4 \
  -out dist/benchmarks/multifield-precision.json
# Add -resume to continue only after verifying the same source/cohort/requests.
```

[Complete responses, logits and comparisons](../benchmarks/data/multifield-precision.json) are committed. Ranked-candidate diagnostics can be recomputed without inference:

```sh
go run ./scripts/decisioncompare \
  -analyse docs/benchmarks/data/multifield-precision.json \
  -out dist/benchmarks/multifield-precision-recomputed.json
```
