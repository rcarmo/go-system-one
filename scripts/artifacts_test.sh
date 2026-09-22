#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
sandbox=$(mktemp -d)
tmp=$sandbox/go-system-one/v1
cleanup() { rm -rf "$sandbox"; }
trap cleanup EXIT

model=fixture-model
tokenizer=fixture-tokenizer
tokenizer_config=fixture-config
chat_template=fixture-template
mkdir -p "$tmp/model" "$tmp/tokenizer"
printf '%s' "$model" > "$tmp/model/gemma-4-12b-it-UD-Q4_K_XL.gguf"
printf '%s' "$tokenizer" > "$tmp/tokenizer/tokenizer.json"
printf '%s' "$tokenizer_config" > "$tmp/tokenizer/tokenizer_config.json"
printf '%s' "$chat_template" > "$tmp/tokenizer/chat_template.jinja"

sha() { printf '%s' "$1" | sha256sum | awk '{print $1}'; }

common=(
  "GO_SYSTEM_ONE_ARTIFACT_DIR=$tmp"
  "GO_SYSTEM_ONE_MODEL_BYTES=${#model}"
  "GO_SYSTEM_ONE_MODEL_SHA256=$(sha "$model")"
  "GO_SYSTEM_ONE_TOKENIZER_SHA256=$(sha "$tokenizer")"
  "GO_SYSTEM_ONE_TOKENIZER_CONFIG_SHA256=$(sha "$tokenizer_config")"
  "GO_SYSTEM_ONE_CHAT_TEMPLATE_SHA256=$(sha "$chat_template")"
)

env "${common[@]}" "$root/scripts/artifacts.sh" verify >/dev/null
paths=$(GO_SYSTEM_ONE_ARTIFACT_DIR="$tmp" "$root/scripts/artifacts.sh" paths)
grep -Fq "GO_SYSTEM_ONE_MODEL=$tmp/model/gemma-4-12b-it-UD-Q4_K_XL.gguf" <<<"$paths"
grep -Fq "GO_SYSTEM_ONE_TOKENIZER_DIR=$tmp/tokenizer" <<<"$paths"

printf 'corrupt' >> "$tmp/tokenizer/tokenizer.json"
if env "${common[@]}" "$root/scripts/artifacts.sh" verify >/dev/null 2>&1; then
  printf 'corrupt tokenizer unexpectedly verified\n' >&2
  exit 1
fi

if GO_SYSTEM_ONE_ARTIFACT_DIR="$tmp" "$root/scripts/artifacts.sh" download >/dev/null 2>&1; then
  printf 'artifact download without licence acknowledgement unexpectedly succeeded\n' >&2
  exit 1
fi

printf 'go-system-one artifact cache v1\n' > "$tmp/.managed-by-go-system-one"
if GO_SYSTEM_ONE_ARTIFACT_DIR="$tmp" "$root/scripts/artifacts.sh" clean >/dev/null 2>&1; then
  printf 'unguarded artifact cleanup unexpectedly succeeded\n' >&2
  exit 1
fi
GO_SYSTEM_ONE_ARTIFACT_DIR="$tmp" CONFIRM_ARTIFACT_DELETE=1 "$root/scripts/artifacts.sh" clean >/dev/null
[[ ! -e "$tmp" ]] || { printf 'guarded artifact cleanup did not remove fixture directory\n' >&2; exit 1; }

printf 'artifact helper synthetic verification passed\n'
