# L2 Adapter Testing Standard

`xlib-standard` 是标准源，只定义 artifact 形状、必需 evidence 和 release 语义。`testkitx` 负责可执行 contract libraries，`xlibgate` 负责 release 裁决。本仓库必须保持 provider-neutral：不连接 provider，不加载 credentials，也不在这里实现 Contract Runner。

## 必需 downstream flow

1. 将 `templates/l2` 复制到 L2 adapter 仓库。
2. 完成 `.agent/l2-capabilities.yaml`，写入 adapter metadata、capability declarations、selected contract packs 和 evidence paths。
3. 先运行 template shape gates：`make l2-capability-check l2-evidence`。
4. 在 downstream 接入可执行的 `testkitx` packs，并把报告保存在 `.agent/evidence/l2`。
5. 按顺序运行 profile 阶段：`l2-contract`、`l2-integration`、`l2-chaos`、`l2-benchmark` 和 `l2-adoption`。
6. 将 evidence 提交给 `xlibgate` 做 release-readiness 裁决，且不得弱化 `xlib-standard` release levels。

## Profile 语义

- `skeleton` 证明 manifest 和 evidence 目录存在。
- `unit` 证明本地 adapter 行为，不依赖 live provider。
- `contract` 使用 `testkitx` 证明已声明的 packs，例如 `common`、`kv` 和 `ttl`。
- `integration`、`chaos` 和 `benchmark` 证明 downstream service 行为，必须记录为 evidence，不得编码进标准源仓库。
- `adoption` 和 `retrospective` 证明 rollout safety、compatibility notes 和发布后复盘。

## Pass criteria

只有当每个已声明 capability 都映射到 registry pack、每个必需 profile 都有机器可读 evidence，且 `xlibgate` 确认请求的 release level 时，downstream adapter 才算 L2-ready。缺失 evidence 是 blocker，不能通过降低标准绕过。
