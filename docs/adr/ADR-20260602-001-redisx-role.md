# ADR-20260602-001: redisx 仓库角色

## 状态

Accepted

## 背景

基础库标准需要一个单一公开仓库承载标准文本、Go 模板、generator、Harness gate 和 Evidence Runtime，避免标准、实现和发布证据分散。

## 决策

[`redisx`](https://github.com/ZoneCNH/redisx) 是基础库 Standard Source、Go Reference Template、generator、Harness 和 Evidence Runtime 的统一仓库。下游基础库和私有业务系统消费该标准与生成结果，不反向成为标准权威源。

## 影响

- `docs/standard/redisx.md` 作为总标准入口。
- `docs/standard/repository-roles.md` 维护仓库职责细分。
- `docs-check`、Harness 和 release Evidence 必须验证标准入口与仓库角色文档存在。
