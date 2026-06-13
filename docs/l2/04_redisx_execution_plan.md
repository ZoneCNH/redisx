# redisx 执行方案：KV / TTL / Persistence

> 文档用途：独立仓库执行方案，可直接作为 Goal / Issue / PR / Harness / Evidence 落地依据。
> 统一原则：禁止 main 直接开发；必须使用 git worktree；没有 Evidence 不允许 DONE；没有 release-readiness 不允许 Release；不得把真实 Redis 连接信息写入源码、文档、日志或 Evidence。

## 1. 定位

`redisx` 是 L2 infrastructure adapter，不是业务缓存模块。它向上暴露稳定 KV / TTL / persistence contract，向下封装 Redis client、连接池、超时、错误映射和序列化边界。

L2-T2 完成口径如下：

```text
capability manifest
  -> contract pack
  -> env-gated live Redis integration
  -> persistence recovery profile
  -> Evidence
  -> release-readiness.json
```

## 2. 能力族

```text
common
kv
ttl
persist
pipeline
lock
stream
pubsub
```

L2-T2 只开放 `common`、`kv`、`ttl`、`persist` 和 pool config pass-through。`lock`、`stream`、`pubsub`、pool exhaustion、chaos 和 benchmark 进入后续 L2-T3/L2-T4。

## 3. L2-T2 Capability Manifest

```yaml
module: redisx
release_level: L2-T2
capabilities:
  common:
    required: true
  kv:
    required: true
  ttl:
    required: true
  persist:
    required: true
  pipeline:
    required: false
  lock:
    required: false
  stream:
    required: false
  pubsub:
    required: false
required_profiles:
  - unit
  - contract
  - integration
  - persistence
```

真实 Redis profile 必须由调用方显式注入 `REDISX_INTEGRATION=1` 和 `REDISX_REDIS_*` 环境变量。公开文档只允许写变量名和占位符，不允许写真实 host、password、TLS material 或 secret file 内容。

## 4. P0 Contract And Integration Coverage

L2-T2 必需覆盖：

- `kv.set_get`
- `kv.delete`
- `kv.exists`
- `kv.not_found`
- `kv.validation.empty_key`
- `kv.context_cancel`
- `ttl.expire`
- `ttl.not_found_after_expire`
- `pool.config_passthrough`
- `integration.redis.ping_health`
- `integration.redis.commands_set_get_ttl_mset_mget_counter_expire_exists_del`
- `integration.redis.client_reconnect`
- `persistence.redis.restart_recovery`

上述 coverage 同时体现在 `.agent/l2-capabilities.yaml`、contract tests、integration report、persistence report 和 compliance matrix 中。

## 5. 错误映射

Redis provider 错误必须映射到稳定 adapter 错误：

| 场景 | 对外语义 |
| --- | --- |
| key 不存在 | `ErrNotFound` 或 contract 指定的 not found 结果 |
| 空 key / 非法 TTL | validation error |
| context timeout / cancel | context error |
| Redis 暂不可用 | transient provider error |
| Close 后继续使用 | closed client error |

contract tests 不依赖具体 Redis 错误字符串，只验证稳定语义。

## 6. 文件与证据布局

```text
.agent/l2-capabilities.yaml
.agent/evidence/l2/README.md
.agent/evidence/l2/compliance-matrix.json
.agent/evidence/l2/integration-report.json
.agent/evidence/l2/persistence-report.json
.agent/evidence/l2/release-readiness.json
test/contract/l2_contract_test.go
test/integration/README.md
pkg/redisx/redis_integration_test.go
scripts/run_redis_integration.sh
scripts/run_redis_persistence_integration.sh
scripts/verify_l2_redisx.py
docker-compose.test.yml
Makefile
```

Evidence 只记录 profile 状态、覆盖项、命令入口和文件路径，不记录真实 Redis 配置值。

## 7. 标准命令

```bash
GOWORK=off make test-unit
GOWORK=off make test-contract
GOWORK=off make test-integration
REDISX_INTEGRATION=1 GOWORK=off make test-integration
REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration
GOWORK=off make l2-check
XLIB_CONTEXT=release_verify GOWORK=off make release-check
```

`GOWORK=off make test-integration` 默认可使用 Docker-backed Redis runner。真实 Redis 验收必须设置 `REDISX_INTEGRATION=1` 并由外部环境提供 `REDISX_REDIS_ADDR`、`REDISX_REDIS_PASSWORD`、`REDISX_REDIS_DB` 和 TLS 相关变量。persistence recovery 使用 `REDISX_PERSISTENCE_INTEGRATION=1` 触发，并生成 `.agent/evidence/l2/persistence-report.json`。

## 8. Evidence Standard

完成声明必须包含：

```text
DONE with evidence:
- .agent/evidence/l2/release-readiness.json
- .agent/evidence/l2/compliance-matrix.json
- .agent/evidence/l2/integration-report.json
- .agent/evidence/l2/persistence-report.json
```

`release-readiness.json` 必须满足：

- `release_level_actual` 为 `L2-T2`。
- `readiness_score` 不低于 release gate 阈值。
- `required_profiles.unit.status` 为 `pass`。
- `required_profiles.contract.status` 为 `pass`。
- `required_profiles.integration.status` 为 `pass`。
- `required_profiles.persistence.status` 为 `pass`。
- `blockers` 为空。

## 9. 分阶段路线

| 阶段 | 范围 | Gate |
| --- | --- | --- |
| L2-T2 | common、kv、ttl、persist、pool config pass-through、env-gated integration、persistence recovery、release-readiness | `GOWORK=off make l2-check` |
| L2-T3 | chaos、benchmark、adoption、layer guard、secret scan | release extended gate |
| L2-T4 | lock、stream、pubsub、traceability、retrospective、factory-grade evidence | full factory-grade gate |

## 10. Rollout

L2-T2 只允许下游依赖 `common`、`kv`、`ttl` 和 `persist`。调用方不得假设 lock、stream、pubsub 或 pool exhaustion 已经完成。

## 11. 特殊约束

- `lock` 不在 L2-T2 中承诺，避免把 Redis 单节点锁误认为分布式一致性锁。
- `pubsub` 不承诺 persistence；需要独立 loss/reconnect 语义。
- `stream` 需要消费组、ack、claim 和 pending list contract，不混入 KV/TTL 验收。
- persistence recovery 只验证隔离测试 key 的写入、重启和读取恢复，不复用生产 keyspace。
- integration runner 必须支持跳过语义，但 release 验收必须显式提供 live Redis profile evidence。

## 12. Acceptance

L2-T2 完成必须同时满足：

- `GOWORK=off make docs-check` 通过。
- `GOWORK=off make test-contract` 通过。
- `GOWORK=off make test-integration` 通过。
- `REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration` 通过。
- `GOWORK=off make l2-check` 通过。
- `XLIB_CONTEXT=release_verify GOWORK=off make release-check` 通过或记录明确 release-scope gap。
- `.agent/evidence/l2/release-readiness.json` 声明 `unit`、`contract`、`integration`、`persistence` 全部 pass。
- `.agent/evidence/l2/compliance-matrix.json` 包含 `kv-persistence-recovery`。
- 公开文档和 Evidence 不包含真实 Redis 凭据、端点或 secret file 内容。
