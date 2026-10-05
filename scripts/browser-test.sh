#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/scripts/project-env.sh"
mkdir -p "$PROFILE_ROOT"
out=$(mktemp -d "$PROFILE_ROOT/browser-runner-XXXXXXXX")
export BROWSER_SERVER_PROFILE_DIR="$out/server"
cd "$root/browser"
{ node --version; git rev-parse HEAD; printf 'Node CPU interval=1000us; heap sampling=524288 bytes\n'; } > "$out/invocation.txt"
printf '%s\n' 'node --cpu-prof --heap-prof node_modules/@playwright/test/cli.js test --config playwright.config.ts' > "$out/command.txt"
# The server has independent Go profiles; browser processes are not Node samples.
status=0
node --cpu-prof --cpu-prof-dir="$out" --heap-prof --heap-prof-dir="$out" node_modules/@playwright/test/cli.js test --config playwright.config.ts 2>&1 | tee "$out/test.log" || status=$?
# Playwright may treat teardown exit failures as successful browser tests.
# Verify this invocation's server receipt, not an earlier successful run.
if [[ ! -f $BROWSER_SERVER_PROFILE_DIR/status.txt || $(cat "$BROWSER_SERVER_PROFILE_DIR/status.txt") != 0 ]]; then
  echo 'Browser-server test/profile gate failed or did not finish' | tee -a "$out/test.log"
  status=1
fi
for file in cpu.pprof heap.pprof cpu.txt alloc_space.txt alloc_objects.txt test.bin; do
  if [[ ! -s $BROWSER_SERVER_PROFILE_DIR/$file ]]; then
    echo "Missing browser-server capture: $file" | tee -a "$out/test.log"
    status=1
  fi
done
printf '%s\n' "$status" > "$out/status.txt"
exit "$status"
