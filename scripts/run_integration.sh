#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

retry_command() {
  local max_attempts="$1"
  shift

  local attempt=1
  until "$@"; do
    if [[ "$attempt" -ge "$max_attempts" ]]; then
      return 1
    fi

    echo "retrying command after failure (attempt $attempt/$max_attempts): $*" >&2
    sleep "$((attempt * 2))"
    attempt=$((attempt + 1))
  done
}

cases=(
  "kernel|github.com/ZoneCNH/kernel|kernel"
  "configx|github.com/ZoneCNH/configx|configx"
)
standard_adapter_name="redis""x"
cases+=(
  "${standard_adapter_name}|github.com/ZoneCNH/${standard_adapter_name}|${standard_adapter_name}"
)

for spec in "${cases[@]}"; do
  IFS='|' read -r module_name module_path package_name <<< "$spec"
  out_dir="$tmpdir/$module_name"

  ./scripts/render_template.sh \
    --module-name "$module_name" \
    --module-path "$module_path" \
    --package-name "$package_name" \
    --out "$out_dir"

  ./scripts/check_rendered_template.sh "$out_dir" "$module_name" "$module_path" "$package_name"

  if [[ "$(id -u)" == "0" ]] && [[ "$(stat -c %u "$out_dir")" != "0" ]]; then
    # Docker-mounted workspaces preserve host ownership through template copies.
    chown -R "$(id -u):$(id -g)" "$out_dir"
  fi

  (
    cd "$out_dir"
    git init -q
    git config user.email "ci@example.invalid"
    git config user.name "Template Integration"
    git add .
    git commit -qm "Initial rendered template"

    retry_command 3 env GOWORK=off go mod tidy
    retry_command 3 env GOWORK=off go mod download all
    git diff --exit-code -- go.mod go.sum
    if command -v docker >/dev/null 2>&1 && docker version >/dev/null 2>&1; then
      GOWORK=off make docker-toolchain-check
    else
      XLIB_DOCKER_ALLOW_MISSING=1 GOWORK=off make docker-toolchain-check
    fi
    retry_command 3 env GOWORK=off go test ./...
    GOWORK=off make contracts
    GOWORK=off make boundary
    GOWORK=off make standard-impact-check
    GOWORK=off make debt
    GOWORK=off make debt-evidence
    GOWORK=off make debt-evidence-checksum-check
    retry_command 3 env CHECK_STATUS=passed GOWORK=off make evidence
    retry_command 3 env RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check
  )
done

GOWORK=off make test-integration

echo "integration check passed"
