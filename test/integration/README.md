# L2 Integration

`redisx` 的 L2 integration profile 通过 env-gated 真实 Redis 测试执行：

`docker-compose.yml` 与 `.devcontainer/devcontainer.json` 已提供非敏感默认端点 `REDISX_REDIS_ADDR`、`REDISX_REDIS_URL`、`REDISX_REDIS_DB`。用户名、密码、token 或 secret 只能来自本地环境或受控 secret store，不写入代码、文档、证据文件或测试输出。

```bash
# Docker/devcontainer defaults already provide endpoint values.
REDISX_INTEGRATION=1 GOWORK=off make test-integration
```

可选环境变量：`REDISX_REDIS_USERNAME`、`REDISX_REDIS_PASSWORD`、`REDISX_REDIS_DB`。认证值只能来自本地环境或受控 secret store，不写入代码、文档或证据文件，也不在测试输出中打印。


## 受控 dev secret 验证

`/home/ZoneCNH/sre/secrets/env/dev.md` 只能作为本地受控凭据来源使用。验证时关闭 shell tracing，只把需要的 `REDISX_REDIS_*` 值注入当前进程；不要 `cat`、echo、tee、提交或粘贴该文件内容，也不要把 secret value 写入 Evidence。`make test-dev-env-integration` 只解析 allowlisted Redis 变量名并写出 redacted-only 报告；若文件可读但不包含支持的 Redis endpoint assignment，报告状态为 `not_applicable`，live Redis 证据仍由 CI Redis service 或 Docker runner 提供。

```bash
set +x
DEV_ENV_FILE=/home/ZoneCNH/sre/secrets/env/dev.md GOWORK=off make test-dev-env-integration
REDISX_INTEGRATION_DOCKER=1 GOWORK=off make test-integration
REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration
```

本地 Docker 验证路径：

```bash
docker compose -f docker-compose.yml -f docker-compose.test.yml up -d redis
docker compose -f docker-compose.yml -f docker-compose.test.yml run --rm \
  -e GOWORK=off \
  -e REDISX_INTEGRATION=1 \
  -e REDISX_REDIS_ADDR \
  toolchain make test-integration
```

持久化恢复 profile 使用临时 Redis 实例开启 AOF/RDB，把永久 key 写入后重启服务并在同一数据卷上验证恢复：

```bash
REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration
```

当前自动 integration runner 覆盖 Ping/Health、KV、TTL、multi-key、counter、hash、list、pipeline、delete、错误映射、lock token compare-release、fixed-window rate-limit TTL window，以及第二个客户端读取第一个客户端写入值的 reconnect proof。`test-persistence-integration` 额外覆盖永久 string、hash、list、counter 和 pipeline writes 在 AOF/RDB-backed Redis restart 后仍可读取，且永久 TTL 仍为永久。Lock token、rate-limit window 和 pub/sub 不作为 durable persistence evidence。
