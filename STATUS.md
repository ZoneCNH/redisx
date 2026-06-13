# redisx 状态

更新日期：2026-06-13

## 当前结论

`redisx` 当前按 Redis adapter 的 L2-T2 目标对齐，release-readiness 结论为通过：

- 目标等级：L2-T2
- 能力范围：KV、TTL、Pool、Persistence / restart recovery
- Contract packs：`common`、`kv`、`pool`、`ttl`
- 必需 profiles：`unit`、`contract`、`integration`、`persistence`
- Release readiness：`release_ready=true`，score `92`

## 功能清单

| 功能 | 状态 | 证据来源 |
| --- | --- | --- |
| Ping / Health / HealthCheck | pass | `.agent/evidence/l2/integration-report.json` |
| KV Set / Get | pass | `.agent/evidence/l2/integration-report.json` |
| TTL permanent / missing / expiring keys | pass | `.agent/evidence/l2/integration-report.json` |
| Multi-key MSet / MGet | pass | `.agent/evidence/l2/integration-report.json` |
| Counter Incr / Decr | pass | `.agent/evidence/l2/integration-report.json` |
| Expire existing / missing keys | pass | `.agent/evidence/l2/integration-report.json` |
| Exists / Delete / Close | pass | `.agent/evidence/l2/integration-report.json` |
| Validation error mapping | pass | `.agent/evidence/l2/integration-report.json` |
| Reconnect / cross-client visibility | pass | `.agent/evidence/l2/integration-report.json` |
| Persistence restart recovery | pass | `.agent/evidence/l2/persistence-report.json` |
| Evidence generation and release-readiness aggregation | pass | `.agent/evidence/l2/release-readiness.json` |

## 持久化对齐

`redisx` 不实现本地持久化层；所有写入和删除命令统一走 Redis 数据面，持久化能力由被测 Redis 后端的 AOF/RDB 配置提供。当前 L2-T2 gate 要求 persistence profile 通过 restart recovery，并证明永久键值和永久 TTL 语义在同一存储上重启后仍保持一致。

| 命令类别 | 持久化边界 | 当前状态 |
| --- | --- | --- |
| Set / MSet | Redis 后端持久化 | supported |
| Incr / Decr | Redis 后端持久化 | supported |
| Expire / TTL-bearing writes | Redis 后端持久化，按 Redis TTL 语义恢复或过期 | supported |
| Delete | Redis 后端持久化删除状态 | supported |
| Read-only commands | 不产生持久化写入 | supported |

## 验证命令

当前状态页与以下 gate 对齐：

```bash
GOWORK=off make test-contract
GOWORK=off make test-integration
REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration
GOWORK=off make l2-check
GOWORK=off make docs-check
go test ./...
go vet ./...
git diff --check
```

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

当前没有 L2-T2 release blocker。提交、PR 和发布动作尚未在本轮执行。
