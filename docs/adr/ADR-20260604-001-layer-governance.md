# ADR-20260604-001: 分层治理边界

## 状态

Accepted

## 背景

`redisx` 作为标准、模板、generator、Harness 和 Evidence 的权威仓库，需要把公开基础库与私有业务系统的职责边界写成可审计规则。L0/L1/L2 可以公开复用；L3 私有业务系统只能消费基础库能力，不能把业务模型、生产密钥、业务 topic、客户语义或部署 wiring 反向写入公开仓库。

## 决策

采用 `redisx -> L0 -> L1 -> L2 -> L3` 的分层治理规则：

- Standard/Runtime：`redisx` 维护标准、模板、generator、Harness、Evidence 和治理 gate。
- L0：`kernel` 提供通用 runtime primitive。
- L1：`configx`、`observex`、`testkitx` 等横切基础库。
- L2：`redisx`、`kafkax`、`postgresx`、`taosx`、`ossx`、`clickhousex`、`natsx` 等基础设施适配库。
- L3 私有：`x.go` 与业务系统负责业务组合、生产配置注入、业务模型和部署 wiring。

依赖方向只能是 L3 -> L2 -> L1 -> L0 -> Standard。公开基础库不得依赖或复制 L3 内容。

## 影响

- `docs/standard/layer-governance-rules.md` 是此 ADR 的执行规则入口。
- `docs-check` 必须覆盖该 ADR 与分层治理文档，防止缺失标准决策记录。
- 下游同步策略和采纳状态必须把 L3 私有验证标为私有消费侧责任，不能写成公开仓库已完成 Evidence。

## 后续要求

任何新增 L1/L2 基础库、L3 私有消费规则或 P0/P1/P2 gate 调整，都必须同步标准文档、Harness/goalcli 检查、下游矩阵和 Evidence 结论。
