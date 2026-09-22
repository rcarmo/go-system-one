#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tmp=$(mktemp -d)
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

repo=$tmp/upstream
mkdir -p "$repo/relevant" "$repo/unrelated"
git -C "$repo" init -q
git -C "$repo" config user.name test
git -C "$repo" config user.email test@example.invalid
printf 'one\n' > "$repo/relevant/file.txt"
printf 'one\n' > "$repo/unrelated/file.txt"
git -C "$repo" add .
git -C "$repo" commit -qm initial
base=$(git -C "$repo" rev-parse HEAD)

printf 'two\n' > "$repo/unrelated/file.txt"
git -C "$repo" commit -qam unrelated
unrelated=$(git -C "$repo" rev-parse HEAD)

printf 'two\n' > "$repo/relevant/file.txt"
git -C "$repo" commit -qam relevant
relevant=$(git -C "$repo" rev-parse HEAD)

printf 'relevant/file.txt\trelevant/file.txt\n' > "$tmp/manifest.tsv"

if GO_PHERENCE_SOURCE="$repo" UPSTREAM_PIN="$base" UPSTREAM_FILES_MANIFEST="$tmp/manifest.tsv" \
  "$root/scripts/upstream-relevant.sh" "$base" >/dev/null 2>&1; then
  printf 'current pin unexpectedly reported a relevant change\n' >&2
  exit 1
else
  rc=$?
  [[ $rc -eq 3 ]] || { printf 'current pin returned %d, expected 3\n' "$rc" >&2; exit 1; }
fi

if GO_PHERENCE_SOURCE="$repo" UPSTREAM_PIN="$base" UPSTREAM_FILES_MANIFEST="$tmp/manifest.tsv" \
  "$root/scripts/upstream-relevant.sh" "$unrelated" >/dev/null 2>&1; then
  printf 'unrelated commit unexpectedly reported a relevant change\n' >&2
  exit 1
else
  rc=$?
  [[ $rc -eq 3 ]] || { printf 'unrelated commit returned %d, expected 3\n' "$rc" >&2; exit 1; }
fi

output=$(GO_PHERENCE_SOURCE="$repo" UPSTREAM_PIN="$base" UPSTREAM_FILES_MANIFEST="$tmp/manifest.tsv" \
  "$root/scripts/upstream-relevant.sh" "$relevant")
[[ "$output" == relevant/file.txt ]] || { printf 'unexpected relevant output: %q\n' "$output" >&2; exit 1; }

printf 'upstream relevance synthetic verification passed\n'
