#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
failed=0

while IFS=$'\t' read -r upstream_import local_import mode; do
  [[ -n "$upstream_import" && ${upstream_import:0:1} != "#" ]] || continue
  matches=$(grep -rI --exclude-dir=.git --exclude-dir=vendor --include='*.go' -n "\"$upstream_import\"" "$root" || true)
  if [[ -n "$matches" ]]; then
    printf 'first-party source still imports internalised package %s:\n%s\n' "$upstream_import" "$matches" >&2
    failed=1
  fi
done < "$root/scripts/local-packages.tsv"

exit "$failed"
