#!/usr/bin/env bash
set -euo pipefail

dist=${1:-dist}
[[ -d "$dist" ]] || { printf 'release directory does not exist: %s\n' "$dist" >&2; exit 1; }

shopt -s nullglob
archives=("$dist"/*.tar.gz)
((${#archives[@]})) || { printf 'no release archives found under %s\n' "$dist" >&2; exit 1; }

for archive in "${archives[@]}"; do
  if tar -tzf "$archive" | grep -Eiq '\.gguf($|\.)|(^|/)(models|checkpoints)/'; then
    printf 'release archive contains a forbidden model/checkpoint path: %s\n' "$archive" >&2
    exit 1
  fi
  tmp=$(mktemp -d)
  tar -xzf "$archive" -C "$tmp"
  while IFS= read -r -d '' path; do
    magic=
    IFS= read -r -N 4 magic < "$path" || true
    if [[ "$magic" == "GGUF" ]]; then
      printf 'release archive contains GGUF magic: %s (%s)\n' "$archive" "${path#$tmp/}" >&2
      rm -rf "$tmp"
      exit 1
    fi
  done < <(find "$tmp" -type f -print0)
  rm -rf "$tmp"
  printf 'verified release archive %s\n' "$archive"
done
