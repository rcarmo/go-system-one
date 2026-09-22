# go-system-one

`go-system-one` serves finite-choice decisions, ordered scores and yes/no probabilities from the Go System One v1 Gemma 4 12B checkpoint.

```sh
make run BACKEND=nvidia LISTEN=127.0.0.1:8080
```

Open `http://127.0.0.1:8080/go-system-one` for the [standalone playground](../../docs/playground.md). Its default TypeSafe view submits `noul`, `choice` and `score` questions to `POST /v1/systemone`, showing yes-probabilities, selected choices, fractional expected scores and token usage. The [API contract](../../docs/systemone-api.md) defines the local confidence approximations.

Select Batch decisions to use `POST /v1/decision` with independent contexts and boolean/enum schemas. Tree-mode results display every allowed outcome and highlight the selected value. Both routes share the scorer and admission gate. Probabilities are over allowed candidates, not calibrated correctness estimates.

![Go System One decision playground](../../docs/images/go-system-one-light-desktop.png)

The command verifies the frozen model, tokenizer, tokenizer configuration and chat-template SHA-256 values before loading them. Use `-verify-artifacts=false` only for development fixtures. `-backend simd` selects the correctness-oracle implementation; the 12B SIMD path is too slow for interactive use.

## Packed execution

NVIDIA tree scoring packs contexts and field branches by default, up to 512 real token rows per group. Use `-packed-token-rows=0` for the serial reference or a positive value up to 512 to bound groups more tightly. Automatic mode (`-1`) selects 512 on NVIDIA and 0 on SIMD. Oversized entries fall back individually; greedy/mixed-mode requests keep the serial scorer.

Packing uses Q8 activations where tiny serial branch batches use F32. The [precision report](../../docs/performance/multifield-precision.md) records winner agreement, losing-rank shifts and probability movement. Returned confidence can change even when the selected answer does not.

## Deployment limits

The server has no authentication or TLS. Its default loopback binding is the supported direct-use configuration. For remote access, put an authenticated TLS reverse proxy in front of every route, restrict the upstream to loopback or a private socket, and apply request-rate and body-size limits at the proxy. Do not expose the command directly to an untrusted network.

Inference uses single admission. A concurrent request receives HTTP 429 with `decision inference busy; retry later`; clients should use bounded backoff rather than parallel retries. The command verifies model artifacts by default, accepts at most a 1 MiB JSON body and shuts down with a 30-second drain window. The NVIDIA configuration is qualified for one RTX 3060 12 GB process with 9,173,277,696 resident projection bytes; co-locating another model or inference worker requires separate VRAM admission measurements.

The endpoint returns constrained model probabilities. They are not calibrated accuracy estimates. Log requests and decisions according to the deployment's data-handling policy, and avoid placing sensitive context text in proxy access logs.

See [`docs/validation/go-system-one-v1-20260921.md`](../../docs/validation/go-system-one-v1-20260921.md) for pins, numerical gates, workload measurements and retention evidence.
