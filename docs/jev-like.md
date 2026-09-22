# Jev-like decision model

Go System One applies a Jev-like finite-choice interface to a pinned Gemma 4 12B instruction backbone. Callers provide context, guidance and typed boolean or enum fields; the scorer follows bounded candidate-token paths and returns one selected value plus a constrained model probability for each field.

Gemma supplies the pretrained language representation and broad world knowledge used to interpret the context and options. Go System One supplies the request schema, prompt contract, candidate trie, prefix reuse, independent suffix scoring, validation and HTTP surface.

The Jev-like description is architectural and behavioural. This repository does not load Jev weights, use a Jev pointer head or claim parity with a released Jev checkpoint. Its correctness gates use the pinned Gemma checkpoint, CPU/SIMD oracle, NVIDIA differential tests and independent llama.cpp fixtures.

## Decision contract

The v1 API supports:

- boolean fields;
- unordered string-enum fields;
- one to 256 independent contexts per request;
- `auto` and explicit `tree` scoring modes;
- bounded candidates and strict request validation.

The model scores only paths admitted by the compiled schema. It does not generate arbitrary prose through `/v1/decision`.

## Limits

The returned probabilities are constrained model probabilities over the admitted candidates. They are not calibrated estimates of real-world correctness. Ordered scores, permutation analysis and labelled calibration need separate contracts and evaluation cohorts.

The [Kev comparison](validation/go-system-one-kev-20260922.md) records related decision-model techniques and why Kev's learned pointer head is a separate checkpoint contract. The [v1 validation report](validation/go-system-one-v1-20260921.md) records the actual Gemma model, tokenizer, numerical and performance pins.
