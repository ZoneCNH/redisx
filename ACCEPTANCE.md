# redisx v1.1.1 Acceptance

## Required local gates

Run from the repository root with `GOWORK=off` unless a target states otherwise:

```bash
GOWORK=off go test ./...
GOWORK=off make coverage-check
GOWORK=off go vet ./...
GOWORK=off make test-integration
REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration
XLIB_CONTEXT=release_verify GOWORK=off make release-check
XLIB_CONTEXT=release_verify GOWORK=off make release-preflight VERSION=v1.1.1
```

## Real Redis acceptance

CI must exercise Redis with a non-secret local service:

```bash
REDISX_INTEGRATION=1 REDISX_REDIS_ADDR=127.0.0.1:6379 REDISX_REDIS_DB=0 GOWORK=off make integration
```

Local dev secret handling must use the redacted wrapper when relying on `/home/ZoneCNH/sre/secrets/env/dev.md`:

```bash
DEV_ENV_FILE=/home/ZoneCNH/sre/secrets/env/dev.md GOWORK=off make test-dev-env-integration
```

The wrapper may print loaded key names, but must never print values from the secret file.

## Release evidence

- `pkg/redisx.Version`, goalcli governance version, `release/manifest/template.json`, docs, and Harness final gate must all point at `v1.1.1`.
- `FEATURES.md`, `ACCEPTANCE.md`, `README.md`, and `CHANGELOG.md` must describe the same Redis runtime surface and secret redaction boundary.
- `release/manifest/latest.json` and checksum files are generated evidence and remain uncommitted source artifacts.
