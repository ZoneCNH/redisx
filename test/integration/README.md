# L2 Integration

本目录保存 `redisx` 的 L2 integration profile 本地检查。

当前仓库通过 Docker/devcontainer 提供本地 Redis 7.2 端点，并暴露非敏感 `REDISX_REDIS_ADDR`、`REDISX_REDIS_URL`、`REDISX_REDIS_DB` 默认值。集成证据写入 `.agent/evidence/l2/integration-report.json`，用于证明 Docker-backed Redis 与 restart smoke 通过。
