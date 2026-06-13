# L2 Evidence

本目录保存 `redisx` 对 xlib-standard L2 契约的本地证据快照。

当前状态声明 L2-T2 readiness 已被评估，`score=100`、`release_ready=true`，并且 `unit`、`contract`、`integration`、`persistence` 四个必需 profile 均有本地证据：

- `release-readiness.json`：记录 L2-T2 score、release_ready 状态与四个必需 profile 的 pass 状态。
- `contract-report.json`：记录 common、kv、ttl、pool contract pack 的本地证据文件。
- `integration-report.json`：记录 env-gated 真实 Redis integration runner 覆盖的 string、TTL、multi-key、counter、hash、list、SetNX、lock 和 pipeline 场景。
- `persistence-report.json`：记录 AOF/RDB-backed Redis restart 后永久 string、MSet、hash、list、counter 和 pipeline 写入恢复的场景。
- `compliance-matrix.json`：逐项记录 common、kv、ttl、pool contract pack 的证据状态。

真实 Redis 测试必须由 `REDISX_INTEGRATION=1` 显式开启。`docker-compose.yml` 与 `.devcontainer/devcontainer.json` 提供非敏感默认端点 `REDISX_REDIS_ADDR`、`REDISX_REDIS_URL`、`REDISX_REDIS_DB`；用户名、密码、token 或 secret 只能来自本地环境或受控 secret store，不得写入证据文件、配置默认值、日志或提交。持久化恢复测试必须由 `REDISX_PERSISTENCE_INTEGRATION=1` 显式开启并使用临时 Redis 实例。
