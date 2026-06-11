# L2 Downstream Adoption

`xlib-standard` 是标准源，只定义 artifact 形状、必需 evidence 和 release 语义。`testkitx` 负责可执行 contract libraries，`xlibgate` 负责 release 裁决。本仓库必须保持 provider-neutral：不连接 provider，不加载 credentials，也不在这里实现 Contract Runner。

Adopting repositories 消费 `xlib-standard` templates 和 registries，但不重新定义 release levels。它们只能在自己的仓库中加入 provider-specific services、credentials loading、concrete compose services 和 Contract Runner wiring。

## Adoption checklist

- 保持 `.agent/l2-capabilities.yaml` 符合 schema。
- 只选择 registry-defined contract packs，除非 backlog item 已提升。
- 将 evidence 保存在 `.agent/evidence/l2` 下。
- 当本地 Makefiles 包装 template targets 时，保留 `xlib-standard` template aliases。
- 每次 release claim 都发布 compliance、compatibility 和 release-readiness reports。

## Template behavior

包含的 template 从本地 shape checks 和 provider-neutral placeholders 开始。Compose file 默认使用 placeholder profile 且不访问网络，让 adopting repositories 能先证明 paths，再添加 services。

## Upgrade path

当 `xlib-standard` 更新 schemas 或 release levels 时，downstream repositories 应先更新 manifests，重新运行 shape checks，然后重新运行可执行的 `testkitx` profiles 和 `xlibgate` adjudication。
