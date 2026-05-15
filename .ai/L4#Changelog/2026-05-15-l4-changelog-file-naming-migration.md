# 2026-05-15 — L4 操作日志文件命名迁移

## 操作内容

按 `.ai` 命名规范，把 `L4#操作日志 (Changelog)` 中带版本号点号的历史文件名收敛为小写连字符格式。

## 文件重命名

| 原文件 | 新文件 |
|--------|--------|
| `2026-05-13-v0.1.0-smart-relay-retry.md` | `2026-05-13-v0-1-0-smart-relay-retry.md` |
| `2026-05-14-v0.1.0-integration-test-results.md` | `2026-05-14-v0-1-0-integration-test-results.md` |
| `2026-05-14-v0.1.1-error-log-split-and-devenv.md` | `2026-05-14-v0-1-1-error-log-split-and-devenv.md` |

## 决策记录

1. 文件名中不使用点号表达版本，避免和 ZERO 文档编号格式混淆。
2. changelog 正文仍可保留真实版本号，例如 `v0.1.0`。
3. `.DS_Store` 已在 `.gitignore` 中忽略，不进入 Git 迁移范围。

