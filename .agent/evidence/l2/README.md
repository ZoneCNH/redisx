# L2 Evidence

本目录保存 `redisx` 对 xlib-standard L2 契约的本地证据快照。

当前状态只声明 L2-T2 readiness 已被评估，不声明 release ready：

- `release-readiness.json`：记录 `unit` 已通过，`contract` 与 `integration` 证据仍缺失。
- `compliance-matrix.json`：逐项记录 common、kv、ttl、pool contract pack 的证据状态。
- `contract-report.json` 与 `integration-report.json` 尚未生成，因此 readiness 中保持 `missing`。

后续接入 testkitx 合同 runner 或真实 Redis integration runner 后，应先生成对应报告，再把 readiness 状态从 `missing` 提升为 `pass`。
