# PATCH-HARNESS-20260605-001 — Redis L2 harness evidence gates

## Problem

Generic contract and evidence gates can pass without asserting Redis-specific contract artifact names or fake-provider isolation.

## Patch

Add or preserve harness checks that assert:

- `contracts/redisx.config.schema.json`, `contracts/redisx.health.schema.json`, `contracts/redisx.errors.yaml`, and `contracts/redisx.metrics.yaml` exist and are exercised by Go tests.
- `testkit.NewFakeRedis()` is public and usable through `redisx.WithProvider`.
- Unit tests do not import go-redis, call `net.Dial`, or require localhost Redis by default.
- Release evidence gates run with `CHECK_STATUS=passed GOWORK=off make evidence` and `RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check`.

## Acceptance

The harness patch is accepted when failed Redis contract/fake-provider assertions fail locally before release evidence is emitted.
