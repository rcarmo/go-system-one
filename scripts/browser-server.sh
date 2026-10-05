#!/usr/bin/env bash
# Playwright signals the entire server process group. Do not forward a second
# TERM to Go after signal.NotifyContext returns: it can kill profile flushing.
set -uo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/scripts/project-env.sh" || exit 2
cd "$root"
mkdir -p "$PROFILE_ROOT"
if [[ -n ${BROWSER_SERVER_PROFILE_DIR:-} ]]; then
  out=$BROWSER_SERVER_PROFILE_DIR
  mkdir "$out" || exit 1
else
  out=$(mktemp -d "$PROFILE_ROOT/browser-server-XXXXXXXX") || exit 1
fi
{ go version; git rev-parse HEAD; printf 'CPU=100Hz\nmemprofilerate=524288\nworkload=Playwright synthetic HTTP fixture\n'; } > "$out/invocation.txt"
if ! go test -mod=vendor -c -o "$out/test.bin" ./webui > "$out/build.log" 2>&1; then
  cat "$out/build.log"; echo 'Build failed: no profiles captured' >&2; exit 1
fi
args=(-test.run '^TestBrowserServer$' -test.count=1 -test.timeout=5m "-test.cpuprofile=$out/cpu.pprof" "-test.memprofile=$out/heap.pprof")
printf '%q ' "$out/test.bin" "${args[@]}" > "$out/command.txt"
trap ':' TERM INT
WEBUI_BROWSER_TEST=1 "$out/test.bin" "${args[@]}" > "$out/test.log" 2>&1 &
pid=$!
status=0
# A trapped signal interrupts Bash's wait before Go finishes. Reap the child
# before inspecting profiles, preserving its exit code rather than wait's 143.
while true; do
  wait "$pid"; status=$?
  kill -0 "$pid" 2>/dev/null || break
done
trap '' TERM INT
for metric in cpu alloc_space alloc_objects; do
  profile="$out/heap.pprof"; options=("-sample_index=$metric")
  if [[ $metric == cpu ]]; then profile="$out/cpu.pprof"; options=(); fi
  if [[ -s $profile ]]; then
    go tool pprof -top -cum "${options[@]}" "$out/test.bin" "$profile" > "$out/$metric.txt" 2>&1 || status=1
    if grep -q 'Total samples = 0' "$out/$metric.txt"; then
      echo "Empty $metric samples; no performance conclusion" >> "$out/warnings.txt"
    fi
  else
    echo "Missing $metric capture" | tee "$out/$metric.txt"; status=1
  fi
done
printf '%s\n' "$status" > "$out/status.txt"
exit "$status"
