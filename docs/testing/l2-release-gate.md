# L2 Release Gate

`xlib-standard` 是标准源，只定义 artifact 形状、必需 evidence 和 release 语义。`testkitx` 负责可执行 contract libraries，`xlibgate` 负责 release 裁决。本仓库必须保持 provider-neutral：不连接 provider，不加载 credentials，也不在这里实现 Contract Runner。

Release levels 定义在 `.agent/registry/l2-release-levels.yaml`，downstream 不得重新定义。`L2-T3` 是第一个 release-allowed level；`L2-T4` 是 factory-grade。

## Level expectations

- `L2-T0` 只证明 skeleton readiness。
- `L2-T1` 要求 unit 和 contract profiles。
- `L2-T2` 增加 integration evidence。
- `L2-T3` 增加 chaos、benchmark 和 adoption evidence，并允许 release。
- `L2-T4` 增加 retrospective evidence，并允许 factory-grade claims。

## Gate behavior

当 required profiles、evidence paths、selected pack reports 或 compatibility notes 缺失时，gates 必须 fail closed。`xlibgate` 根据此 registry 裁决 evidence；`xlib-standard` 只定义 release semantics 和 template targets。

## Template targets

`make l2-release-readiness` 检查本地 standards shape，并指向 downstream 的 `xlibgate`。Compatibility aliases `l2-release-readiness-check`、`l2-manifest-check` 和 `l2-evidence-check` 会继续保留，以支持现有 workflows。

Release claims 的 evidence paths 应保持在 `.agent/evidence/l2` 下，除非 `xlib-standard` 修改 schema。
