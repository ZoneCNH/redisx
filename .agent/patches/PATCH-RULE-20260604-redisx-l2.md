# PATCH-RULE-20260604 redisx L2

## Rule

A Redis factory cannot be marked L2 closeout-complete unless the default client path is offline-safe and Redis-specific contracts are present in the repository.

## Required evidence

- Public config/options binder or equivalent importable API.
- Redis error taxonomy identifiers exposed from `pkg/redisx` and documented under `contracts/`.
- Health and metrics contracts tied to public struct fields/constants.
- Public fake Redis provider in `testkit` with golden or contract assertions.
- Release evidence gates pass with `GOWORK=off` and generated `release/manifest/latest.json*` files are not committed.

## Enforcement note

If a future task adds a network Redis adapter, require an explicit option/provider selection and preserve the no-default-dial test.
