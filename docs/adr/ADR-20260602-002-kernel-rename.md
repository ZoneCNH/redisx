# ADR-20260602-002 kernel 默认下游名

## 状态

已采纳。

## 背景

旧示例名 `foundationx` 容易与当前标准源身份混淆。L0 下游库需要一个稳定、简短、与标准工厂职责分离的名称，用于承载最小公共内核能力。

## 决策

默认 L0 下游库名称采用 `kernel`。`foundationx` 仅保留在迁移文档、历史审计和旧兼容说明中，不作为新模板渲染目标或新文档主身份。

生成示例必须使用：

```bash
scripts/render_template.sh \
  --module-name kernel \
  --module-path github.com/ZoneCNH/kernel \
  --package-name kernel \
  --out ../kernel
```

## 影响

下游矩阵、模板生成契约、README、release evidence 和同步策略必须以 `kernel` 作为 L0 默认目标。新增文档不得把 `foundationx` 描述为当前默认下游库。

## 验证

验证入口包括 `GOWORK=off make docs-check`、`GOWORK=off make standard-impact-check` 和模板渲染相关测试。
