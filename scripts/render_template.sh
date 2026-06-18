#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/render_template.sh --module-name NAME --module-path PATH --package-name NAME --out DIR

Renders redisx into a concrete base library by copying the repository,
moving pkg/redisx to pkg/<package>, and replacing template identifiers.
USAGE
}

module_name=""
module_path=""
package_name=""
out_dir=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --module-name)
      module_name="${2:-}"
      shift 2
      ;;
    --module-path)
      module_path="${2:-}"
      shift 2
      ;;
    --package-name)
      package_name="${2:-}"
      shift 2
      ;;
    --out)
      out_dir="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$module_name" || -z "$module_path" || -z "$package_name" || -z "$out_dir" ]]; then
  echo "ERROR: --module-name, --module-path, --package-name and --out are required" >&2
  usage >&2
  exit 2
fi

if [[ "$package_name" =~ [^a-zA-Z0-9_] || "$package_name" =~ ^[0-9] ]]; then
  echo "ERROR: --package-name must be a valid Go package identifier" >&2
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_abs="$(realpath "$repo_root")"
out_abs="$(realpath -m "$out_dir")"

if [[ "$out_abs" == "$repo_abs" || "$out_abs" == "$repo_abs"/* ]]; then
  echo "ERROR: output directory must be outside the template repository: $out_abs" >&2
  exit 2
fi

if [[ -e "$out_abs" && ! -d "$out_abs" ]]; then
  echo "ERROR: output path exists but is not a directory: $out_abs" >&2
  exit 2
fi

if [[ -e "$out_abs/.git" || -e "$out_abs/go.mod" ]]; then
  echo "ERROR: output directory looks like an existing repository: $out_abs" >&2
  exit 2
fi

if [[ -d "$out_abs" ]] && find "$out_abs" -mindepth 1 -maxdepth 1 | read -r _; then
  echo "ERROR: output directory must be empty: $out_abs" >&2
  exit 2
fi

mkdir -p "$out_abs"
out_dir="$out_abs"

copy_from_live_tree() {
  (
    cd "$repo_root"
    tar \
    --exclude='./.git' \
    --exclude='./.omc' \
    --exclude='./.omx' \
    --exclude='./.worktree' \
    --exclude='./.agent/inbox' \
    --exclude='./docs/adr' \
    --exclude='./docs/goal.md' \
    --exclude='./tmp' \
    --exclude='./dist' \
    --exclude='./node_modules' \
    --exclude='./coverage.out' \
    --exclude='./coverage.*' \
    --exclude='./*.coverprofile' \
    --exclude='./profile.cov' \
    --exclude='./release/manifest/latest.json' \
    --exclude='./release/manifest/latest.json.sha256' \
    --exclude='./release/standard-impact/latest.md' \
    --exclude='./release/downstream-sync/latest.md' \
    --exclude='./release/debt/latest.json' \
    --exclude='./release/debt/latest.md' \
    --exclude='./release/debt/latest.json.sha256' \
    -cf - .
  ) | (
    cd "$out_dir"
    tar -xf -
  )
}

prune_render_omissions() {
  rm -rf "$out_dir/.omc"
  rm -rf "$out_dir/.omx"
  rm -rf "$out_dir/.worktree"
  rm -rf "$out_dir/.agent/inbox"
  rm -rf "$out_dir/docs/adr"
  rm -f "$out_dir/docs/goal.md"
  rm -f "$out_dir/release/manifest/latest.json"
  rm -f "$out_dir/release/manifest/latest.json.sha256"
  rm -f "$out_dir/release/standard-impact/latest.md"
  rm -f "$out_dir/release/downstream-sync/latest.md"
  rm -f "$out_dir/release/debt/latest.json"
  rm -f "$out_dir/release/debt/latest.md"
  rm -f "$out_dir/release/debt/latest.json.sha256"
}

copy_from_git_archive() {
  git -C "$repo_root" archive --format=tar HEAD | (
    cd "$out_dir"
    tar -xf -
  )
  prune_render_omissions
}

use_git_archive=0
if [[ "${XLIB_RENDER_FORCE_GIT_ARCHIVE:-0}" == "1" ]]; then
  use_git_archive=1
elif git -C "$repo_root" rev-parse --is-inside-work-tree >/dev/null 2>&1 && \
  [[ -z "$(git -C "$repo_root" status --porcelain=v1 --untracked-files=no)" ]]; then
  use_git_archive=1
fi

if [[ "$use_git_archive" == "1" ]]; then
  copy_from_git_archive
else
  copy_from_live_tree
fi

# Raw inbox archives are intentionally omitted from rendered downstream repos.
# Keep the rendered control-plane index aligned with that reduced file set.
index_path="$out_dir/.agent/index.yaml"
if [[ -f "$index_path" ]]; then
  awk '
    /^  - path: \.agent\/inbox\// {
      skip = 1
      next
    }
    skip && /^    / {
      next
    }
    {
      skip = 0
      print
    }
  ' "$index_path" > "$index_path.tmp"
  mv "$index_path.tmp" "$index_path"
fi

if [[ "$package_name" != "redisx" ]]; then
  mkdir -p "$out_dir/pkg"
  mv "$out_dir/pkg/redisx" "$out_dir/pkg/$package_name"
fi

replace_in_text_files() {
  local find_text="$1"
  local replace_text="$2"

  while IFS= read -r -d '' file; do
    FIND_TEXT="$find_text" REPLACE_TEXT="$replace_text" perl -0pi -e 's/\Q$ENV{FIND_TEXT}\E/$ENV{REPLACE_TEXT}/g' "$file"
  done < <(
    find "$out_dir" -type f \( \
      -name '*.go' -o \
      -name '*.md' -o \
      -name '*.json' -o \
      -name '*.py' -o \
      -name '*.sh' -o \
      -name '*.yml' -o \
      -name '*.yaml' -o \
      -name '.env.example' -o \
      -name 'Makefile' -o \
      -name 'go.mod' \
    \) -print0
  )
}

downstream_standard_target_files=(
  "$out_dir/.agent/registries/downstream-adoption-status.yaml"
  "$out_dir/.agent/registries/downstream-registry.yaml"
  "$out_dir/docs/downstream-matrix.md"
  "$out_dir/scripts/check_standard_impact.sh"
)

protect_downstream_standard_target_tokens() {
  if [[ "$package_name" == "redisx" ]]; then
    return
  fi

  local sentinel="__XLIB_STANDARD_TARGET_CANONICAL_ADAPTER__"
  local file
  for file in "${downstream_standard_target_files[@]}"; do
    [[ -f "$file" ]] || continue
    SENTINEL="$sentinel" perl -0pi -e 's/redisx/$ENV{SENTINEL}/g' "$file"
  done
}

restore_downstream_standard_target_tokens() {
  if [[ "$package_name" == "redisx" ]]; then
    return
  fi

  local sentinel="__XLIB_STANDARD_TARGET_CANONICAL_ADAPTER__"
  local file
  for file in "${downstream_standard_target_files[@]}"; do
    [[ -f "$file" ]] || continue
    SENTINEL="$sentinel" perl -0pi -e 's/\Q$ENV{SENTINEL}\E/redisx/g' "$file"
  done
}

rename_if_exists() {
  local from="$1"
  local to="$2"

  if [[ "$from" == "$to" || ! -e "$from" ]]; then
    return
  fi
  if [[ -e "$to" ]]; then
    echo "ERROR: rendered path already exists: $to" >&2
    exit 1
  fi
  mkdir -p "$(dirname "$to")"
  mv "$from" "$to"
}

rename_rendered_template_paths() {
  local target_upper="$1"

  if [[ "$package_name" == "redisx" ]]; then
    return
  fi

  rename_if_exists "$out_dir/contracts/redisx.config.schema.json" "$out_dir/contracts/$package_name.config.schema.json"
  rename_if_exists "$out_dir/contracts/redisx.health.schema.json" "$out_dir/contracts/$package_name.health.schema.json"
  rename_if_exists "$out_dir/contracts/redisx.errors.yaml" "$out_dir/contracts/$package_name.errors.yaml"
  rename_if_exists "$out_dir/contracts/redisx.metrics.yaml" "$out_dir/contracts/$package_name.metrics.yaml"
  rename_if_exists "$out_dir/scripts/verify_l2_redisx.py" "$out_dir/scripts/verify_l2_$package_name.py"
  rename_if_exists "$out_dir/.agent/evidence/GOAL-20260604-REDISX-L2-STANDARD-FACTORY" "$out_dir/.agent/evidence/GOAL-20260604-$target_upper-L2-STANDARD-FACTORY"
  rename_if_exists "$out_dir/.agent/retrospectives/RETRO-20260604-redisx-l2.md" "$out_dir/.agent/retrospectives/RETRO-20260604-$package_name-l2.md"
  rename_if_exists "$out_dir/.agent/patches/PATCH-PROMPT-20260604-redisx-l2.md" "$out_dir/.agent/patches/PATCH-PROMPT-20260604-$package_name-l2.md"
  rename_if_exists "$out_dir/.agent/patches/PATCH-HARNESS-20260604-redisx-l2.md" "$out_dir/.agent/patches/PATCH-HARNESS-20260604-$package_name-l2.md"
  rename_if_exists "$out_dir/.agent/patches/PATCH-RULE-20260604-redisx-l2.md" "$out_dir/.agent/patches/PATCH-RULE-20260604-$package_name-l2.md"
}

protect_downstream_standard_target_tokens
replace_in_text_files 'redisx' "$module_name"
replace_in_text_files 'github.com/ZoneCNH/redisx' "$module_path"
replace_in_text_files 'redisx' "$package_name"
replace_in_text_files 'github.com/ZoneCNH/redisx' "$module_path"
replace_in_text_files 'github.com/ZoneCNH/redisx' "$module_path"
replace_in_text_files 'redisx' "$module_name"
replace_in_text_files 'redisx' "$module_name"
package_title="$(printf '%s%s' "$(printf '%s' "${package_name:0:1}" | tr '[:lower:]' '[:upper:]')" "${package_name:1}")"
package_upper="$(printf '%s' "$package_name" | tr '[:lower:]' '[:upper:]')"
replace_in_text_files 'redisx_' "${package_name}_"
replace_in_text_files 'Redisx' "$package_title"
replace_in_text_files 'REDISX' "$package_upper"
replace_in_text_files 'redisx' "$package_name"
rename_rendered_template_paths "$package_upper"
restore_downstream_standard_target_tokens

(
  cd "$out_dir"
  gofmt -w ./pkg ./internal ./contracts ./examples ./testkit
)

echo "rendered $module_name at $out_dir"
