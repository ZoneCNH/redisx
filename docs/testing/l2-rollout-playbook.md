# L2 Rollout Playbook

`xlib-standard` 是标准源，只定义 artifact 形状、必需 evidence 和 release 语义。`testkitx` 负责可执行 contract libraries，`xlibgate` 负责 release 裁决。本仓库必须保持 provider-neutral：不连接 provider，不加载 credentials，也不在这里实现 Contract Runner。

## Sequence

1. 将 `templates/l2` 复制到 L2 repository。
2. 填写 adapter metadata、declared capabilities、selected contract packs 和 evidence report paths。
3. 运行 `make l2-capability-check l2-evidence` 证明 local shape。
4. 在 downstream repository 接入 `testkitx` 并生成 contract reports。
5. 按 target release level 要求添加 integration、chaos、benchmark 和 adoption evidence。
6. 运行 `make l2-release-readiness`，并将 evidence bundle 提交给 `xlibgate`。
7. 记录 blockers，不要弱化 release criteria。

## Rollout controls

在 manifest 和 evidence directory 稳定前，从 `L2-T0` 开始。每次只提升一个 release level，保留 failed evidence 和 review notes。对共享 families 或 pack selections 的 adapters 保持 compatibility notes，以便跨 downstream repositories 比较 regressions。

## Stop conditions

当 selected packs 缺少可执行的 `testkitx` coverage、required reports 缺失，或 provider-specific behavior 无法用现有 pack semantics 表达时，停止 rollout。通过 `xlib-standard` backlog 升级 gap，而不是添加本地例外。

Release claims 的 evidence paths 应保持在 `.agent/evidence/l2` 下，除非 `xlib-standard` 修改 schema。
