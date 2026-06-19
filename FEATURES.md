# FEATURES.md

Current release anchor: `v1.1.1`.

`redisx` is the L2 Redis adapter reference library. This matrix records the public feature surface and the gate that proves each area before production release.

| Area | Supported feature surface | Required evidence |
| --- | --- | --- |
| Connection/runtime | Redis client construction, `Ping`, `Health`, close semantics, and error mapping. | `GOWORK=off make test`; `REDISX_INTEGRATION=1 GOWORK=off make test-integration` |
| KV and TTL | String `Set` / `Get` / `Delete`, expiring values, multi-key reads and writes. | Unit tests plus live Redis integration report under `.agent/evidence/l2/` |
| Counters | Increment/decrement helpers and integer conversion error paths. | `GOWORK=off make coverage-check`; integration counter coverage |
| Hash/List | Hash field reads/writes/deletes and list push/pop/range operations. | `REDISX_INTEGRATION=1 GOWORK=off make test-integration` |
| Pipeline writes | Batched write execution with result/error propagation. | Unit pipeline tests and integration pipeline coverage |
| Cache-aside | `JSONCodec`, generic `Codec[T]`, `Cache[T]`, and `NewCacheClient[T]` helpers on string values. | `GOWORK=off make test`; API docs in `docs/api.md` |
| Coordination | Lock token compare-release and fixed-window rate limit keys. These are TTL-scoped coordination states, not durable persistence promises. | Integration lock/rate-limit coverage; docs exclude them from durable recovery |
| Durable persistence | Permanent string, hash, list, counter, and pipeline writes survive Redis restart under the persistence profile. Pub/sub is out of scope. | `REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration` |
| Release/version evidence | `pkg/redisx.Version`, goalcli governance, release manifest template, docs, Harness, and release gate examples all track `v1.1.1`. | `GOWORK=off go test ./cmd/goalcli -run TestVersionConstantsTrackChangelogRelease -count=1` |
| Secret handling | Runtime credentials come only from local environment or controlled secret stores. Secret values are not committed, printed, logged, or embedded in evidence. | Review `test/integration/README.md`; run integration with env injection only |

See [ACCEPTANCE.md](ACCEPTANCE.md) for the production-readiness acceptance matrix and [STATUS.md](STATUS.md) for the current snapshot.
