# L2 Capability Manifest

`xlib-standard` 是标准源，只定义 artifact 形状、必需 evidence 和 release 语义。`testkitx` 负责可执行 contract libraries，`xlibgate` 负责 release 裁决。本仓库必须保持 provider-neutral：不连接 provider，不加载 credentials，也不在这里实现 Contract Runner。

manifest schema 是 `.agent/schemas/l2-capabilities.schema.json`；downstream template 是 `templates/l2/.agent/l2-capabilities.yaml`。template 刻意保持本地化和 provider-neutral，让新 adapter 可以先证明 shape，再接入 `testkitx`。

## 必需 sections

- `schema_version` 标识标准 schema version。
- `layer` 必须是 `L2`。
- `adapter` 记录 name、module、family 和 owners。
- `capabilities` 声明每个 adapter capability 及其 family 和 status。
- `contract_packs` 将 capabilities 映射到 `xlib-standard` registry packs。
- `evidence` 记录 required profiles、output directory 和 report paths。

## Invariants

每个 selected pack 必须存在于 `.agent/registry/l2-contract-packs.yaml`。Evidence paths 应保持在 `.agent/evidence/l2` 下。Manifest 只描述 adapter 意图；live connection details、runtime credential loading 和 runner wiring 属于 adopting repository，不应提交到 `xlib-standard` artifacts。

## Local checks

在 template 中运行 `make l2-capability-check`，确认 manifest 存在。Downstream repositories 随后应先按 schema 校验 manifest，再调用 `testkitx` 或 `xlibgate`。
