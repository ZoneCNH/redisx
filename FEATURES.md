# redisx v1.1.1 Features

## Production-ready Redis surface

- String, TTL, multi-key, counter, hash, list, pipeline, cache-aside codec, lock token, and fixed-window rate-limit helpers remain the supported `pkg/redisx` public surface for v1.1.1.
- Durable Redis evidence covers non-TTL string, hash, list, counter, and pipeline writes across restart recovery.
- TTL-scoped locks, rate-limit windows, and pub/sub remain transient state and are not represented as durable persistence guarantees.

## Release and governance gates

- `GOWORK=off make coverage-check` enforces 100% coverage on the configured runtime/API package set.
- `GOWORK=off make integration` now has deterministic CI Redis service coverage through `.github/workflows/integration.yml`.
- `GOWORK=off make test-dev-env-integration` provides a local, redacted bridge to `/home/ZoneCNH/sre/secrets/env/dev.md` without printing or committing secret values.
- `XLIB_CONTEXT=release_verify GOWORK=off make release-preflight VERSION=v1.1.1` is the release preflight anchor.

## Evidence boundaries

- Runtime evidence remains under `.agent/evidence/l2/` and records profile status, coverage names, and environment variable names only.
- Secret values, passwords, TLS material, and raw dev env file contents are excluded from docs, logs, release manifests, and commits.
