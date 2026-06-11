# ADR-20260602-002: 默认下游名迁移为 kernel

## 状态

Accepted

## 背景

旧默认下游名容易与标准仓库职责混淆；L0 runtime primitive 需要一个独立、稳定、可公开消费的名称。

## 决策

默认 L0 下游名称采用 `kernel`。`kernel` 承载通用 runtime primitive；`redisx` 继续作为标准、模板、generator、Harness 和 Evidence 的权威仓库。

## 影响

- `docs/downstream-matrix.md` 与 `.agent/registries/downstream-adoption-status.yaml` 使用 `kernel` 表示默认 L0 下游。
- 旧名称只保留在迁移说明中，不作为新 Evidence 或 release manifest 的默认下游名。
- 模板、docs-check 和 downstream sync 结论必须优先使用 `kernel`。
