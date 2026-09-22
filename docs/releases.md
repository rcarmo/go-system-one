# Releasing Go System One

Go System One releases contain source documentation and a statically built service binary. They never contain GGUF models, tokenizer artifacts, checkpoints, profiles or secrets.

## Local release build

```sh
make package VERSION=v1.0.0
```

The target runs `make check`, then builds Linux archives for:

- `amd64`;
- `arm64`;
- `riscv64`.

Each archive contains:

```text
go-system-one-VERSION-linux-ARCH/
├── go-system-one
├── LICENSE
├── README.md
└── docs/
```

`dist/SHA256SUMS` records archive checksums. `scripts/check-release-archives.sh` rejects model/checkpoint paths and scans extracted files for GGUF magic.

Verify a local build with:

```sh
(cd dist && sha256sum -c SHA256SUMS)
./scripts/check-release-archives.sh dist
```

## GitHub release

Push a signed or annotated `v*` tag after `main` CI and the required released-model hardware gate pass:

```sh
git tag -s v1.0.0 -m 'Go System One v1.0.0'
git push origin v1.0.0
```

[`.github/workflows/release.yml`](../.github/workflows/release.yml) builds and audits all three archives on Ubuntu 24.04, then creates the GitHub release with generated notes, archives and `SHA256SUMS`. It does not download or access model artifacts.

The workflow supports a build-only manual dry run:

```sh
gh workflow run release.yml -f version=snapshot
```

Manual runs execute the same archive and checksum checks but skip publication because no `v*` tag triggered them.

## Acceptance checklist

Before tagging:

1. `main` is clean and synchronized with `origin/main`.
2. `make check`, `make race` and `make cross-build` pass.
3. `make hardware-check MODEL=... TOKENIZER_DIR=...` passes on authorized NVIDIA hardware.
4. Performance claims have a committed validation record and bounded raw samples.
5. `make package VERSION=...` and checksum/archive checks pass.
6. The accepted canonical [`go-pherence`](https://github.com/rcarmo/go-pherence) source pin in `scripts/upstream.env` matches the relevant implementation/evaluation revision.
