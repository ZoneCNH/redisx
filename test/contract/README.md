# L2 Contract

本目录保存 `redisx` 的 L2 contract profile 本地检查。

当前测试只验证 `.agent/l2-capabilities.yaml` 与 `.agent/evidence/l2/release-readiness.json` 的形状和真实状态，避免在缺少 testkitx 合同 runner 时误报通过。后续接入可执行 contract pack 后，应生成 `.agent/evidence/l2/contract-report.json`，再提升 readiness 中的 `contract` 状态。
