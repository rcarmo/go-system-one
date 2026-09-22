# Standalone repository validation — 22 September 2026

> Historical validation record: results and implementation descriptions below refer to the recorded revisions. See the [current API documentation](../README.md) and [refreshed benchmarks](../benchmarks/README.md) for the present service.

The first `rcarmo/go-system-one` extraction builds without a neighbouring `go-pherence` checkout and preserves the pinned released-model decision.

## Source and dependency pins

| Item | Value |
|---|---|
| Standalone commit | `9874e5024d1ec855a9f342fc1e7f09d991b0b324` |
| Product source commit | `go-pherence@559d10b4eb558da20552edf442dbd46ccbee78c1` |
| Shared runtime module | `github.com/rcarmo/go-pherence v0.0.0-20260922112537-559d10b4eb55` |
| `go.mod` SHA-256 | `94c62b0998d9240080d767e3a154257d598e213f41dac27967d9929f331d57f8` |
| `go.sum` SHA-256 | `892d25bebffe055519f2d48f567ee4aeb11828acffd4b8ca9df2e4f404884255` |
| `vendor/modules.txt` SHA-256 | `13bbe935255f587e928d8a3d2f4a42376a1e735890e62a27906122ceaf51ab36` |
| Go toolchain used locally | Go 1.26.3 |

The repository vendors 1,147 dependency files occupying 18 MiB in the working tree. The snapshot includes dependency licences and the RISC-V assembly header that `go mod vendor` does not discover automatically.

## Offline and compile gates

The following gates passed from the standalone checkout:

```sh
GOPROXY=off GOSUMDB=off go test -mod=vendor ./...
go test -race ./model/gosystemone ./internal/httpinput ./webui
go vet ./...
go build ./...
GOPROXY=off GOSUMDB=off GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go build -mod=vendor -o /tmp/go-system-one-arm64 ./cmd/go-system-one
GOPROXY=off GOSUMDB=off GOOS=linux GOARCH=riscv64 CGO_ENABLED=0 \
  go build -mod=vendor -o /tmp/go-system-one-riscv64 ./cmd/go-system-one
```

GitHub Actions run [`35731940802`](https://github.com/rcarmo/go-system-one/actions/runs/35731940802) passed the host test, race, vet and build job plus both cross-build jobs. ARM64 and RISC-V results are compile-only.

## Released-model gate

The pinned Gemma 4 12B GGUF and tokenizer from the [v1 validation record](go-system-one-v1-20260921.md) were available locally. This command ran against the vendored runtime:

```sh
GO_SYSTEM_ONE_MODEL=/tmp/qev-gemma4-12b/gemma-4-12b-it-UD-Q4_K_XL.gguf \
GO_SYSTEM_ONE_TOKENIZER_DIR=/tmp/qev-gemma4-12b/tokenizer \
  go test -mod=vendor ./model/gosystemone \
  -run '^TestGoSystemOneNVIDIAReleasedModelMatchesPinnedLlamaCppDecision$' \
  -count=1 -v
```

The test passed on the NVIDIA GeForce RTX 3060 in 20.92 seconds. It reproduced the pinned llama.cpp boolean decision under the fixture's numerical contract. This run validates extraction and dependency pinning; it is not a new latency benchmark.

## Update automation

`scripts/update-upstream.sh` was exercised in a separate clean clone with the current full upstream commit. It copied every manifest-listed file, regenerated the same module pseudo-version and vendor tree, and passed `make check` without source drift.

The weekly updater opens a pull request for review. It does not write to `go-pherence` or commit directly to `main`. Hardware parity remains a manual acceptance gate for upstream model, tokenizer, scorer, PTX and dispatch changes.
