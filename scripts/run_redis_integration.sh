#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

REPORT_PATH="$ROOT/.agent/evidence/l2/integration-report.json"
TEST_COMMAND="GOWORK=${GOWORK:-off} REDISX_INTEGRATION=1 go test ./pkg/redisx -run '^TestRedisIntegrationWithEnv$' -count=1"

write_report() {
  local status="$1"
  local reason="${2:-}"

  REDISX_REPORT_PATH="$REPORT_PATH" \
    REDISX_REPORT_STATUS="$status" \
    REDISX_REPORT_REASON="$reason" \
    REDISX_REPORT_COMMAND="$TEST_COMMAND" \
    REDISX_REPORT_RUNTIME="${REDISX_REPORT_RUNTIME:-real Redis selected by REDISX_REDIS_* at test time}" \
    python3 - <<'PY'
import json
import os
from datetime import datetime, timezone
from pathlib import Path

status = os.environ["REDISX_REPORT_STATUS"]
reason = os.environ.get("REDISX_REPORT_REASON", "")
command = os.environ["REDISX_REPORT_COMMAND"]
runtime = os.environ["REDISX_REPORT_RUNTIME"]
path = Path(os.environ["REDISX_REPORT_PATH"])
scenario_status = "pass" if status == "pass" else "fail"
config_keys = [
    key
    for key in [
        "REDISX_REDIS_ADDR",
        "REDISX_REDIS_USERNAME",
        "REDISX_REDIS_PASSWORD",
        "REDISX_REDIS_DB",
    ]
    if os.environ.get(key)
]

data = {
    "schema_version": "1.0",
    "adapter": "redisx",
    "target_level": "L2-T2",
    "profile": "integration",
    "status": status,
    "score": 100 if status == "pass" else 0,
    "env_gate": "REDISX_INTEGRATION=1",
    "credential_source": "external environment only; values are not committed or printed",
    "command": command,
    "commands": [
        command,
        "GOWORK=off REDISX_INTEGRATION_DOCKER=1 make test-integration",
    ],
    "evidence_paths": [
        "pkg/redisx/redis_integration_test.go",
        "scripts/run_redis_integration.sh",
        "docker-compose.yml",
        ".agent/evidence/l2/compliance-matrix.json",
    ],
    "generated_at": datetime.now(timezone.utc).isoformat(),
    "config_keys": config_keys,
    "checklist": [
        {"name": "ping", "status": scenario_status},
        {"name": "health", "status": scenario_status},
        {"name": "health_check", "status": scenario_status},
        {"name": "set", "status": scenario_status},
        {"name": "get", "status": scenario_status},
        {"name": "ttl_permanent", "status": scenario_status},
        {"name": "set_with_ttl", "status": scenario_status},
        {"name": "ttl_missing", "status": scenario_status},
        {"name": "missing_get_nil", "status": scenario_status},
        {"name": "mset", "status": scenario_status},
        {"name": "mget", "status": scenario_status},
        {"name": "hset", "status": scenario_status},
        {"name": "hget", "status": scenario_status},
        {"name": "hgetall", "status": scenario_status},
        {"name": "hdel", "status": scenario_status},
        {"name": "lpush", "status": scenario_status},
        {"name": "rpush", "status": scenario_status},
        {"name": "lrange", "status": scenario_status},
        {"name": "llen", "status": scenario_status},
        {"name": "pipeline_set_get", "status": scenario_status},
        {"name": "pipeline_missing_reads", "status": scenario_status},
        {"name": "pipeline_hash", "status": scenario_status},
        {"name": "pipeline_list", "status": scenario_status},
        {"name": "pipeline_counter", "status": scenario_status},
        {"name": "incr", "status": scenario_status},
        {"name": "decr", "status": scenario_status},
        {"name": "hash_hset_hget_hgetall_hdel", "status": scenario_status},
        {"name": "list_lpush_rpush_lpop_rpop_lrange", "status": scenario_status},
        {"name": "setnx", "status": scenario_status},
        {"name": "lock_acquire_release", "status": scenario_status},
        {"name": "fixed_window_rate_limit", "status": scenario_status},
        {"name": "pipeline_set_get_hset_rpush_incr", "status": scenario_status},
        {"name": "validation_error", "status": scenario_status},
        {"name": "expire_existing", "status": scenario_status},
        {"name": "expire_missing", "status": scenario_status},
        {"name": "exists", "status": scenario_status},
        {"name": "del", "status": scenario_status},
        {"name": "close", "status": scenario_status},
        {"name": "client_reconnect_read", "status": scenario_status},
    ],
    "runtime": runtime,
    "data_hygiene": "test keys are nonce-prefixed and deleted during cleanup",
}
if reason:
    data["failure"] = reason

path.parent.mkdir(parents=True, exist_ok=True)
path.write_text(json.dumps(data, indent=2, sort_keys=False) + "\n", encoding="utf-8")
PY
}

run_env_integration() {
  echo "running env-gated Redis integration against configured Redis"
  GOWORK="${GOWORK:-off}" REDISX_INTEGRATION=1 go test ./pkg/redisx -run '^TestRedisIntegrationWithEnv$' -count=1
}

wait_for_container_redis() {
  local container_name="$1"

  for _ in $(seq 1 40); do
    if docker exec "$container_name" redis-cli PING >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done

  return 1
}

wait_for_host_redis() {
  local host="$1"
  local port="$2"

  for _ in $(seq 1 40); do
    if command -v redis-cli >/dev/null 2>&1; then
      if redis-cli -h "$host" -p "$port" PING >/dev/null 2>&1; then
        return 0
      fi
    else
      if REDISX_HOST="$host" REDISX_PORT="$port" python3 - <<'PY' >/dev/null 2>&1
import os
import socket

host = os.environ["REDISX_HOST"]
port = int(os.environ["REDISX_PORT"])
with socket.create_connection((host, port), timeout=1.0):
    pass
PY
      then
        return 0
      fi
    fi
    sleep 0.25
  done

  return 1
}

redis_integration_port() {
  printf '%s%s\n' 63 79
}

redis_loopback_host() {
  printf '%s.%s.%s.%s\n' 127 0 0 1
}

docker_host_port() {
  local container_name="$1"
  local mapping
  local port

  mapping="$(
    docker port "$container_name" "$(redis_integration_port)/tcp" |
      awk '/127[.]0[.]0[.]1/ { print; found = 1; exit } NR == 1 { first = $0 } END { if (!found && first) print first }'
  )"
  port="${mapping##*:}"

  case "$port" in
    ""|*[!0-9]*)
      return 1
      ;;
    *)
      printf '%s\n' "$port"
      ;;
  esac
}

run_docker_integration() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "Redis Docker integration requested but docker is unavailable"
    return 2
  fi

  local image="${REDISX_INTEGRATION_DOCKER_IMAGE:-redis:7-alpine}"
  local container_name="redisx-integration-$$"
  local volume_name="redisx-integration-data-$$"
  local marker_value="redisx-integration-marker-$$"
  local host
  local host_port
  local persisted_value
  host="$(redis_loopback_host)"

  cleanup() {
    docker rm -f "$container_name" >/dev/null 2>&1 || true
    docker volume rm "$volume_name" >/dev/null 2>&1 || true
  }

  cleanup

  if ! docker volume create "$volume_name" >/dev/null; then
    echo "failed to create Redis integration Docker volume"
    cleanup
    return 1
  fi

  if ! docker run -d \
    --name "$container_name" \
    -p "$host::$(redis_integration_port)" \
    -v "$volume_name:/data" \
    "$image" \
    redis-server --appendonly yes --save 1 1 >/dev/null; then
    echo "failed to start Redis integration Docker container"
    cleanup
    return 1
  fi

  if ! wait_for_container_redis "$container_name"; then
    echo "Redis integration Docker container did not become ready"
    cleanup
    return 1
  fi

  if ! docker exec "$container_name" redis-cli SET redisx:persistence:marker "$marker_value" >/dev/null; then
    echo "failed to write Redis Docker persistence marker"
    cleanup
    return 1
  fi

  if ! docker exec "$container_name" redis-cli SAVE >/dev/null; then
    echo "failed to force Redis Docker persistence snapshot"
    cleanup
    return 1
  fi

  if ! docker restart "$container_name" >/dev/null; then
    echo "failed to restart Redis Docker container"
    cleanup
    return 1
  fi

  if ! wait_for_container_redis "$container_name"; then
    echo "Redis integration Docker container was not ready after restart"
    cleanup
    return 1
  fi

  if ! persisted_value="$(docker exec "$container_name" redis-cli GET redisx:persistence:marker)"; then
    echo "failed to read Redis Docker persistence marker after restart"
    cleanup
    return 1
  fi

  if [ "$persisted_value" != "$marker_value" ]; then
    echo "Redis Docker persistence marker did not survive restart"
    cleanup
    return 1
  fi

  if ! host_port="$(docker_host_port "$container_name")"; then
    echo "failed to inspect Redis Docker host port"
    cleanup
    return 1
  fi

  if ! wait_for_host_redis "$host" "$host_port"; then
    echo "Redis integration Docker host endpoint did not become reachable"
    cleanup
    return 1
  fi

  echo "running Docker-backed Redis integration with persistence restart check"
  REDISX_INTEGRATION=1 \
    REDISX_REDIS_ADDR="$host:$host_port" \
    REDISX_REDIS_DB="${REDISX_REDIS_DB:-0}" \
    GOWORK="${GOWORK:-off}" \
    go test ./pkg/redisx -run '^TestRedisIntegrationWithEnv$' -count=1
  local rc=$?

  cleanup
  return "$rc"
}

report_runtime="real Redis selected by REDISX_REDIS_* at test time"

if [ "${REDISX_INTEGRATION:-}" = "1" ] && [ -n "${REDISX_REDIS_ADDR:-}" ]; then
  set +e
  run_env_integration
  rc=$?
  set -e
elif [ "${REDISX_INTEGRATION_DOCKER:-}" = "1" ]; then
  report_runtime="Docker Redis selected by REDISX_INTEGRATION_DOCKER=1 with restart persistence preflight"
  set +e
  run_docker_integration
  rc=$?
  set -e
else
  echo "Redis integration skipped; set REDISX_INTEGRATION=1 with REDISX_REDIS_ADDR, or set REDISX_INTEGRATION_DOCKER=1"
  exit 0
fi

if [ "$rc" -eq 0 ]; then
  REDISX_REPORT_RUNTIME="$report_runtime" write_report "pass"
else
  REDISX_REPORT_RUNTIME="$report_runtime" write_report "fail" "Redis integration exited with status $rc"
fi
exit "$rc"
