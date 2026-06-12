#!/usr/bin/env bash
set -euo pipefail

run_go_integration() {
  REDISX_INTEGRATION=1 GOWORK="${GOWORK:-off}" go test ./pkg/redisx -run TestRedisIntegrationWithEnv -count=1
}

wait_for_container_redis() {
  local container="$1"
  local attempt

  for attempt in $(seq 1 30); do
    if docker exec "$container" redis-cli ping >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "Redis container did not become ready" >&2
  return 1
}

run_docker_integration() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker Redis integration requested but docker is not installed" >&2
    return 1
  fi
  if ! docker version >/dev/null 2>&1; then
    echo "Docker Redis integration requested but docker is unavailable" >&2
    return 1
  fi

  local data_dir container port marker actual
  data_dir="$(mktemp -d)"
  container="redisx-integration-$RANDOM-$$"
  trap 'docker rm -f "$container" >/dev/null 2>&1 || true; rm -rf "$data_dir"' EXIT

  docker run -d --rm \
    --name "$container" \
    -p 127.0.0.1::6379 \
    -v "$data_dir:/data" \
    redis:7-alpine \
    redis-server --appendonly yes --save 1 1 >/dev/null

  wait_for_container_redis "$container"
  port="$(docker port "$container" 6379/tcp | awk -F: 'NR == 1 {print $NF}')"
  if [ -z "$port" ]; then
    echo "Could not determine Redis container host port" >&2
    return 1
  fi

  marker="redisx-persistence-$RANDOM-$$"
  docker exec "$container" redis-cli SET redisx:persistence:marker "$marker" >/dev/null
  docker exec "$container" redis-cli SAVE >/dev/null
  docker restart "$container" >/dev/null
  wait_for_container_redis "$container"

  actual="$(docker exec "$container" redis-cli GET redisx:persistence:marker)"
  if [ "$actual" != "$marker" ]; then
    echo "Redis persistence restart check failed" >&2
    return 1
  fi

  REDISX_INTEGRATION=1 \
    REDISX_REDIS_ADDR="127.0.0.1:$port" \
    REDISX_REDIS_DB="${REDISX_REDIS_DB:-0}" \
    GOWORK="${GOWORK:-off}" \
    go test ./pkg/redisx -run TestRedisIntegrationWithEnv -count=1
}

if [ "${REDISX_INTEGRATION:-}" = "1" ] && [ -n "${REDISX_REDIS_ADDR:-}" ]; then
  echo "running env-gated Redis integration against configured Redis"
  run_go_integration
  exit 0
fi

if [ "${REDISX_INTEGRATION_DOCKER:-}" = "1" ]; then
  echo "running Docker Redis integration with persistence restart check"
  run_docker_integration
  exit 0
fi

echo "Redis integration skipped; set REDISX_INTEGRATION=1 with REDISX_REDIS_ADDR or REDISX_INTEGRATION_DOCKER=1"
