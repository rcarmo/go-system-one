#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/scripts/upstream.env"

revision=${1:-}
source_repo=${GO_PHERENCE_SOURCE:-}
pinned_revision=${UPSTREAM_PIN:-$UPSTREAM_COMMIT}
manifest=${UPSTREAM_FILES_MANIFEST:-$root/scripts/upstream-files.tsv}
if [[ -z "$revision" ]]; then
  printf 'usage: %s <full-go-pherence-commit>\n' "$0" >&2
  exit 2
fi
if [[ ! "$revision" =~ ^[0-9a-f]{40}$ ]]; then
  printf 'upstream revision must be a full lowercase commit hash: %s\n' "$revision" >&2
  exit 2
fi

cleanup() {
  if [[ -n ${tmp:-} ]]; then
    rm -rf "$tmp"
  fi
}
trap cleanup EXIT

if [[ -z "$source_repo" ]]; then
  tmp=$(mktemp -d)
  git clone --filter=blob:none --no-checkout "$UPSTREAM_REPOSITORY" "$tmp/go-pherence" >/dev/null
  source_repo=$tmp/go-pherence
fi

git -C "$source_repo" cat-file -e "$pinned_revision^{commit}"
git -C "$source_repo" cat-file -e "$revision^{commit}"

changed=0
while IFS=$'\t' read -r source_path destination_path; do
  [[ -n "$source_path" && ${source_path:0:1} != "#" ]] || continue
  old_blob=$(git -C "$source_repo" rev-parse "$pinned_revision:$source_path" 2>/dev/null || printf missing)
  new_blob=$(git -C "$source_repo" rev-parse "$revision:$source_path" 2>/dev/null || printf missing)
  if [[ "$old_blob" != "$new_blob" ]]; then
    printf '%s\n' "$source_path"
    changed=1
  fi
done < "$manifest"

if [[ $changed -eq 1 ]]; then
  exit 0
fi
exit 3
