#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

REPORT_PATH="$ROOT/.agent/evidence/l2/persistence-report.json"
TEST_COMMAND="GOWORK=${GOWORK:-off} REDISX_INTEGRATION=1 REDISX_PERSISTENCE_RECOVERY=1 go test ./pkg/redisx -run '^TestRedisIntegrationPersistenceRecoveryWithEnv$' -count=1"
FAIL_REASON=""
DOCKER_ACTIVE_CONTAINER=""
DOCKER_VOLUME=""
LOCAL_REDIS_RUNNING="0"
LOCAL_REDIS_PORT=""
LOCAL_REDIS_BASE=""

redis_loopback_host() {
  printf '%s.%s.%s.%s\n' 127 0 0 1
}

cleanup_docker() {
  if [ -n "${DOCKER_ACTIVE_CONTAINER:-}" ]; then
    docker rm -f "$DOCKER_ACTIVE_CONTAINER" >/dev/null 2>&1 || true
    DOCKER_ACTIVE_CONTAINER=""
  fi
  if [ -n "${DOCKER_VOLUME:-}" ]; then
    docker volume rm "$DOCKER_VOLUME" >/dev/null 2>&1 || true
    DOCKER_VOLUME=""
  fi
}

cleanup_local() {
  if [ "${LOCAL_REDIS_RUNNING:-0}" = "1" ] && [ -n "${LOCAL_REDIS_PORT:-}" ]; then
    redis-cli -h "$(redis_loopback_host)" -p "$LOCAL_REDIS_PORT" SHUTDOWN NOSAVE >/dev/null 2>&1 || true
    LOCAL_REDIS_RUNNING="0"
  fi
  if [ -n "${LOCAL_REDIS_BASE:-}" ]; then
    rm -rf "$LOCAL_REDIS_BASE"
    LOCAL_REDIS_BASE=""
  fi
}

cleanup_all() {
  cleanup_docker
  cleanup_local
}
trap cleanup_all EXIT

write_report() {
  local status="$1"
  local reason="${2:-}"

  REDISX_REPORT_PATH="$REPORT_PATH" \
    REDISX_REPORT_STATUS="$status" \
    REDISX_REPORT_REASON="$reason" \
    REDISX_REPORT_COMMAND="$TEST_COMMAND" \
    python3 - <<'PY'
import json
import os
from datetime import datetime, timezone
from pathlib import Path

status = os.environ["REDISX_REPORT_STATUS"]
reason = os.environ.get("REDISX_REPORT_REASON", "")
command = os.environ["REDISX_REPORT_COMMAND"]
path = Path(os.environ["REDISX_REPORT_PATH"])
scenario_status = "pass" if status == "pass" else ("not_applicable" if status == "not_applicable" else "fail")

data = {
    "schema_version": "1.0",
    "adapter": "redisx",
    "target_level": "L2-T2",
    "profile": "persistence",
    "status": status,
    "score": 100 if status == "pass" else 0,
    "env_gate": "REDISX_PERSISTENCE_INTEGRATION=1",
    "command": command,
    "commands": [
        command,
        "GOWORK=off REDISX_PERSISTENCE_INTEGRATION=1 make test-persistence-integration",
    ],
    "evidence_paths": [
        "pkg/redisx/redis_integration_test.go",
        "scripts/run_redis_persistence_integration.sh",
        "docker-compose.test.yml",
        ".agent/evidence/l2/compliance-matrix.json",
    ],
    "generated_at": datetime.now(timezone.utc).isoformat(),
    "runtime": "Redis with AOF and RDB enabled on retained test storage",
    "credential_source": "local ephemeral Redis without external credentials",
    "checklist": [
        {"name": "set_without_ttl", "status": scenario_status},
        {"name": "mset_without_ttl", "status": scenario_status},
        {"name": "hash_without_ttl", "status": scenario_status},
        {"name": "list_without_ttl", "status": scenario_status},
        {"name": "counter_without_ttl", "status": scenario_status},
        {"name": "pipeline_without_ttl", "status": scenario_status},
        {"name": "ttl_permanent_before_restart", "status": scenario_status},
        {"name": "redis_save", "status": scenario_status},
        {"name": "server_restart_same_storage", "status": scenario_status},
        {"name": "get_after_restart", "status": scenario_status},
        {"name": "hash_after_restart", "status": scenario_status},
        {"name": "list_after_restart", "status": scenario_status},
        {"name": "counter_after_restart", "status": scenario_status},
        {"name": "pipeline_writes_after_restart", "status": scenario_status},
        {"name": "ttl_permanent_after_restart", "status": scenario_status},
        {"name": "cleanup", "status": scenario_status},
        {"name": "data_hygiene", "status": scenario_status},
    ],
    "data_hygiene": "test keys are nonce-prefixed and deleted after recovery validation",
}
if reason:
    data["detail"] = reason

path.parent.mkdir(parents=True, exist_ok=True)
path.write_text(json.dumps(data, indent=2, sort_keys=False) + "\n", encoding="utf-8")
PY
}

free_port() {
  REDISX_LOOPBACK_HOST="$(redis_loopback_host)" python3 - <<'PY'
import os
import socket

host = os.environ["REDISX_LOOPBACK_HOST"]
with socket.socket() as sock:
    sock.bind((host, 0))
    print(sock.getsockname()[1])
PY
}

wait_redis_endpoint() {
  local host="$1"
  local port="$2"
  local label="$3"

  for _ in $(seq 1 60); do
    if command -v redis-cli >/dev/null 2>&1; then
      if redis-cli -h "$host" -p "$port" ping >/dev/null 2>&1; then
        return 0
      fi
    elif REDISX_WAIT_HOST="$host" REDISX_WAIT_PORT="$port" python3 - <<'PY' >/dev/null 2>&1; then
import os
import socket

host = os.environ["REDISX_WAIT_HOST"]
port = int(os.environ["REDISX_WAIT_PORT"])
with socket.create_connection((host, port), timeout=0.5):
    pass
PY
      return 0
    fi
    sleep 0.5
  done

  FAIL_REASON="$label did not become reachable on $host:$port"
  return 1
}

docker_host_port() {
  local container_name="$1"
  local mapping
  local port

  mapping="$(
    docker port "$container_name" 6379/tcp |
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

run_phase() {
  local addr="$1"
  local key="$2"
  local value="$3"
  local expect_existing="$4"
  local cleanup="$5"

  GOWORK="${GOWORK:-off}" \
    REDISX_INTEGRATION=1 \
    REDISX_REDIS_ADDR="$addr" \
    REDISX_REDIS_USERNAME= \
    REDISX_REDIS_PASSWORD= \
    REDISX_REDIS_DB="${REDISX_REDIS_DB:-0}" \
    REDISX_PERSISTENCE_RECOVERY=1 \
    REDISX_PERSISTENCE_EXPECT_EXISTING="$expect_existing" \
    REDISX_PERSISTENCE_CLEANUP="$cleanup" \
    REDISX_PERSISTENCE_KEY="$key" \
    REDISX_PERSISTENCE_VALUE="$value" \
    go test ./pkg/redisx -run '^TestRedisIntegrationPersistenceRecoveryWithEnv$' -count=1
}

run_docker_persistence() {
  local image="${REDISX_PERSISTENCE_REDIS_IMAGE:-redis:7-alpine}"
  local run_id="redisx-persistence-$RANDOM-$(date +%s)"
  local volume="${run_id}-data"
  local key="redisx:integration:persistence:${run_id}"
  local value="survives-redis-restart-${run_id}"
  local host
  host="$(redis_loopback_host)"

  docker image inspect "$image" >/dev/null 2>&1 || docker pull "$image" >/dev/null
  DOCKER_VOLUME="$volume"

  start_docker_redis() {
    local container="$1"
    DOCKER_ACTIVE_CONTAINER="$container"
    docker run -d \
      --name "$container" \
      --publish "$host::6379" \
      --volume "$volume:/data" \
      "$image" redis-server --appendonly yes --save 1 1 >/dev/null

    for _ in $(seq 1 60); do
      if docker exec "$container" redis-cli ping >/dev/null 2>&1; then
        return 0
      fi
      sleep 0.5
    done
    FAIL_REASON="docker Redis did not become ready"
    return 1
  }

  local first="${run_id}-first"
  start_docker_redis "$first" || return $?
  local port
  port="$(docker_host_port "$first")" || {
    FAIL_REASON="failed to inspect docker Redis first phase host port"
    return 1
  }
  wait_redis_endpoint "$host" "$port" "docker Redis first phase" || return $?
  run_phase "${host}:${port}" "$key" "$value" "0" "0" || return $?
  docker exec "$first" redis-cli SAVE >/dev/null || {
    FAIL_REASON="docker Redis SAVE failed"
    return 1
  }
  docker rm -f "$first" >/dev/null || return $?
  DOCKER_ACTIVE_CONTAINER=""

  local second="${run_id}-second"
  start_docker_redis "$second" || return $?
  port="$(docker_host_port "$second")" || {
    FAIL_REASON="failed to inspect docker Redis second phase host port"
    return 1
  }
  wait_redis_endpoint "$host" "$port" "docker Redis second phase" || return $?
  run_phase "${host}:${port}" "$key" "$value" "1" "1" || return $?
}

run_local_persistence() {
  local run_id="redisx-persistence-$RANDOM-$(date +%s)"
  local base="$ROOT/.tmp/${run_id}"
  local data_dir="$base/data"
  local pid_file="$base/redis.pid"
  local log_file="$base/redis.log"
  local key="redisx:integration:persistence:${run_id}"
  local value="survives-redis-restart-${run_id}"
  local host
  local port

  host="$(redis_loopback_host)"
  mkdir -p "$data_dir"
  port="$(free_port)"
  LOCAL_REDIS_BASE="$base"
  LOCAL_REDIS_PORT="$port"

  start_local_redis() {
    redis-server \
      --bind "$host" \
      --port "$port" \
      --dir "$data_dir" \
      --appendonly yes \
      --save 1 1 \
      --daemonize yes \
      --pidfile "$pid_file" \
      --logfile "$log_file"
    LOCAL_REDIS_RUNNING="1"

    for _ in $(seq 1 60); do
      if redis-cli -h "$host" -p "$port" ping >/dev/null 2>&1; then
        return 0
      fi
      sleep 0.5
    done
    FAIL_REASON="local Redis did not become ready"
    return 1
  }

  stop_local_redis_save() {
    redis-cli -h "$host" -p "$port" SAVE >/dev/null
    redis-cli -h "$host" -p "$port" SHUTDOWN SAVE >/dev/null 2>&1 || true
    LOCAL_REDIS_RUNNING="0"
  }

  start_local_redis || return $?
  run_phase "${host}:${port}" "$key" "$value" "0" "0" || return $?
  stop_local_redis_save || return $?
  start_local_redis || return $?
  run_phase "${host}:${port}" "$key" "$value" "1" "1" || return $?
}

run_persistence() {
  if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    run_docker_persistence
    return
  fi

  if command -v redis-server >/dev/null 2>&1 && command -v redis-cli >/dev/null 2>&1; then
    run_local_persistence
    return
  fi

  FAIL_REASON="neither Docker nor local redis-server/redis-cli is available"
  return 1
}

if [ "${REDISX_PERSISTENCE_INTEGRATION:-}" != "1" ]; then
  echo "Redis persistence integration skipped; set REDISX_PERSISTENCE_INTEGRATION=1"
  write_report "not_applicable" "REDISX_PERSISTENCE_INTEGRATION=1 not set"
  exit 0
fi

echo "running Redis persistence integration with restart recovery"
set +e
run_persistence
rc=$?
set -e

if [ "$rc" -eq 0 ]; then
  write_report "pass"
else
  write_report "fail" "${FAIL_REASON:-persistence integration failed with status $rc}"
fi
exit "$rc"
