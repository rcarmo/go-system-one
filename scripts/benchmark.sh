#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
artifact_dir=${GO_SYSTEM_ONE_ARTIFACT_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/go-system-one/v1}
model=${MODEL:-$artifact_dir/model/gemma-4-12b-it-UD-Q4_K_XL.gguf}
tokenizer_dir=${TOKENIZER_DIR:-$artifact_dir/tokenizer}
backend=${BACKEND:-nvidia}
listen=${BENCHMARK_LISTEN:-127.0.0.1:18081}
requests=${BENCHMARK_REQUESTS:-100}
warmup=${BENCHMARK_WARMUP:-1}
out=${BENCHMARK_OUT:-$root/dist/benchmarks/nvidia-http.json}
binary=$root/bin/go-system-one-benchmark
log=${BENCHMARK_LOG:-$root/dist/benchmarks/server.log}

if [[ "$listen" != 127.0.0.1:* && "$listen" != localhost:* ]]; then
  printf 'benchmark: BENCHMARK_LISTEN must be loopback: %s\n' "$listen" >&2
  exit 1
fi
status_url=http://$listen/go-system-one/v1/status
if curl --fail --silent --show-error --max-time 1 "$status_url" >/dev/null 2>&1; then
  printf 'benchmark: refusing occupied endpoint %s\n' "$status_url" >&2
  exit 1
fi
if [[ "$backend" == nvidia && ${ALLOW_BUSY_GPU:-0} != 1 ]] && command -v nvidia-smi >/dev/null 2>&1; then
  if nvidia-smi --query-compute-apps=pid --format=csv,noheader,nounits 2>/dev/null | grep -Eq '[0-9]'; then
    printf 'benchmark: NVIDIA device is busy; stop other compute processes or set ALLOW_BUSY_GPU=1\n' >&2
    exit 1
  fi
fi

cleanup() {
  if [[ -n ${pid:-} ]] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

GO_SYSTEM_ONE_ARTIFACT_DIR="$artifact_dir" GO_SYSTEM_ONE_MODEL="$model" GO_SYSTEM_ONE_TOKENIZER_DIR="$tokenizer_dir" \
  "$root/scripts/artifacts.sh" verify

mkdir -p "$root/bin" "$(dirname "$out")" "$(dirname "$log")"
(
  cd "$root"
  go build -mod=vendor -trimpath -o "$binary" ./cmd/go-system-one
)
"$binary" -model "$model" -tokenizer-dir "$tokenizer_dir" -backend "$backend" -listen "$listen" >"$log" 2>&1 &
pid=$!

for _ in $(seq 1 300); do
  if curl --fail --silent --show-error "$status_url" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    cat "$log" >&2
    printf 'benchmark: server exited before becoming ready\n' >&2
    exit 1
  fi
  sleep 1
done
curl --fail --silent --show-error "$status_url" >/dev/null || {
  cat "$log" >&2
  printf 'benchmark: server readiness timeout\n' >&2
  exit 1
}
kill -0 "$pid" 2>/dev/null || {
  cat "$log" >&2
  printf 'benchmark: benchmark server is not running after readiness\n' >&2
  exit 1
}

(
  cd "$root"
  go run ./scripts/benchmarks collect \
    -url "http://$listen/v1/decision" \
    -request docs/benchmarks/request.json \
    -out "$out" \
    -warmup "$warmup" \
    -n "$requests"
)
printf 'benchmark samples written to %s\n' "$out"
