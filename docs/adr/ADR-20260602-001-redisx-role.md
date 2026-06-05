# ADR-20260602-001 redisx 仓库身份

## 状态

已采纳。

## 背景

仓库需要同时保存标准文本、Go 参考模板、生成器、Harness gate 和 release Evidence。如果这些职责拆散，标准变更、模板实现和发布证据容易产生漂移。

## 决策

`redisx` 作为统一标准源仓库，承担五类职责：

- Standard Source
- Go Reference Template
- Generator
- Harness
- Evidence Runtime

这些职责必须在同一 release gate 中验证。公共标准以 `docs/standard/` 为入口，机器执行面以 `Makefile`、`cmd/goalcli`、`scripts/` 和 `.agent/harness/harness.yaml` 为入口，release 证据以 `release/manifest/template.json` 与生成的 manifest 为入口。

## 影响

标准文本、参考实现和生成器不能被视为独立交付物。任何修改公共 API、模板结构、Harness gate 或 Evidence 协议的变更，都必须同步检查文档、contracts、测试和 release evidence。

## 验证

验证入口包括 `GOWORK=off make docs-check`、`GOWORK=off make contracts`、`GOWORK=off make standard-impact-check` 和 `GOWORK=off make score`。
