# L2 Compatibility Matrix

`xlib-standard` 是标准源，只定义 artifact 形状、必需 evidence 和 release 语义。`testkitx` 负责可执行 contract libraries，`xlibgate` 负责 release 裁决。本仓库必须保持 provider-neutral：不连接 provider，不加载 credentials，也不在这里实现 Contract Runner。

Compatibility 应围绕 adapter family、selected contract packs、test profiles、target release level 和 observed behavior notes 记录。Golden sample registry `.agent/registry/l2-golden-samples.yaml` 列出 redisx、postgresx、kafkax、natsx、ossx、clickhousex 和 taosx 的初始 standards-only expectations。

## Matrix dimensions

- Adapter family 和 module identity。
- Selected packs，例如 `common`、`kv`、`ttl`、`sql`、`pubsub` 或 `objectstore`。
- 请求的 release level 所需 profiles。
- Evidence paths 和 report statuses。
- 用于 version drift、semantic gaps 和 migration risks 的 compatibility notes。

## Review 用途

matrix 帮助 reviewers 在不读取 provider-specific implementation details 的情况下比较 adapters。Compatibility claim 必须链接到 evidence；unsupported capabilities 应明确写出，不能通过省略 rows 隐藏。

## Evolution

新的 pack candidates 和 behavior gaps 应先进入 registry backlog。一旦提升为 first-class pack semantics，downstream repositories 更新 manifests、重新生成 evidence，并刷新 compatibility matrices。

Release claims 的 evidence paths 应保持在 `.agent/evidence/l2` 下，除非 `xlib-standard` 修改 schema。
