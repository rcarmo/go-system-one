# JevBench public evidence

This directory records 231 unique public task attempts against Go System One at `a4e49833c85c96b775f2f2240f129279001f3167`, using JevBench `f79a1cab94ab9a5879383b7ef9ee1805b9dc2d84`. Read the [evaluation report](../../jevbench-public-20260923.md) before comparing the scores.

* `records.jsonl`: upstream per-item output plus tier, question type and source chunk. Failed HTTP responses remain failed/wrong.
* `raw-evidence.jsonl`: exact adapter request and response, HTTP status, original raw hash and task/chunk references. All task text is public and MIT-licensed; see `JEVBENCH-LICENSE.txt`.
* `summary.json`: upstream aggregates, separate stopped hard run, diagnostic continuation, comparison counts, source pins and original chunk manifests. Missing completion fields on the easy and hard-diag6 manifests reflect interruptions; their saved results were not overwritten.
* `token-lengths.jsonl`: post-run tokenizer-only diagnosis, with the exact compiled shared/state/branch lengths. It does not feed predictions or change outcomes.
* `thermal-starts.jsonl` and `gpu-telemetry.jsonl`: per-request cooldown observations and sampled device data.
* `protocol/`: exact collection and aggregation sources used in this workspace, including their absolute paths. The Go length-audit probe is stored as `.go.txt` so default tests do not compile or run it. No probe entered the server build.
* `audit.py`: portable, offline re-scoring and provenance check against a local checkout of the pinned JevBench revision.

```sh
python audit.py --suite /path/to/jevbench
sha256sum -c SHA256SUMS
```

The standard hard run stopped after three HTTP 400 responses. Continuing after input refusals was a separate diagnostic, not an upstream benchmark rule change. Every remaining item was attempted once without retry, truncation or model changes. Official JevBench Score, rank and local cost remain null.
