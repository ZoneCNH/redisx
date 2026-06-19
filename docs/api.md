# API 模板

## 占位符

- `redisx`：生成的仓库名称。
- `github.com/ZoneCNH/redisx`：生成的 Go module 路径。
- `redisx`：生成的包名。

## 公共 API

- `Config`：由用户显式提供的配置。
- `Validate`：拒绝无效配置，并返回 `ErrorKindValidation`。
- `Sanitize`：在日志或 Evidence 采集前屏蔽敏感值。
- `New`：基于显式配置创建客户端；拒绝 `nil`、canceled 和 expired context；成功时记录 `client_created_total`。
- `Close`：释放资源，并且必须幂等；成功首次关闭时记录 `client_closed_total`。
- `HealthCheck`：报告客户端健康状态，JSON 字段必须匹配 `contracts/health.schema.json`；当本次检查的 context deadline 预算短于 `Config.Timeout` 时返回 `degraded`。
- `Error`：稳定 error contract，支持 `errors.Is` / `errors.As` 和 `IsKind`。
- `NewError` / `WrapError`：创建或包装稳定错误，包装时必须保留 cause。
- `Metrics`：注入式指标钩子；指标名必须匹配 `contracts/metrics.md`。
- `Version`：发布版本。
- `Ping` / `Health`：验证 Redis 连接与运行时健康。
- `Get` / `Set` / `Del` / `Exists` / `Expire` / `TTL`：字符串 KV、删除、存在性和 TTL 操作。
- `MGet` / `MSet`：multi-key 读写。
- `Incr` / `Decr`：Redis counter 操作。
- `HSet` / `HGet` / `HGetAll` / `HDel`：hash write/read/delete primitives。
- `LPush` / `RPush` / `LPop` / `RPop` / `LRange`：list write/read primitives。
- `SetNX`：条件写入，供 lock 与调用方幂等场景使用。
- `Pipeline`、`PipelineCommand`、`PipelineResult` 和 `Pipeline*` op 常量：批量执行 Set/MSet/HSet/RPush/Incr 写命令。
- `KeyBuilder`：稳定 key namespace/prefix builder。
- `Codec[T]`、`JSONCodec[T]`、`Cache[T]`：cache-aside 编解码与 `GetOrLoad` helper。
- `NewLock` / `Lock`：自动生成 token 的 `SET NX` lock acquire/release。
- `FixedWindowRateLimiter` / `RateLimitResult`：fixed-window counter + TTL rate limit helper。

生成的基础库不得依赖 `x.go`。

### Redis v1.1.1 primitives

`pkg/redisx` 的 Redis adapter 公共面包含：

- String / TTL：`Set`、`Get`、`MSet`、`MGet`、`Delete`、`Exists`、`Expire`、`TTL`。
- Counter：`Incr`、`Decr`。
- Hash：`HSet`、`HGet`、`HGetAll`、`HDel`。
- List：`LPush`、`RPush`、`LRange`、`LPop`、`RPop`。
- Pipeline：`Pipeline` 支持 `PipelineSet`、`PipelineMSet`、`PipelineHSet`、`PipelineRPush` 和 `PipelineIncr`。
- Coordination helpers：`AcquireLock` / `ReleaseLock` 使用显式 token compare-release；`NewLock` / `Lock` 生成并保留 token；`FixedWindowRateLimit` 返回 allowed、remaining、count 和 reset window。二者都是 TTL-scoped 临时状态，不作为 durable persistence 证据。
- Cache-aside helpers：`NewCacheClient[T]`、`JSONCodec[T]`、`Get`、`Set`、`GetOrLoad` 和 `KeyBuilder` 基于 Redis string values 提供 typed codec helper。

Durable Redis persistence evidence 覆盖永久 string、hash、list、counter 和 pipeline writes 的 restart recovery。Pub/sub 不属于当前 v1.1.1 durable write surface。

## 生成对齐

使用 `scripts/render_template.sh` 生成具体基础库时，公共包目录会从 `pkg/redisx` 移动到 `pkg/redisx`，代码 imports、文档占位符和 module path 会同步替换。
