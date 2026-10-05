#!/usr/bin/env bash
# Source before tools, or execute as: scripts/project-env.sh <command> [args...].
# Keep installed model assets and retained validation evidence out of this tree.
project_env() {
  local relative component path repo
  repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
  source "$repo/scripts/project-tmp.sh"
  # Preserve the caller's platform temp root across nested Make/helper calls.
  export GO_SYSTEM_ONE_ORIGINAL_TMPDIR="${GO_SYSTEM_ONE_ORIGINAL_TMPDIR-${TMPDIR:-}}"
  # Resolve once, before assigning TMPDIR to a unique child run.
  PROJECT_TMP_ROOT=$(project_tmp_resolve go-system-one) || return
  export PROJECT_TMP_ROOT
  project_tmp_init "$PROJECT_TMP_ROOT" || return
  # Check every component below the project parent, including existing symlinks.
  project_owned_dir() {
    local target=$1 current=${PROJECT_TMP_ROOT%/*} part
    local -a parts
    [[ -n $current ]] || current=/
    [[ $target == "$PROJECT_TMP_ROOT" || $target == "$PROJECT_TMP_ROOT/"* ]] || return 2
    [[ ! -L $current ]] || { echo "Refusing symlink: $current" >&2; return 2; }
    mkdir -p "$current" || return
    local rest=${target#"${current%/}/"}
    IFS=/ read -r -a parts <<< "$rest"
    for part in "${parts[@]}"; do
      [[ -n $part && $part != . && $part != .. ]] || return 2
      current="${current%/}/$part"
      [[ ! -L $current ]] || { echo "Refusing symlink: $current" >&2; return 2; }
      if [[ -e $current ]]; then
        [[ -d $current && -O $current && -w $current && -x $current ]] || { echo "Not an owned directory: $current" >&2; return 2; }
      else
        mkdir "$current" || return
      fi
    done
  }
  for relative in cache/go-build cache/go-mod cache/go-path cache/xdg cache/bun cache/npm cache/playwright cache/cuda build runs/tools; do
    project_owned_dir "$PROJECT_TMP_ROOT/$relative" || return
  done
  if [[ -z ${PROJECT_RUN_DIR:-} ]]; then
    PROJECT_RUN_DIR=$(mktemp -d "$PROJECT_TMP_ROOT/runs/tools/run-XXXXXXXX") || return
  fi
  [[ $PROJECT_RUN_DIR == "$PROJECT_TMP_ROOT/runs/"*/* && $PROJECT_RUN_DIR != *'/../'* ]] || return 2
  project_owned_dir "$PROJECT_RUN_DIR" || return
  export PROJECT_RUN_DIR
  export TMPDIR="$PROJECT_RUN_DIR" TMP="$PROJECT_RUN_DIR" TEMP="$PROJECT_RUN_DIR" GOTMPDIR="$PROJECT_RUN_DIR"
  export GOCACHE="$PROJECT_TMP_ROOT/cache/go-build" GOMODCACHE="$PROJECT_TMP_ROOT/cache/go-mod" GOPATH="$PROJECT_TMP_ROOT/cache/go-path"
  export GO_SYSTEM_ONE_ARTIFACT_DIR="${GO_SYSTEM_ONE_ARTIFACT_DIR:-$HOME/.cache/go-system-one/v1}"
  # Explicit artifact overrides also serve isolated synthetic artifact tests.
  # Real model assets must remain in durable storage (see AGENTS.md).
  export XDG_CACHE_HOME="$PROJECT_TMP_ROOT/cache/xdg" BUN_INSTALL_CACHE_DIR="$PROJECT_TMP_ROOT/cache/bun" npm_config_cache="$PROJECT_TMP_ROOT/cache/npm"
  export PLAYWRIGHT_BROWSERS_PATH="$PROJECT_TMP_ROOT/cache/playwright" CUDA_CACHE_PATH="$PROJECT_TMP_ROOT/cache/cuda"
  export BUILD_DIR="$PROJECT_TMP_ROOT/build" DIST_DIR="$PROJECT_TMP_ROOT/build/dist"
  if [[ -z ${EVIDENCE_ROOT:-} ]]; then
    if [[ ${GITHUB_ACTIONS:-} == true ]]; then
      EVIDENCE_ROOT="${GITHUB_WORKSPACE:?}/go-system-one"
    elif [[ -d /workspace/notes && -w /workspace/notes ]]; then
      EVIDENCE_ROOT=/workspace/notes/validation/go-system-one
    else
      EVIDENCE_ROOT="$repo/validation-evidence"
    fi
  fi
  export EVIDENCE_ROOT
  export PROFILE_ROOT="${PROFILE_ROOT:-$EVIDENCE_ROOT/profiles}"
  for path in "$EVIDENCE_ROOT" "$PROFILE_ROOT"; do
    project_path_usable "$path" || { echo "Unsafe retained evidence path: $path" >&2; return 2; }
  done
  case "$EVIDENCE_ROOT/" in "$PROJECT_TMP_ROOT/"*) echo 'Evidence must be outside disposable scratch' >&2; return 2;; esac
  case "$PROFILE_ROOT/" in "$PROJECT_TMP_ROOT/"*) echo 'Profiles must be outside disposable scratch' >&2; return 2;; esac
}
project_env || { status=$?; return "$status" 2>/dev/null || exit "$status"; }
if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
  if [[ ${1:-} == --run ]]; then printf '%s\n' "$PROJECT_RUN_DIR"; else exec "$@"; fi
fi
