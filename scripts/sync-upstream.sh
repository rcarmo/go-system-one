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
else
  tmp=$(mktemp -d)
  git clone --filter=blob:none --no-checkout "$UPSTREAM_REPOSITORY" "$tmp/go-pherence"
  source_repo="$tmp/go-pherence"
fi

git -C "$source_repo" cat-file -e "$revision^{commit}"
if [[ -n "$(git -C "$source_repo" status --porcelain)" ]]; then
  printf 'refusing to sync from a dirty upstream checkout: %s\n' "$source_repo" >&2
  exit 1
fi

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

gofmt -w "$root/cmd" "$root/model/gosystemone" "$root/webui" "$root/internal/httpinput"

printf 'synced declared paths from go-pherence %s\n' "$(git -C "$source_repo" rev-parse "$revision^{commit}")"
printf 'review the diff, update scripts/upstream.env and go.mod deliberately, then run make check\n'
