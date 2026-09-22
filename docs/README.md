# Go System One documentation

Go System One serves boolean/enum batches through `/v1/decision` and TypeSafe-style Noul, Choice and Score questions through `/v1/systemone`. Both routes use the pinned Gemma runtime and one admission gate.

## Operate the service

- [Jev-like decision model](jev-like.md) — finite-choice behaviour, Gemma world knowledge and compatibility limits.
- [TypeSafe question types](systemone-api.md) — `noul`, `choice` and `score` through `/v1/systemone`.
- [External model artifacts](artifacts.md) — exact model/tokenizer pins, download, verification and cleanup.
- [Decision playground](playground.md) — TypeSafe questions and batch decisions, OS themes, desktop/mobile captures and browser tests.
- [Benchmarks](benchmarks/README.md) — committed latency samples, generated charts, workload matrix and fresh-run commands.
- [Packed Gemma scoring](performance/packed-decisions.md) — implementation, precision trade-offs and development measurements.
- [Q5 staging](performance/q5-staged.md) and [512-column chunks](performance/q5-chunk512.md) — accepted kernel changes, measured gains and rejected variants.
- [Release process](releases.md) — local packages, version tags, archive contents, checksums and publication gates.

## Maintain the source

- [Updating from go-pherence](upstream.md) — one-way manifest updates from the canonical reference repository.

[`go-pherence`](https://github.com/rcarmo/go-pherence) remains the canonical reference for implementation lineage and relevant future changes. This repository keeps an immutable accepted source pin and imports only manifest-listed paths.

## Validation evidence

The dated reports below preserve their tested revisions and results. Current API behaviour is documented above; the [benchmark page](benchmarks/README.md) identifies the source and binary for every refreshed dataset.

- [Multi-field precision comparison](performance/multifield-precision.md) — winner agreement, lower-rank shifts and probability movement for serial and packed execution.
- [Go System One v1](validation/go-system-one-v1-20260921.md) — model/oracle pins, numerical gates, workload matrix and NVIDIA performance.
- [Standalone repository extraction](validation/standalone-repository-20260922.md) — source/dependency boundaries, offline builds and CI.
- [Standalone accelerated NVIDIA run](validation/standalone-nvidia-f65652f6-20260922.md) — complete 100-request sample.
- [Standalone final state](validation/standalone-final-20260922.md) — fully local source, lifecycle, release and relevance-aware update evidence.
- [Kev comparison](validation/go-system-one-kev-20260922.md) — architecture/API comparison and adopted delimiter protection.
