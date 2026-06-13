# L2 Contract

本目录保存 `redisx` 的 L2 contract profile 本地检查。

`l2_contract_test.go` 验证：

- `.agent/l2-capabilities.yaml` 声明了 L2、redisx identity、common/kv/ttl/persist/pool contract packs 与必需 profiles。
- Manifest 不包含 provider 边界禁止的明文连接键。
- `.agent/evidence/l2/release-readiness.json` 达到 L2-T2 最低 readiness score，并且 `unit`、`contract`、`integration`、`persistence` 证据均为 `pass`。
- readiness 中声明为 `pass` 的 `.agent/...` 证据文件实际存在。

运行：

```bash
GOWORK=off go test ./test/contract -count=1
REDISX_PERSISTENCE_INTEGRATION=1 GOWORK=off make test-persistence-integration
GOWORK=off make l2-check
```
