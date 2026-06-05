# redisx 标准

`redisx` 是 `github.com/ZoneCNH/redisx` 仓库中的标准权威源、Go Reference Template、Generator、Harness 和 Evidence Runtime，仓库 URL 为 `https://github.com/ZoneCNH/redisx`。当前仓库既保存标准正文，也保存可渲染的 `redisx` 参考实现；下游库通过模板渲染获得自己的 `pkg/<package-name>`、文档、contracts、release manifest 和 `.agent` 控制面。

本标准约束 `redisx` 作为 L2 基础设施适配层的最小可交付形态。它的目标不是提供完整 Redis SDK，而是证明一个基础设施 adapter 可以被 `redisx` 标准工厂生成、测试、发布、审计和下游采纳。

## 标准角色

`redisx` 在本仓库中承担五类职责：

| 职责 | 说明 |
| --- | --- |
| Standard Source | 定义公共 API、模块边界、分层、文档、release gate 和完成证据。 |
| Go Reference Template | 在 `pkg/redisx` 保存可编译、可测试、可渲染的 Go 参考实现。 |
| Generator | 通过 `scripts/render_template.sh` 把模板渲染为 `kernel`、`configx`、`observex`、`redisx` 等目标库。 |
| Harness | 通过 `Makefile`、`cmd/goalcli`、`scripts/` 和 `.agent/harness/harness.yaml` 执行 fmt、vet、lint、test、race、boundary、security、contracts、docs、dependency、standard-impact、score 和 evidence gate。 |
| Evidence | 通过 `release/manifest/template.json`、`make evidence`、checksum 和 release evidence gate 证明每次发布的源码、工具、契约、检查结果和分数。 |

## L2 边界

作为 L2 基础设施 adapter，`redisx` 只能表达 Redis 连接、配置、健康检查、metrics、错误分类、生命周期和测试夹具。它不得承载业务 key 命名策略、应用缓存策略、业务消息 schema、业务 repository 或 `x.go` 反向依赖。

运行时配置必须由 `redisx.Config`、`redisx.Option` 或下游显式 binder 注入。基础库不得隐式读取 `/home/k8s/secrets/env/*`，也不得提交真实凭据或把 password、token、DSN 明文写入日志、manifest、README、PR、Issue 或 Evidence。

## 公共 API

参考实现的公共面保持窄接口：

- `redisx.Config` 定义 name、timeout 和 secret 等配置输入，并提供 `Validate` 与 `Sanitize`。
- `redisx.New(ctx, cfg, opts...)` 构造 client，并在 context、配置和 metrics 维度执行校验。
- `Client.Close(ctx)` 提供幂等关闭语义。
- `Client.HealthCheck(ctx)` 输出 name、status、message、checked_at、latency_ms 和 metadata。
- `redisx.Error`、`ErrorKind`、`IsKind` 提供稳定错误分类。
- `redisx.Metrics` 与内置 metric 名称提供供应商无关观测 contract。

第三方 provider 只能作为内部实现细节出现，公共 API 不得泄漏 provider-specific 类型。未来接入真实 Redis provider 时，adapter 必须保持可替换，并继续通过 fake/testkit 和边界 gate 证明隔离。

## 模板和 generator

`scripts/render_template.sh` 是标准渲染入口。它必须从 Git 跟踪源码生成目标仓库，跳过未跟踪临时文件，并保持 module path、package name、README、docs、contracts、examples、testkit、release manifest 和 `.agent` 结构一致。

模板渲染的验收重点：

- 目标 `go.mod` 使用调用方传入的 module path。
- `pkg/redisx` 被渲染为 `pkg/<package-name>`。
- `docs/standard/`、`contracts/`、`scripts/`、`examples/`、`testkit/` 和 `.agent/` 在目标仓库中可用。
- 渲染结果不得继承未跟踪测试探针、release 产物或本地 Agent runtime 状态。

## Harness

发布前必须以 `GOWORK=off` 运行 required gate：

```bash
GOWORK=off make fmt
GOWORK=off make vet
GOWORK=off make lint
GOWORK=off make test
GOWORK=off make race
GOWORK=off make boundary
GOWORK=off make security
GOWORK=off make contracts
GOWORK=off make docs-check
GOWORK=off make dependency-check
GOWORK=off make standard-impact-check
GOWORK=off make score
CHECK_STATUS=passed GOWORK=off make evidence
RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check
```

`cmd/goalcli` 是 gate 的统一实现面；shell 脚本只作为兼容层或子检查实现。任何新增 gate 必须同步 Makefile、`.agent/registries/command-registry.yaml`、`.agent/harness/harness.yaml` 和相关文档。

## Evidence

`release/manifest/latest.json` 与 `release/manifest/latest.json.sha256` 是生成产物，必须保持被 `.gitignore` 忽略。完成声明必须使用实际 gate 输出，并包含 `DONE with evidence:`，不能把文档、目标态 registry 或未运行的 downstream smoke 当作完成证明。

manifest 至少覆盖 module、commit、tree_sha、source_digest、dependencies、tools、contracts、checks、score、workflow 和 standard_impact。release evidence check 必须校验 manifest 与当前仓库事实一致。

## 下游采纳

`redisx` 可以作为 `kernel`、`configx`、`observex`、`postgresx`、`redisx`、`kafkax`、`taosx`、`ossx`、`clickhousex` 等目标库的模板来源。下游采纳状态只能由对应仓库的真实 Evidence、CI 输出、PR 或 artifact 证明。

`x.go` 只能作为基础库消费方。没有外部下游证据时，采纳状态必须保持 `not_adopted`、`not_run` 或 `blocked`，不得升级为 adopted。

## 复盘与回写

每次执行中发现的模板缺口、generator 缺口、Harness 缺口、文档缺口或规则缺口，都必须记录为 retrospective、patch candidate 或 issue candidate，并回写到标准源维护流程。`redisx` 的长期价值来自可复制的标准工厂能力，而不是单次 Redis 封装功能。
