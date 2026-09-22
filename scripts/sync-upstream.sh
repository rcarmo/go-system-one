#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/scripts/upstream.env"

upstream=${GO_PHERENCE_SOURCE:-}
revision=${1:-$UPSTREAM_COMMIT}

cleanup() {
  if [[ -n ${tmp:-} ]]; then
    rm -rf "$tmp"
  fi
}
trap cleanup EXIT

if [[ -n "$upstream" ]]; then
  if [[ ! -d "$upstream/.git" ]]; then
    printf 'GO_PHERENCE_SOURCE is not a Git checkout: %s\n' "$upstream" >&2
    exit 1
  fi
  source_repo=$(cd "$upstream" && pwd)
  if [[ -n "$(git -C "$source_repo" status --porcelain)" ]]; then
    printf 'refusing to sync from a dirty upstream checkout: %s\n' "$source_repo" >&2
    exit 1
  fi
else
  tmp=$(mktemp -d)
  git clone --filter=blob:none --no-checkout "$UPSTREAM_REPOSITORY" "$tmp/go-pherence"
  source_repo="$tmp/go-pherence"
  git -C "$source_repo" checkout --detach "$revision"
fi

git -C "$source_repo" cat-file -e "$revision^{commit}"

while IFS=$'\t' read -r source_path destination_path; do
  [[ -n "$source_path" && ${source_path:0:1} != "#" ]] || continue
  mkdir -p "$root/$(dirname "$destination_path")"
  git -C "$source_repo" show "$revision:$source_path" > "$root/$destination_path"
done < "$root/scripts/upstream-files.tsv"

# These packages are owned here. Keep their imports local after copying them from
# the monorepo; heavyweight inference packages remain on the pinned core module.
find "$root/cmd" "$root/model/gosystemone" "$root/webui" "$root/internal/httpinput" \
  -type f -name '*.go' -print0 | xargs -0 sed -i \
  -e 's#github.com/rcarmo/go-pherence/model/gosystemone#github.com/rcarmo/go-system-one/model/gosystemone#g' \
  -e 's#github.com/rcarmo/go-pherence/internal/httpinput#github.com/rcarmo/go-system-one/internal/httpinput#g' \
  -e 's#github.com/rcarmo/go-pherence/webui#github.com/rcarmo/go-system-one/webui#g'

# Preserve every package boundary already internalised by this repository.
while IFS=$'\t' read -r upstream_import local_import mode; do
  [[ -n "$upstream_import" && ${upstream_import:0:1} != "#" ]] || continue
  mapfile -d '' files < <(grep -rlZ --exclude-dir=.git --exclude-dir=vendor --include='*.go' "\"$upstream_import\"" "$root" || true)
  if ((${#files[@]})); then
    sed -i "s#\"$upstream_import\"#\"$local_import\"#g" "${files[@]}"
  fi
done < "$root/scripts/local-packages.tsv"

# Use standalone artifact environment names in imported Go tests and docs.
mapfile -d '' artifact_files < <(grep -rlZ --include='*.go' --include='*.md' \
  'GO_PHERENCE_GO_SYSTEM_ONE_GEMMA4_12B' "$root/model" "$root/loader" "$root/docs" "$root/cmd" || true)
if ((${#artifact_files[@]})); then
  sed -i \
    -e 's/GO_PHERENCE_GO_SYSTEM_ONE_GEMMA4_12B_TOKENIZER/GO_SYSTEM_ONE_TOKENIZER_DIR/g' \
    -e 's/GO_PHERENCE_GO_SYSTEM_ONE_GEMMA4_12B/GO_SYSTEM_ONE_MODEL/g' \
    "${artifact_files[@]}"
fi

# Preserve standalone documentation links after importing monorepo-relative text.
sed -i \
  -e 's#\[Kev porting roadmap\](../models/kev-porting-roadmap.md)#[upstream Kev porting roadmap](https://github.com/rcarmo/go-pherence/blob/'"$revision"'/docs/models/kev-porting-roadmap.md)#' \
  "$root/docs/validation/go-system-one-kev-20260922.md"
sed -i \
  -e 's#\[CPU SIMD gap note\](../../docs/performance/gemma4-cpu-simd-gap.md)#[upstream CPU SIMD gap note](https://github.com/rcarmo/go-pherence/blob/'"$revision"'/docs/performance/gemma4-cpu-simd-gap.md)#' \
  "$root/loader/gguf/README.md"
sed -i \
  -e 's#See the \[asset migration notes\](../docs/guides/model-assets.md) for older checkouts\.#See the [external artifact contract](../docs/artifacts.md) for pinned model and tokenizer requirements.#' \
  "$root/model/README.md"
sed -Ei \
  -e 's#go-pherence@[0-9a-f]{40}#go-pherence@'"$revision"'#' \
  -e 's#go-pherence/commit/[0-9a-f]{40}#go-pherence/commit/'"$revision"'#' \
  "$root/README.md"
if [[ -f "$root/cmd/go-system-one/README.md" ]]; then
  perl -0pi -e 's#```sh\ngo run \./cmd/llm/go-system-one \\\n  -model .*?\n```#```sh\nmake run BACKEND=nvidia LISTEN=127.0.0.1:8080\n```#s' "$root/cmd/go-system-one/README.md"
  sed -i \
    -e 's#../../../docs/images/go-system-one-playground.png#../../docs/images/go-system-one-light-desktop.png#' \
    -e 's#../../../docs/validation/go-system-one-v1-20260921.md#../../docs/validation/go-system-one-v1-20260921.md#' \
    "$root/cmd/go-system-one/README.md"
fi

mapfile -d '' go_files < <(find "$root" -type f -name '*.go' -not -path "$root/.git/*" -not -path "$root/vendor/*" -print0)
if ((${#go_files[@]})); then
  gofmt -w "${go_files[@]}"
fi

printf 'synced declared paths from go-pherence %s\n' "$(git -C "$source_repo" rev-parse "$revision^{commit}")"
printf 'review the diff, update scripts/upstream.env, then run make check\n'
