# L2 Integration

`redisx` 的 L2 integration profile 通过 env-gated 真实 Redis 测试执行：

```bash
REDISX_INTEGRATION=1 \
REDISX_REDIS_ADDR=127.0.0.1:6379 \
GOWORK=off make test-integration
```

可选环境变量：`REDISX_REDIS_USERNAME`、`REDISX_REDIS_PASSWORD`、`REDISX_REDIS_DB`。连接值只能来自本地环境或受控 secret store，不写入代码、文档示例之外的证据文件，也不在测试输出中打印。

本地 Docker 验证路径：

```bash
docker compose -f docker-compose.yml -f docker-compose.test.yml up -d redis
docker compose -f docker-compose.yml -f docker-compose.test.yml run --rm \
  -e GOWORK=off \
  -e REDISX_INTEGRATION=1 \
  -e REDISX_REDIS_ADDR=redis:6379 \
  toolchain make test-integration
```

如需人工验证服务重启恢复，可在第一次通过后执行：

```bash
docker compose -f docker-compose.yml -f docker-compose.test.yml restart redis
docker compose -f docker-compose.yml -f docker-compose.test.yml run --rm \
  -e GOWORK=off \
  -e REDISX_INTEGRATION=1 \
  -e REDISX_REDIS_ADDR=redis:6379 \
  toolchain make test-integration
```

当前自动 integration runner 覆盖 Ping/Health、KV、TTL、multi-key、counter、delete、错误映射，以及第二个客户端读取第一个客户端写入值的 reconnect persistence proof；外部 Redis server restart 不由默认测试自动触发。
