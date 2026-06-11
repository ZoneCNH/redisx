# redisx 错误契约

## 占位符

- `redisx`
- `redisx`

## `ErrorKind`

`ErrorKind` 是调用方分支判断的稳定分类。调用方应使用 `IsKind(err, ErrorKind...)`，或通过 `errors.Is(err, redisx.Err...)` 匹配 Redis-specific sentinel，不依赖错误字符串。

| `ErrorKind` | 字符串 | 典型场景 | Retryable |
| --- | --- | --- | --- |
| `ErrorKindConfig` | `config` | 配置来源或配置装载失败。 | 否 |
| `ErrorKindValidation` | `validation` | 配置字段缺失、格式非法、调用参数非法。 | 否 |
| `ErrorKindConnection` | `connection` | 连接建立失败或连接不可用。 | 通常是 |
| `ErrorKindUnavailable` | `unavailable` | 依赖暂不可用。 | 视场景 |
| `ErrorKindTimeout` | `timeout` | context deadline exceeded 或外部超时。 | 是 |
| `ErrorKindAuth` | `auth` | Redis 认证、授权失败。 | 否 |
| `ErrorKindNetwork` | `network` | 网络读写或传输层失败。 | 通常是 |
| `ErrorKindReadOnly` | `read_only` | Redis 只读节点拒绝写入。 | 通常是 |
| `ErrorKindLoading` | `loading` | Redis 节点仍在加载数据。 | 是 |
| `ErrorKindTryAgain` | `try_again` | Redis 返回临时重试类错误。 | 是 |
| `ErrorKindClusterMoved` | `cluster_moved` | Redis Cluster `MOVED` 重定向。 | 是 |
| `ErrorKindClusterAsk` | `cluster_ask` | Redis Cluster `ASK` 重定向。 | 是 |
| `ErrorKindConflict` | `conflict` | 幂等冲突、资源状态冲突。 | 否 |
| `ErrorKindRateLimit` | `rate_limit` | 限流或配额耗尽。 | 是 |
| `ErrorKindInternal` | `internal` | 未分类内部错误。 | 否 |
| `ErrorKindCanceled` | `canceled` | context canceled。 | 否 |
| `ErrorKindNil` | `nil` | Redis missing key / nil reply。 | 否 |
| `ErrorKindClosed` | `closed` | Client 或 provider 已关闭。 | 否 |
| `ErrorKindInvalidConfig` | `invalid_config` | 绑定后的配置不满足 Redis contract。 | 否 |
| `ErrorKindProvider` | `provider` | provider 返回的未分类错误。 | 视场景 |

## Redis-specific identifiers

`RedisErrorID` 既是错误标识符，也是 `errors.Is` 的 sentinel 目标。`ErrorIdentifierForKind` 将 `ErrorKind` 映射为这些 Redis 标识符。

| `RedisErrorID` | 字符串 | 对应 `ErrorKind` |
| --- | --- | --- |
| `ErrNil` | `redis.nil` | `ErrorKindNil` |
| `ErrTimeout` | `redis.timeout` | `ErrorKindTimeout` |
| `ErrCanceled` | `redis.canceled` | `ErrorKindCanceled` |
| `ErrNetwork` | `redis.network` | `ErrorKindNetwork` |
| `ErrAuth` | `redis.auth` | `ErrorKindAuth` |
| `ErrReadOnly` | `redis.read_only` | `ErrorKindReadOnly` |
| `ErrLoading` | `redis.loading` | `ErrorKindLoading` |
| `ErrTryAgain` | `redis.try_again` | `ErrorKindTryAgain` |
| `ErrClusterMoved` | `redis.cluster_moved` | `ErrorKindClusterMoved` |
| `ErrClusterAsk` | `redis.cluster_ask` | `ErrorKindClusterAsk` |
| `ErrConnectionClosed` | `redis.connection_closed` | `ErrorKindConnection` / `ErrorKindClosed` |
| `ErrInvalidConfig` | `redis.invalid_config` | `ErrorKindConfig` / `ErrorKindValidation` / `ErrorKindInvalidConfig` |
| `ErrProvider` | `redis.provider` | provider、unavailable、conflict、rate limit、internal fallback |

## 约束

- 公共错误必须使用 `Error`、`NewError` 或 `WrapError` 表达稳定 contract。
- 包装错误必须保留 cause，使调用方可以使用 `errors.Is` / `errors.As`。
- 调用方按 `IsKind(err, ErrorKind...)`、`errors.Is(err, redisx.Err...)` 或 `ErrorIdentifierForKind` 做分支判断，不依赖错误字符串。
- 错误可以安全纳入 Evidence，但不得包含原始凭据、生产连接串或业务私密数据。
- 生成的库不得使用 `x.go` 业务模型。
