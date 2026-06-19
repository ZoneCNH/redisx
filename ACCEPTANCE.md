# ACCEPTANCE.md

Production readiness acceptance for release `v1.1.1`.

The stop condition is a clean commit containing synchronized release anchors, root feature/acceptance documentation, no committed secret values, and fresh verification evidence from `GOWORK=off` commands.

| Gate | Acceptance criterion | Evidence command / source |
| --- | --- | --- |
| Version anchors | Latest changelog entry, `pkg/redisx.Version`, goalcli governance version, release manifest template, Harness final gate, and release docs all use `v1.1.1`. | `GOWORK=off go test ./cmd/goalcli -run TestVersionConstantsTrackChangelogRelease -count=1` |
| Documentation completeness | Root docs include `FEATURES.md` and `ACCEPTANCE.md`; docs-check tracks them with the standard docs set. | `GOWORK=off make docs-check` |
| Unit/runtime coverage | Unit and contract behavior pass under workspace isolation. Redis runtime/API coverage remains governed by the 100% coverage gate. | `GOWORK=off make test`; `GOWORK=off make coverage-check` |
| Live Redis integration | Ping/Health, KV, TTL, multi-key, counter, hash, list, pipeline, delete, reconnect, lock, and rate-limit behavior pass against Redis. | `REDISX_INTEGRATION=1 GOWORK=off make test-integration` |
| Persistence integration | Permanent string, hash, list, counter, and pipeline writes survive Redis restart. TTL-scoped lock/rate-limit keys and pub/sub are excluded from durable persistence. | `REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration` |
| Release preflight | Release branch is clean, versioned, changelogged, lintable, and ready for manifest/release evidence generation. | `XLIB_CONTEXT=release_verify GOWORK=off make release-preflight VERSION=v1.1.1` from clean `main` |
| CI/CD safety | Integration and release workflows use `GOWORK=off`; release evidence remains redacted and reproducible. | `.github/workflows/integration.yml`, `.github/workflows/release.yml`, `.agent/release/release-required-gates.yaml` |
| Secret hygiene | `/home/ZoneCNH/sre/secrets/env/dev.md` may be used only as a controlled local credential source. Do not `cat`, echo, tee, log, commit, or paste secret values. | Redacted review only; integration commands consume exported `REDISX_REDIS_*` values |

## Safe dev secret procedure

Use local credentials without printing them. Keep shell tracing disabled and record only pass/fail status in evidence.

```bash
set +x
# Load only the REDISX_REDIS_* values from the approved local secret flow for
# /home/ZoneCNH/sre/secrets/env/dev.md. Do not print the file or variable values.
REDISX_INTEGRATION=1 GOWORK=off make test-integration
REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration
```

If the controlled file is not in shell-env format, use the local secret manager/loader for that environment and export only the required `REDISX_REDIS_*` variables into the current process.
