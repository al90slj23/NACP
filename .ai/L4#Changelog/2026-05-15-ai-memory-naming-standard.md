# 2026-05-15 — AI 记忆体系命名规范

## 操作内容

新增 NACP `.ai` 六层记忆体系命名规范，确认当前 `.ai` 文件命名尚未完全遵循 ZERO 框架，需要先建立本地命名规则和迁移清单，再分批执行重命名。

## 文件变更

- 新增 `.ai/L3#Standards/standards/10.ai-memory-01.naming.md`
- 更新 `.ai/L2#Index/toc.md`

## 决策记录

1. 不立即批量重命名旧文件，避免 `.ai` 内部引用断裂。
2. 先固化命名规范，再按迁移清单逐批 `git mv`。
3. `L3#Standards (Standards)` 采用 ZERO 分类编号。
4. `L5#Knowledge (Knowledge)` 在 ZERO 基础上增加 NACP 本地约定：`{领域}-{主题}-{文档类型}.md`。
5. 后续 CAF 文档和 NewAPI 系统图谱文档必须按新命名规则创建。

