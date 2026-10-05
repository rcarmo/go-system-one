#!/usr/bin/env bash
# Static/environment checks only; Go tests run through test-profile.sh.
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$root/scripts/project-env.sh"
for name in TMPDIR TMP TEMP GOTMPDIR GOCACHE GOMODCACHE GOPATH XDG_CACHE_HOME BUN_INSTALL_CACHE_DIR npm_config_cache PLAYWRIGHT_BROWSERS_PATH CUDA_CACHE_PATH; do
  [[ ${!name} == "$PROJECT_TMP_ROOT/"* ]] || { echo "$name escaped project scratch" >&2; exit 1; }
done
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
mkdir "$fixture/owned"
ln -s "$fixture/owned" "$fixture/link"
if (project_owned_dir "$fixture/link/child") >/dev/null 2>&1; then
  echo 'symlink path accepted' >&2; exit 1
fi
if (project_owned_dir "$fixture/../escape") >/dev/null 2>&1; then
  echo 'parent traversal accepted' >&2; exit 1
fi
if PROJECT_TMP_ROOT="$fixture/wrong" bash "$root/scripts/project-env.sh" true >/dev/null 2>&1; then
  echo 'ad-hoc project root accepted' >&2; exit 1
fi
# Fallback resolution is tested beneath this isolated fixture; no real /tmp mutation.
mkdir -p "$fixture/workspace/tmp" "$fixture/runner" "$fixture/original"
resolved=$(unset PROJECT_TMP_ROOT GO_SYSTEM_ONE_ORIGINAL_TMPDIR; RUNNER_TEMP="$fixture/runner" TMPDIR="$fixture/original" project_tmp_resolve go-system-one "$fixture/workspace/tmp")
[[ $resolved == "$fixture/workspace/tmp/go-system-one" ]]
resolved=$(unset PROJECT_TMP_ROOT GO_SYSTEM_ONE_ORIGINAL_TMPDIR; RUNNER_TEMP="$fixture/runner" TMPDIR="$fixture/original" project_tmp_resolve go-system-one "$fixture/absent/host/tmp")
[[ $resolved == "$fixture/runner/go-system-one" ]]
resolved=$(unset PROJECT_TMP_ROOT RUNNER_TEMP GO_SYSTEM_ONE_ORIGINAL_TMPDIR; TMPDIR="$fixture/original" project_tmp_resolve go-system-one "$fixture/absent/host/tmp")
[[ $resolved == "$fixture/original/go-system-one" ]]
resolved=$(unset PROJECT_TMP_ROOT RUNNER_TEMP TMPDIR GO_SYSTEM_ONE_ORIGINAL_TMPDIR; project_tmp_resolve go-system-one "$fixture/absent/host/tmp")
[[ $resolved == /tmp/go-system-one ]]
PROJECT_TMP_ROOT="$fixture/owned/go-system-one" project_tmp_resolve go-system-one >/dev/null
for invalid in relative/go-system-one "$fixture/owned/../go-system-one" "$fixture/link/go-system-one" ''; do
  if (PROJECT_TMP_ROOT="$invalid" project_tmp_resolve go-system-one) >/dev/null 2>&1; then
    echo "invalid override accepted: $invalid" >&2; exit 1
  fi
done
resolved=$(unset PROJECT_TMP_ROOT RUNNER_TEMP; GO_SYSTEM_ONE_ORIGINAL_TMPDIR="$fixture/original" TMPDIR="$fixture/owned" project_tmp_resolve go-system-one "$fixture/absent/host/tmp")
[[ $resolved == "$fixture/original/go-system-one" ]]
# Re-sourcing must preserve the root rather than append under the new TMPDIR.
before=$PROJECT_TMP_ROOT
source "$root/scripts/project-env.sh"
[[ $PROJECT_TMP_ROOT == "$before" ]]
# Clean is deliberately limited; refusing an arbitrary target must preserve data.
printf keep > "$fixture/owned/sentinel"
if "$root/scripts/clean-build.sh" "$fixture/owned" >/dev/null 2>&1; then
  echo 'arbitrary cleanup target accepted' >&2; exit 1
fi
[[ $(cat "$fixture/owned/sentinel") == keep ]]
clean_root="$fixture/cleanup/go-system-one"
mkdir -p "$clean_root/build" "$clean_root/cache" "$clean_root/runs"
printf keep > "$clean_root/cache/sentinel"
printf keep > "$clean_root/runs/sentinel"
printf remove > "$clean_root/build/generated"
PROJECT_TMP_ROOT="$clean_root" "$root/scripts/clean-build.sh"
[[ ! -e $clean_root/build && -f $clean_root/cache/sentinel && -f $clean_root/runs/sentinel ]]
ln -s "$fixture/owned" "$clean_root/build"
if PROJECT_TMP_ROOT="$clean_root" "$root/scripts/clean-build.sh" >/dev/null 2>&1; then
  echo 'symlink cleanup target accepted' >&2; exit 1
fi
[[ $(cat "$fixture/owned/sentinel") == keep ]]
echo 'Project path routing, symlink/traversal rejection and cleanup scope passed.'
