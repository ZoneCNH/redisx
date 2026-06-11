# PATCH-HARNESS-20260604 redisx L2

## Harness gap

Existing harnesses verified generic contracts but did not fail when Redis-specific contracts or fake Redis test surfaces were absent.

## Patch

Add targeted harness assertions for Redis-labeled factory goals:

- `contracts/redisx.config.schema.json` maps to public `redisx.Options`.
- `contracts/redisx.health.schema.json` requires `component`, `latency_ms`, and `error_class` fields.
- `contracts/redisx.errors.yaml` documents all public Redis error identifiers.
- `contracts/redisx.metrics.yaml` documents all public Redis/client metric constants.
- `testkit.NewFakeRedis()` satisfies `redisx.Provider` and runs without Redis network configuration.

## Evidence command

Use `GOWORK=off go test ./contracts ./pkg/redisx ./testkit` as the narrow closeout harness, followed by full `GOWORK=off go test ./...`.
