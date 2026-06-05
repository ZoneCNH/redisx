# RETRO-20260604 redisx L2 standard factory closeout

## Outcome

GOAL-20260604-REDISX-L2-STANDARD-FACTORY 的 closeout 缺口集中在 Redis 专属公开契约、fake provider 测试入口和治理证据链。本次补齐 `redisx.Options` 绑定面、Redis 错误标识符、`contracts/redisx.*` 契约、`testkit.NewFakeRedis()` 和 REQ-001..014 审计证据。

## What worked

- 维持默认内存 provider，不引入真实 Redis 网络依赖。
- 以契约测试锁住新 JSON/YAML 契约与公开常量的映射关系。
- 用 golden 测试锁住 fake Redis 的最小语义，避免生成器回归时静默漂移。

## What changed

- API：新增 `redisx.Options`、`NewWithOptions`、`ErrorIdentifier` 常量和 `ErrorIdentifierForKind`。
- Contracts：新增 Redis 专属 config、health、errors、metrics 契约。
- Testkit：新增 `NewFakeRedis()` 和 fake Redis golden/客户端测试。
- Governance：新增 prompt/harness/rule patches 与 task audit，并登记 `.agent/index.yaml`。

## Verification gate

- `GOWORK=off go test ./contracts ./pkg/redisx ./testkit`
- `GOWORK=off go test ./...`
- `GOWORK=off go vet ./...`
- `GOWORK=off go build ./...`
- `CHECK_STATUS=passed GOWORK=off make evidence`
- `RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check`

## Follow-up

If a real Redis adapter is added later, keep it behind explicit provider configuration and extend `contracts/redisx.errors.yaml` with adapter-specific mappings without changing the default no-dial behavior.
