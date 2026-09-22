# Go System One

Go System One is a native Go service for finite boolean and enum decisions using the pinned Gemma 4 12B instruction checkpoint. It provides `POST /v1/decision` and an embedded browser playground at `/go-system-one`.

[![Go System One decision playground](docs/images/go-system-one-light-desktop.png)](docs/playground.md)

The service runs through a portable CPU/SIMD correctness path or NVIDIA Driver API/PTX. It does not use CGo, a llama.cpp runtime wrapper or a production CUDA toolkit.

## Source boundary

[`go-pherence`](https://github.com/rcarmo/go-pherence) is the canonical reference for Go System One implementation lineage, shared-kernel evolution and relevant future changes. This repository is the independently buildable and releasable projection of that work: it owns the decision contract, HTTP service, playground, model orchestration, loaders, tensor/runtime packages, portable SIMD, NVIDIA Driver API runtime, embedded PTX and platform adapters.

The standalone build has no compile-time dependency on `go-pherence`; `vendor/` contains third-party modules and their licences only. [`go-pherence@788f22402d928004a597b1446568f0e0595dfb1c`](https://github.com/rcarmo/go-pherence/commit/788f22402d928004a597b1446568f0e0595dfb1c) is the current immutable source provenance. [`scripts/sync-upstream.sh`](scripts/sync-upstream.sh) copies manifest-listed files from a reviewed upstream commit in one direction and never writes to that repository.

## Setup and build

Go 1.26.2 or the version declared in `go.mod` is required.

```sh
make prerequisites
make setup
```

The model and tokenizer remain external. Their exact repositories, revisions, filenames, byte count and SHA-256 pins are listed in [`docs/artifacts.md`](docs/artifacts.md). The [documentation index](docs/README.md) links operational, update, release and validation records.

```sh
make artifacts-info
make artifacts-download ACCEPT_GEMMA_LICENSE=1 HF_TOKEN="$HF_TOKEN"
make artifacts-verify
```

## Run

```sh
make run BACKEND=nvidia LISTEN=127.0.0.1:8080
```

Use `MODEL=/path/to/model.gguf TOKENIZER_DIR=/path/to/tokenizer` with `make run`, `make artifacts-verify` or `make hardware-check` to use an existing external store.

Open `http://127.0.0.1:8080/go-system-one`. The decision endpoint is `POST /v1/decision`. Desktop and mobile captures, theme behaviour and screenshot provenance are documented in [`docs/playground.md`](docs/playground.md).

The command verifies the frozen model, tokenizer, tokenizer configuration and chat-template SHA-256 values before loading them. Model weights and GGUF files are never shipped in this repository or release packages. Repository checks reject tracked GGUF filenames and GGUF file magic. Use `-verify-artifacts=false` only for development fixtures. `BACKEND=simd make run` selects the correctness-oracle implementation; the 12B SIMD path is too slow for interactive use.

The server has no authentication or TLS. Bind it to loopback or put an authenticated reverse proxy in front of every route.

## Test

```sh
make test
make race
make check
make cross-build
make hardware-check  # requires verified artifacts and NVIDIA hardware
```

The default suite is offline and uses synthetic or frozen repository fixtures. Released-model NVIDIA parity is opt-in because it requires the pinned checkpoint, tokenizer and suitable hardware. See the [v1 model validation](docs/validation/go-system-one-v1-20260921.md) for artifact pins and oracle data, the [standalone repository validation](docs/validation/standalone-repository-20260922.md) for extraction and CI evidence, and the [accelerated standalone NVIDIA run](docs/validation/standalone-nvidia-f65652f6-20260922.md) for the complete 100-request sample.

## Update from go-pherence

```sh
./scripts/update-upstream.sh <full-go-pherence-commit>
```

This command copies manifest-listed source files, records the immutable source commit, regenerates third-party `vendor/` and runs the offline checks. [`scripts/sync-upstream.sh`](scripts/sync-upstream.sh) is the lower-level copy-only command and accepts `GO_PHERENCE_SOURCE=/path/to/go-pherence` for a clean local checkout. The weekly workflow compares only manifest-listed upstream blobs, so unrelated canonical work does not advance the pin or open a pull request. See [`docs/upstream.md`](docs/upstream.md) for the acceptance procedure.

## Project lifecycle

Run `make help` for the complete target list. The Makefile covers prerequisites, setup, vendoring, build, install/uninstall, serving, tests, race and coverage runs, policy checks, cross-builds, artifact management, hardware parity, upstream source updates, release packaging and cleanup.

`make package` writes code-only Linux archives under `dist/`. It never includes model or tokenizer artifacts. Tag-driven publication and build-only dry runs are documented in [`docs/releases.md`](docs/releases.md).

## Licence

Go System One is available under the [MIT licence](LICENSE).
