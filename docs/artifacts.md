# External model artifacts

Go System One requires one GGUF model and three tokenizer-sidecar files. They are not distributed in this repository or its release archives.

## Frozen v1 artifacts

| Artifact | Repository and revision | Bytes | SHA-256 |
|---|---|---:|---|
| `gemma-4-12b-it-UD-Q4_K_XL.gguf` | `unsloth/gemma-4-12b-it-GGUF@fc034cfff751157913579611efad8462ac1be606` | 7,366,423,360 | `90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e` |
| `tokenizer.json` | `google/gemma-4-12b-it@707f0a3b8a3c7ad586ed01e27eafbad8a27dd0f7` | — | `cc8d3a0ce36466ccc1278bf987df5f71db1719b9ca6b4118264f45cb627bfe0f` |
| `tokenizer_config.json` | same revision | — | `a62f4e85a47c0c136edaaa3a4f591fd6783717299a9def47e5ad03a49f6a5eb9` |
| `chat_template.jinja` | same revision | — | `ae53464bf3be25802b3a5b37def7fd89667067d7577049b3b2d74c4d8de4c6d4` |

The model and tokenizer licences and access conditions are set by their Hugging Face repositories. Accept those terms before downloading. `make artifacts-download` requires `ACCEPT_GEMMA_LICENSE=1` to make that acknowledgement explicit. Set `HF_TOKEN` or `HUGGINGFACE_TOKEN` when the repositories require authentication.

## Default location

The helper uses:

```text
${XDG_CACHE_HOME:-$HOME/.cache}/go-system-one/v1/
├── model/gemma-4-12b-it-UD-Q4_K_XL.gguf
└── tokenizer/
    ├── tokenizer.json
    ├── tokenizer_config.json
    └── chat_template.jinja
```

Override the root with `ARTIFACT_DIR` in Make commands or `GO_SYSTEM_ONE_ARTIFACT_DIR` when calling the helper directly. Existing stores can set `MODEL` and `TOKENIZER_DIR` without copying files.

## Download and verify

```sh
make artifacts-info
make artifacts-download ACCEPT_GEMMA_LICENSE=1 HF_TOKEN="$HF_TOKEN"
make artifacts-verify
```

Downloads use exact revision URLs and temporary `.part` files. Interrupted downloads remain resumable. Every completed download is checked for the pinned byte count where defined and SHA-256 before use.

To use an existing store:

```sh
make artifacts-verify \
  MODEL=/srv/models/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  TOKENIZER_DIR=/srv/models/gemma-4-12b-it-tokenizer
```

The server independently repeats these checks by default before parsing either artifact.

## Hardware validation and serving

```sh
make hardware-check MODEL=/srv/models/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  TOKENIZER_DIR=/srv/models/gemma-4-12b-it-tokenizer

make run BACKEND=nvidia LISTEN=127.0.0.1:8080 \
  MODEL=/srv/models/gemma-4-12b-it-UD-Q4_K_XL.gguf \
  TOKENIZER_DIR=/srv/models/gemma-4-12b-it-tokenizer
```

`hardware-check` runs artifact verification, the original boolean llama.cpp parity fixture and the two-context/two-field fixture.

## Removal

Artifacts are outside `make clean`. Remove the default or overridden artifact directory only with explicit confirmation:

```sh
make artifacts-clean CONFIRM_ARTIFACT_DELETE=1
```

`make distclean` applies the same confirmation requirement. The helper deletes only directories containing its `.managed-by-go-system-one` ownership marker; it refuses arbitrary paths even when confirmation is set.
