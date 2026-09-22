# Go System One

Go System One runs Gemma 4 12B as a finite-choice model. It scores boolean and string-enum values and returns the selected value plus the probability of every scored outcome.

[![Go System One playground showing outcome probabilities](docs/images/go-system-one-light-desktop.png)](docs/playground.md)

## How it was built

1. **Gemma baseline.** We fixed the Gemma 4 12B model, tokenizer, chat template and quantised GGUF file.
2. **llama.cpp prototype.** We used the same model to define the prompt, candidate paths, expected decisions and reference probabilities. The pinned request took 96.0 ms.
3. **Native Go runtime.** We replaced llama.cpp with Go code that loads GGUF files, reuses the shared prompt prefix and scores only paths allowed by the schema.
4. **Direct GPU execution.** Go loads embedded, hand-tuned PTX through the NVIDIA Driver API. The runtime uses no CGo, llama.cpp or CUDA toolkit.

The GPU work keeps weights and KV data on the device, packs independent candidate branches and projects only the vocabulary rows needed for each choice.

The first native NVIDIA version took about 521 ms on the pinned RTX 3060 request. Initial tuning reduced it to about 136 ms. The current Go/PTX runtime has an 81.13 ms median.

[![Latency from the llama.cpp prototype to hand-tuned Go/PTX](docs/benchmarks/latency-comparison.svg)](docs/benchmarks/README.md)

| Runtime | Warm latency |
|---|---:|
| llama.cpp prototype | 96.0 ms |
| Early native Go | 520.8–521.9 ms |
| Native Go after initial tuning | 135.3–136.7 ms |
| Hand-tuned Go/PTX | 81.13 ms median; 81.59 ms p95; 82.15 ms p99 |

These figures use one model, request and host. See the [benchmark data and method](docs/benchmarks/README.md).

## Model contract

`POST /v1/decision` accepts context plus boolean or unordered string-enum fields. Output is restricted to the values in the schema.

*Jev-like* refers to this finite-choice behaviour. The model uses Gemma weights; it has no Jev weights, Jev training or Jev pointer head. Returned probabilities are not calibrated estimates of real-world accuracy. See the [model contract](docs/jev-like.md).

## Build and run

Go 1.26.2 or the version in `go.mod` is required.

```sh
make prerequisites
make setup
make artifacts-download ACCEPT_GEMMA_LICENSE=1 HF_TOKEN="$HF_TOKEN"
make artifacts-verify
make run BACKEND=nvidia LISTEN=127.0.0.1:8080
```

Open `http://127.0.0.1:8080/go-system-one`. Model files remain outside the repository and release archives. Their revisions and hashes are in [`docs/artifacts.md`](docs/artifacts.md).

The server has no authentication or TLS. Bind it to loopback or use an authenticated reverse proxy.

## Test

```sh
make check
make browser-test
make cross-build
make hardware-check  # requires the pinned model and NVIDIA hardware
```

The repository builds without `go-pherence`. Reviewed source updates use the one-way [import process](docs/upstream.md). Validation and release records are listed in the [documentation index](docs/README.md).

## Licence

[MIT](LICENSE)
