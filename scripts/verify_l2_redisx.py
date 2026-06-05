#!/usr/bin/env python3
"""Validate redisx L2 shape without requiring provider connections."""
from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / ".agent" / "l2-capabilities.yaml"
EVIDENCE_DIR = ROOT / ".agent" / "evidence" / "l2"
READINESS = EVIDENCE_DIR / "release-readiness.json"
COMPLIANCE = EVIDENCE_DIR / "compliance-matrix.json"

REQUIRED_STANDARD_FILES = [
    ".agent/gates/l2gate.yaml",
    ".agent/registry/l2-contract-packs.yaml",
    ".agent/registry/l2-capability-families.yaml",
    ".agent/registry/l2-golden-samples.yaml",
    ".agent/registry/l2-release-levels.yaml",
    ".agent/schemas/l2-capabilities.schema.json",
    ".agent/schemas/l2-contract-packs.schema.json",
    ".agent/schemas/l2-release-readiness.schema.json",
    ".agent/schemas/l2-compliance-matrix.schema.json",
]
REQUIRED_PACKS = {"common", "kv", "ttl", "pool"}
REQUIRED_PROFILES = {"unit", "contract", "integration"}
ALLOWED_CAPABILITY_STATUS = {"declared", "implemented", "unsupported", "deprecated"}
ALLOWED_EVIDENCE_STATUS = {"pass", "fail", "missing", "not_applicable"}
FORBIDDEN_MANIFEST_KEYS = {
    "provider_endpoint",
    "provider_credentials",
    "secret",
    "password",
    "token",
}


def rel(path: Path) -> str:
    return str(path.relative_to(ROOT))


def load_json(path: Path) -> Any:
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def require_file(path: Path) -> None:
    require(path.exists(), f"missing required file: {rel(path)}")


def strip_scalar(value: str) -> str:
    value = value.split("#", 1)[0].strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
        return value[1:-1]
    return value


def split_key_value(text: str) -> tuple[str, str]:
    key, _, value = text.partition(":")
    return key.strip(), strip_scalar(value)


def top_level_scalar(lines: list[str], name: str) -> str | None:
    prefix = f"{name}:"
    for line in lines:
        if line.startswith(prefix):
            return strip_scalar(line[len(prefix) :])
    return None


def top_level_block(lines: list[str], name: str) -> list[str]:
    start = -1
    marker = f"{name}:"
    for index, line in enumerate(lines):
        if line.startswith(marker):
            start = index + 1
            break
    if start < 0:
        return []

    end = len(lines)
    for index in range(start, len(lines)):
        line = lines[index]
        if line and not line.startswith(" ") and not line.startswith("#"):
            end = index
            break
    return lines[start:end]


def scalar_list(block: list[str], indent: int = 2) -> list[str]:
    marker = " " * indent + "- "
    return [strip_scalar(line[len(marker) :]) for line in block if line.startswith(marker)]


def parse_map_block(block: list[str], indent: int = 2) -> dict[str, str]:
    prefix = " " * indent
    result: dict[str, str] = {}
    for line in block:
        if not line.startswith(prefix) or line.startswith(prefix + " ") or ":" not in line:
            continue
        key, value = split_key_value(line.strip())
        result[key] = value
    return result


def parse_capabilities(block: list[str]) -> list[dict[str, str]]:
    capabilities: list[dict[str, str]] = []
    current: dict[str, str] | None = None
    for line in block:
        if line.startswith("  - "):
            if current is not None:
                capabilities.append(current)
            current = {}
            item = line[4:].strip()
            if item and ":" in item:
                key, value = split_key_value(item)
                current[key] = value
            continue
        if current is not None and line.startswith("    ") and ":" in line:
            key, value = split_key_value(line.strip())
            current[key] = value
    if current is not None:
        capabilities.append(current)
    return capabilities


def nested_scalar_list(block: list[str], name: str) -> list[str]:
    marker = f"  {name}:"
    start = -1
    for index, line in enumerate(block):
        if line.startswith(marker):
            start = index + 1
            break
    if start < 0:
        return []

    nested: list[str] = []
    for line in block[start:]:
        if line.startswith("  ") and not line.startswith("    "):
            break
        nested.append(line)
    return scalar_list(nested, indent=4)


def manifest_keys(lines: list[str]) -> list[str]:
    keys: list[str] = []
    for line in lines:
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if stripped.startswith("- "):
            stripped = stripped[2:].strip()
        if ":" in stripped:
            key, _ = split_key_value(stripped)
            keys.append(key)
    return keys


def load_manifest(path: Path) -> dict[str, Any]:
    lines = path.read_text(encoding="utf-8").splitlines()
    adapter = parse_map_block(top_level_block(lines, "adapter"))
    capabilities = parse_capabilities(top_level_block(lines, "capabilities"))
    contract_packs = scalar_list(top_level_block(lines, "contract_packs"))
    evidence_block = top_level_block(lines, "evidence")
    evidence = parse_map_block(evidence_block)
    evidence["required_profiles"] = nested_scalar_list(evidence_block, "required_profiles")
    evidence["reports"] = nested_scalar_list(evidence_block, "reports")

    return {
        "schema_version": top_level_scalar(lines, "schema_version"),
        "layer": top_level_scalar(lines, "layer"),
        "adapter": adapter,
        "capabilities": capabilities,
        "contract_packs": contract_packs,
        "evidence": evidence,
        "_keys": manifest_keys(lines),
    }


def validate_manifest() -> dict[str, Any]:
    require_file(MANIFEST)
    for item in REQUIRED_STANDARD_FILES:
        require_file(ROOT / item)

    manifest = load_manifest(MANIFEST)
    require(isinstance(manifest, dict), "L2 manifest must be a YAML object")
    require(manifest.get("schema_version") == "1.0", "manifest schema_version must be 1.0")
    require(manifest.get("layer") == "L2", "manifest layer must be L2")

    adapter = manifest.get("adapter", {})
    require(adapter.get("name") == "redisx", "adapter.name must be redisx")
    require(adapter.get("module") == "github.com/ZoneCNH/redisx", "adapter.module must match redisx module")
    require(adapter.get("family") == "key_value", "adapter.family must be key_value")

    forbidden = [key for key in manifest["_keys"] if key.lower() in FORBIDDEN_MANIFEST_KEYS]
    require(not forbidden, f"manifest contains forbidden provider-boundary keys: {forbidden}")

    capabilities = manifest.get("capabilities", [])
    require(isinstance(capabilities, list) and capabilities, "manifest capabilities must be a non-empty list")
    invalid_status = [
        capability.get("name")
        for capability in capabilities
        if capability.get("status") not in ALLOWED_CAPABILITY_STATUS
    ]
    require(not invalid_status, f"capabilities with invalid status: {invalid_status}")

    packs = set(manifest.get("contract_packs", []))
    require(REQUIRED_PACKS <= packs, f"manifest missing contract packs: {sorted(REQUIRED_PACKS - packs)}")

    evidence = manifest.get("evidence", {})
    profiles = set(evidence.get("required_profiles", []))
    require(REQUIRED_PROFILES <= profiles, f"manifest missing required profiles: {sorted(REQUIRED_PROFILES - profiles)}")
    require(evidence.get("output_dir") == ".agent/evidence/l2", "manifest evidence.output_dir must be .agent/evidence/l2")
    reports = set(evidence.get("reports", []))
    for report in [
        ".agent/evidence/l2/contract-report.json",
        ".agent/evidence/l2/integration-report.json",
        ".agent/evidence/l2/compliance-matrix.json",
        ".agent/evidence/l2/release-readiness.json",
    ]:
        require(report in reports, f"manifest missing evidence report: {report}")

    return {
        "adapter": adapter["name"],
        "packs": sorted(packs),
        "profiles": sorted(profiles),
        "capabilities": len(capabilities),
        "standard_files": len(REQUIRED_STANDARD_FILES),
    }


def evidence_status(entries: list[dict[str, Any]], profile: str) -> str | None:
    for entry in entries:
        if entry.get("profile") == profile:
            return entry.get("status")
    return None


def validate_readiness() -> dict[str, Any]:
    require_file(READINESS)
    data = load_json(READINESS)
    require(data.get("schema_version") == "1.0", "readiness schema_version must be 1.0")
    require(data.get("adapter") == "redisx", "readiness adapter must be redisx")
    require(data.get("target_level") == "L2-T2", "readiness target_level must be L2-T2")
    require(isinstance(data.get("score"), int), "readiness score must be an integer")
    require(0 <= data["score"] <= 100, "readiness score must be between 0 and 100")

    profiles = set(data.get("profiles", []))
    require(REQUIRED_PROFILES <= profiles, f"readiness missing profiles: {sorted(REQUIRED_PROFILES - profiles)}")

    entries = data.get("evidence", [])
    require(isinstance(entries, list) and entries, "readiness evidence must be a non-empty list")
    for entry in entries:
        require(entry.get("status") in ALLOWED_EVIDENCE_STATUS, f"invalid readiness evidence status: {entry}")
        status = entry.get("status")
        path = entry.get("path", "")
        if status == "pass" and path.startswith(".agent/"):
            require((ROOT / path).exists(), f"passing evidence path does not exist: {path}")

    statuses = {profile: evidence_status(entries, profile) for profile in REQUIRED_PROFILES}
    require(statuses["unit"] == "pass", "unit evidence must currently be pass")
    release_ready = all(statuses.get(profile) == "pass" for profile in REQUIRED_PROFILES) and data["score"] >= 75
    return {
        "target_level": data["target_level"],
        "score": data["score"],
        "statuses": statuses,
        "release_ready": release_ready,
    }


def validate_compliance() -> dict[str, Any]:
    require_file(COMPLIANCE)
    data = load_json(COMPLIANCE)
    require(data.get("schema_version") == "1.0", "compliance schema_version must be 1.0")
    require(data.get("adapter") == "redisx", "compliance adapter must be redisx")
    rows = data.get("rows", [])
    require(isinstance(rows, list) and rows, "compliance rows must be a non-empty list")

    seen_packs: set[str] = set()
    by_status: dict[str, int] = {}
    for row in rows:
        for field in ["requirement", "contract_pack", "profile", "evidence", "status"]:
            require(row.get(field), f"compliance row missing {field}: {row}")
        require(row["status"] in ALLOWED_EVIDENCE_STATUS, f"invalid compliance row status: {row}")
        require(row["contract_pack"] in REQUIRED_PACKS, f"unexpected compliance contract pack: {row}")
        seen_packs.add(row["contract_pack"])
        by_status[row["status"]] = by_status.get(row["status"], 0) + 1

    require(REQUIRED_PACKS <= seen_packs, f"compliance missing packs: {sorted(REQUIRED_PACKS - seen_packs)}")
    return {"rows": len(rows), "packs": sorted(seen_packs), "status_counts": by_status}


def validate_evidence() -> dict[str, Any]:
    require(EVIDENCE_DIR.exists(), "missing L2 evidence directory")
    return {
        "readiness": validate_readiness(),
        "compliance": validate_compliance(),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest-only", action="store_true")
    parser.add_argument("--evidence-only", action="store_true")
    parser.add_argument("--readiness-only", action="store_true")
    args = parser.parse_args()

    selected = {
        "manifest": args.manifest_only,
        "evidence": args.evidence_only,
        "readiness": args.readiness_only,
    }
    run_all = not any(selected.values())

    result: dict[str, Any] = {"status": "PASS"}
    if run_all or args.manifest_only:
        result["manifest"] = validate_manifest()
    if run_all or args.evidence_only:
        result["evidence"] = validate_evidence()
    if args.readiness_only:
        result["readiness"] = validate_readiness()

    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
