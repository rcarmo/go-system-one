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
# Policy amendment: CI ignores an available workspace; local ignores CI temp vars.
mkdir -p "$fixture/workspace/tmp" "$fixture/runner" "$fixture/original"
resolve_fixture() (
  unset PROJECT_TMP_ROOT PROJECT_TMP_BASE CI GITHUB_ACTIONS GITLAB_CI TF_BUILD CIRCLECI
  export PROJECT_ORIGINAL_TMPDIR="$fixture/original" RUNNER_TEMP="$fixture/runner"
  case "$1" in
    local) project_tmp_resolve go-system-one "$fixture/workspace/tmp";;
    local-absent) project_tmp_resolve go-system-one "$fixture/absent/host/tmp";;
    ci) CI=true project_tmp_resolve go-system-one "$fixture/workspace/tmp";;
    ci-original) unset RUNNER_TEMP; CI=true project_tmp_resolve go-system-one "$fixture/workspace/tmp";;
    ci-system) unset RUNNER_TEMP; PROJECT_ORIGINAL_TMPDIR= CI=true project_tmp_resolve go-system-one "$fixture/workspace/tmp";;
  esac
)
[[ $(resolve_fixture local) == "$fixture/workspace/tmp/go-system-one" ]]
[[ $(resolve_fixture local-absent) == /tmp/go-system-one ]]
[[ $(resolve_fixture ci) == "$fixture/runner/go-system-one" ]]
[[ $(resolve_fixture ci-original) == "$fixture/original/go-system-one" ]]
[[ $(resolve_fixture ci-system) == /tmp/go-system-one ]]
(
  unset PROJECT_TMP_ROOT PROJECT_TMP_BASE
  [[ $(PROJECT_TMP_BASE="$fixture/owned" project_tmp_resolve go-system-one) == "$fixture/owned/go-system-one" ]]
  PROJECT_TMP_BASE="$fixture/owned" PROJECT_TMP_ROOT="$fixture/owned/go-system-one" project_tmp_resolve go-system-one >/dev/null
  if PROJECT_TMP_BASE="$fixture/owned" PROJECT_TMP_ROOT="$fixture/other/go-system-one" project_tmp_resolve go-system-one >/dev/null 2>&1; then exit 1; fi
  for invalid in relative "$fixture/link" "$fixture/owned/../elsewhere" ''; do
    if PROJECT_TMP_BASE="$invalid" project_tmp_resolve go-system-one >/dev/null 2>&1; then exit 1; fi
  done
)
for invalid in relative/go-system-one "$fixture/owned/../go-system-one" "$fixture/link/go-system-one" ''; do
  if (unset PROJECT_TMP_BASE; PROJECT_TMP_ROOT="$invalid" project_tmp_resolve go-system-one) >/dev/null 2>&1; then
    echo "invalid override accepted: $invalid" >&2; exit 1
  fi
done
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
