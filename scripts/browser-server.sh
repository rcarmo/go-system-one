#!/usr/bin/env bash
# Playwright sends TERM; allow the fixture to flush profiles before analysis.
set -uo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/scripts/project-env.sh" || exit 2
cd "$root"
mkdir -p "$PROFILE_ROOT"
out=$(mktemp -d "$PROFILE_ROOT/browser-server-XXXXXXXX") || exit 1
{ go version; git rev-parse HEAD; printf 'CPU=100Hz\nmemprofilerate=524288\nworkload=Playwright synthetic HTTP fixture\n'; } > "$out/invocation.txt"
if ! go test -mod=vendor -c -o "$out/test.bin" ./webui > "$out/build.log" 2>&1; then
  cat "$out/build.log"; echo 'Build failed: no profiles captured' >&2; exit 1
fi
args=(-test.run '^TestBrowserServer$' -test.count=1 -test.timeout=5m "-test.cpuprofile=$out/cpu.pprof" "-test.memprofile=$out/heap.pprof")
printf '%q ' "$out/test.bin" "${args[@]}" > "$out/command.txt"
WEBUI_BROWSER_TEST=1 "$out/test.bin" "${args[@]}" > "$out/test.log" 2>&1 &
pid=$!
trap 'kill -TERM "$pid" 2>/dev/null || true' TERM INT
status=0
wait "$pid" || status=$?
if kill -0 "$pid" 2>/dev/null; then wait "$pid"; status=$?; fi
for metric in cpu alloc_space alloc_objects; do
  profile="$out/heap.pprof"; options=("-sample_index=$metric")
  if [[ $metric == cpu ]]; then profile="$out/cpu.pprof"; options=(); fi
  if [[ -s $profile ]]; then
    go tool pprof -top -cum "${options[@]}" "$out/test.bin" "$profile" > "$out/$metric.txt" 2>&1 || status=1
  else
    echo "Missing $metric capture" | tee "$out/$metric.txt"; status=1
  fi
done
printf '%s\n' "$status" > "$out/status.txt"
exit "$status"
