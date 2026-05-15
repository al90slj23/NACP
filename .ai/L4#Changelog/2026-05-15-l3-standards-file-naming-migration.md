# 2026-05-15 — L3 规范文件命名迁移

## 操作内容

按 `10.ai-memory-01.naming.md` 的第一批迁移清单，完成 `L3#完整规范 (Standards)` 文件命名收敛。

## 文件重命名

| 原文件 | 新文件 |
|--------|--------|
| `01.arch-01.relay-retry.md` | `01.arch-01.smart-failover-trace.md` |
| `01.arch-02.baseline.md` | `01.arch-02.newapi-v0-13-2-baseline.md` |
| `02.dev-01.json-usage.md` | `02.backend-01.json-wrapper-usage.md` |
| `02.dev-02.db-compat.md` | `02.backend-02.database-compatibility.md` |
| `02.dev-03.dto-pointer.md` | `02.backend-03.relay-dto-explicit-zero-values.md` |
| `03.quality-01.git.md` | `06.quality-01.git-workflow.md` |

## 同步更新

- 更新 `.ai/L2#Index/toc.md`
- 更新 `.ai/L4#Changelog/2026-05-13-project-init.md` 中的当前路径引用
- 更新部分 L3 文件标题，使标题与新文件名一致
- 更新 `.ai/L3#Standards/standards/10.ai-memory-01.naming.md`，把迁移清单标记为迁移记录

## 决策记录

1. L3 是规范源头，优先于 L5 迁移。
2. 迁移使用 `git mv`，保留 Git 历史。
3. 历史迁移表和 changelog 允许保留旧文件名，作为审计对照。
4. 后续新增 L3 规范必须直接使用 ZERO 分类编号命名。

