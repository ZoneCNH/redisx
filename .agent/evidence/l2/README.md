# L2 Evidence

本目录保存 `redisx` 对 xlib-standard L2 契约的本地证据快照。

当前状态声明 L2-T2 readiness 已完成并可用于 v1.0.0 release：

- `release-readiness.json`：记录 `score=100`、`release_ready=true`，`unit`、`contract`、`integration` 均为 `pass`。
- `compliance-matrix.json`：逐项记录 common、kv、ttl、pool contract pack 的证据状态，所有 release 行均为 `pass`。
- `contract-report.json`：记录 contract profile 的命令与证据路径。
- `integration-report.json`：记录 Docker-backed Redis endpoint 与 restart smoke 证据。

dev Docker/devcontainer 默认暴露非敏感 Redis 端点变量：`REDISX_REDIS_ADDR=redis:6379`、`REDISX_REDIS_URL=redis://redis:6379/0`、`REDISX_REDIS_DB=0`。本证据不暴露 password、token 或 secret 默认值。
