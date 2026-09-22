# Standalone NVIDIA validation — 22 September 2026

The standalone repository reproduced the accelerated `go-pherence@f65652f6d9d8aa45c2f5eae68603e995f27b5b75` decision path with a warm median of 81.69 ms over 100 HTTP requests.

## Pins and host

| Item | Value |
|---|---|
| `go-system-one` commit | `a53c9d07351124c14c30e35c51ce15ec6377694e` |
| Shared runtime | `github.com/rcarmo/go-pherence v0.0.0-20260922131709-f65652f6d9d8` |
| Model | `gemma-4-12b-it-UD-Q4_K_XL.gguf`, SHA-256 `90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e` |
| Device | NVIDIA GeForce RTX 3060, 12,288 MiB, compute capability 8.6 |
| Driver | 580.173.02 |
| Reported process GPU memory | 9,364 MiB |

The request SHA-256 was `6e97caea4389295e11a5284174f310f851db2cf5bb5b2299cbd271c4c0af1aa1`. It contained one context, one boolean field, tree mode and prompt caching, matching the v1 performance fixture.

## Command

The service was built entirely from this repository's vendored closure:

```sh
go build -mod=vendor -o /tmp/go-system-one-standalone ./cmd/go-system-one
/tmp/go-system-one-standalone \
  -model /tmp/qev-gemma4-12b/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  -tokenizer-dir /tmp/qev-gemma4-12b/tokenizer \
  -backend nvidia \
  -listen 127.0.0.1:49140
```

One request warmed the service. The next 100 requests ran sequentially through `POST /v1/decision`. The measured interval is the `timings.total_ms` value returned by the handler.

## Results

| Statistic | `total_ms` |
|---|---:|
| Minimum | 80.397981 |
| Median | 81.691360 |
| p95 | 84.484299 |
| p99 | 85.875650 |
| Maximum | 86.011112 |

The complete sorted sample set is checked in as [`data/nvidia-http-100-f65652f6-20260922.json`](data/nvidia-http-100-f65652f6-20260922.json), SHA-256 `00415eebaaff4786a45231b5a0f66eee0f816a97c23ef1a2692749869a4a843c`.

The final response selected `true` with constrained probability `0.9999999999887976`. The pinned released-model parity test passed three consecutive times before the HTTP run. No numerical tolerance changed.

The median is 1.04 ms slower than the source repository's 80.65 ms median and remains 14.31 ms, or 14.9%, faster than the pinned 96.0 ms llama.cpp worker result. Host activity and run order were not independently controlled beyond sequential requests and an otherwise idle GPU; this is a reproduction sample, not a throughput or multi-user benchmark.
