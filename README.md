# Go System One

Go System One uses Gemma 4 12B to choose between a fixed set of boolean or enum values. We started with a fixed Gemma baseline, built a llama.cpp prototype to define the expected results and speed, then replaced it with a native Go runtime.

The current NVIDIA path loads hand-tuned PTX kernels directly from Go through the NVIDIA Driver API. It needs no CGo, llama.cpp or CUDA toolkit at run time, and reduced the pinned RTX 3060 request from about 521 ms in the first native version to a 77.91 ms median.

[![Go System One playground showing outcome probabilities](docs/images/go-system-one-light-desktop.png)](docs/playground.md)

## Run

Go 1.26.2 or the version in `go.mod` is required. Model files are downloaded separately under the [Gemma licence](docs/artifacts.md).

```sh
make setup
make artifacts-download ACCEPT_GEMMA_LICENSE=1 HF_TOKEN="$HF_TOKEN"
make run BACKEND=nvidia LISTEN=127.0.0.1:8080
```

Open `http://127.0.0.1:8080/go-system-one`.

## Documentation

- [Model and API](docs/jev-like.md)
- [Benchmarks](docs/benchmarks/README.md)
- [Model files and hashes](docs/artifacts.md)
- [Tests and validation](docs/README.md)
- [Source updates](docs/upstream.md)
- [Releases](docs/releases.md)

## Licence

[MIT](LICENSE)
