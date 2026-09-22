#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tmp=$(mktemp -d)
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

mkdir -p "$tmp/src" "$tmp/pkg"
printf 'fixture\n' > "$tmp/src/file.go"
printf 'source/file.go\tsrc/file.go\n' > "$tmp/files.tsv"
printf 'github.com/rcarmo/go-pherence/pkg\tgithub.com/rcarmo/go-system-one/pkg\n' > "$tmp/packages.tsv"

UPSTREAM_FILES_MANIFEST="$tmp/files.tsv" LOCAL_PACKAGES_MANIFEST="$tmp/packages.tsv" \
  MANIFEST_DESTINATION_ROOT="$tmp" "$root/scripts/check-manifests.sh"

printf 'source/file.go\tsrc/file.go\nsource/other.go\tsrc/file.go\n' > "$tmp/files.tsv"
if UPSTREAM_FILES_MANIFEST="$tmp/files.tsv" LOCAL_PACKAGES_MANIFEST="$tmp/packages.tsv" \
  MANIFEST_DESTINATION_ROOT="$tmp" "$root/scripts/check-manifests.sh" >/dev/null 2>&1; then
  printf 'duplicate destination unexpectedly passed manifest validation\n' >&2
  exit 1
fi

printf '../escape\tsrc/file.go\n' > "$tmp/files.tsv"
if UPSTREAM_FILES_MANIFEST="$tmp/files.tsv" LOCAL_PACKAGES_MANIFEST="$tmp/packages.tsv" \
  MANIFEST_DESTINATION_ROOT="$tmp" "$root/scripts/check-manifests.sh" >/dev/null 2>&1; then
  printf 'unsafe source path unexpectedly passed manifest validation\n' >&2
  exit 1
fi

printf 'manifest synthetic verification passed\n'
