# Redisx v1.1.1 验收标准

v1.1.1 发布候选必须在 `GOWORK=off` 下完成以下验证，并保留 redacted-only evidence。

## 必需验证命令

```bash
GOWORK=off go test ./...
GOWORK=off make coverage-check
GOWORK=off make test-contract
GOWORK=off REDISX_INTEGRATION=1 REDISX_REDIS_ADDR=127.0.0.1:6379 REDISX_REDIS_DB=0 make test-integration
GOWORK=off REDISX_PERSISTENCE_INTEGRATION=1 make test-persistence-integration
GOWORK=off make docs-check
XLIB_CONTEXT=release_verify GOWORK=off make release-check
XLIB_CONTEXT=release_verify GOWORK=off make release-preflight VERSION=v1.1.1
```

## CI 验收

- Pull request integration workflow 必须启动 Redis service，并显式设置 `REDISX_INTEGRATION=1`、`REDISX_REDIS_ADDR` 和 `REDISX_REDIS_DB`。
- Persistence integration 必须覆盖 Redis restart recovery；本地脚本可使用 Docker 或本机 `redis-server`。
- Coverage gate 必须满足 `COVERAGE_MIN=100.0`，不能通过降低阈值通过。

## 密钥与外部 dev env 约束

- `/home/ZoneCNH/sre/secrets/env/dev.md` 只能作为外部、未跟踪的运行时输入；不得复制、提交或在日志中打印其值。
- 允许在 evidence 中记录已配置的变量名（例如 `REDISX_REDIS_ADDR`、`REDISX_REDIS_DB`），但必须 redacted value。
- 如果该文件不包含 Redis 连接变量，集成验证必须改用本机 Redis service 或 CI Redis service，而不是输出文件内容。

## 完成条件

- Version anchors 一致指向 `v1.1.1`：`redisx.Version`、goalcli governance release、release manifest template、README/API/release docs 与 changelog。
- `FEATURES.md` 和 `ACCEPTANCE.md` 位于仓库根目录并描述当前 release surface 与验收命令。
- 所有验证失败都必须修复或以明确、可复现的环境缺口记录为 `Not-tested`。
