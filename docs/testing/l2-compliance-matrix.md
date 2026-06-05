# L2 Compliance Matrix

`xlib-standard` 是标准源，只定义 artifact 形状、必需 evidence 和 release 语义。`testkitx` 负责可执行 contract libraries，`xlibgate` 负责 release 裁决。本仓库必须保持 provider-neutral：不连接 provider，不加载 credentials，也不在这里实现 Contract Runner。

Compliance matrix 映射 requirement、contract pack、profile、evidence path、status 和 release-level impact。它的 schema 是 `.agent/schemas/l2-compliance-matrix.schema.json`。

## 必需 mapping

每一行都应标识 adapter capability、承载语义预期的 registry pack、证明该能力的 profile，以及支撑 status 的 evidence file。Statuses 为 `pass`、`fail`、`missing` 或 `not_applicable`。

## Matrix rules

- Manifest 中声明的 capability 应出现在 matrix 中。
- Selected contract pack 应至少有一行带 evidence 的记录。
- `missing` 和 `fail` rows 必须阻塞受影响的 release level。
- `not_applicable` 必须由 adapter family 或显式 unsupported capability status 说明。

## Review 用途

`xlibgate` 和 human reviewers 应能只通过 matrix 与链接的 evidence 重建 release decision，而不需要读取 downstream implementation code。

Release claims 的 evidence paths 应保持在 `.agent/evidence/l2` 下，除非 `xlib-standard` 修改 schema。
