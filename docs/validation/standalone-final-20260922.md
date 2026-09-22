# Standalone final-state validation — 22 September 2026

> Historical validation record: results and implementation descriptions below refer to the recorded revisions. See the [current API documentation](../README.md) and [refreshed benchmarks](../benchmarks/README.md) for the present service.

Go System One now builds entirely from first-party source in `rcarmo/go-system-one`; `go-pherence` remains the canonical reference and one-way source origin, not a Go module dependency.

## Source and dependency boundary

| Item | Value |
|---|---|
| Accepted canonical source | `go-pherence@788f22402d928004a597b1446568f0e0595dfb1c` |
| Standalone implementation evidence | `e3840373e08337552cbf6793b78b276623d6923a` |
| Release-workflow evidence | `ae5701c95245e9f8a858a35a01457e82764932d4` |
| `go.mod` SHA-256 | `b8a8a960f27a8c1f7af938333163b3eddbf57fbe4dc2656878a6a0ecad227270` |
| `go.sum` SHA-256 | `763717617f7382964077b1880d40b38fe68436102440a3d400f6aaa8820a78e9` |
| `vendor/modules.txt` SHA-256 | `fb933c7d1ef4a43fa2549737855e509e59372722f9bc07eff7de6589b0505f60` |
| Source-file manifest SHA-256 | `1a481ea456fc480688fc6d9bd0931b7447872aa5435cc9626e10a5a1d3dbddd5` |
| Package mapping SHA-256 | `2c149a41b0de65b781f30daa8036e9781a624dc47f42fd0d2da81f973fd0a2bb` |

`go.mod` contains four third-party dependencies: PureGo, `x/sys`, `x/text` and YAML v3. No Go import, module requirement, checksum entry or vendored package points at `github.com/rcarmo/go-pherence`. `vendor/` contains 532 third-party files occupying 12 MiB.

No GGUF file exists in Git. Repository and release checks reject GGUF-like names, checkpoint directories and GGUF file magic. The exact external model/tokenizer contract is in [`../artifacts.md`](../artifacts.md).

## Local gates

The following passed after full source extraction:

```sh
make check
CGO_ENABLED=1 go test ./... -count=1
CGO_ENABLED=0 go test ./... -count=1
make race
make cross-build
```

The ARM64 and RISC-V results are compile-only. CGo and pure-Go host suites both passed. Vendoring was regenerated twice with the same aggregate file hash.

Both pinned released-model gates passed through `make hardware-check` on the RTX 3060, including artifact hashes, the original boolean fixture and the two-context/two-field multi-token fixture.

## Lifecycle and release gates

The Make lifecycle covers prerequisites, setup, vendoring, build, install/uninstall, serving, test/race/coverage, repository checks, cross-builds, external artifacts, hardware parity, source updates, release packaging and guarded cleanup.

Local release tests built Linux AMD64, ARM64 and RISC-V archives, generated `SHA256SUMS`, verified every checksum and scanned extracted files for forbidden model paths and GGUF magic.

GitHub Actions release dry run [`35746532506`](https://github.com/rcarmo/go-system-one/actions/runs/35746532506) built and audited all three code-only archives. The publication step was correctly skipped because the run used manual dispatch instead of a `v*` tag.

## Update automation

A clean-clone update to the accepted canonical pin produced no diff after sync, import rewriting, `go mod tidy`, third-party vendoring and `make check`.

The relevance checker compares Git blob IDs only for the paths in `scripts/upstream-files.tsv`. Live workflow run [`35746258604`](https://github.com/rcarmo/go-system-one/actions/runs/35746258604) resolved a newer canonical commit containing unrelated work, found no manifest-listed change, and skipped both import and pull-request creation.

The command README change in canonical commit `788f2240` was added to the manifest and imported. Qwen3-TTS paths were outside the manifest and did not enter this repository.

## UI evidence

The refined playground source came from `go-pherence@8629232b14440f4a9aa06cfb6d6003c1302c8cb9`. The committed screenshots are:

- light desktop, 1440 × 1100, SHA-256 `20efa66f80f963fefa40175b1756dac9caa03fb6d134a0da8a3c5d1c3a736631`;
- dark mobile, 390 × 1588 output, SHA-256 `6aeb50d478ad3b683adf0bd044e177364c7f68ab054ccb47044b1b1413735317`.

The capture and OS-theme contract is documented in [`../playground.md`](../playground.md).
