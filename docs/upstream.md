# Updating from go-pherence

`go-system-one` imports reviewed source from `go-pherence` in one direction. The update process never modifies the upstream checkout and never adds `go-pherence` as a Go module dependency.

## Pinned source

[`scripts/upstream.env`](../scripts/upstream.env) records the base source commit. [`scripts/upstream-files.tsv`](../scripts/upstream-files.tsv) maps upstream paths to local destinations. Its optional third column pins one file to another full commit. This permits a reviewed hand-off without importing unrelated files changed between the base and hand-off commits. [`scripts/local-packages.tsv`](../scripts/local-packages.tsv) maps upstream import paths to this module path after copying.

All compiled first-party packages live in this repository. `vendor/` contains third-party modules and licences only. Local packing, PTX and TypeSafe adapter changes can extend or modify imported files; an upstream pin records lineage, not equality with the current tree. Review incoming copies against those local changes before accepting an update.

## Import a revision

Use a full commit hash for the complete update:

```sh
./scripts/update-upstream.sh <full-go-pherence-commit>
```

The command requires a clean `go-system-one` tree. It copies the manifest, records the source pin, rewrites first-party imports, runs `go mod tidy`, regenerates third-party `vendor/` and runs `make check`.

Use the lower-level copy-only command when reviewing source before accepting the pin:

```sh
./scripts/sync-upstream.sh <full-go-pherence-commit>
```

The copy command makes a temporary partial clone by default. To avoid a network fetch, point it at an existing clean checkout:

```sh
GO_PHERENCE_SOURCE=/path/to/go-pherence \
  ./scripts/sync-upstream.sh <full-go-pherence-commit>
```

The checkout must have no tracked or untracked changes. This prevents an uncommitted experiment from entering a release snapshot.

## Review and accept

1. Read the complete diff. Resolve repository-specific imports without changing public contracts.
2. Add new imported files to `scripts/upstream-files.tsv`. Add package-path mappings to `scripts/local-packages.tsv` when a new package enters this module.
3. Update `scripts/upstream.env` for a complete import. For a selective hand-off, add its full commit to the affected manifest rows and leave the base pin unchanged.
4. Confirm no source imports `github.com/rcarmo/go-pherence/...` and no `go-pherence` requirement appears in `go.mod`, `go.sum` or `vendor/modules.txt`.
5. Run `go mod tidy`, `./scripts/vendor.sh` and `make check`.
6. Run released-model parity when model, tokenizer, scorer, PTX or dispatch behaviour changed.
7. Record new performance samples in `docs/benchmarks/data/`, refresh the benchmark tables and generated charts, and preserve historical datasets. Use `docs/performance/` for experiment decisions and `docs/validation/` for pinned correctness reports.
8. Commit copied source, the source pin, third-party dependency changes and evidence as one reviewable change.

## Automation contract

[`.github/workflows/upstream-update.yml`](../.github/workflows/upstream-update.yml) checks the upstream `main` head weekly and can also run manually for an explicit full commit. Before importing, `scripts/upstream-relevant.sh` compares Git blob IDs only for paths in `scripts/upstream-files.tsv`. If none changed, the workflow leaves the accepted pin unchanged and creates no pull request. Relevant changes open or update `automation/go-pherence-update`; the workflow never commits directly to `main`.

An automated update job must:

- use an explicit full upstream commit;
- skip commits that do not change a manifest-listed source path;
- refuse dirty source and destination checkouts;
- include the generated source diff and pin change;
- keep the standalone module free of `go-pherence` package dependencies;
- run offline checks;
- leave hardware tests marked as required when it cannot run them;
- avoid force pushes and writes to `go-pherence`;
- require review before merge.

The updater does not infer that every upstream change is relevant. The file manifest is the relevance boundary.
