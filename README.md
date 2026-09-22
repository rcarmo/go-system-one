# Go System One

Go System One is a native Go service for finite boolean and enum decisions using the pinned Gemma 4 12B instruction checkpoint. It provides `POST /v1/decision` and an embedded browser playground at `/go-system-one`.

The service runs through a portable CPU/SIMD correctness path or NVIDIA Driver API/PTX. It does not use CGo, a llama.cpp runtime wrapper or a production CUDA toolkit.

## Source boundary

This repository owns the complete Go System One build: decision contract, HTTP service, playground, model orchestration, loaders, tensor/runtime packages, portable SIMD, NVIDIA Driver API runtime, embedded PTX and platform adapters. It has no compile-time dependency on `go-pherence`; `vendor/` contains third-party modules and their licences only.

[`go-pherence@9ada94d175cbb82dc78e27f2aa8df5b129e30676`](https://github.com/rcarmo/go-pherence/commit/9ada94d175cbb82dc78e27f2aa8df5b129e30676) is the current immutable source provenance. [`scripts/sync-upstream.sh`](scripts/sync-upstream.sh) copies manifest-listed files from a reviewed upstream commit in one direction and never writes to that repository.

## Build

Go 1.26.2 or the version declared in `go.mod` is required.

```sh
go build ./cmd/go-system-one
```

## Run

```sh
go run ./cmd/go-system-one \
  -model /path/to/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  -tokenizer-dir /path/to/gemma-4-12b-it-tokenizer \
  -backend nvidia \
  -listen 127.0.0.1:8080
```

Open `http://127.0.0.1:8080/go-system-one`. The decision endpoint is `POST /v1/decision`. Desktop and mobile captures, theme behaviour and screenshot provenance are documented in [`docs/playground.md`](docs/playground.md).

The command verifies the frozen model, tokenizer, tokenizer configuration and chat-template SHA-256 values before loading them. Model weights and GGUF files are never shipped in this repository; provide them through external paths. Repository checks reject tracked GGUF filenames and GGUF file magic. Use `-verify-artifacts=false` only for development fixtures. `-backend simd` selects the correctness-oracle implementation; the 12B SIMD path is too slow for interactive use.

The server has no authentication or TLS. Bind it to loopback or put an authenticated reverse proxy in front of every route.

## Test

```sh
make check
go test -race ./model/gosystemone ./internal/httpinput ./webui
```

The default suite is offline and uses synthetic or frozen repository fixtures. Released-model NVIDIA parity is opt-in because it requires the pinned checkpoint, tokenizer and suitable hardware. See the [v1 model validation](docs/validation/go-system-one-v1-20260921.md) for artifact pins and oracle data, the [standalone repository validation](docs/validation/standalone-repository-20260922.md) for extraction and CI evidence, and the [accelerated standalone NVIDIA run](docs/validation/standalone-nvidia-f65652f6-20260922.md) for the complete 100-request sample.

## Update from go-pherence

```sh
./scripts/update-upstream.sh <full-go-pherence-commit>
```

This command copies the manifest-listed source files, records the immutable source commit, regenerates third-party `vendor/` and runs the offline checks. [`scripts/sync-upstream.sh`](scripts/sync-upstream.sh) is the lower-level copy-only command and accepts `GO_PHERENCE_SOURCE=/path/to/go-pherence` for a clean local checkout. A weekly GitHub workflow runs the full updater and opens a review pull request when the source pin changes. See [`docs/upstream.md`](docs/upstream.md) for the acceptance procedure.

## Licence

Go System One is available under the [MIT licence](LICENSE).
