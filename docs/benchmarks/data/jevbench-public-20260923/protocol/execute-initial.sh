#!/usr/bin/env bash
set -euo pipefail
root=/workspace/tmp/gso-jevbench
py=/tmp/gso-jevbench-venv/bin/python
if nvidia-smi --query-compute-apps=pid --format=csv,noheader,nounits | grep -Eq '[0-9]'; then echo 'GPU busy'; exit 1; fi
if curl -fsS --max-time 1 http://127.0.0.1:18087/go-system-one/v1/status >/dev/null 2>&1; then echo 'port occupied'; exit 1; fi
"$root/server" -model /tmp/qev-gemma4-12b/gemma-4-12b-it-UD-Q4_K_XL.gguf -tokenizer-dir /tmp/qev-gemma4-12b/tokenizer -listen 127.0.0.1:18087 > "$root/server.log" 2>&1 & server=$!
monitor=''; runner=''
cleanup(){
 [ -z "$runner" ] || kill "$runner" 2>/dev/null || true
 [ -z "$monitor" ] || kill "$monitor" 2>/dev/null || true
 kill "$server" 2>/dev/null || true
 wait "$server" 2>/dev/null || true
}
trap cleanup EXIT
for i in $(seq 1 180); do
 if curl -fsS http://127.0.0.1:18087/go-system-one/v1/status > "$root/status.json" 2>/dev/null; then break; fi
 kill -0 "$server" || exit 1; sleep 1
done
"$py" "$root/monitor.py" "$server" "$root/runner.pid" & monitor=$!
for tier in easy original hard; do
 "$py" -u "$root/run.py" "$tier" > "$root/$tier.log" 2>&1 & runner=$!
 echo "$runner" > "$root/runner.pid"
 result=0; wait "$runner" || result=$?
 runner=''; cat "$root/$tier.log"
 test ! -f "$root/ABORTED" || exit 70
 if [ "$result" != 0 ]; then echo "Tier $tier stopped with status $result"; exit "$result"; fi
done
