# redisx L2 基础设施适配层标准工厂 Goal 可执行方案

> Goal ID: `GOAL-20260604-REDISX-L2-STANDARD-FACTORY`  
> Spec ID: `SPEC-redisx-l2-v1.0`  
> Design ID: `DESIGN-redisx-l2-v1.0`  
> Plan ID: `PLAN-GOAL-20260604-REDISX-L2-STANDARD-FACTORY-v1.0`  
> Runtime: Goal Runtime Prompt v3.1  
> 目标仓库: `github.com/ZoneCNH/redisx`  
> 标准源: `github.com/ZoneCNH/xlib-standard`  
> 生成目标: `module=github.com/ZoneCNH/redisx`, `package=redisx`, `layer=L2`  
> 执行日期: 2026-06-04 Asia/Tokyo

---

## 0. 当前结论

`redisx` 不应该继续作为“零散 Redis 封装库”推进，而应该被升级为由 `xlib-standard` 控制的 **L2 基础设施适配层标准工厂实例**。

最终结果不是“能连 Redis 的 Go 包”，而是：

1. 一个独立仓库、独立版本、独立 Release Evidence 的 L2 标准适配库。
2. 运行时只依赖标准允许的 L0/L1 契约：`kernel`、`configx`、`observex`。
3. 测试、Harness、Release Gate、Evidence、下游采纳状态、回滚、复盘全部纳入 `.agent` 控制面。
4. Redis 具体实现被隔离在 provider adapter 内部，公共 API 不泄漏第三方客户端细节。
5. 不包含业务 key 语义、不包含应用缓存策略、不反向依赖 `x.go`、不读取生产密钥。
6. 每次发布必须通过 `DONE with evidence:` 证明，而不是口头声称完成。

---

## 1. 源状态确认

### 1.1 `xlib-standard` 当前事实

`xlib-standard` 当前承担五类职责：

- `Standard Source`
- `Go Reference Template`
- `Generator`
- `Harness`
- `Evidence Runtime`

它把公共 API、配置、错误、健康检查、metrics、测试、release Evidence、Goal Runtime 和下游生成规则放在同一套可验证工件中维护。

当前标准源已经把 `redisx` 列为 L2 目标库：

| 库 | module path | package | layer | 允许依赖 | 禁止依赖 |
|---|---|---|---|---|---|
| `redisx` | `github.com/ZoneCNH/redisx` | `redisx` | L2 | `kernel`、`configx`、`observex` | 业务 key 语义、应用缓存策略 |

关键含义：

- `redisx` 是标准目标库，不是临时实验仓库。
- `redisx` 当前应接受 `xlib-standard` 的 generator、Harness、Evidence 和 release gate 控制。
- 当前下游矩阵显示 `redisx` adoption 状态仍应按 `not_adopted` / `not_run` 处理，不能宣称已采纳。
- `redisx` 必须生成可编译 module、package、README 和 docs。
- `redisx` 不得导入 `x.go`，不得读取 `/home/k8s/secrets/env/*`。

### 1.2 `redisx` 当前事实

当前 `redisx` 仓库仅有极简 README，尚未形成：

- Go module
- 标准目录结构
- `.agent` 控制面
- Harness gate
- contracts
- docs
- release manifest
- Evidence
- testkit
- CI
- downstream adoption proof

因此本方案按“从标准源生成并升级”为主路径，而不是在当前空壳上继续手写累加。

---

## 2. 问题的底层本质

表面问题：

> 封装 Redis 公共基础开发库。

真实问题：

> 如何把一个具体基础设施适配库纳入可复制、可验证、可发布、可下游采纳、可自我进化的标准工厂体系。

底层本质不是 Redis，而是 **标准控制权**：

- 谁定义边界？
- 谁定义 API 契约？
- 谁定义依赖白名单？
- 谁定义测试与 Evidence？
- 谁定义“完成”？
- 谁负责下游同步？
- 发现问题后如何反向沉淀到 `xlib-standard`？

因此 `redisx` 必须从“代码库”升级为：

```text
xlib-standard 标准源
  -> generator 渲染 redisx
  -> redisx 独立实现 Redis adapter
  -> Harness 验证边界、测试、安全、文档、发布
  -> Evidence 证明完成
  -> downstream adoption 证明可被 x.go / 其他库消费
  -> Retrospective 反向修复 xlib-standard / Harness / Rules
```

---

## 3. 不可再拆解的基本真理

1. **基础库的价值来自稳定契约，不来自内部实现。**  
   `redisx` 可以替换 provider，但不能随意破坏公共 API。

2. **L2 只能适配基础设施，不得承载业务语义。**  
   Redis key 如何命名、缓存什么、TTL 策略如何随业务变化，属于应用层或业务层，不属于 `redisx`。

3. **配置必须显式。**  
   `redisx` 不得隐式读取生产密钥、环境目录、全局配置或 `x.go` runtime。

4. **可观测性必须供应商无关。**  
   metrics / tracing / logging 使用 `observex` 契约，不把 OpenTelemetry、Prometheus 或具体 vendor 变成上层硬依赖。

5. **生命周期必须可关闭。**  
   Redis client、pubsub、stream consumer、background refresh 等资源必须能被 context / lifecycle 控制关闭。

6. **错误必须可分类。**  
   timeout、network、auth、redis nil、cluster moved、try again、loading、readonly、circuit open 等不能全变成普通 `error`。

7. **Evidence 是完成的一部分。**  
   没有 release manifest、gate 输出、checksum、score、workflow artifact，就不能宣称 release ready。

8. **独立仓库必须独立发布。**  
   `redisx` 不应等待 monorepo 统一发布，也不应依赖 `x.go` 发布节奏。

9. **标准工厂的复利来自反向沉淀。**  
   `redisx` 中发现的模板缺口、Harness 缺口、契约缺口必须形成 Patch 回写 `xlib-standard`。

---

## 4. 被误认为真理的常见假设

| 常见假设 | 判断 | 正确处理 |
|---|---|---|
| Redis 封装就是包一层 `go-redis` | 假 | 需要标准 API、config、health、metrics、error、lifecycle、Evidence |
| Redis key 规范应放进 `redisx` | 假 | `redisx` 只能提供 key 类型和操作原语，业务 key schema 放在调用方 |
| 基础库可以直接读取环境变量 | 假 | 必须通过显式 Options / configx 注入 |
| 只要 `go test ./...` 通过就能发布 | 假 | 还需要 boundary、security、contracts、docs、dependency、standard-impact、score、evidence |
| L2 可以随意依赖所有 L1 | 假 | 当前标准源允许 runtime 依赖 `kernel`、`configx`、`observex`；其他 L1 需先更新标准源 |
| Redis client 选型一旦确定就不可替换 | 假 | provider 必须内部隔离，公共 API 不绑定 provider |
| Integration test 必须连生产 Redis | 假 | 必须用 mock / fake / ephemeral Redis，不得默认触达生产 |
| 下游能编译就是采纳成功 | 假 | 需要 downstream adoption status、Evidence、artifact、commit、workflow proof |
| 文档写了就代表标准落地 | 假 | 必须有命令、测试、manifest、checksum、score 证明 |

---

## 5. 可以被打破的限制

### 5.1 打破“手写每个基础库”的限制

使用 `xlib-standard` generator 生成基础结构，redisx 只补充 Redis 专属实现。

### 5.2 打破“每个库单独发明 Harness”的限制

复用标准源 Harness，redisx 只增加 Redis-specific gates：

- fake Redis contract tests
- optional ephemeral Redis integration
- key/value roundtrip tests
- connection mode tests
- metrics contract tests
- secret redaction tests

### 5.3 打破“基础库只交付代码”的限制

每个 release 交付：

- source code
- contracts
- docs
- test evidence
- release manifest
- checksum
- score
- standard impact
- downstream sync plan
- retrospective patch

### 5.4 打破“Redis 封装绑定第三方库”的限制

公共 API 面向 `redisx.Client`、`redisx.Options`、`redisx.Commander`、`redisx.HealthChecker`，第三方 provider 位于 `internal/provider/goredis`，未来可替换为其他 provider。

### 5.5 打破“下游采纳靠人工记忆”的限制

建立 adoption registry：

```yaml
libraries:
  redisx:
    module: github.com/ZoneCNH/redisx
    layer: L2
    adoption_status: adopted|not_adopted|blocked
    evidence_state: passed|failed|not_run
    last_release: v0.1.0
    last_evidence_artifact: <workflow artifact or local path>
    downstream_consumers:
      - x.go
      - market-data
      - macro-data
```

---

## 6. 从零设计的新方案

### 6.1 目标态架构

```text
                          ┌──────────────────────────────┐
                          │        xlib-standard          │
                          │ Standard / Generator / Gate   │
                          │ Evidence Runtime / Rules      │
                          └───────────────┬──────────────┘
                                          │ render + sync
                                          ▼
┌──────────────────────────────────────────────────────────────────┐
│                         redisx repo                              │
│                                                                  │
│  .agent/               Goal Runtime v3.1 控制面                  │
│  contracts/            config / metrics / errors / release schema │
│  docs/                 spec / design / api / config / testing     │
│  pkg/redisx            public API                                │
│  internal/provider     go-redis adapter, fake adapter             │
│  internal/redaction    secret/key/value redaction                 │
│  testkit/              fake Redis, assertions, golden fixtures    │
│  examples/             basic / config / health / metrics          │
│  release/manifest      manifest template + generated Evidence     │
│  scripts/              gate, evidence, integration helpers        │
│  .github/workflows     CI / release / security                    │
└───────────────┬──────────────────────────────────────────────────┘
                │ allowed runtime deps
                ▼
┌─────────────┐   ┌──────────────┐   ┌──────────────┐
│   kernel    │   │   configx    │   │   observex   │
│ error/life  │   │ explicit cfg │   │ metrics/logs │
└─────────────┘   └──────────────┘   └──────────────┘

                │ consumed by
                ▼
┌──────────────────────────────────────────────────────────────────┐
│ downstream: x.go / market-data / macro-data / regime services     │
│ only composition layer; no reverse dependency into redisx          │
└──────────────────────────────────────────────────────────────────┘
```

### 6.2 分层规则

```text
L0 kernel
  - error
  - lifecycle
  - context
  - clock
  - shutdown
  - validation primitive

L1 configx / observex / testkitx / resiliencx / schedulex
  - 横切能力
  - 标准契约
  - 不绑定业务

L2 redisx
  - Redis adapter
  - connection / health / metrics / error mapping
  - KV / atomic / pubsub / stream primitive
  - no business key strategy
  - no x.go import
```

当前执行中，`redisx` runtime 依赖以当前标准源为准：

```text
allowed runtime deps:
  - github.com/ZoneCNH/kernel
  - github.com/ZoneCNH/configx
  - github.com/ZoneCNH/observex

test-only allowed:
  - github.com/ZoneCNH/testkitx   # 若标准源已允许 test-only
  - miniredis / ephemeral Redis    # 仅测试，不进入生产路径

conditional future runtime deps:
  - github.com/ZoneCNH/resiliencx  # 需要先由 xlib-standard 更新 downstream matrix
  - github.com/ZoneCNH/schedulex   # 需要先由 xlib-standard 更新 downstream matrix
```

### 6.3 Redis provider 策略

默认建议：

```text
public redisx API
  -> internal provider interface
  -> internal/provider/goredis
  -> github.com/redis/go-redis/v9
```

原因：

- 官方 Redis Go client。
- API 简单，生态成熟。
- 支持 pool、cluster、sentinel、pubsub、pipeline、transactions。
- 容易被 Go 团队理解和维护。

但必须满足：

- `redisx` 公共 API 不暴露 `*redis.Client`。
- `go-redis` 类型只能出现在 `internal/provider/goredis` 或明确的 optional adapter 包中。
- 如果未来迁移到其他 provider，不应影响调用方主要 API。

---

## 7. Goal Runtime v3.1 对象模型

### 7.1 Goal

```yaml
goal_id: GOAL-20260604-REDISX-L2-STANDARD-FACTORY
title: Upgrade redisx into xlib-standard controlled L2 infrastructure adapter factory
mode: full
owner: ZoneCNH
target_repo: github.com/ZoneCNH/redisx
standard_source: github.com/ZoneCNH/xlib-standard
state: INIT
```

### 7.2 Spec

```yaml
spec_id: SPEC-redisx-l2-v1.0
scope:
  include:
    - independent Go module
    - xlib-standard generated skeleton
    - L2 Redis adapter public API
    - explicit config
    - lifecycle and close
    - health check
    - metrics/logging/tracing contract
    - error mapping
    - fake/testkit
    - optional integration test
    - release Evidence
    - downstream adoption proof
  exclude:
    - business cache policy
    - business key schema
    - x.go import
    - production secret reading
    - hidden global singleton
    - real production Redis connection in default tests
```

### 7.3 Design

```yaml
design_id: DESIGN-redisx-l2-v1.0
sections:
  - repository bootstrap
  - package API
  - config model
  - provider isolation
  - observability contract
  - error taxonomy
  - testing strategy
  - release evidence
  - downstream adoption
  - self-improving loop
```

### 7.4 Plan

```yaml
plan_id: PLAN-GOAL-20260604-REDISX-L2-STANDARD-FACTORY-v1.0
milestones:
  - M0 standard-source bootstrap
  - M1 contract and boundary
  - M2 API and provider
  - M3 config/health/observability
  - M4 testkit and integration
  - M5 release gate and evidence
  - M6 downstream adoption
  - M7 retrospective and standard patch
```

### 7.5 State Machine

```text
INIT
  -> CONTEXT_READY
  -> GOAL_READY
  -> SPEC_READY
  -> DESIGN_READY
  -> PLAN_READY
  -> TASKS_READY
  -> EXECUTING
  -> VERIFYING
  -> REVIEWING
  -> RELEASING
  -> RETROSPECTING
  -> DONE
```

异常状态：

```text
BLOCKED
FAILED
NEEDS_RESEARCH
NEEDS_DECISION
NEEDS_REPLAN
NEEDS_ROLLBACK
NEEDS_HUMAN_APPROVAL
INCONSISTENT_STATE
```

---

## 8. Requirement / Acceptance Criteria

### REQ-001: 标准源生成

`redisx` 必须由 `xlib-standard` generator 生成标准结构。

Acceptance Criteria:

- `go.mod` module = `github.com/ZoneCNH/redisx`
- package = `redisx`
- `.agent/` 存在并包含 Goal Runtime v3.1 控制面
- `docs/`、`contracts/`、`scripts/`、`release/manifest/`、`examples/`、`testkit/` 存在
- `GOWORK=off go test ./...` 可运行

Evidence:

- generator command output
- generated file list
- `go test ./...` output
- commit SHA

---

### REQ-002: L2 边界

`redisx` 只能作为 Redis 基础设施适配层。

Acceptance Criteria:

- 不导入 `x.go`
- 不包含业务 key 命名策略
- 不包含应用缓存策略
- 不包含业务 message schema
- 不读取 `/home/k8s/secrets/env/*`
- 不依赖业务 repository

Evidence:

- `make boundary`
- grep / AST import scan
- docs boundary check

---

### REQ-003: 显式配置

所有配置必须通过 `redisx.Options` 或 `configx` binder 显式注入。

Acceptance Criteria:

- 无隐式 env read
- 无 package-level default global client
- password/token 支持 redaction
- options validate 使用 `kernel` validation primitive
- 配置 schema 存在：`contracts/redisx.config.schema.json`

Evidence:

- config unit tests
- redaction tests
- secret scan
- config schema contract check

---

### REQ-004: 生命周期

Client 必须可关闭、可检测状态、可由 context 控制。

Acceptance Criteria:

- `Client.Close(ctx)` 或 `Close() error`
- `Ping(ctx)` / `Health(ctx)`
- 超时从 context 或 options 注入
- 关闭后操作返回可分类错误
- 不泄漏 goroutine

Evidence:

- lifecycle tests
- race test
- close-after-use tests

---

### REQ-005: Redis 基础操作

MVA 必须支持最小 KV 原语。

Acceptance Criteria:

- `Get(ctx, key)`
- `Set(ctx, key, value, ttl)`
- `Del(ctx, keys...)`
- `Exists(ctx, keys...)`
- `Expire(ctx, key, ttl)`
- `TTL(ctx, key)`
- `MGet(ctx, keys...)`
- `MSet(ctx, pairs)`
- `Incr(ctx, key)`
- `Decr(ctx, key)`

Evidence:

- fake provider tests
- optional real Redis integration tests
- examples/basic smoke

---

### REQ-006: Provider 隔离

第三方 Redis client 不得泄漏到公共 API。

Acceptance Criteria:

- public API 不出现 `*redis.Client`、`redis.Cmd`、`redis.Options`
- provider 位于 `internal/provider/goredis`
- provider interface 位于 internal 或 narrow public extension point
- 单元测试可用 fake provider

Evidence:

- AST check
- boundary gate
- provider fake tests

---

### REQ-007: Health Contract

`redisx` 必须输出标准健康状态。

Acceptance Criteria:

- `Health(ctx)` 返回 component、status、latency、error class、checked_at
- Ping timeout 可配置
- auth/network/timeout 可分类
- 不输出 password、full DSN、敏感 key

Evidence:

- health tests
- examples/health smoke
- redaction tests

---

### REQ-008: Observability Contract

所有操作必须有供应商无关观测点。

Acceptance Criteria:

- operation count
- duration histogram
- error count
- pool stats
- health status gauge
- optional tracing span
- logs 不含 value / password / secret

Suggested metrics:

```yaml
redisx_operations_total:
  labels: [operation, status]
redisx_operation_duration_seconds:
  labels: [operation]
redisx_errors_total:
  labels: [operation, error_class]
redisx_pool_connections:
  labels: [state]
redisx_health_status:
  labels: [component]
```

Evidence:

- metrics contract file
- observability unit tests
- examples/metrics smoke

---

### REQ-009: Error Taxonomy

错误必须映射到稳定错误类别。

Acceptance Criteria:

```text
ErrNil
ErrTimeout
ErrCanceled
ErrNetwork
ErrAuth
ErrReadOnly
ErrLoading
ErrTryAgain
ErrClusterMoved
ErrClusterAsk
ErrConnectionClosed
ErrInvalidConfig
ErrProvider
```

Evidence:

- provider error mapping tests
- public error docs
- golden errors fixture

---

### REQ-010: Testkit

必须提供 fake / contract helper。

Acceptance Criteria:

- `testkit.NewFakeRedis()` 或等价 fake provider
- golden fixture
- contract assertions
- 不默认连接真实 Redis
- integration 需要显式 env flag

Evidence:

- `make test`
- `make golden`
- `REDISX_INTEGRATION=1 make integration` optional proof

---

### REQ-011: Harness

redisx 必须拥有与标准源兼容的 Harness gates。

Acceptance Criteria:

Required:

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

Extended:

```bash
GOWORK=off make property
GOWORK=off make golden
GOWORK=off make fuzz-smoke
GOWORK=off make ci-extended
REDISX_INTEGRATION=1 GOWORK=off make integration
```

Final:

```bash
XLIB_CONTEXT=release_verify GOWORK=off make release-check
XLIB_CONTEXT=release_verify GOWORK=off make release-final-check
XLIB_CONTEXT=release_verify GOWORK=off make release-preflight VERSION=v0.1.0
```

Evidence:

- CI artifacts
- release manifest
- score output
- checksum

---

### REQ-012: Release Evidence

每个 release 必须独立生成 Evidence。

Acceptance Criteria:

- `release/manifest/latest.json` generated but not committed
- `release/manifest/latest.json.sha256` generated but not committed
- manifest contains module, commit, tree_sha, source_digest, dependencies, tools, contracts, checks, score, workflow, standard_impact
- `DONE with evidence:` 使用实际 gate 输出

Evidence:

- manifest artifact
- checksum artifact
- workflow artifact URL or local evidence path

---

### REQ-013: Downstream Adoption

redisx release 后必须记录下游采纳状态。

Acceptance Criteria:

- `xlib-standard` adoption registry 更新为 `redisx adopted` 需要真实 redisx Evidence
- `x.go` 如采用，只能作为调用方，不得让 `redisx` 反向依赖
- 下游 adoption PR 必须有独立 Evidence
- 未采纳时明确 `not_adopted` / `blocked owner`

Evidence:

- downstream adoption status file
- downstream CI result
- adoption PR link / artifact
- release note

---

### REQ-014: Self-improving

redisx 执行后必须形成复盘资产。

Acceptance Criteria:

- retrospective file
- prompt patch
- harness patch
- rule patch
- new issue candidates
- standard source patch candidates

Evidence:

- `.agent/retrospectives/RETRO-20260604-redisx-l2.md`
- `.agent/patches/PATCH-PROMPT-*.md`
- `.agent/patches/PATCH-HARNESS-*.md`
- `.agent/patches/PATCH-RULE-*.md`

---

## 9. 目标仓库结构

推荐目标结构：

```text
redisx/
  .agent/
    runtime/
      goal-runtime.md
      object-model.md
      state-machine.md
      rollback-protocol.md
    harness/
      harness.yaml
    evidence/
      evidence-protocol.md
      truth-state.yaml
      GOAL-20260604-REDISX-L2-STANDARD-FACTORY/
    registries/
      command-registry.yaml
      issue-registry.yaml
      adoption-status.yaml
    traceability/
      traceability-matrix.md
      risk-register.md
      decision-log.md
    release/
      release-template.md
    retrospectives/
    patches/
  .github/
    workflows/
      ci.yml
      release-check.yml
      security.yml
  contracts/
    redisx.config.schema.json
    redisx.metrics.yaml
    redisx.errors.yaml
    redisx.health.schema.json
  docs/
    README.md
    goal/
      redisx-l2-standard-factory.md
    spec.md
    design.md
    api.md
    config.md
    errors.md
    observability.md
    testing.md
    release.md
    downstream-adoption.md
  examples/
    basic/
    config/
    health/
    metrics/
  internal/
    provider/
      provider.go
      goredis/
        client.go
        errors.go
        options.go
      fake/
        client.go
    redaction/
    health/
    metrics/
  pkg/
    redisx/
      client.go
      options.go
      errors.go
      health.go
      metrics.go
      commands.go
      lifecycle.go
  release/
    manifest/
      template.json
      .gitkeep
    standard-impact/
      .gitkeep
    downstream-sync/
      .gitkeep
  scripts/
    check_boundary.sh
    check_secrets.sh
    run_integration.sh
    generate_evidence.sh
  testkit/
    fake.go
    assertions.go
    golden/
  Makefile
  go.mod
  go.sum
  README.md
  CHANGELOG.md
  LICENSE
```

---

## 10. Public API 初始设计

### 10.1 Options

```go
package redisx

type Mode string

const (
    ModeStandalone Mode = "standalone"
    ModeSentinel   Mode = "sentinel"
    ModeCluster    Mode = "cluster"
)

type Options struct {
    Name      string
    Mode      Mode
    Addresses []string

    Username string
    Password SecretString
    DB       int

    DialTimeout  time.Duration
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    PoolSize     int
    MinIdleConns int
    MaxRetries   int

    TLS *TLSOptions

    Observability ObservabilityOptions
    Health        HealthOptions
}

type TLSOptions struct {
    Enabled            bool
    ServerName         string
    InsecureSkipVerify bool
}

type HealthOptions struct {
    PingTimeout time.Duration
}

type ObservabilityOptions struct {
    ComponentName string
    EnableMetrics bool
    EnableTracing bool
}
```

### 10.2 Client

```go
type Client interface {
    Ping(ctx context.Context) error
    Health(ctx context.Context) HealthStatus

    Get(ctx context.Context, key string) (Value, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Del(ctx context.Context, keys ...string) (int64, error)
    Exists(ctx context.Context, keys ...string) (int64, error)
    Expire(ctx context.Context, key string, ttl time.Duration) (bool, error)
    TTL(ctx context.Context, key string) (time.Duration, error)

    MGet(ctx context.Context, keys ...string) ([]Value, error)
    MSet(ctx context.Context, values map[string][]byte) error

    Incr(ctx context.Context, key string) (int64, error)
    Decr(ctx context.Context, key string) (int64, error)

    Close(ctx context.Context) error
}
```

### 10.3 Constructor

```go
func NewClient(ctx context.Context, opts Options, deps Dependencies) (Client, error)
```

`Dependencies` 只允许注入标准契约：

```go
type Dependencies struct {
    Logger  observex.Logger
    Metrics observex.Meter
    Tracer  observex.Tracer
    Clock   kernel.Clock
}
```

如果 `observex` 尚未稳定，先以 narrow interface 暂存，并在标准源稳定后替换。

### 10.4 Value

```go
type Value struct {
    Bytes []byte
    Nil   bool
}
```

不直接返回业务 struct；JSON decode 属于 helper，不成为强业务模型。

---

## 11. Redis 专属边界

### 11.1 允许

- 连接管理
- standalone / sentinel / cluster
- ping / health
- KV 原语
- TTL / expire
- atomic counter
- pipeline primitive
- pubsub primitive
- stream primitive
- error classification
- observability
- redaction
- fake / testkit
- optional distributed lock primitive，前提是严格文档声明其语义限制

### 11.2 禁止

- `user:{id}:profile` 这类业务 key 模板
- 默认 TTL 策略
- 应用缓存预热策略
- 业务对象 JSON schema
- x.go import
- profile runtime
- 隐式读取生产 secrets
- 默认连接生产 Redis
- 自动创建业务 key namespace
- 在日志中输出 value、password、完整 DSN

---

## 12. gstack 设计

```text
G0 Standard Source
  xlib-standard: standards, generator, harness, evidence protocol

G1 Repo Runtime
  redisx: .agent, Makefile, CI, docs, contracts, manifest template

G2 Adapter Core
  redisx public API, provider interface, options, errors, health

G3 Provider Implementation
  internal/provider/goredis, fake provider, integration provider

G4 Verification Fabric
  unit, race, boundary, security, contracts, docs, integration, score

G5 Evidence Ledger
  release manifest, checksum, workflow artifact, DONE with evidence

G6 Adoption Layer
  x.go/market-data/macro-data adoption PR, downstream status registry

G7 Self-improving Loop
  retrospective, prompt patch, harness patch, rule patch, xlib-standard patch
```

---

## 13. Superpowers

### 13.1 Boundary Sentinel

自动阻断：

- `x.go` import
- business key patterns
- `/home/k8s/secrets/env/*` 真实内容
- global singleton
- public API 泄漏 provider 类型

### 13.2 Evidence Ledger

每个 issue / PR / release 必须记录：

- command
- result
- artifact
- checksum
- known gaps
- next action

### 13.3 Contract Lock

contracts 变更必须同步：

- docs
- tests
- examples
- traceability matrix
- release manifest template
- downstream sync plan

### 13.4 Adoption Radar

每次 release 后检查：

- `redisx` 是否被 `x.go` 或其他服务采用
- adoption 是否有 CI Evidence
- blocked 原因是什么
- 是否需要 `xlib-standard` 同步

### 13.5 AutoResearch Guard

当出现以下未知项时进入 `NEEDS_RESEARCH`：

- Redis provider 版本变化
- Redis 8.x 命令兼容性变化
- go-redis API breaking change
- `kernel/configx/observex` contract 未稳定
- L1 dependency matrix 未授权
- CI action pinning 策略变化

### 13.6 Compound Engineering Loop

每个 redisx 交付不只是修 redisx，还要产出：

- 一条可复用标准
- 一个 Harness 检查
- 一个文档模板
- 一个 issue 模板
- 一个 release evidence 模板
- 一个 downstream adoption 模板

---

## 14. Harness Gates 设计

### 14.1 Semantic Gates

| Gate | 检查内容 |
|---|---|
| `goal-check` | Goal 对象完整，ID、状态机、scope、DoD 完整 |
| `spec-check` | Requirements、AC、非目标、边界完整 |
| `design-check` | provider isolation、config、health、observability、error mapping |
| `traceability-check` | REQ -> AC -> Design -> Task -> Test -> Evidence 完整 |
| `docs-check` | 文档入口、命名、标准源引用、release protocol |
| `adoption-check` | adoption status 不得无证据升级 |

### 14.2 Executable Gates

| Gate | 命令 |
|---|---|
| format | `GOWORK=off make fmt` |
| vet | `GOWORK=off make vet` |
| lint | `GOWORK=off make lint` |
| unit | `GOWORK=off make test` |
| race | `GOWORK=off make race` |
| security | `GOWORK=off make security` |
| contracts | `GOWORK=off make contracts` |
| boundary | `GOWORK=off make boundary` |
| dependency | `GOWORK=off make dependency-check` |
| score | `GOWORK=off go run ./cmd/goalcli score --min 9.8` |
| evidence | `CHECK_STATUS=passed GOWORK=off make evidence` |

### 14.3 Hybrid Gates

| Gate | 检查内容 |
|---|---|
| `release-check` | CI + docs + contracts + dependency + standard impact + evidence |
| `release-final-check` | release-check + score + clean workspace + manifest checksum |
| `downstream-sync-plan` | standard impact + downstream status + blocked owner |
| `integration` | generator + fake + optional ephemeral Redis |

---

## 15. Evidence Protocol

### 15.1 完成声明格式

```text
DONE with evidence:
- scope: goal
- commit: <sha>
- branch: goal/redisx-l2-standard-factory
- tag: v0.1.0 or not_created
- gates:
  - GOWORK=off make fmt: passed <log/artifact>
  - GOWORK=off make vet: passed <log/artifact>
  - GOWORK=off make lint: passed <log/artifact>
  - GOWORK=off make test: passed <log/artifact>
  - GOWORK=off make race: passed <log/artifact>
  - GOWORK=off make boundary: passed <log/artifact>
  - GOWORK=off make security: passed <log/artifact>
  - GOWORK=off make contracts: passed <log/artifact>
  - GOWORK=off make docs-check: passed <log/artifact>
  - GOWORK=off make dependency-check: passed <log/artifact>
  - GOWORK=off make standard-impact-check: passed <log/artifact>
  - GOWORK=off make score: passed score >= 9.8
  - CHECK_STATUS=passed GOWORK=off make evidence: passed
  - RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check: passed
- artifacts:
  - release/manifest/latest.json: release manifest artifact, not committed
  - release/manifest/latest.json.sha256: checksum artifact, not committed
  - release/standard-impact/latest.md: standard impact report
  - release/downstream-sync/latest.md: downstream sync report
  - .agent/evidence/GOAL-20260604-REDISX-L2-STANDARD-FACTORY/: issue/goal evidence
- known gaps:
  - none
```

### 15.2 禁止

- 禁止用“已实现”代替 Evidence。
- 禁止 skipped required gate 写 passed。
- 禁止 dirty workspace 宣称 release-final ready。
- 禁止把本地生成 smoke 当成真实下游 adoption。
- 禁止把 `xlib-standard` registry 目标态当作 redisx 已采纳证据。

---

## 16. Worktree / Branch / PR 铁律

### 16.1 禁止 main 开发

所有实现必须通过 worktree 分支：

```bash
git clone git@github.com:ZoneCNH/redisx.git
cd redisx
git checkout main
git pull --ff-only

git worktree add ../redisx-goal-20260604 \
  -b goal/redisx-l2-standard-factory

cd ../redisx-goal-20260604
```

### 16.2 本地 hooks

```bash
make install-hooks
make doctor-hooks
```

如果当前 redisx 尚未生成 Makefile，则先完成标准源生成，再运行 hooks。

### 16.3 Commit 规则

每个 PR 至少包含：

- issue ID
- task ID
- evidence path
- gate result
- known gaps

Commit message 示例：

```text
feat(redisx): bootstrap xlib-standard L2 runtime

Goal: GOAL-20260604-REDISX-L2-STANDARD-FACTORY
Task: TASK-GOAL-20260604-REDISX-L2-STANDARD-FACTORY-001
Evidence: .agent/evidence/GOAL-20260604-REDISX-L2-STANDARD-FACTORY/TASK-001.md
```

---

## 17. 标准源生成步骤

### 17.1 准备标准源

```bash
git clone git@github.com:ZoneCNH/xlib-standard.git ../xlib-standard
cd ../xlib-standard
git checkout main
git pull --ff-only
GOWORK=off make release-final-check
```

如果 `release-final-check` 失败：

- 不要继续生成。
- 记录 `NEEDS_STANDARD_SOURCE_FIX`。
- 先修复 `xlib-standard` 或明确使用哪个 commit 作为标准源基线。

### 17.2 生成 redisx

```bash
cd ../xlib-standard

scripts/render_template.sh \
  --module-name redisx \
  --module-path github.com/ZoneCNH/redisx \
  --package-name redisx \
  --out ../redisx-goal-20260604
```

### 17.3 生成后修复

检查：

```bash
cd ../redisx-goal-20260604

grep -R "templatex\|xlib-standard\|foundationx\|baselib-template" -n \
  README.md docs .agent contracts pkg internal examples || true

grep -R "github.com/bytechainx/x.go\|/home/k8s/secrets/env" -n . || true

GOWORK=off go test ./...
```

必须把模板残留全部替换为 redisx 当前身份。

---

## 18. 实施里程碑

### M0: Context Recovery / 标准源对齐

目标：

- 确认 xlib-standard 当前可用。
- 确认 redisx 当前空壳状态。
- 明确标准源 commit。

交付：

- `.agent/evidence/context-recovery.md`
- `docs/goal/redisx-l2-standard-factory.md`
- initial issue registry

Gate：

```bash
GOWORK=off make docs-check
```

---

### M1: Repo Bootstrap / 控制面落地

目标：

- 用 generator 生成 redisx 标准结构。
- 初始化 go.mod、Makefile、CI、contracts、docs、.agent。

交付：

- repo structure
- Makefile
- `.github/workflows/*`
- `.agent/*`
- `release/manifest/template.json`

Gate：

```bash
GOWORK=off make fmt
GOWORK=off make vet
GOWORK=off make test
```

---

### M2: Boundary / Contract Lock

目标：

- 固定 L2 边界。
- 禁止 x.go、业务 key、secret、global singleton。
- 建立 contracts。

交付：

- `docs/spec.md`
- `docs/design.md`
- `docs/api.md`
- `contracts/redisx.config.schema.json`
- `contracts/redisx.metrics.yaml`
- `contracts/redisx.errors.yaml`

Gate：

```bash
GOWORK=off make boundary
GOWORK=off make contracts
GOWORK=off make security
```

---

### M3: API / Provider Adapter

目标：

- 实现公共 API。
- 实现 provider interface。
- 实现 `internal/provider/goredis`。
- 实现 fake provider。

交付：

- `pkg/redisx/*.go`
- `internal/provider/provider.go`
- `internal/provider/goredis/*.go`
- `internal/provider/fake/*.go`

Gate：

```bash
GOWORK=off make test
GOWORK=off make race
```

---

### M4: Config / Health / Observability

目标：

- 显式配置。
- health contract。
- metrics / tracing / logging contract。
- secret redaction。

交付：

- `pkg/redisx/options.go`
- `pkg/redisx/health.go`
- `pkg/redisx/metrics.go`
- `internal/redaction/*`
- examples/config
- examples/health
- examples/metrics

Gate：

```bash
GOWORK=off make test
GOWORK=off make contracts
GOWORK=off make security
```

---

### M5: Testkit / Integration

目标：

- fake / golden / property / fuzz smoke。
- optional ephemeral Redis integration。

交付：

- `testkit/fake.go`
- `testkit/assertions.go`
- `testkit/golden/*`
- `scripts/run_integration.sh`
- `examples/basic`

Gate：

```bash
GOWORK=off make golden
GOWORK=off make property
GOWORK=off make fuzz-smoke
REDISX_INTEGRATION=1 GOWORK=off make integration
```

---

### M6: Release Evidence

目标：

- 完成 release manifest。
- 完成 release score。
- 完成 checksum。
- 完成 release final gate。

交付：

- generated `release/manifest/latest.json`
- generated `release/manifest/latest.json.sha256`
- `release/standard-impact/latest.md`
- `release/downstream-sync/latest.md`

Gate：

```bash
GOWORK=off make dependency-check
GOWORK=off make standard-impact-check
GOWORK=off make downstream-sync-plan
GOWORK=off go run ./cmd/goalcli score --min 9.8
CHECK_STATUS=passed GOWORK=off make evidence
RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check
XLIB_CONTEXT=release_verify GOWORK=off make release-final-check
```

---

### M7: Downstream Adoption

目标：

- 更新 xlib-standard adoption status。
- 至少创建一个 downstream smoke consumer。
- 若 x.go 暂不采纳，明确 blocked owner。

交付：

- `docs/downstream-adoption.md`
- adoption registry update proposal
- downstream PR / smoke evidence
- known gap if blocked

Gate：

```bash
GOWORK=off make downstream-sync-plan
GOWORK=off make integration DOWNSTREAM=kernel
```

---

### M8: Self-improving

目标：

- 把 redisx 实施中发现的问题反向沉淀。
- 形成 prompt/harness/rule patch。
- 创建 xlib-standard patch issue。

交付：

- `RETRO-20260604-redisx-l2.md`
- `PATCH-PROMPT-20260604-redisx-l2.md`
- `PATCH-HARNESS-20260604-redisx-l2.md`
- `PATCH-RULE-20260604-redisx-l2.md`
- `xlib-standard` follow-up issues

Gate：

```bash
GOWORK=off make retrospective-check
```

---

## 19. Task 分解

### TASK-001: 创建 Goal 文档

```yaml
task_id: TASK-GOAL-20260604-REDISX-L2-STANDARD-FACTORY-001
title: Create executable goal document
owner: agent
status: todo
```

DoD:

- Goal / Spec / Design / Plan / Tasks 完整
- Traceability Matrix 初始化
- Risk Register 初始化
- Evidence 目录初始化

---

### TASK-002: 标准源基线锁定

DoD:

- 记录 `xlib-standard` commit SHA
- 记录 `xlib-standard` release gate 状态
- 若失败，进入 `NEEDS_STANDARD_SOURCE_FIX`

---

### TASK-003: 生成 redisx 标准结构

DoD:

- generator 成功
- module path 正确
- package name 正确
- 无模板残留
- `go test ./...` 通过

---

### TASK-004: 建立 Redis L2 边界

DoD:

- docs/spec.md 写明 include/exclude
- boundary gate 能拦截 x.go import
- boundary gate 能拦截 business key policy
- boundary gate 能拦截 secret path content

---

### TASK-005: 设计并实现 Options

DoD:

- 显式配置结构
- config schema
- validation
- redaction
- no env implicit read

---

### TASK-006: 设计并实现 Client API

DoD:

- Client interface
- constructor
- lifecycle
- KV MVA
- no provider leak

---

### TASK-007: 实现 go-redis provider

DoD:

- internal only
- error mapping
- timeout/context
- standalone MVA
- cluster/sentinel config placeholders or implementation

---

### TASK-008: 实现 fake provider 和 testkit

DoD:

- unit test 不依赖真实 Redis
- fake supports MVA commands
- testkit assertions
- golden fixtures

---

### TASK-009: Health / Metrics / Trace

DoD:

- health status contract
- metrics contract
- observability tests
- examples/health and examples/metrics

---

### TASK-010: Security / Redaction

DoD:

- password redacted
- DSN redacted
- key optional redaction
- values never logged by default
- secret scan passed

---

### TASK-011: CI / Harness

DoD:

- required gates in Makefile
- GitHub workflows
- local hooks
- score gate
- release gate

---

### TASK-012: Release Evidence

DoD:

- manifest generated
- checksum generated
- score >= 9.8
- workflow artifact or local artifact recorded
- `DONE with evidence:` completed

---

### TASK-013: Downstream Adoption

DoD:

- adoption status documented
- downstream smoke attempted
- if not adopted, blocked owner recorded
- no false adoption claim

---

### TASK-014: Retrospective and Patches

DoD:

- retro completed
- prompt patch
- harness patch
- rule patch
- xlib-standard follow-up issue candidates

---

## 20. Traceability Matrix

| Requirement | Acceptance Criteria | Design Section | Task | Test | Evidence | Status |
|---|---|---|---|---|---|---|
| REQ-001 | generator creates redisx module | Repo Bootstrap | TASK-002/003 | go test ./... | generator log | closed |
| REQ-002 | no x.go / no business policy | Boundary | TASK-004 | boundary scan | boundary output | closed |
| REQ-003 | explicit config | Config | TASK-005 | config tests | schema + test log | partial |
| REQ-004 | lifecycle close | Client API | TASK-006 | lifecycle tests | unit/race log | closed |
| REQ-005 | KV MVA | Client API | TASK-006/007 | KV tests | fake/integration log | partial |
| REQ-006 | provider isolation | Provider | TASK-007 | AST/boundary | boundary output | closed |
| REQ-007 | health | Health | TASK-009 | health tests | examples/health | closed |
| REQ-008 | observability | Metrics | TASK-009 | metrics tests | contract output | closed |
| REQ-009 | error taxonomy | Errors | TASK-007 | error golden | golden output | closed |
| REQ-010 | testkit | Testing | TASK-008 | fake/golden | testkit output | closed |
| REQ-011 | harness | Harness | TASK-011 | make gates | CI artifact | closed |
| REQ-012 | release evidence | Release | TASK-012 | release-check | manifest + sha | partial |
| REQ-013 | downstream adoption | Adoption | TASK-013 | downstream smoke | adoption registry | not adopted |
| REQ-014 | self-improving | Retro | TASK-014 | retrospective-check | patch files | closed |

---

### 20.1 Closeout Audit — 2026-06-05

本轮 closeout 已同步 `xlib-standard` L2 standard-source 的 registry、schema、gate、testing docs、contract test 入口和 readiness/compliance evidence snapshot，并修复本地 L2 verifier 为 stdlib-only 运行路径。详细审计记录见 `docs/goal/redisx-l2-audit-20260605.md`。

当前矩阵只关闭本地已有证据的范围：`l2-check`、`test-contract`、`contracts`、`docs-check`、`command-registry`、`go test ./...` 和 verifier 均已通过。`REQ-003`、`REQ-005` 与 `REQ-012` 保持 partial，因为真实 Redis-specific public config、provider-backed contract report、integration report 和 release-ready evidence 仍未完成。`REQ-013` 保持 `not adopted`，不得在缺少 downstream repo/commit 证据时声明 adoption。

---

## 21. Risk Register

| Risk ID | Risk | Impact | Probability | Mitigation | Gate |
|---|---|---:|---:|---|---|
| RISK-001 | xlib-standard generator 当前不支持 L2 redisx 专属结构 | High | Medium | 先生成通用模板，再补 Redis delta，并回写 generator patch | generator gate |
| RISK-002 | kernel/configx/observex 尚未稳定发布 | High | Medium | 暂用 narrow interface + replace only in branch；release 前必须锁定 tag | dependency-check |
| RISK-003 | redisx 公共 API 泄漏 go-redis 类型 | High | Medium | AST boundary gate | boundary |
| RISK-004 | 误把业务缓存策略写入 redisx | High | Medium | docs + grep + review | boundary/docs |
| RISK-005 | 默认测试连接生产 Redis | Critical | Low | integration 必须 opt-in | test/security |
| RISK-006 | secret 泄漏到 logs/manifest | Critical | Medium | redaction + secret scan | security |
| RISK-007 | release Evidence 生成但被提交 | Medium | Medium | `.gitignore` + release-evidence-check | release |
| RISK-008 | downstream adoption 被误宣称 | High | Medium | adoption registry 必须 evidence-backed | adoption |
| RISK-009 | go-redis API 版本变化 | Medium | Medium | provider isolation + dependency automation | dependency |
| RISK-010 | L2 依赖 resiliencx 但标准源未允许 | Medium | High | 先在 xlib-standard 更新 downstream matrix，再进入 redisx runtime deps | standard-impact |
| RISK-011 | main 分支直接开发 | High | Medium | hooks + worktree gate | governance |
| RISK-012 | score < 9.8 仍发布 | High | Low | release-final-check fail-fast | score |

---

## 22. Decision Log / ADR

### ADR-20260604-001: redisx 使用 xlib-standard generator 初始化

Decision:

- 使用 `xlib-standard` generator 生成 redisx 标准结构。
- 不在当前空壳 README 仓库上手写累加。

Reason:

- 确保 `.agent`、Harness、Evidence、contracts、release gate 与标准源一致。

Status:

- accepted

---

### ADR-20260604-002: redisx 默认 provider 采用 go-redis v9

Decision:

- 默认 provider 使用 `github.com/redis/go-redis/v9`。
- Provider 隔离在 `internal/provider/goredis`。

Reason:

- 官方 Redis Go client，生态成熟，功能覆盖 standalone / cluster / sentinel / pubsub / pipeline。
- 内部隔离后可替换，不污染公共 API。

Status:

- accepted with provider isolation

---

### ADR-20260604-003: 不实现业务缓存策略

Decision:

- `redisx` 不提供业务 key schema、默认缓存 TTL、业务缓存刷新策略。

Reason:

- L2 只能是基础设施 adapter。
- 应用缓存策略属于调用方。

Status:

- accepted

---

### ADR-20260604-004: Integration test 默认关闭

Decision:

- 默认 `make test` 使用 fake provider。
- 真实 Redis integration 通过 `REDISX_INTEGRATION=1` 显式开启。

Reason:

- 避免测试触达生产 Redis。
- 保持 CI 快速稳定。

Status:

- accepted

---

### ADR-20260604-005: resiliencx 不作为当前 runtime deps 强制引入

Decision:

- 当前 runtime deps 严格遵循标准源允许依赖：`kernel`、`configx`、`observex`。
- retry/timeout/circuit/bulkhead/rate limit 先暴露 hook / options，不直接依赖 `resiliencx`。
- 若标准源后续允许 `resiliencx` 作为 L1 runtime dependency，再引入。

Reason:

- 防止 L2 擅自突破标准源依赖矩阵。

Status:

- accepted

---

## 23. API 迭代路线

### v0.1.0 MVA

- repo bootstrap
- config
- lifecycle
- health
- metrics contract
- error taxonomy
- basic KV
- fake provider
- go-redis standalone provider
- release evidence

### v0.2.0

- cluster mode
- sentinel mode
- pipeline primitive
- scan helper
- richer pool stats
- testcontainers / docker compose integration

### v0.3.0

- pubsub primitive
- stream primitive
- distributed lock primitive with warning docs
- Lua script primitive
- OpenTelemetry bridge via observex if standard allows

### v0.4.0

- resiliencx integration if downstream matrix allows
- circuit/bulkhead/rate-limit policy injection
- downstream x.go adoption proof
- standard factory pattern generalized to other L2 adapters

---

## 24. 最小可行行动 MVA

MVA 不是完整 Redis SDK，而是能证明 `redisx` 已成为 L2 标准工厂实例。

### MVA 必须完成

1. `redisx` 从 `xlib-standard` 生成标准结构。
2. `go.mod` 正确。
3. `.agent` 控制面存在。
4. Required Harness gates 可运行。
5. Public API 包含 Options / Client / Health / Errors。
6. fake provider 支持 KV MVA。
7. go-redis provider 支持 standalone KV MVA。
8. docs / contracts / examples 完整。
9. release manifest 可生成。
10. `DONE with evidence:` 可填写。

### MVA 不做

- 业务缓存策略
- 复杂分布式锁
- Redis Streams 完整消费框架
- Pub/Sub 重连策略完整工程化
- x.go 深度接入
- resiliencx runtime dependency

---

## 25. 1 天行动计划

目标：把 redisx 从空壳推进到标准源控制的可执行骨架。

### Day 1 Checklist

1. 锁定 `xlib-standard` commit。
2. 创建 redisx worktree 分支。
3. 用 generator 渲染 redisx。
4. 修正模板残留。
5. 建立 docs/goal。
6. 初始化 requirements / traceability / risk register。
7. 初始化 contracts。
8. 确认 `go test ./...` 可跑。
9. 建立 boundary gate 初版。
10. 提交 PR-1：bootstrap control plane。

### Day 1 Commands

```bash
git clone git@github.com:ZoneCNH/redisx.git
cd redisx
git pull --ff-only
git worktree add ../redisx-goal-20260604 -b goal/redisx-l2-standard-factory

git clone git@github.com:ZoneCNH/xlib-standard.git ../xlib-standard
cd ../xlib-standard
git pull --ff-only
GOWORK=off make release-final-check

scripts/render_template.sh \
  --module-name redisx \
  --module-path github.com/ZoneCNH/redisx \
  --package-name redisx \
  --out ../redisx-goal-20260604

cd ../redisx-goal-20260604
GOWORK=off go test ./...
GOWORK=off make docs-check || true
GOWORK=off make boundary || true
```

### Day 1 Evidence

```text
.agent/evidence/GOAL-20260604-REDISX-L2-STANDARD-FACTORY/day1.md
```

---

## 26. 7 天行动计划

目标：完成 v0.1.0 MVA，可发布候选。

### Day 1-2: Bootstrap + Boundary

- generator
- docs
- contracts
- boundary/security gates
- CI baseline

### Day 3: API + Config

- Options
- validation
- redaction
- constructor
- no global singleton

### Day 4: Provider + Fake

- provider interface
- fake provider
- go-redis standalone provider
- error mapping

### Day 5: Health + Observability

- Ping
- Health
- metrics
- tracing/logging hook
- examples

### Day 6: Testing + Integration

- unit
- race
- golden
- property
- fuzz-smoke
- optional ephemeral Redis integration

### Day 7: Release Evidence + PR Review

- release manifest
- checksum
- score >= 9.8
- release-final-check
- PR summary
- DONE with evidence

---

## 27. 30 天行动计划

目标：从 redisx 单库完成，扩展为 L2 基础设施适配层标准工厂范式。

### Week 1: redisx v0.1.0

- 完成 MVA。
- 独立 release。
- Evidence 完整。

### Week 2: redisx v0.2.0

- cluster / sentinel
- pipeline
- scan
- stronger integration
- dependency automation

### Week 3: 下游采纳

- x.go or market-data smoke adoption
- adoption registry
- downstream PR evidence
- blocked owner if not adopted

### Week 4: 反向沉淀

- xlib-standard generator patch
- L2 adapter template patch
- redisx learnings -> kafkax/postgresx/taosx standard patches
- Create L2 common checklist
- Define L2 factory scorecard

---

## 28. 指标体系

### 28.1 工程质量指标

| Metric | Target |
|---|---:|
| release score | >= 9.8 |
| required gates pass rate | 100% |
| public API provider leakage | 0 |
| x.go imports | 0 |
| secret scan findings | 0 |
| business key policy findings | 0 |
| test race failures | 0 |
| generated manifest checksum mismatch | 0 |

### 28.2 Redis 功能指标

| Metric | Target |
|---|---:|
| KV MVA command coverage | >= 10 commands |
| error taxonomy coverage | >= 12 classes |
| fake provider parity for MVA | 100% |
| health check latency budget | configurable |
| default tests requiring real Redis | 0 |

### 28.3 工厂复利指标

| Metric | Target |
|---|---:|
| reusable xlib patches | >= 3 |
| harness patch candidates | >= 1 |
| rule patch candidates | >= 1 |
| downstream adoption evidence | >= 1 or explicit blocked owner |
| new L2 template improvements | >= 3 |

---

## 29. 迭代优化机制

### 29.1 每个 PR 后

更新：

- traceability matrix
- issue registry
- evidence
- risk register
- decision log

### 29.2 每个 Release 后

生成：

- release manifest
- checksum
- release score
- standard impact
- downstream sync plan
- retrospective

### 29.3 每次失败后

失败 Evidence 必须记录：

```text
- command
- exit code
- key error
- affected requirement
- unaffected scope
- fix condition
- owner
- next task
```

### 29.4 每月

执行 L2 factory review：

- redisx 是否仍符合 xlib-standard？
- xlib-standard 是否需要吸收 redisx learnings？
- 是否可以复制到 `postgresx/kafkax/taosx/ossx/clickhousex/natsx`？
- 下游 adoption 是否真实发生？
- 哪些 gates 误报 / 漏报？
- 哪些 docs 过时？

---

## 30. Issue / PR 执行序列

### Issue 1 / PR 1: Bootstrap redisx standard runtime

Scope:

- generator
- repo structure
- .agent
- docs/goal
- Makefile/CI skeleton

DoD:

- `go test ./...`
- `docs-check`
- evidence day1

---

### Issue 2 / PR 2: Lock L2 boundaries and contracts

Scope:

- boundary gate
- contracts
- docs/spec
- docs/design
- security/redaction rules

DoD:

- `make boundary`
- `make contracts`
- `make security`

---

### Issue 3 / PR 3: Implement API and fake provider

Scope:

- Options
- Client
- Value
- Error taxonomy
- fake provider

DoD:

- unit tests
- race tests
- fake parity tests

---

### Issue 4 / PR 4: Implement go-redis provider

Scope:

- internal provider
- standalone connection
- KV operations
- error mapping
- close lifecycle

DoD:

- provider tests
- optional integration tests
- no public provider leakage

---

### Issue 5 / PR 5: Health and observability

Scope:

- health
- metrics
- traces/logs
- examples

DoD:

- contracts pass
- examples smoke
- redaction pass

---

### Issue 6 / PR 6: Release Evidence

Scope:

- release manifest
- score
- standard impact
- downstream sync plan
- release-final-check

DoD:

- `DONE with evidence:`
- score >= 9.8
- clean workspace

---

### Issue 7 / PR 7: Downstream adoption

Scope:

- xlib-standard adoption status proposal
- downstream smoke
- x.go consumer boundary

DoD:

- adoption evidence or blocked owner
- no false adoption claim

---

### Issue 8 / PR 8: Self-improving patches

Scope:

- retrospective
- prompt/harness/rule patch
- generator improvement issue

DoD:

- patches created
- xlib-standard issue candidates filed

---

## 31. Release Gate 标准

### 31.1 Pre-release

```bash
GOWORK=off make ci
GOWORK=off make ci-extended
GOWORK=off make dependency-check
GOWORK=off make standard-impact-check
GOWORK=off make downstream-sync-plan
GOWORK=off make docs-check
GOWORK=off go run ./cmd/goalcli score --min 9.8
```

### 31.2 Release

```bash
XLIB_CONTEXT=release_verify GOWORK=off make release-check
XLIB_CONTEXT=release_verify GOWORK=off make release-final-check
XLIB_CONTEXT=release_verify GOWORK=off make release-preflight VERSION=v0.1.0
```

### 31.3 Tag

```bash
git tag -a v0.1.0 -m "redisx v0.1.0: xlib-standard L2 factory MVA"
git push origin v0.1.0
```

Tag 前必须确认：

- workspace clean
- release-final-check passed
- release manifest artifact saved
- changelog updated
- no generated manifest committed
- no secret
- no false downstream adoption claim

---

## 32. Rollback Protocol

如果 release 后失败：

1. 标记 release 为 `suspect`。
2. 不删除失败 Evidence。
3. 创建 hotfix branch：

```bash
git worktree add ../redisx-hotfix-v0.1.1 -b hotfix/redisx-v0.1.1
```

4. 修复并生成新 Evidence。
5. 发布 `v0.1.1`。
6. 在 retrospective 中记录：
   - 失败原因
   - gate 为什么没拦住
   - 新增 Harness patch
   - 新增 Rule patch

---

## 33. Security Policy

### 33.1 Secret

禁止：

- 输出 Redis password
- 输出完整 DSN
- 输出 `/home/k8s/secrets/env/*` 真实内容
- 把 secret 写入 README、tests、manifest、PR、Issue

允许：

- 文档中把 `/home/k8s/secrets/env/*` 作为调用方部署路径名出现
- 使用 mock secret：`REDACTED` / `example-password`

### 33.2 Logs

默认日志只能包含：

- component
- operation
- status
- duration
- error class
- redacted address
- database number if non-sensitive

默认日志不得包含：

- key value
- Redis value
- password
- auth token
- full URI

### 33.3 Config

配置必须显式：

```go
redisx.NewClient(ctx, redisx.Options{
    Addresses: []string{"127.0.0.1:6379"},
    Password: redisx.SecretString("REDACTED"),
}, deps)
```

---

## 34. Docs DoD

必须有：

- `README.md`
- `docs/spec.md`
- `docs/design.md`
- `docs/api.md`
- `docs/config.md`
- `docs/errors.md`
- `docs/observability.md`
- `docs/testing.md`
- `docs/release.md`
- `docs/downstream-adoption.md`
- `CHANGELOG.md`

README 必须说明：

- redisx 是 L2 基础设施适配库
- allowed dependencies
- forbidden responsibilities
- installation
- basic usage
- config
- health
- metrics
- testing
- release evidence
- security

---

## 35. AutoResearch Protocol

当遇到以下问题，不能猜：

| Trigger | Action |
|---|---|
| go-redis 最新版本/Go 版本要求不确定 | 查官方 repo / pkg.go.dev |
| Redis server version compatibility 不确定 | 查 Redis / go-redis docs |
| L1 契约未稳定 | 查对应 repo release / xlib downstream matrix |
| Harness 命令和当前 repo 不一致 | 查 xlib-standard Makefile / docs / .agent registry |
| CI action 版本不确定 | 查 GitHub workflow / action release |
| dependency vulnerability | 查 govulncheck / advisory |
| downstream adoption 状态不确定 | 查 adoption registry / downstream CI |

输出格式：

```text
NEEDS_RESEARCH:
- question:
- source checked:
- finding:
- decision:
- evidence:
- follow-up:
```

---

## 36. Final 推荐路径

最佳路径：

```text
先标准源生成
  -> 再锁边界
  -> 再做 MVA API
  -> 再做 provider
  -> 再做 health/metrics
  -> 再做 release Evidence
  -> 再做 downstream adoption
  -> 最后把经验回写 xlib-standard
```

不要走：

```text
直接手写 Redis client wrapper
  -> 补几个测试
  -> 写 README
  -> 打 tag
```

原因：

- 会继续制造零散基础设施库。
- 无法证明边界。
- 无法证明下游采纳。
- 无法复用到其他 L2 adapter。
- 无法形成 Compound Engineering 复利。

---

## 37. Agent 可执行 Master Prompt

下面这段可以直接交给 Codex / Agent Team 执行。

```text
You are executing GOAL-20260604-REDISX-L2-STANDARD-FACTORY.

Mission:
Upgrade github.com/ZoneCNH/redisx from a minimal Redis wrapper repository into an xlib-standard controlled L2 infrastructure adapter standard factory instance.

Authoritative standard source:
github.com/ZoneCNH/xlib-standard

Target module:
github.com/ZoneCNH/redisx

Target package:
redisx

Execution mode:
Goal Runtime Prompt v3.1 Full Mode.

Mandatory runtime chain:
Goal -> Context Recovery -> Spec -> Design -> Plan -> Tasks -> Execution -> Verification -> Evidence -> Review -> Release -> Retrospective -> Self-improving.

Hard constraints:
1. Do not develop on main.
2. Use git worktree.
3. Do not import x.go.
4. Do not read /home/k8s/secrets/env/*.
5. Do not implement business key semantics.
6. Do not implement application cache policy.
7. Do not expose go-redis types in public API.
8. Do not claim DONE without Evidence.
9. Do not commit release/manifest/latest.json or latest.json.sha256.
10. Do not mark downstream adoption as adopted without external downstream repo Evidence.

Required steps:
1. Lock xlib-standard commit and verify standard source gate.
2. Generate redisx standard structure with scripts/render_template.sh.
3. Replace all template identity leftovers with redisx identity.
4. Create docs/goal/redisx-l2-standard-factory.md.
5. Create SPEC-redisx-l2-v1.0 and DESIGN-redisx-l2-v1.0.
6. Implement redisx Options, Client, Value, HealthStatus, error taxonomy.
7. Implement provider interface and internal/provider/goredis using github.com/redis/go-redis/v9.
8. Implement fake provider and testkit.
9. Implement explicit config validation and redaction.
10. Implement health and observability contracts using kernel/configx/observex interfaces only.
11. Add boundary gate for x.go imports, provider leakage, business key policy, production secret leakage, hidden global singleton.
12. Add unit, race, golden, property and optional integration tests.
13. Run all required gates:
    - GOWORK=off make fmt
    - GOWORK=off make vet
    - GOWORK=off make lint
    - GOWORK=off make test
    - GOWORK=off make race
    - GOWORK=off make boundary
    - GOWORK=off make security
    - GOWORK=off make contracts
    - GOWORK=off make docs-check
    - GOWORK=off make dependency-check
    - GOWORK=off make standard-impact-check
    - GOWORK=off make downstream-sync-plan
    - GOWORK=off go run ./cmd/goalcli score --min 9.8
    - CHECK_STATUS=passed GOWORK=off make evidence
    - RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check
    - XLIB_CONTEXT=release_verify GOWORK=off make release-final-check
14. Produce DONE with evidence.
15. Produce retrospective and prompt/harness/rule patch candidates for xlib-standard.

Definition of Done:
DONE with evidence is allowed only when:
- release manifest exists as generated artifact and is not committed
- manifest checksum exists as generated artifact and is not committed
- score >= 9.8
- required gates passed
- workspace clean for release-final-check
- boundary has zero P0 findings
- security has zero secret findings
- public API does not leak provider
- downstream adoption is either proven by evidence or explicitly marked not_adopted/blocked
- retrospective and self-improving patches are created
```

---

## 38. 最终交付清单

### Code

- `pkg/redisx`
- `internal/provider`
- `internal/redaction`
- `testkit`
- `examples`

### Contracts

- `contracts/redisx.config.schema.json`
- `contracts/redisx.metrics.yaml`
- `contracts/redisx.errors.yaml`
- `contracts/redisx.health.schema.json`

### Docs

- `README.md`
- `docs/spec.md`
- `docs/design.md`
- `docs/api.md`
- `docs/config.md`
- `docs/errors.md`
- `docs/observability.md`
- `docs/testing.md`
- `docs/release.md`
- `docs/downstream-adoption.md`

### Harness

- `Makefile`
- `scripts/*`
- `.github/workflows/*`
- `.agent/harness/harness.yaml`

### Evidence

- `.agent/evidence/GOAL-20260604-REDISX-L2-STANDARD-FACTORY/*`
- `release/manifest/latest.json`
- `release/manifest/latest.json.sha256`
- `release/standard-impact/latest.md`
- `release/downstream-sync/latest.md`
- `DONE with evidence:`

### Self-improving

- retrospective
- prompt patch
- harness patch
- rule patch
- xlib-standard issue candidates

---

## 39. 深度遗漏检查表

### Standard Source

- [ ] xlib-standard commit recorded
- [ ] xlib-standard generator used
- [ ] template identity replaced
- [ ] generated redisx compiles
- [ ] downstream matrix checked

### Layering

- [ ] L2 role documented
- [ ] allowed deps enforced
- [ ] x.go import banned
- [ ] business key policy banned
- [ ] application cache strategy banned

### API

- [ ] Options
- [ ] Client
- [ ] Value
- [ ] HealthStatus
- [ ] Error taxonomy
- [ ] Close lifecycle
- [ ] no provider leak

### Redis

- [ ] standalone
- [ ] fake provider
- [ ] KV MVA
- [ ] TTL
- [ ] MGET/MSET
- [ ] counter
- [ ] error mapping
- [ ] optional integration

### Config

- [ ] explicit options
- [ ] config schema
- [ ] validation
- [ ] redaction
- [ ] no implicit env

### Observability

- [ ] metrics
- [ ] tracing hook
- [ ] logger hook
- [ ] health
- [ ] no sensitive logs

### Testing

- [ ] unit
- [ ] race
- [ ] fake
- [ ] golden
- [ ] property
- [ ] fuzz smoke
- [ ] optional integration
- [ ] examples smoke

### Security

- [ ] secret scan
- [ ] redaction
- [ ] no DSN leak
- [ ] no value logs
- [ ] no production connection default

### Harness

- [ ] fmt
- [ ] vet
- [ ] lint
- [ ] test
- [ ] race
- [ ] boundary
- [ ] security
- [ ] contracts
- [ ] docs-check
- [ ] dependency-check
- [ ] standard-impact-check
- [ ] score
- [ ] evidence
- [ ] release-evidence-check
- [ ] release-final-check

### Release

- [ ] manifest generated
- [ ] checksum generated
- [ ] manifest ignored
- [ ] checksum ignored
- [ ] score >= 9.8
- [ ] changelog updated
- [ ] clean workspace
- [ ] tag only after final gate

### Downstream

- [ ] adoption registry not falsely upgraded
- [ ] downstream smoke attempted
- [ ] blocked owner if not adopted
- [ ] x.go remains consumer only

### Self-improving

- [ ] retrospective
- [ ] prompt patch
- [ ] harness patch
- [ ] rule patch
- [ ] xlib-standard follow-up issues

---

## 40. 最终推荐路径

最终推荐：

1. **先把 redisx 标准化，不急着扩展 Redis 功能。**
2. **v0.1.0 只证明它是合格 L2 标准工厂实例。**
3. **v0.2.0 再扩展 cluster/sentinel/pipeline。**
4. **v0.3.0 再扩展 pubsub/stream/lock。**
5. **所有 cross-cutting 能力必须优先来自 L1 标准契约。**
6. **任何 redisx 中发现的标准缺口都必须反向沉淀到 xlib-standard。**

一句话：

> redisx 的第一目标不是“封装 Redis”，而是成为 `xlib-standard` 可复制 L2 adapter 工厂的第一个强证据样本。
