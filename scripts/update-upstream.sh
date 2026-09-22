#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/scripts/upstream.env"

revision=${1:-}
if [[ -z "$revision" ]]; then
  printf 'usage: %s <full-go-pherence-commit>\n' "$0" >&2
  exit 2
fi
if [[ ! "$revision" =~ ^[0-9a-f]{40}$ ]]; then
  printf 'upstream revision must be a full lowercase commit hash: %s\n' "$revision" >&2
  exit 2
fi
if [[ -n "$(git -C "$root" status --porcelain)" ]]; then
  printf 'refusing to update a dirty go-system-one checkout\n' >&2
  exit 1
fi

tmp=$(mktemp -d)
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

git clone --filter=blob:none --no-checkout "$UPSTREAM_REPOSITORY" "$tmp/go-pherence"
git -C "$tmp/go-pherence" checkout --detach "$revision"
resolved=$(git -C "$tmp/go-pherence" rev-parse HEAD)
commit_epoch=$(git -C "$tmp/go-pherence" show -s --format=%ct HEAD)
committed_at=$(date -u -d "@$commit_epoch" '+%Y-%m-%dT%H:%M:%SZ')
pseudo_time=$(date -u -d "@$commit_epoch" '+%Y%m%d%H%M%S')
short=${resolved:0:12}

GO_PHERENCE_SOURCE="$tmp/go-pherence" "$root/scripts/sync-upstream.sh" "$resolved"

cat > "$root/scripts/upstream.env" <<EOF
# Source revision for product-owned files and the temporary core module dependency.
UPSTREAM_REPOSITORY=$UPSTREAM_REPOSITORY
UPSTREAM_COMMIT=$resolved
UPSTREAM_COMMITTED_AT=$committed_at
EOF

cd "$root"
go mod edit -require="github.com/rcarmo/go-pherence@v0.0.0-${pseudo_time}-${short}"
go mod tidy
"$root/scripts/vendor.sh"
make check

printf '\nupdated go-system-one working tree to go-pherence %s\n' "$resolved"
printf 'review the diff and run the released-model hardware gate before merging runtime changes\n'
