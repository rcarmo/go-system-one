#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/project-tmp.sh"
PROJECT_TMP_ROOT=$(project_tmp_resolve go-system-one)
# Explicit operator cleanup only. Never delete run roots, caches or evidence.
case ${1:-build} in
  build) target="$PROJECT_TMP_ROOT/build";;
  dist) target="$PROJECT_TMP_ROOT/build/dist";;
  *) echo 'Only build or dist cleanup is supported' >&2; exit 2;;
esac
# Non-creating validation rejects symlinks and ownership problems.
project_path_usable "$target" || { echo "Unsafe cleanup path: $target" >&2; exit 2; }
[[ -e $target ]] || exit 0
rm -rf -- "$target"
