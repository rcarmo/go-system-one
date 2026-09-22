#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tmp=$(mktemp -d)
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

cd "$root"
go run ./scripts/benchmarks render -out "$tmp"
for name in latency-comparison.svg warm-latency.svg workload-matrix.svg; do
  cmp "$tmp/$name" "$root/docs/benchmarks/$name" || {
    printf 'benchmark chart is stale: docs/benchmarks/%s; run make benchmark-charts\n' "$name" >&2
    exit 1
  }
done
printf 'benchmark charts match committed data\n'
