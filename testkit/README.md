# Testkit 测试工具

为生成的基础库提供可复用测试夹具、断言和 Redis 语义 fake。

## 契约

- `Config(name string)` 返回带 `Name` 和 `Timeout` 的最小有效配置。
- `NewFakeRedis()` 返回公开的 `redisx.Provider`，底层使用内存实现，不连接真实 Redis。
- `RequireNoError(t, err)` 在 `err == nil` 时保持静默，在非空错误时终止当前测试。
- `RequireGolden(t, path, actual)` 读取 golden 文件并比较实际输出；不一致时报告 expected / actual 上下文。

## 回归覆盖

`fixture_test.go` 锁定 `Config("fixture")` 的字段和 `Validate` 结果，并验证 `RequireNoError(t, nil)` 可用。`golden_test.go` 锁定 golden 断言的匹配路径。`fakeredis_test.go` 用 golden 输出锁定 `NewFakeRedis()` 的 Ping、KV、批量读取、计数、TTL、计数器、删除和关闭错误语义，并验证默认 `redisx.New` 在存在 `REDIS_URL` / `REDIS_ADDR` 时仍使用内存 provider，不默认拨号真实 Redis。

生成的库应保持此包独立于 `x.go` 和业务特定模型。
