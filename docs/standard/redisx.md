# redisx 基础库总标准

[`redisx`](https://github.com/ZoneCNH/redisx) 是基础库 Standard Source、Go Reference Template、generator、Harness 和 Evidence Runtime 的统一实现仓库。本文是 `redisx` 仓库内的 L0/L1/L2 基础库总标准入口；具体 gate、发布证据和下游采纳状态仍以相邻标准文档、`.agent/registries/*`、`release/manifest/*` 与 `docs/downstream-matrix.md` 为准。

## 仓库职责

- **Standard Source**：维护基础库公共契约、分层规则、module 边界、完成定义和 release gate；不得把标准权威源分散到下游仓库。
- **模板 / generator**：提供 Go 基础库模板与渲染逻辑，确保 module path、package name、README/docs 和 release Evidence 的替换规则可审计。
- **Harness**：提供 `GOWORK=off` 的本地和 CI gate，包括 boundary、contracts、docs-check、dependency-check、standard-impact-check、score 与 release-check。
- **Evidence**：维护 release manifest、checksum、standard impact report、DOWNSTREAM 采纳结论、score 和 DONE with evidence 声明的机器可验证协议。

## 基础库契约

每个基础库层必须明确：

1. module path、package name、公开 API 和禁止导出的实现细节；
2. 配置、错误、健康检查、metrics、安全和测试边界；
3. 与 `kernel`、L1/L2 基础库和 `x.go` 的依赖方向；
4. release Evidence、downstream sync 和标准影响结论；
5. 模板生成后的文档、Harness gate 和 Evidence artifact 是否与标准保持同步。

`redisx` 作为标准实现仓库可以保存模板、generator、Harness 和 Evidence 代码；生成库或下游仓库只能采纳对应契约和生成结果，不能反向修改标准定义。

## 必跑 gate

发布式变更至少保留以下命令的证据：

```bash
GOWORK=off make boundary
GOWORK=off make contracts
GOWORK=off make docs-check
GOWORK=off make dependency-check
GOWORK=off make standard-impact-check
GOWORK=off go run ./cmd/goalcli score --min 9.8
```

涉及模板、generator、Harness 或 Evidence Runtime 的变更还必须补充对应的 targeted Go test、渲染检查、release manifest 校验和 downstream adoption 结论。
