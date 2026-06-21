# Redisx v1.1.2 功能清单

`redisx` v1.1.2 的发布面聚焦 Redis L2-T2 生产就绪：显式配置、可验证写入语义、运行时健康信号、以及不泄露密钥的 CI/集成证据。

## Runtime 能力

- 显式 Redis 配置：地址、用户名、密码、DB、TLS、超时和连接池参数均由调用方或外部环境提供；库不内置生产凭证。
- KV / TTL：字符串读写、过期时间、存在性判断与删除语义覆盖基础缓存和状态存储场景。
- Multi-key / pipeline：批量读写与 pipeline writes 支持低往返延迟的多键操作。
- Counter / hash / list：计数器、Hash 字段、List push/pop/range 操作覆盖常见 Redis 数据结构。
- Cache-aside：`JSONCodec`、`Codec[T]`、`Cache[T]` 与 `NewCacheClient[T]` 支持 typed JSON cache-aside。
- Coordination：自动 token lock、显式 token `AcquireLock` / `ReleaseLock`、fixed-window rate limiter 都使用 TTL-scoped key。
- Health / observability：health check、metrics hooks 和 typed error/sanitize 边界用于 runtime diagnostics。

## 发布边界

- Durable persistence evidence 覆盖 string、hash、list、counter 和 pipeline writes 的 restart recovery。
- Pub/sub 不属于 v1.1.2 durable write surface。
- 集成证据只允许记录 profile、命令、覆盖项和环境变量名；不得记录 Redis 密码、API key 或 `/home/ZoneCNH/sre/secrets/env/dev.md` 的值。

## CI / Release gates

- `GOWORK=off make coverage-check` 必须保持 configured package set 的 100% coverage gate。
- GitHub Integration workflow 使用 Redis service 运行 live integration profile，并运行 persistence restart recovery profile。
- `make test-dev-env-integration` 可读取外部 dev env 文件、生成 redacted-only 配置报告，并在未发现 Redis endpoint 时返回 `not_applicable`。
- Release readiness 由 unit、contract、integration、persistence、docs、release preflight 共同证明；接受标准见 `ACCEPTANCE.md`。
