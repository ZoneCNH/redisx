# Testkit 测试工具

为生成的基础库提供可复用测试夹具、断言和 Redis 语义 fake。

## 契约

- `Config(name string)` 返回带 `Name` 和 `Timeout` 的最小有效配置。
- `NewFakeRedis()` 返回公开的 `redisx.Provider`，底层使用内存实现，不连接真实 Redis。
- `RequireNoError(t, err)` 在 `err == nil` 时保持静默，在非空错误时终止当前测试。
- `RequireGolden(t, path, actual)` 读取 golden 文件并比较实际输出；不一致时报告 expected / actual 上下文。
- `NewFakeRedis()` 返回公开的 `redisx.Provider` fake，使用独立内存存储，不读取 Redis 环境变量、不打开网络连接。
- `NewClientWithFakeRedis(ctx, cfg, opts...)` 创建显式 fake provider 支撑的 `redisx.Client`。

## 回归覆盖

`fixture_test.go` 锁定 `Config("fixture")` 的字段和 `Validate` 结果，并验证 `RequireNoError(t, nil)` 可用。`golden_test.go` 锁定 golden 断言的匹配路径。`fake_redis_test.go` 锁定 fake provider 的 Ping、Set/Get、MSet/MGet、Exists、Del、Expire/TTL、Incr/Decr、Close、实例隔离和 no-real-Redis 默认路径证据。生成后的基础库需要保留这组最小测试，以防测试夹具随包名替换、配置 contract 或稳定输出漂移。

生成的库应保持此包独立于 `x.go` 和业务特定模型。
