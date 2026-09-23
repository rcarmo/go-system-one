#!/usr/bin/env bash
set -euo pipefail
root=/workspace/tmp/gso-long-context
if nvidia-smi --query-compute-apps=pid --format=csv,noheader | grep -Eq '[0-9]';then echo GPU-busy;exit 1;fi
"$root/server-final" -model /tmp/qev-gemma4-12b/gemma-4-12b-it-UD-Q4_K_XL.gguf -tokenizer-dir /tmp/qev-gemma4-12b/tokenizer -listen 127.0.0.1:18088 > "$root/recheck-server-$1.log" 2>&1 & p=$!
trap 'kill "$p" 2>/dev/null || true; wait "$p" 2>/dev/null || true' EXIT
for i in $(seq 1 180);do curl -fsS http://127.0.0.1:18088/go-system-one/v1/status >/dev/null 2>&1 && break; kill -0 "$p";sleep 1;done
bun "$root/recheck.ts" "$1" "$2"
