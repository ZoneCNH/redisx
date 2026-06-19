#!/usr/bin/env bash
set -euo pipefail

# Load only redisx integration variables from a local dev env file without
# printing values. The default file is intentionally external to the repo and
# must never be committed or echoed in CI logs.
DEV_ENV_FILE="${DEV_ENV_FILE:-/home/ZoneCNH/sre/secrets/env/dev.md}"

allowed_key() {
  case "$1" in
    REDISX_REDIS_ADDR|REDISX_REDIS_URL|REDISX_REDIS_USERNAME|REDISX_REDIS_PASSWORD|REDISX_REDIS_DB|REDISX_REDIS_TLS|REDISX_REDIS_CLIENT_NAME)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

trim() {
  local value="$1"
  value="${value#${value%%[![:space:]]*}}"
  value="${value%${value##*[![:space:]]}}"
  printf '%s' "$value"
}

strip_quotes() {
  local value="$1"
  if [[ "$value" == \"*\" && "$value" == *\" ]]; then
    value="${value:1:${#value}-2}"
  elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
    value="${value:1:${#value}-2}"
  fi
  printf '%s' "$value"
}

loaded_keys=()
if [ -f "$DEV_ENV_FILE" ]; then
  while IFS= read -r raw_line || [ -n "$raw_line" ]; do
    line="$(trim "$raw_line")"
    [ -z "$line" ] && continue
    [[ "$line" == \#* ]] && continue

    key=""
    value=""
    if [[ "$line" =~ ^export[[:space:]]+([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
      key="${BASH_REMATCH[1]}"
      value="${BASH_REMATCH[2]}"
    elif [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
      key="${BASH_REMATCH[1]}"
      value="${BASH_REMATCH[2]}"
    elif [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*:[[:space:]]*(.*)$ ]]; then
      key="${BASH_REMATCH[1]}"
      value="${BASH_REMATCH[2]}"
    elif [[ "$line" =~ ^\|[[:space:]]*([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*\|[[:space:]]*(.*)[[:space:]]*\|$ ]]; then
      key="${BASH_REMATCH[1]}"
      value="${BASH_REMATCH[2]}"
    fi

    [ -z "$key" ] && continue
    allowed_key "$key" || continue
    value="$(strip_quotes "$(trim "$value")")"
    [ -z "$value" ] && continue
    printf -v "$key" '%s' "$value"
    export "$key"
    loaded_keys+=("$key")
  done < "$DEV_ENV_FILE"
else
  echo "dev Redis env file not found at DEV_ENV_FILE path; set REDISX_REDIS_* directly or create the file" >&2
fi

if [ ${#loaded_keys[@]} -gt 0 ]; then
  printf 'loaded dev Redis integration key names:'
  printf ' %s' "${loaded_keys[@]}"
  printf '\n'
fi

if [ -z "${REDISX_REDIS_ADDR:-}" ] && [ -z "${REDISX_REDIS_URL:-}" ]; then
  echo "no Redis endpoint configured; expected REDISX_REDIS_ADDR or REDISX_REDIS_URL from env or DEV_ENV_FILE (values are not printed)" >&2
  exit 2
fi

GOWORK="${GOWORK:-off}" REDISX_INTEGRATION=1 ./scripts/run_redis_integration.sh
