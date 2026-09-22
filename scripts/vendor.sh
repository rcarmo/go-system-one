#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

go mod vendor

# All first-party packages are local. The manifest remains an update/audit map;
# vendor contains third-party modules only.
if grep -q '^github.com/rcarmo/go-pherence/' "$root/vendor/modules.txt" 2>/dev/null; then
  printf 'go-pherence packages unexpectedly remain in vendor/modules.txt\n' >&2
  exit 1
fi
if [[ -d "$root/vendor/github.com/rcarmo/go-pherence" ]]; then
  printf 'go-pherence source unexpectedly remains under vendor/\n' >&2
  exit 1
fi

printf 'vendored third-party dependencies from go.mod\n'
