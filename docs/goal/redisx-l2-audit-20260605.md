# redisx L2 Standard Factory 审计 — 2026-06-05

## 范围

本审计覆盖 `docs/goal/goal.md` 中 REQ-001..REQ-014 的当前 closeout 状态，并同步 `xlib-standard` L2 standard-source 更新后的本地缺口。结论只基于本仓库当前公共 API、测试和 Evidence 工件，不声明真实 Redis 集成或 downstream adoption 已完成。

## 摘要

`redisx` 已具备 Redis-specific 的本地 contract 基础：模块身份、生命周期、健康、错误分类、metrics、fake provider/testkit 和 KV/TTL 单元测试均有源码或测试证据。同步后新增了 L2 manifest、contract pack registry、schema、L2 gate、release-readiness/compliance snapshot、contract test 入口、测试目录占位说明、CI workflow 和 `Makefile` L2 targets。

当前 public config 仍是模板级 `Config{Name, Timeout, Secret}`，`Options{Config, Metrics, Provider}` 只绑定配置、metrics 和 provider override；仓库没有公开 Redis address、username/password、DB、TLS 或 pool 参数，也没有默认网络 dial。L2-T2 已完成本地评估，但不是 release-ready：contract report、integration report 和 provider-backed pool contract 仍为 missing/pending。

## L2 同步结果

| 工件 | 状态 | 说明 |
| --- | --- | --- |
| `.agent/l2-capabilities.yaml` | 已新增 | 声明 adapter 为 `redisx` / `github.com/ZoneCNH/redisx` / `key_value`，contract packs 为 `common`、`kv`、`ttl`、`pool`。 |
| `.agent/gates/l2gate.yaml` | 已新增 | 保留 L2 gate 结构，供后续统一 gate runtime 接入。 |
| `.agent/registry/l2-*.yaml` | 已同步 | 来自 `xlib-standard` L2 standard-source registry。 |
| `.agent/schemas/l2-*.schema.json` | 已同步 | 来自 `xlib-standard` L2 standard-source schema。 |
| `.agent/evidence/l2/release-readiness.json` | 已新增 | `target_level=L2-T2`，unit pass，contract/integration missing，`release_allowed=false`。 |
| `.agent/evidence/l2/compliance-matrix.json` | 已新增 | 记录 common/kv 已有本地证据，以及 ttl/pool contract 缺口。 |
| `test/contract/l2_contract_test.go` | 已新增 | 固定 L2 manifest、readiness snapshot 和 provider-boundary 禁止项。 |
| `test/{integration,chaos,benchmark,adoption}/README.md` | 已新增 | 明确当前 pending 范围，避免误报。 |
| `docker-compose.test.yml` | 已新增 | 为后续真实 Redis integration profile 预留本地依赖。 |
| `.github/workflows/l2-gates.yml` | 已新增 | 运行 `GOWORK=off make l2-check` 与 `GOWORK=off make test-contract`。 |
| `Makefile` L2 targets | 已新增 | 暴露 `l2-check`、`l2-manifest-check`、`l2-evidence-check`、`l2-readiness-status`、`test-contract` 等入口。 |

## 需求审计

| Req | 状态 | 证据 |
| --- | --- | --- |
| REQ-001 standard-source generation | 满足当前 closeout | Module/package identity 仍为 `github.com/ZoneCNH/redisx`；Redis-specific contracts、tests、testkit 和 L2 registry/schema 已落地。 |
| REQ-002 L2 boundary | 满足 | 公共 API 保持 package-local；provider 实现在 `redisx.Provider` / `internal/provider` 边界内；未引入业务层或 `x.go` 依赖。 |
| REQ-003 explicit config | 部分满足 | 当前公开配置只有 `Config{Name, Timeout, Secret}` 与 `Options{Config, Metrics, Provider}`；尚未公开 Redis address、DB、TLS 或 pool 配置。 |
| REQ-004 lifecycle | 满足当前 API | `New`、`Close`、`Ping`、`Health` 行为由本地测试覆盖，health schema 保持稳定。 |
| REQ-005 Redis KV operations | 满足本地 fake-provider surface | fake provider 测试覆盖 get/set、mget、TTL、close behavior 和 missing-key classification；真实 Redis contract runner 仍 pending。 |
| REQ-006 provider isolation | 满足 | `testkit.NewFakeRedis()` 返回内存 provider；未公开 go-redis 类型，未加入默认 Redis 网络连接。 |
| REQ-007 health contract | 满足 | `HealthStatus` 与 `contracts/redisx.health.schema.json` 固定 health payload shape。 |
| REQ-008 observability | 满足 | Redis metric constants 由 `contracts/redisx.metrics.yaml` 验证；labels 仍为 `op`、`kind`、`name`、`status`。 |
| REQ-009 error taxonomy | 满足 | `ErrorKind` 与 `RedisErrorID` 已覆盖 Redis nil、timeout、canceled、network、auth、read-only、loading、try-again、cluster moved/ask、closed、invalid config 和 provider fallback。 |
| REQ-010 testkit | 满足 | `testkit.NewFakeRedis()` 可通过 `redisx.WithProvider` 注入，无需 Redis 环境变量。 |
| REQ-011 harness | 满足本地 gates | `l2-check`、`test-contract`、`contracts`、`docs-check` 与 `go test ./...` 可作为当前本地验证入口。 |
| REQ-012 release evidence | 部分满足 | L2 readiness snapshot 已生成并明确 `contract` / `integration` missing；不得将其声明为 L2-T2 release-ready。 |
| REQ-013 downstream adoption | 未声明 | 本 closeout 不声明 downstream adoption；后续需要下游库真实渲染、运行和报告证据。 |
| REQ-014 self-improving artifacts | 满足当前 closeout | Goal-specific retrospective、prompt/harness/rule patch 与 L2 同步工件已纳入本地审计链。 |

## 剩余缺口

- 需要生成真实 `contract-report.json`，覆盖 `common`、`kv`、`ttl` 和 `pool` contract pack。
- 需要运行真实 Redis integration profile 并生成 `integration-report.json`。
- 需要决定 public config 是否引入 Redis-specific address、auth、DB、TLS、timeout 和 pool 参数；在决策前不得把这些配置写入已完成证据。
- 需要 downstream adoption proof 后，才能把 REQ-013 从未声明改为通过。
