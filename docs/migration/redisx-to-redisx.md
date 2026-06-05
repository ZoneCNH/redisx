# redisx 迁移指南

## 目的

本文记录旧 `redisx` 身份迁移到当前 `redisx` 标准工厂身份的规则。旧名仅允许出现在迁移、审计和历史决策语境中；新增标准、README、release note 和 PR 描述默认使用当前仓库身份。

## 迁移规则

- 标准源仓库统一为 `https://github.com/ZoneCNH/redisx`。
- 仓库职责统一为 Standard Source、Go Reference Template、Generator、Harness 和 Evidence Runtime。
- 默认 L0 下游库名称使用 `kernel`，不再使用 `foundationx` 作为新目标名。
- `pkg/redisx` 是参考模板包；渲染后必须替换为 `pkg/<package-name>`。
- module path、package name、README、docs、contracts、examples、testkit、release manifest 和 `.agent` 控制面必须由 `scripts/render_template.sh` 生成或验证。

## 禁止事项

- 不得在新增文档中把旧名作为当前身份。
- 不得让基础库依赖 `x.go` 或私有业务仓库。
- 不得把迁移叙事写入公共 API、测试输出、release manifest 或 Evidence artifact，除非该 artifact 明确记录历史兼容性。

## 验证

迁移后至少运行：

```bash
GOWORK=off make docs-check
GOWORK=off make boundary
GOWORK=off make contracts
GOWORK=off make standard-impact-check
```

发布前还必须生成 release Evidence，并在完成声明中包含 `DONE with evidence:`。
