#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

DEV_ENV_FILE="${DEV_ENV_FILE:-/home/ZoneCNH/sre/secrets/env/dev.md}"
REPORT_PATH="$ROOT/.agent/evidence/l2/dev-env-config-report.json"
EXPORT_PATH="$(mktemp)"
TEST_COMMAND="GOWORK=${GOWORK:-off} REDISX_INTEGRATION=1 go test ./pkg/redisx -run '^TestRedisIntegrationWithEnv$' -count=1"

cleanup() {
  rm -f "$EXPORT_PATH"
}
trap cleanup EXIT

set +e
python3 - "$DEV_ENV_FILE" "$EXPORT_PATH" "$REPORT_PATH" "$TEST_COMMAND" <<'PY'
import json
import re
import shlex
import sys
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import unquote, urlparse

dev_env_file = Path(sys.argv[1])
export_path = Path(sys.argv[2])
report_path = Path(sys.argv[3])
test_command = sys.argv[4]

key_map = {
    "REDISX_REDIS_ADDR": "REDISX_REDIS_ADDR",
    "REDIS_ADDR": "REDISX_REDIS_ADDR",
    "REDISX_REDIS_USERNAME": "REDISX_REDIS_USERNAME",
    "REDIS_USERNAME": "REDISX_REDIS_USERNAME",
    "REDIS_USER": "REDISX_REDIS_USERNAME",
    "REDISX_REDIS_PASSWORD": "REDISX_REDIS_PASSWORD",
    "REDIS_PASSWORD": "REDISX_REDIS_PASSWORD",
    "REDISX_REDIS_DB": "REDISX_REDIS_DB",
    "REDIS_DB": "REDISX_REDIS_DB",
    "REDISX_REDIS_URL": "REDIS_URL",
    "REDIS_URL": "REDIS_URL",
}
host_keys = {"REDIS_HOST", "REDISX_REDIS_HOST"}
port_keys = {"REDIS_PORT", "REDISX_REDIS_PORT"}
assign_re = re.compile(
    r"(?:^|[\s`|])(?:export\s+)?([A-Z][A-Z0-9_]*)\s*=\s*(\"[^\"]*\"|'[^']*'|[^`\s|]+)"
)


def write_stable_json_report(path, data):
    try:
        existing = json.loads(path.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError, OSError):
        existing = None

    generated_at = datetime.now(timezone.utc).isoformat()
    if isinstance(existing, dict):
        existing_body = dict(existing)
        existing_body.pop("generated_at", None)
        data_body = dict(data)
        data_body.pop("generated_at", None)
        if existing_body == data_body and existing.get("generated_at"):
            generated_at = existing["generated_at"]

    data["generated_at"] = generated_at
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=False) + "\n", encoding="utf-8")


def write_report(status, reason, matched_keys=None, config_keys=None):
    scenario_status = "pass" if status == "pass" else status
    file_status = "fail" if status == "fail" else "pass"
    endpoint_status = "not_applicable" if status == "not_applicable" else scenario_status
    data = {
        "schema_version": "1.0",
        "adapter": "redisx",
        "target_level": "L2-T2",
        "profile": "dev-env-config",
        "status": status,
        "score": 100 if status in {"pass", "not_applicable"} else 0,
        "env_gate": "DEV_ENV_FILE",
        "credential_source": "external dev env only; values are not committed or printed",
        "command": test_command,
        "commands": [
            "GOWORK=off make test-dev-env-integration",
            test_command,
        ],
        "evidence_paths": [
            "scripts/run_dev_redis_integration.sh",
            "scripts/run_redis_integration.sh",
            "pkg/redisx/redis_integration_test.go",
            "test/integration/README.md",
            ".agent/evidence/l2/dev-env-config-report.json",
        ],
        "config_keys": sorted(config_keys or []),
        "matched_keys": sorted(matched_keys or []),
        "checklist": [
            {"name": "dev_env_file_readable", "status": file_status},
            {"name": "supported_redis_endpoint_detected", "status": endpoint_status},
            {"name": "secret_values_redacted", "status": "pass"},
        ],
        "runtime": "dev env configuration probe; secret values are held in process env only when a supported endpoint is detected",
        "data_hygiene": "report records variable names only; Redis values are never printed or persisted",
    }
    if reason:
        data["reason"] = reason
    write_stable_json_report(report_path, data)


def parse_value(raw):
    raw = raw.strip().rstrip(",;")
    if not raw:
        return ""
    try:
        parsed = shlex.split(raw, comments=False, posix=True)
    except ValueError:
        parsed = []
    if parsed:
        return parsed[0]
    return raw.strip("\"'")


def parse_assignments(text):
    found = {}
    for match in assign_re.finditer(text):
        key = match.group(1)
        value = parse_value(match.group(2))
        if value:
            found[key] = value
    return found


def parse_url(value):
    parsed = urlparse(value)
    if parsed.scheme not in {"redis", "rediss"} or not parsed.hostname:
        return {}
    result = {"REDISX_REDIS_ADDR": f"{parsed.hostname}:{parsed.port or 6379}"}
    if parsed.username:
        result["REDISX_REDIS_USERNAME"] = unquote(parsed.username)
    if parsed.password:
        result["REDISX_REDIS_PASSWORD"] = unquote(parsed.password)
    db = parsed.path.lstrip("/")
    if db:
        result["REDISX_REDIS_DB"] = db
    return result


def shell_export_line(key, value):
    return f"export {key}={shlex.quote(value)}"


if not dev_env_file.is_file():
    write_report("fail", "DEV_ENV_FILE is missing or not a regular file")
    sys.exit(2)

raw_text = dev_env_file.read_text(encoding="utf-8", errors="replace")
assignments = parse_assignments(raw_text)
matched_keys = []
config = {}
host = ""
port = ""

for source_key, value in assignments.items():
    if source_key in key_map:
        matched_keys.append(source_key)
        canonical_key = key_map[source_key]
        if canonical_key == "REDIS_URL":
            config.update(parse_url(value))
        else:
            config[canonical_key] = value
    elif source_key in host_keys:
        matched_keys.append(source_key)
        host = value
    elif source_key in port_keys:
        matched_keys.append(source_key)
        port = value

if "REDISX_REDIS_ADDR" not in config and host and port:
    config["REDISX_REDIS_ADDR"] = f"{host}:{port}"

config_keys = set(config)
if "REDISX_REDIS_ADDR" not in config:
    write_report(
        "not_applicable",
        "DEV_ENV_FILE is readable but contains no supported Redis endpoint assignment",
        matched_keys=matched_keys,
        config_keys=config_keys,
    )
    sys.exit(0)

export_lines = [
    shell_export_line(key, config[key])
    for key in [
        "REDISX_REDIS_ADDR",
        "REDISX_REDIS_USERNAME",
        "REDISX_REDIS_PASSWORD",
        "REDISX_REDIS_DB",
    ]
    if key in config and config[key]
]
export_path.write_text("\n".join(export_lines) + "\n", encoding="utf-8")
write_report(
    "configured",
    "supported Redis endpoint assignment detected; running redacted live integration",
    matched_keys=matched_keys,
    config_keys=config_keys,
)
sys.exit(10)
PY
classifier_rc=$?
set -e

case "$classifier_rc" in
  0)
    echo "dev env Redis integration not applicable; no supported Redis endpoint assignment found"
    exit 0
    ;;
  10)
    echo "running dev env Redis integration with redacted configuration"
    ;;
  *)
    echo "dev env Redis integration configuration check failed"
    exit "$classifier_rc"
    ;;
esac

# The export file is generated from allowlisted keys only and is removed by trap.
# Do not enable shell tracing around this source operation.
set +x
. "$EXPORT_PATH"

set +e
GOWORK="${GOWORK:-off}" REDISX_INTEGRATION=1 go test ./pkg/redisx -run '^TestRedisIntegrationWithEnv$' -count=1
rc=$?
set -e

REDISX_DEV_ENV_REPORT="$REPORT_PATH" REDISX_DEV_ENV_STATUS="$rc" python3 - <<'PY'
import json
import os
from datetime import datetime, timezone
from pathlib import Path

path = Path(os.environ["REDISX_DEV_ENV_REPORT"])
rc = int(os.environ["REDISX_DEV_ENV_STATUS"])
data = json.loads(path.read_text(encoding="utf-8"))


def write_stable_json_report(path, data):
    try:
        existing = json.loads(path.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError, OSError):
        existing = None

    generated_at = datetime.now(timezone.utc).isoformat()
    if isinstance(existing, dict):
        existing_body = dict(existing)
        existing_body.pop("generated_at", None)
        data_body = dict(data)
        data_body.pop("generated_at", None)
        if existing_body == data_body and existing.get("generated_at"):
            generated_at = existing["generated_at"]

    data["generated_at"] = generated_at
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=False) + "\n", encoding="utf-8")


if rc == 0:
    data["status"] = "pass"
    data["score"] = 100
    data["reason"] = "dev env backed Redis integration passed"
    checklist_status = "pass"
else:
    data["status"] = "fail"
    data["score"] = 0
    data["reason"] = f"dev env backed Redis integration exited with status {rc}"
    checklist_status = "fail"
for item in data.get("checklist", []):
    if item.get("name") in {"supported_redis_endpoint_detected", "dev_env_file_readable"}:
        item["status"] = checklist_status
write_stable_json_report(path, data)
PY

exit "$rc"
