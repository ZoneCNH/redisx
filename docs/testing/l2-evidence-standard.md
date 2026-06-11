# L2 Evidence Standard

`xlib-standard` 是标准源，只定义 artifact 形状、必需 evidence 和 release 语义。`testkitx` 负责可执行 contract libraries，`xlibgate` 负责 release 裁决。本仓库必须保持 provider-neutral：不连接 provider，不加载 credentials，也不在这里实现 Contract Runner。

Evidence 必须可复现，尽可能机器可读，并链接到 manifest capabilities、selected contract packs、required profiles 和 release-level decisions。Canonical downstream output root 是 `.agent/evidence/l2`。

## Minimum evidence set

- 每个 selected pack 的 contract report。
- 映射 requirements、packs、profiles、evidence paths 和 status 的 compliance matrix。
- 当请求的 release level 需要时，提供 integration、chaos、benchmark 和 adoption reports。
- 提供用于 `xlibgate` adjudication 的 release-readiness summary。

## Evidence quality

Evidence 应命名 manifest version、adapter module、selected packs、生成 report 的 command 或 workflow、timestamp 以及 pass/fail status。缺失文件或空 placeholder 不是 passing evidence。Standards templates 可以包含 `.gitkeep` files 来建立目录，但 downstream release claims 必须包含真实 reports。

## Failure handling

当 profile 失败时，保留失败 evidence 并记录 blocker。不要删除 profile、未经 review 降低 target release level，或向 `xlib-standard` 添加 provider-specific bypasses。
