# JevBench v1.4.0 public evidence

All 231 public items were evaluated once on service `b18ee0d4748bac436999aa72c000e06406c3cce6` with JevBench `2fa63fa3226cb369795525ed011800f57dcbd894`. There are 196 correct answers, 231 strict-valid responses and no failures. See the [report](../../jevbench-v140-public.md) for scope and timing limits.

`records.jsonl` contains scored outcomes; `raw-evidence.jsonl` contains exact public requests/responses and original hashes. `summary.json` retains aggregates, source pins and chunk manifests. `thermal-starts.jsonl` and `gpu-telemetry.jsonl` retain cooling observations. `protocol/` contains the exact collection scripts, with original workspace paths. The one interrupted chunk has ten complete durable records and was resumed without repeating them.

```sh
python audit.py --suite /path/to/jevbench-v1.4.0
sha256sum -c SHA256SUMS
```

The audit is offline and verifies the pinned upstream scorer against all outcomes and aggregates. Public task text is covered by `JEVBENCH-LICENSE.txt`. No model artifacts or private benchmark items are included. Official score, rank and cost are null; v1.4 requires sealed measurements unavailable here.
