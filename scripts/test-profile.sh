#!/usr/bin/env bash
# One invocation per package: go test cannot profile ./... into separate files.
set -uo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/scripts/project-env.sh" || exit 2
cd "$root"
GO=${GO:-go}
export GO_PHERENCE_DISABLE_NVIDIA=${GO_PHERENCE_DISABLE_NVIDIA-1}
patterns=(); flags=()
while (($#)); do
  if [[ $1 == -- ]]; then shift; flags=("$@"); break; fi
  patterns+=("$1"); shift
done
((${#patterns[@]})) || patterns=(./...)
for flag in "${flags[@]}"; do
  case "$flag" in
    -cpuprofile*|-memprofile*|-memprofilerate*|-o|-o=*|-outputdir*|-coverprofile*|-fuzz*)
      echo 'Profile paths are managed here; fuzz workers require explicit per-worker capture.' >&2; exit 2;;
  esac
done
mkdir -p "$PROFILE_ROOT" || exit 1
run=$(mktemp -d "$PROFILE_ROOT/run-$(date -u +%Y%m%dT%H%M%SZ)-XXXXXXXX") || exit 1
printf 'Profiles: %s\n' "$run"
{
  "$GO" version
  git rev-parse HEAD
  git status --short
  printf 'scratch=%s\nCPU=100Hz\nmemprofilerate=%s\nGOMAXPROCS=%s\n' "$TMPDIR" "${MEMPROFILE_RATE:-524288}" "${GOMAXPROCS:-default}"
  printf 'flags:'; printf ' %q' "${flags[@]}"; printf '\n'
  "$GO" env GOOS GOARCH GOCACHE GOMODCACHE
} > "$run/invocation.txt" 2>&1
git diff HEAD > "$run/source.patch"
git ls-files --others --exclude-standard > "$run/untracked.txt"
while IFS= read -r file; do
  [[ -f $file ]] || continue
  mkdir -p "$run/untracked-source/$(dirname "$file")"
  cp "$file" "$run/untracked-source/$file"
done < "$run/untracked.txt"
if ! "$GO" list -mod=vendor -f '{{.ImportPath}}|{{if or .TestGoFiles .XTestGoFiles}}tests{{else}}build-only{{end}}' "${patterns[@]}" > "$run/packages.txt" 2> "$run/list.log"; then
  cat "$run/list.log" >&2; echo 'No profiles captured: package discovery failed.' >&2; exit 1
fi
failed=0
printf 'package\ttest_status\tprofile_status\n' > "$run/status.tsv"
while IFS='|' read -r package kind; do
  out="$run/$package"; mkdir -p "$out"
  args=(test -mod=vendor "$package" -count=1 -timeout=6m "${flags[@]}" "-o=$out/test.bin" "-cpuprofile=$out/cpu.pprof" "-memprofile=$out/heap.pprof" "-memprofilerate=${MEMPROFILE_RATE:-524288}")
  if [[ ${PROFILE_COVERAGE:-0} == 1 ]]; then args+=("-coverprofile=$out/coverage.out"); fi
  printf '%q ' "$GO" "${args[@]}" > "$out/command.txt"; printf '\n' >> "$out/command.txt"
  mkdir -p "$out/subprocess"
  PROFILE_SUBPROCESS_DIR="$out/subprocess" "$GO" "${args[@]}" > "$out/test.log" 2>&1
  status=$?; cat "$out/test.log"; analysis=0
  if [[ $kind == tests ]]; then
    for metric in cpu alloc_space alloc_objects; do
      profile="$out/heap.pprof"; options=("-sample_index=$metric")
      if [[ $metric == cpu ]]; then profile="$out/cpu.pprof"; options=(); fi
      if [[ -s $profile && -s $out/test.bin ]]; then
        "$GO" tool pprof -top -cum "${options[@]}" "$out/test.bin" "$profile" > "$out/$metric.txt" 2>&1 || analysis=1
        if grep -q 'Total samples = 0' "$out/$metric.txt"; then
          echo "$package: empty $metric samples; no performance conclusion" | tee -a "$run/warnings.txt"
        fi
      else
        echo "Missing $metric capture (build failure or interrupted process)" | tee "$out/$metric.txt"
        analysis=1
      fi
    done
    { for metric in cpu alloc_space alloc_objects; do echo "=== $metric ==="; head -22 "$out/$metric.txt"; done; } > "$out/analysis.txt"
    shopt -s nullglob
    for child in "$out"/subprocess/*.cpu.pprof; do
      prefix=${child%.cpu.pprof}
      "$GO" tool pprof -top -cum "$out/test.bin" "$child" > "$prefix.cpu.txt" 2>&1 || analysis=1
      for metric in alloc_space alloc_objects; do
        "$GO" tool pprof -top -cum "-sample_index=$metric" "$out/test.bin" "$prefix.heap.pprof" > "$prefix.$metric.txt" 2>&1 || analysis=1
      done
      if grep -q 'Total samples = 0' "$prefix.cpu.txt"; then
        echo "$prefix: empty subprocess CPU samples (skip-only workload)" | tee -a "$run/warnings.txt"
      fi
    done
    shopt -u nullglob
  fi
  if [[ ${PROFILE_COVERAGE:-0} == 1 && -s $out/coverage.out ]]; then
    if [[ ! -s $run/coverage.out ]]; then head -1 "$out/coverage.out" > "$run/coverage.out"; fi
    tail -n +2 "$out/coverage.out" >> "$run/coverage.out"
  fi
  printf '%s\t%s\t%s\n' "$package" "$status" "$analysis" >> "$run/status.tsv"
  ((status == 0 && analysis == 0)) || failed=1
done < "$run/packages.txt"
if [[ -s $run/coverage.out ]]; then "$GO" tool cover -func="$run/coverage.out" > "$run/coverage.txt"; fi
printf 'Result: %s; profiles and cumulative tables: %s\n' "$failed" "$run"
echo 'Review CPU/allocation tables and compare equivalent workloads; subprocess activity is not covered by parent profiles.'
exit "$failed"
