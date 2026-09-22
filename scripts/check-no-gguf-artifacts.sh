#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

failed=0
while IFS= read -r -d '' path; do
  lower=${path,,}
  if [[ "$lower" == *.gguf || "$lower" == *.gguf.* ]]; then
    printf 'tracked GGUF artifact is forbidden: %s\n' "$path" >&2
    failed=1
    continue
  fi
  if [[ -f "$path" ]]; then
    magic=
    IFS= read -r -N 4 magic < "$path" || true
    if [[ "$magic" == "GGUF" ]]; then
      printf 'tracked file contains GGUF magic and is forbidden: %s\n' "$path" >&2
      failed=1
    fi
  fi
done < <(git ls-files -z)

exit "$failed"
