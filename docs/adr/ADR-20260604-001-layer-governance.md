# ADR-20260604-001 分层治理

## 状态

已采纳。

## 背景

`redisx` 同时作为 Standard Source、Go Reference Template、Generator、Harness 和 Evidence Runtime。为避免标准库、下游基础库与私有业务组合层之间出现反向依赖，仓库需要一份可由文档、policy、schema 和 gate 共同引用的分层治理决策。

本 ADR 与 `docs/standard/layer-governance-rules.md`、`.agent/policies/layer-governance.yaml` 和 `contracts/layer-governance.schema.json` 保持一致。

## 决策

采用固定依赖方向 `L3>L2>L1>L0>Standard`，并将 `redisx` 归入 L2 基础库层。

- Standard：标准源与模板协议，定义生成、治理和证据边界。
- L0：`kernel`，提供最小公共内核能力。
- L1：`configx`、`observex`、`testkitx`，提供配置、观测和测试支撑。
- L2：`redisx`、`kafkax`、`postgresx`、`taosx`、`ossx`、`clickhousex`、`natsx`，提供具体基础设施客户端与测试夹具。
- L3：`x.go` 及私有业务仓库，只能组合调用 L2 及以下能力，不得被基础库依赖。

P0 规则禁止私有业务边界泄漏与反向依赖，且不允许临时例外。P1 规则要求下游采用流程跟随标准变更。P2 规则要求迭代证据可回放、可审计。

## 影响

`redisx` 的公共 API、模板渲染、Harness gate 与 Evidence manifest 必须保持 L2 边界：可以依赖 Standard、L0 和 L1 能力，不得依赖 L3 私有业务层或 `x.go` 内部包。

任何更新分层表、依赖方向或治理规则的变更，必须同步更新：

- `docs/standard/layer-governance-rules.md`
- `.agent/policies/layer-governance.yaml`
- `contracts/layer-governance.schema.json`
- 相关 Harness gate 与 release evidence

## 验证

验证入口包括 `make boundary`、`make contracts`、`make docs-check`、`make standard-impact-check` 和 `make score`。发布前必须将这些 gate 的结果纳入 Evidence，并保留可复现命令与输出摘要。
