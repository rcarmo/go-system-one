# AGENTS.md — go-system-one

This repository owns the Go System One decision service, its public contract, playground, pinned evaluations and release packaging.

## Source ownership

- `go-system-one` is an independent repository. Commit and push changes here.
- `go-pherence` owns the shared model, loader, SIMD and NVIDIA runtime while those packages remain external dependencies.
- Never edit, commit, push, reset or clean a neighbouring `go-pherence` checkout while working here.
- Pull product-owned files through `scripts/sync-upstream.sh`. The sync direction is one way: `go-pherence` to `go-system-one`.
- Pin every upstream import to an immutable full commit. Do not track an upstream branch or use an uncommitted checkout as release evidence.
- A local `replace` directive is allowed only for temporary development and must not be committed.

## Repository layout

```text
cmd/go-system-one/       HTTP service entry point
model/gosystemone/       decision contract, validation, scorer adapters and tests
internal/httpinput/      bounded JSON request decoding
webui/                   embedded standalone playground and status endpoint
docs/validation/         model pins, oracle results and benchmark evidence
scripts/                 one-way upstream sync and validation helpers
```

## Required tooling

- Go 1.26.2 or the version declared in `go.mod`.
- `gofmt`, `go test`, `go vet` and `go build` from the same Go toolchain.
- An NVIDIA driver is required only for the opt-in NVIDIA released-model gate. Production PTX loads through the Driver API without CGo or a CUDA toolkit.

## Before editing

1. Read the relevant files and tests.
2. Search all callers before changing a public function, schema field or route.
3. Treat `POST /v1/decision`, `/go-system-one`, artifact hashes, candidate order and validation limits as public contracts.
4. Check `scripts/upstream-files.tsv` when adding, moving or removing a product-owned file that still originates in `go-pherence`.
5. Keep ordinary tests offline and deterministic. Released-model and hardware tests must remain opt-in.

## Correctness rules

- The portable CPU/SIMD scorer is the numerical oracle.
- Preserve deterministic candidate ordering, request validation, cancellation, single admission and returned-output ownership.
- Do not weaken numerical tolerances to admit an optimisation.
- Pin external oracle code by repository and full revision. Record model and tokenizer filenames, byte counts and SHA-256 digests.
- Compare semantic output separately from timings. Never assert wall-clock latency in a correctness test.
- Use synthetic repository-owned fixtures for the default suite. An absent released model may skip only with an exact setup instruction.
- Validate delimiter forgery before tokenisation. User text must not be able to inject model control markers.

## Performance rules

- Optimise measured whole-request bottlenecks. A kernel microbenchmark does not establish service improvement.
- Record hardware, driver, source revision, model hash, warm/cold state, sample count, min/median/p95/max latency and resident memory.
- Keep model weights immutable and reusable scratch session-owned and bounded.
- Do not add speculative concurrency, unbounded caches, CGo, a llama.cpp runtime wrapper or a production CUDA-toolkit dependency.
- Retain only changes that preserve the released-model decision and accepted numerical parity while improving repeated warm requests.

## Upstream synchronisation

Run the sync against a clean upstream commit:

```sh
./scripts/sync-upstream.sh <full-go-pherence-commit>
```

Set `GO_PHERENCE_SOURCE=/path/to/go-pherence` to use an existing clean checkout. The script refuses a dirty source tree. After syncing:

1. Review every changed file.
2. Update `scripts/upstream.env` to the reviewed full commit and UTC commit time.
3. Update the pinned `go-pherence` pseudo-version in `go.mod` when shared runtime changes are required.
4. Run `go mod tidy` and `make check`.
5. Run the opt-in released-model gate on authorised hardware when scorer, tokenizer, model or backend code changed.
6. Commit the source pin, copied changes, dependency pin and validation evidence together.

The script must never write to the upstream checkout or create commits there.

## Validation before commit

Run:

```sh
make check
```

This includes formatting checks, tests, vet and a full build. Also run:

```sh
git diff --check
go test -race ./model/gosystemone ./internal/httpinput ./webui
```

For NVIDIA or released-model changes, set the artifact paths documented in `docs/validation/go-system-one-v1-20260921.md` and run the named opt-in parity test. Record skipped hardware gates as skipped, not passed.

## Git workflow

- Never use `git rebase`; use merge or `git pull --no-rebase`.
- Commit as `Rui Carmo <rui.carmo@gmail.com>`. Configure local and global Git identity before committing.
- Keep commits focused and use `scope: concise change` subjects.
- Do not commit model weights, tokenizers, profiles, generated benchmark databases, secrets or temporary probe code.
- Push only this repository. Verify the remote branch and CI after pushing.
