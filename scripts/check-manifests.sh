#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
files_manifest=${UPSTREAM_FILES_MANIFEST:-$root/scripts/upstream-files.tsv}
packages_manifest=${LOCAL_PACKAGES_MANIFEST:-$root/scripts/local-packages.tsv}
failed=0

safe_relative() {
  local path=$1
  [[ -n "$path" && "$path" != /* && "$path" != ".." && "$path" != ../* && "$path" != */../* && "$path" != */.. ]]
}

check_duplicates() {
  local file=$1 field=$2 label=$3 duplicates
  duplicates=$(awk -F'\t' -v field="$field" 'NF >= 2 && $1 !~ /^#/ { print $field }' "$file" | sort | uniq -d)
  if [[ -n "$duplicates" ]]; then
    printf 'duplicate %s entries in %s:\n%s\n' "$label" "$file" "$duplicates" >&2
    failed=1
  fi
}

check_duplicates "$files_manifest" 1 'upstream source path'
check_duplicates "$files_manifest" 2 'local destination path'
check_duplicates "$packages_manifest" 1 'upstream package import'
check_duplicates "$packages_manifest" 2 'local package import'

while IFS=$'\t' read -r source destination revision extra; do
  [[ -n "$source" && ${source:0:1} != "#" ]] || continue
  if [[ -z "$destination" || -n "${extra:-}" || ( -n "${revision:-}" && ! "$revision" =~ ^[0-9a-f]{40}$ ) ]]; then
    printf 'invalid file manifest row: %q -> %q revision=%q extra=%q\n' "$source" "$destination" "${revision:-}" "${extra:-}" >&2
    failed=1
    continue
  fi
  if ! safe_relative "$source" || ! safe_relative "$destination"; then
    printf 'unsafe file manifest path: %q -> %q\n' "$source" "$destination" >&2
    failed=1
  fi
  destination_root=${MANIFEST_DESTINATION_ROOT:-$root}
  if [[ ! -f "$destination_root/$destination" ]]; then
    printf 'file manifest destination is missing: %s\n' "$destination" >&2
    failed=1
  fi
done < "$files_manifest"

while IFS=$'\t' read -r upstream_import local_import mode extra; do
  [[ -n "$upstream_import" && ${upstream_import:0:1} != "#" ]] || continue
  if [[ -z "$local_import" || -n "${extra:-}" || ( -n "${mode:-}" && "$mode" != "local-only" ) ]]; then
    printf 'invalid package manifest row: %q -> %q mode=%q extra=%q\n' "$upstream_import" "$local_import" "${mode:-}" "${extra:-}" >&2
    failed=1
    continue
  fi
  if [[ "$upstream_import" != github.com/rcarmo/go-pherence/* ]]; then
    printf 'unexpected upstream package import: %s\n' "$upstream_import" >&2
    failed=1
  fi
  if [[ "$local_import" != github.com/rcarmo/go-system-one/* ]]; then
    printf 'unexpected local package import: %s\n' "$local_import" >&2
    failed=1
  fi
  relative=${local_import#github.com/rcarmo/go-system-one/}
  destination_root=${MANIFEST_DESTINATION_ROOT:-$root}
  if [[ ! -d "$destination_root/$relative" ]]; then
    printf 'local package directory is missing: %s\n' "$relative" >&2
    failed=1
  fi
done < "$packages_manifest"

exit "$failed"
