# Go System One

Go System One is a native Go service for finite boolean and enum decisions using the pinned Gemma 4 12B instruction checkpoint. It provides `POST /v1/decision` and an embedded browser playground at `/go-system-one`.

The service runs through a portable CPU/SIMD correctness path or NVIDIA Driver API/PTX. It does not use CGo, a llama.cpp runtime wrapper or a production CUDA toolkit.

## Current source boundary

This repository owns the decision contract, request validation, scorer adapters, HTTP service, web interface, frozen oracle fixtures and evaluation records. The first standalone revision pins the shared model, tokenizer, SIMD and NVIDIA runtime to [`go-pherence@559d10b4eb558da20552edf442dbd46ccbee78c1`](https://github.com/rcarmo/go-pherence/commit/559d10b4eb558da20552edf442dbd46ccbee78c1).

The shared runtime is an ordinary immutable Go module dependency. Its complete build closure is checked into `vendor/`, including licences, so a clone can build offline without a sibling checkout. [`scripts/sync-upstream.sh`](scripts/sync-upstream.sh) copies the declared product-owned files from a reviewed `go-pherence` commit without writing to that repository.

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

Open `http://127.0.0.1:8080/go-system-one`. The decision endpoint is `POST /v1/decision`.

The command verifies the frozen model, tokenizer, tokenizer configuration and chat-template SHA-256 values before loading them. Use `-verify-artifacts=false` only for development fixtures. `-backend simd` selects the correctness-oracle implementation; the 12B SIMD path is too slow for interactive use.

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

This command copies the manifest-listed product files, updates the immutable module pin, regenerates `vendor/` and runs the offline checks. [`scripts/sync-upstream.sh`](scripts/sync-upstream.sh) is the lower-level copy-only command and accepts `GO_PHERENCE_SOURCE=/path/to/go-pherence` for a clean local checkout. A weekly GitHub workflow runs the full updater and opens a review pull request when the upstream pin changes. See [`docs/upstream.md`](docs/upstream.md) for the acceptance procedure.

## Licence

Go System One is available under the [MIT licence](LICENSE).
