#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/scripts/project-env.sh"
mkdir -p "$PROFILE_ROOT"
out=$(mktemp -d "$PROFILE_ROOT/browser-runner-XXXXXXXX")
cd "$root/browser"
{ node --version; git rev-parse HEAD; printf 'Node CPU interval=1000us; heap sampling=524288 bytes\n'; } > "$out/invocation.txt"
printf '%s\n' 'node --cpu-prof --heap-prof node_modules/@playwright/test/cli.js test --config playwright.config.ts' > "$out/command.txt"
# The server has independent Go profiles; browser processes are not Node samples.
node --cpu-prof --cpu-prof-dir="$out" --heap-prof --heap-prof-dir="$out" node_modules/@playwright/test/cli.js test --config playwright.config.ts 2>&1 | tee "$out/test.log"
