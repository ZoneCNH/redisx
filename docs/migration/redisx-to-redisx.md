# redisx 名称迁移说明

## 状态

已完成兼容迁移记录。

## 范围

本文保留旧文档链接的迁移入口。当前标准仓库、模板、generator、Harness 和 Evidence Runtime 的权威名称均为 [`redisx`](https://github.com/ZoneCNH/redisx)。若旧材料出现同名迁移占位，按当前 `redisx` 仓库身份解释，不引入新的 module path 或下游名称。

## 规则

- 标准入口以 `docs/standard/README.md` 和 `docs/standard/redisx.md` 为准。
- 默认下游名称按 ADR-20260602-002 迁移为 `kernel`。
- release Evidence、downstream matrix 和 docs-check 不接受缺失迁移链接作为完成状态。
