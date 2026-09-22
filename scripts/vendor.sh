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

printf 'vendored dependencies from go.mod\n'
