# 2026-05-15 — L5 知识文档命名迁移

## 操作内容

按 `10.ai-memory-01.naming.md` 的第一批 L5 迁移清单，完成 `L5#知识图谱 (Knowledge)` 文件命名收敛。

## 文件重命名

| 原文件 | 新文件 |
|--------|--------|
| `business-requirements-v1.md` | `nacp-business-requirements-v1.md` |
| `channel-affinity-skipretry-analysis.md` | `channel-affinity-skip-retry-analysis.md` |
| `data-analysis-stream-window2.md` | `data-stream-window-2-analysis.md` |
| `integration-test-plan.md` | `test-integration-plan.md` |
| `nacp-full-changelog-for-review.md` | `nacp-full-changelog-review.md` |
| `nacp-smart-failover-trace.md` | `sft-smart-failover-trace-analysis.md` |
| `nacp-test-execution-plan.md` | `test-nacp-online-execution-plan.md` |
| `nacp-v0.13.2-regression-test-plan.md` | `test-newapi-v0-13-2-regression-plan.md` |
| `relay-retry-mechanism-analysis.md` | `newapi-relay-retry-mechanism-analysis.md` |

## 同步更新

- 更新 `.ai/L5#Knowledge/README.md`
- 更新测试计划文档中的输入依据引用
- 更新 `.ai/L3#Standards/standards/10.ai-memory-01.naming.md`，把 L5 迁移清单标记为迁移记录

## 决策记录

1. `channel-affinity-analysis.md` 已符合“领域-主题-文档类型”形式，暂不重命名。
2. `nacp-smart-failover-trace.md` 是本轮新增未入 Git 的文件，使用 `apply_patch` 移动到目标文件名。
3. L5 历史迁移表和 changelog 允许保留旧文件名，作为审计对照。
4. 后续 CAF、新系统图谱、深度测试计划必须直接使用 L5 新命名规则。

