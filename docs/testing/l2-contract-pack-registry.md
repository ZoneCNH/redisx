# L2 Contract Pack Registry

`xlib-standard` 是标准源，只定义 artifact 形状、必需 evidence 和 release 语义。`testkitx` 负责可执行 contract libraries，`xlibgate` 负责 release 裁决。本仓库必须保持 provider-neutral：不连接 provider，不加载 credentials，也不在这里实现 Contract Runner。

Registry 是 `.agent/registry/l2-contract-packs.yaml`，并由 `.agent/schemas/l2-contract-packs.schema.json` 校验。Pack entries 定义 family、title、required profiles、required evidence 和 capability names。

## Pack fields

- `family` 将 capabilities 分组为 common、key-value、relational、messaging、streaming、storage、analytics 或 time-series domains。
- `profiles` 命名证明该 pack 所需的 evidence stages。
- `required_evidence` 命名 downstream execution 预期产出的 report classes。
- `capabilities` 列出该 pack 覆盖的 semantic operations。

## Backlog discipline

`extension_backlog` 记录尚未成为 first-class pack definitions，或仍需要 downstream evidence maturity 的标准候选项。当 leader QA 需要显式跟踪时，`ttl` 这类条目可能出现在 backlog 中，即使已有初始 pack。Backlog entries 只是 planning markers，不是 executable tests。

## Ownership boundary

`xlib-standard` 可以新增、重命名或退役 pack definitions。`testkitx` 实现可执行 checks。Downstream adapters 选择 packs 并发布 evidence；它们不得在本地重新定义 pack semantics。

Release claims 的 evidence paths 应保持在 `.agent/evidence/l2` 下，除非 `xlib-standard` 修改 schema。
