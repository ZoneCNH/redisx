# redisx

`redisx` 是 FoundationX 的 **L2 Redis adapter 基础库**，为上层服务提供 KV/TTL/Hash/List/Pipeline/Cache-aside/Lock/RateLimit/Pool/Persistence 等 Redis 能力封装。

本仓库遵循 [xlib-standard](https://github.com/ZoneCNH/xlib-standard) 治理协议，但不是标准源、不是 generator、不是模板仓库。

## 项目概述

公开包：`pkg/redisx`。提供标准化的 Redis 访问层，覆盖数据结构操作、缓存策略、分布式锁和连接池管理。

## 硬性约束

- **禁止依赖 `x.go`** 或任何业务缓存 key/schema
- **不在公开 API 中泄露 go-redis 具体类型**
- **生产凭证通过 Config 显式注入**，不在源码、日志或 artifact 中硬编码
- **不创建隐藏全局客户端**——所有 Client 显式构造，显式 Close

## 编辑前基线确认

> 同步自 ZoneCNH/CLAUDE.md 工作流规则（PR #340-#343）。

- **编辑前先 `git log --oneline -5`，然后 `Read` 确认目标文件当前内容**——禁止假设文件仍是自己记忆中的状态。
- **对文档中的代码事实声称，核对源码后再提交**——用 `grep` 确认字段存在，用 `head` 确认文档不是占位符。
- **先列验证清单，再列变更清单**——先确定需要查什么，验证完再按变更清单编辑。

## 语言规则（全局强制）

1. **回答语言**：所有对话回复默认使用中文，除非用户明确要求使用其他语言。
2. **文档语言**：所有仓库文档默认使用中文叙述。
3. **代码注释**：Go 源码注释默认使用中文。导出符号的 godoc 注释可保留英文，内部代码一律中文。
4. **保留原文的例外**：代码标识符、命令、路径、包名、Go module 路径、外部专有名词、协议固定短语和 git 提交标题保留原文。
5. **提交信息**：正文和 trailer 使用中文；标题保留英文以兼容工具链。
