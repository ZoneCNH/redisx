# redisx 状态

更新日期：2026-06-21

## 当前结论

`redisx` 当前按 Redis adapter 的 L2-T2 目标对齐，release-readiness 结论为通过：

- 目标等级：L2-T2
- 能力范围：KV、TTL、Hash、List、Pipeline、Cache-aside helpers、TTL-scoped lock/rate limit、Pool、Persistence / restart recovery
- Contract packs：`common`、`kv`、`pool`、`ttl`；v1 public helper surface includes hash/list/pipeline/cache/lock/rate-limit primitives
- 必需 profiles：`unit`、`contract`、`integration`、`persistence`
- Release readiness：`release_ready=true`，L2 readiness score `92`；release score gate 为 `10/10`
- 当前分支：`redisx`
- Agent team：本轮 OMX team 分为 repo exploration、test coverage、provider/runtime implementation 三条 lane，5/5 tasks completed，0 pending，0 in_progress，0 failed

## 功能清单

| 功能 | 状态 | 证据来源 |
| --- | --- | --- |
| Ping / Health / HealthCheck | pass | `.agent/evidence/l2/integration-report.json` |
| KV Set / Get | pass | `.agent/evidence/l2/integration-report.json` |
| TTL permanent / missing / expiring keys | pass | `.agent/evidence/l2/integration-report.json` |
| Multi-key MSet / MGet | pass | `.agent/evidence/l2/integration-report.json` |
| Hash HSet / HGet / HGetAll / HDel | pass | `pkg/redisx/primitives_test.go`, `pkg/redisx/redis_integration_test.go` |
| List LPush / RPush / LRange / LPop / RPop | pass | `pkg/redisx/primitives_test.go`, `pkg/redisx/redis_integration_test.go` |
| Pipeline Set / MSet / HSet / RPush / Incr | pass | `pkg/redisx/primitives_test.go`, `pkg/redisx/redis_integration_test.go` |
| Counter Incr / Decr | pass | `.agent/evidence/l2/integration-report.json`, `pkg/redisx/redis_integration_test.go` |
| Expire existing / missing keys | pass | `.agent/evidence/l2/integration-report.json` |
| Exists / Delete / Close | pass | `.agent/evidence/l2/integration-report.json` |
| Validation error mapping | pass | `.agent/evidence/l2/integration-report.json` |
| Reconnect / cross-client visibility | pass | `.agent/evidence/l2/integration-report.json` |
| Persistence restart recovery for strings, hashes, lists, counters, and pipeline writes | pass | `.agent/evidence/l2/persistence-report.json`, `pkg/redisx/redis_integration_test.go` |
| Cache-aside JSON codec helpers | pass | `pkg/redisx/primitives_test.go` |
| Lock token acquire/release | pass | `NewLock` 生成 token；`AcquireLock` / `ReleaseLock` 接受显式 token；`pkg/redisx/primitives_test.go`, `pkg/redisx/redis_integration_test.go`; TTL-scoped, not durable evidence |
| Fixed-window rate limiting | pass | `pkg/redisx/primitives_test.go`, `pkg/redisx/redis_integration_test.go`; TTL-scoped, not durable evidence |
| Evidence generation and release-readiness aggregation | pass | `.agent/evidence/l2/release-readiness.json` |

## 持久化对齐

`redisx` 不实现本地持久化层；所有写入和删除命令统一走 Redis 数据面，持久化能力由被测 Redis 后端的 AOF/RDB 配置提供。当前 v1.1.2 release gate 要求 persistence profile 通过 restart recovery，并证明永久 string、hash、list、counter 和 pipeline 写入在同一存储上重启后仍保持一致。TTL-scoped lock token 和 fixed-window rate-limit key 是临时协调状态，不作为 durable persistence 证据；pub/sub 语义不在当前公共持久化承诺中。

| 命令类别 | 持久化边界 | 当前状态 |
| --- | --- | --- |
| Set / MSet | Redis 后端持久化 | supported |
| HSet / HDel | Redis 后端持久化 | supported |
| LPush / RPush / LPop / RPop | Redis 后端持久化 | supported |
| Pipeline Set / MSet / HSet / RPush / Incr | Redis 后端按命令语义持久化 | supported |
| Incr / Decr | Redis 后端持久化 | supported |
| Expire / TTL-bearing writes | Redis 后端持久化，按 Redis TTL 语义恢复或过期 | supported |
| Lock token / fixed-window rate limit | TTL-scoped 临时状态；只验证 token compare-delete 与 window TTL 行为，不作为 durable evidence | supported, non-durable |
| Delete | Redis 后端持久化删除状态 | supported |
| Read-only commands | 不产生持久化写入 | supported |

## 验证命令

当前状态页与以下 gate 对齐：

```bash
GOWORK=off make test-contract
REDISX_INTEGRATION_DOCKER=1 GOWORK=off make test-integration
REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration
GOWORK=off make l2-check
GOWORK=off make docs-check
GOWORK=off make coverage-check
GOWORK=off make lint
GOWORK=off make race
GOWORK=off make security
GOWORK=off make contracts
GOWORK=off make score-check
XLIB_CONTEXT=release_verify GOWORK=off make release-check
GOWORK=off go test ./...
GOWORK=off go test ./... -race -count=1
GOWORK=off make coverage-check
GOWORK=off go vet ./...
git diff --check
```

本轮 release team 验证说明：开发环境未暴露兼容的 `REDISX_REDIS_*` 连接变量，因此未声明新的 live Redis integration / persistence evidence；公开文档和 evidence 不记录真实 Redis 地址、密码、TLS material 或本地 secret 路径。CI/CD 配置已收紧为 fail-closed：`coverage-check` 已纳入 `ci`、`context-release`、Harness 和 release required gates，Worktree Guard push 事件不再误跑 PR source branch 检查，Auto Patch workflow 使用可用的 golangci-lint action 版本。`GOWORK=off make coverage-check` 对 Redis runtime/API 发布面执行 100% 覆盖率门禁，`GOWORK=off make fmt vet lint test race coverage-check` 已通过。

## Evidence

Redis L2 evidence 固定在 `.agent/evidence/l2/`：

- `.agent/evidence/l2/compliance-matrix.json`
- `.agent/evidence/l2/integration-report.json`
- `.agent/evidence/l2/persistence-report.json`
- `.agent/evidence/l2/release-readiness.json`

`release-readiness.json` 汇总 `unit`、`contract`、`integration`、`persistence` profile；公开 evidence 只记录 profile 状态、覆盖项、占位符变量名和 gate 结论。

## 安全边界

- 真实 Redis 连接信息只允许从外部环境注入。
- 文档、日志和 evidence 只能记录 `REDISX_REDIS_*` 变量名，不得记录真实 Redis URL、host 值、password 值、TLS material 或本地 secret 文件路径。
- Integration profile 需要显式启用 `REDISX_INTEGRATION=1`。
- Persistence recovery profile 需要显式启用 `REDISX_PERSISTENCE_INTEGRATION=1`，并使用保留存储验证重启恢复。

## 关联文档

- `README.md`
- `docs/l2/04_redisx_execution_plan.md`
- `docs/release.md`
- `docs/test-strategy.md`
- `test/integration/README.md`
- `.agent/evidence/l2/README.md`

## 未完成事项

当前没有 L2-T2 运行时验收 blocker。发布 tag 仍受 `release-preflight` 约束：必须在干净且与 `origin/main` 对齐的 `main` 分支上执行，当前 feature branch 只能完成验证、提交和 PR 准备。
