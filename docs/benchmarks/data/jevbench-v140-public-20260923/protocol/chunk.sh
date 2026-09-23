#!/usr/bin/env bash
set -euo pipefail
root=/workspace/tmp/gso-jevbench-v140
py=/tmp/gso-jevbench-venv/bin/python
tier=$1; chunk=$2; limit=${3:-24}
log="$root/chunks/$tier-$chunk"; mkdir -p "$log"
if nvidia-smi --query-compute-apps=pid --format=csv,noheader,nounits | grep -Eq '[0-9]'; then echo 'GPU busy'; exit 1; fi
if curl -fsS --max-time 1 http://127.0.0.1:18087/go-system-one/v1/status >/dev/null 2>&1; then echo 'port occupied'; exit 1; fi
"$root/server" -model /tmp/qev-gemma4-12b/gemma-4-12b-it-UD-Q4_K_XL.gguf -tokenizer-dir /tmp/qev-gemma4-12b/tokenizer -listen 127.0.0.1:18087 > "$log/server.log" 2>&1 & server=$!
monitor=''; runner=''
cleanup(){
 [ -z "$runner" ] || kill "$runner" 2>/dev/null || true
 [ -z "$monitor" ] || kill "$monitor" 2>/dev/null || true
 kill "$server" 2>/dev/null || true
 wait "$server" 2>/dev/null || true
}
trap cleanup EXIT
for i in $(seq 1 180); do
 if curl -fsS http://127.0.0.1:18087/go-system-one/v1/status > "$log/status.json" 2>/dev/null; then break; fi
 kill -0 "$server" || exit 1; sleep 1
done
"$py" "$root/monitor.py" "$server" "$log/runner.pid" "$log" & monitor=$!
"$py" -u "$root/run.py" "$tier" "$chunk" "$limit" > "$log/runner.log" 2>&1 & runner=$!
echo "$runner" > "$log/runner.pid"
result=0; wait "$runner" || result=$?
runner=''; cat "$log/runner.log"
test ! -f "$log/ABORTED" || exit 70
exit "$result"
