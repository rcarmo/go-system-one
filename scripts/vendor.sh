#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

go mod vendor

# `go mod vendor` copies files selected by Go packages. The RISC-V assembly in
# go-pherence includes this adjacent header, which is invisible to that package
# selection. Copy it from the exact module version in go.mod.
module_dir=$(go list -mod=mod -m -f '{{.Dir}}' github.com/rcarmo/go-pherence)
install -D -m 0644 \
  "$module_dir/backends/spacemit/ime2/ime2_isa.h" \
  "$root/vendor/github.com/rcarmo/go-pherence/backends/spacemit/ime2/ime2_isa.h"

# Shared kernels move into this module incrementally. Rewrite the retained
# upstream closure to use the local implementation, then remove the duplicate
# vendored package and its package line from modules.txt.
while IFS=$'\t' read -r upstream_import local_import; do
  [[ -n "$upstream_import" && ${upstream_import:0:1} != "#" ]] || continue
  mapfile -d '' files < <(grep -rlZ --include='*.go' "\"$upstream_import\"" "$root/vendor/github.com/rcarmo/go-pherence" || true)
  if ((${#files[@]})); then
    sed -i "s#\"$upstream_import\"#\"$local_import\"#g" "${files[@]}"
  fi
  relative=${upstream_import#github.com/rcarmo/go-pherence/}
  package_dir="$root/vendor/github.com/rcarmo/go-pherence/$relative"
  if [[ -d "$package_dir" ]]; then
    find "$package_dir" -maxdepth 1 -type f -delete
    rmdir "$package_dir" 2>/dev/null || true
  fi
  sed -i "\\#^$upstream_import\$#d" "$root/vendor/modules.txt"
done < "$root/scripts/local-packages.tsv"

printf 'vendored dependencies from go.mod with local package overrides\n'
