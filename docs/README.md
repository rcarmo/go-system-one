# Go System One documentation

These documents define the standalone service, its external artifacts, browser playground, source-update process, releases and validation evidence.

## Operate the service

- [External model artifacts](artifacts.md) — exact model/tokenizer pins, download, verification and cleanup.
- [Decision playground](playground.md) — routes, theme contract, desktop/mobile captures and browser-test provenance.
- [Release process](releases.md) — local packages, version tags, archive contents, checksums and publication gates.

## Maintain the source

- [Updating from go-pherence](upstream.md) — one-way manifest updates from the canonical reference repository.

[`go-pherence`](https://github.com/rcarmo/go-pherence) remains the canonical reference for implementation lineage and relevant future changes. This repository keeps an immutable accepted source pin and imports only manifest-listed paths.

## Validation evidence

- [Go System One v1](validation/go-system-one-v1-20260921.md) — model/oracle pins, numerical gates, workload matrix and NVIDIA performance.
- [Standalone repository extraction](validation/standalone-repository-20260922.md) — source/dependency boundaries, offline builds and CI.
- [Standalone accelerated NVIDIA run](validation/standalone-nvidia-f65652f6-20260922.md) — complete 100-request sample.
- [Kev comparison](validation/go-system-one-kev-20260922.md) — architecture/API comparison and adopted delimiter protection.
