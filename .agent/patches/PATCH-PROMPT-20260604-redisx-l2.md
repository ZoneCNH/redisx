# PATCH-PROMPT-20260604 redisx L2

## Trigger

A L2 standard factory closeout claims Redis readiness without Redis-specific public contracts, testkit fake provider, or REQ audit evidence.

## Patch

Prompt future agents to verify all Redis-labeled factory goals include:

1. A public options/config binding surface.
2. Redis-specific config, health, errors, and metrics contract files.
3. A public fake Redis provider for tests.
4. Evidence that default tests do not dial real Redis.
5. A REQ-001..014 audit linked from `.agent/index.yaml`.

## Validator prompt

Before accepting closeout, ask: "Can a downstream package import only public redisx/testkit APIs and run contract tests offline with GOWORK=off?" If not, the closeout is incomplete.
