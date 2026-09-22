#!/usr/bin/env bash
set -euo pipefail

command=${1:-info}
cache_root=${XDG_CACHE_HOME:-$HOME/.cache}
artifact_dir=${GO_SYSTEM_ONE_ARTIFACT_DIR:-$cache_root/go-system-one/v1}
model_dir=$artifact_dir/model
tokenizer_dir=${GO_SYSTEM_ONE_TOKENIZER_DIR:-$artifact_dir/tokenizer}

model_repo=unsloth/gemma-4-12b-it-GGUF
model_revision=fc034cfff751157913579611efad8462ac1be606
model_file=gemma-4-12b-it-UD-Q4_K_XL.gguf
model_bytes=${GO_SYSTEM_ONE_MODEL_BYTES:-7366423360}
model_sha256=${GO_SYSTEM_ONE_MODEL_SHA256:-90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e}

tokenizer_repo=google/gemma-4-12b-it
tokenizer_revision=707f0a3b8a3c7ad586ed01e27eafbad8a27dd0f7
tokenizer_files=(tokenizer.json tokenizer_config.json chat_template.jinja)
tokenizer_sha256=(
  "${GO_SYSTEM_ONE_TOKENIZER_SHA256:-cc8d3a0ce36466ccc1278bf987df5f71db1719b9ca6b4118264f45cb627bfe0f}"
  "${GO_SYSTEM_ONE_TOKENIZER_CONFIG_SHA256:-a62f4e85a47c0c136edaaa3a4f591fd6783717299a9def47e5ad03a49f6a5eb9}"
  "${GO_SYSTEM_ONE_CHAT_TEMPLATE_SHA256:-ae53464bf3be25802b3a5b37def7fd89667067d7577049b3b2d74c4d8de4c6d4}"
)

model_path=${GO_SYSTEM_ONE_MODEL:-$model_dir/$model_file}

die() {
  printf 'artifacts: %s\n' "$*" >&2
  exit 1
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -- "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -- "$1" | awk '{print $1}'
  else
    die 'install sha256sum or shasum'
  fi
}

valid_file() {
  local path=$1 want_sha=$2 want_bytes=${3:--1} got_sha got_bytes
  [[ -f "$path" ]] || return 1
  got_bytes=$(wc -c < "$path" | tr -d '[:space:]')
  [[ $want_bytes -lt 0 || $got_bytes -eq $want_bytes ]] || return 1
  got_sha=$(sha256_file "$path")
  [[ "$got_sha" == "$want_sha" ]]
}

verify_file() {
  local path=$1 want_sha=$2 want_bytes=${3:--1} got_sha got_bytes
  [[ -f "$path" ]] || die "missing $path"
  got_bytes=$(wc -c < "$path" | tr -d '[:space:]')
  if [[ $want_bytes -ge 0 && $got_bytes -ne $want_bytes ]]; then
    die "$path has $got_bytes bytes; expected $want_bytes"
  fi
  got_sha=$(sha256_file "$path")
  [[ "$got_sha" == "$want_sha" ]] || die "$path SHA-256 is $got_sha; expected $want_sha"
  printf 'verified %s (%s bytes)\n' "$path" "$got_bytes"
}

verify_all() {
  verify_file "$model_path" "$model_sha256" "$model_bytes"
  local i
  for i in "${!tokenizer_files[@]}"; do
    verify_file "$tokenizer_dir/${tokenizer_files[$i]}" "${tokenizer_sha256[$i]}"
  done
}

download_file() {
  local url=$1 destination=$2 tmp=$2.part
  local -a auth=()
  local token=${HF_TOKEN:-${HUGGINGFACE_TOKEN:-}}
  [[ -n "$token" ]] && auth=(-H "Authorization: Bearer $token")
  mkdir -p "$(dirname "$destination")"
  printf 'downloading %s\n' "$destination"
  curl --fail --location --retry 4 --retry-delay 2 --continue-at - \
    "${auth[@]}" --output "$tmp" "$url" || \
      die "download failed; partial data remains at $tmp; gated repositories may require HF_TOKEN or HUGGINGFACE_TOKEN"
  mv "$tmp" "$destination"
}

download_all() {
  [[ ${ACCEPT_GEMMA_LICENSE:-0} == 1 ]] || die 'set ACCEPT_GEMMA_LICENSE=1 after accepting the Gemma/Hugging Face repository terms'
  command -v curl >/dev/null 2>&1 || die 'curl is required'
  mkdir -p "$artifact_dir"
  printf 'go-system-one artifact cache v1\n' > "$artifact_dir/.managed-by-go-system-one"

  if ! valid_file "$model_path" "$model_sha256" "$model_bytes"; then
    download_file \
      "https://huggingface.co/$model_repo/resolve/$model_revision/$model_file?download=true" \
      "$model_path"
  fi

  local i path
  for i in "${!tokenizer_files[@]}"; do
    path=$tokenizer_dir/${tokenizer_files[$i]}
    if ! valid_file "$path" "${tokenizer_sha256[$i]}"; then
      download_file \
        "https://huggingface.co/$tokenizer_repo/resolve/$tokenizer_revision/${tokenizer_files[$i]}?download=true" \
        "$path"
    fi
  done
  verify_all
}

print_info() {
  cat <<EOF
Artifact directory: $artifact_dir
Model:             $model_repo@$model_revision
Model file:        $model_path
Model bytes:       $model_bytes
Model SHA-256:     $model_sha256
Tokenizer:         $tokenizer_repo@$tokenizer_revision
Tokenizer dir:     $tokenizer_dir

Artifacts are external to the repository. Downloading requires explicit
ACCEPT_GEMMA_LICENSE=1 and may require HF_TOKEN or HUGGINGFACE_TOKEN.
EOF
}

case "$command" in
  info)
    print_info
    ;;
  paths)
    printf 'GO_SYSTEM_ONE_MODEL=%q\n' "$model_path"
    printf 'GO_SYSTEM_ONE_TOKENIZER_DIR=%q\n' "$tokenizer_dir"
    ;;
  download)
    download_all
    ;;
  verify)
    verify_all
    ;;
  clean)
    [[ ${CONFIRM_ARTIFACT_DELETE:-0} == 1 ]] || die 'set CONFIRM_ARTIFACT_DELETE=1 to delete the external artifact directory'
    [[ -f "$artifact_dir/.managed-by-go-system-one" ]] || die "refusing unmanaged artifact path $artifact_dir"
    rm -rf -- "$artifact_dir"
    printf 'removed %s\n' "$artifact_dir"
    ;;
  *)
    die "usage: $0 {info|paths|download|verify|clean}"
    ;;
esac
