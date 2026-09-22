# Updating from go-pherence

`go-system-one` imports reviewed work from `go-pherence` in one direction. The update process never modifies the upstream checkout.

## Pinned inputs

[`scripts/upstream.env`](../scripts/upstream.env) records the accepted source commit. [`scripts/upstream-files.tsv`](../scripts/upstream-files.tsv) maps product-owned upstream paths to their local destinations. `go.mod` separately pins the shared runtime module to an immutable pseudo-version.

These pins may advance together when an upstream change touches both product and runtime code. A documentation-only product update need not change the module dependency.

## Import a revision

Use a full commit hash:

```sh
./scripts/sync-upstream.sh <full-go-pherence-commit>
```

The script makes a temporary partial clone by default. To avoid a network fetch, point it at an existing clean checkout:

```sh
GO_PHERENCE_SOURCE=/path/to/go-pherence \
  ./scripts/sync-upstream.sh <full-go-pherence-commit>
```

The checkout must have no tracked or untracked changes. This prevents an uncommitted experiment from entering a release snapshot.

## Review and accept

1. Read the complete diff. Resolve repository-specific imports without changing the public API.
2. Add new product-owned files to `scripts/upstream-files.tsv`. Remove mappings only when this repository deliberately replaces or deletes the corresponding feature.
3. Update `scripts/upstream.env` with the accepted full commit and its UTC commit time.
4. If shared model, loader, SIMD or NVIDIA code changed, update the `go-pherence` pseudo-version in `go.mod` to the same accepted commit.
5. Run `go mod tidy` and `make check`.
6. Run the released-model parity gate when model, tokenizer, scorer, PTX or dispatch behaviour changed.
7. Record performance claims in `docs/validation/` with raw sample distributions and exact hardware details.
8. Commit the copied source, both pins and evidence as one reviewable change.

## Automation contract

An automated update job may open a pull request, but it must:

- use an explicit full upstream commit;
- refuse a dirty source checkout;
- include the generated diff and pin changes;
- run the offline checks;
- leave hardware tests marked as required when it cannot run them;
- avoid force pushes and writes to `go-pherence`;
- require review before merge.

The updater does not infer that every upstream change is relevant. The manifest is the relevance boundary.
