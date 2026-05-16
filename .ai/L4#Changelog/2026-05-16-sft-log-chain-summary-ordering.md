# 2026-05-16 — SFT 日志链路结构与摘要排序说明回流

## 背景

排查 `/console/log` 中 `20/50` 摘要行、`51/52/21/29/59` 展开行和 `logs.id` 的关系时，确认：

1. `20/50` 不是数据库真实日志行。
2. `51/52/21/29/59` 由 `logs.type + trace_role + trace_seq` 派生显示。
3. 同链路依赖 `request_id/trace_id` 关联。
4. 当前链路内排序依赖 `trace_seq -> trace_sibling_seq -> id`。
5. 当前主表 `20/50` 摘要排序继承 `/api/log/grouped` 原始行 `id DESC` 下首次出现的位置。

## 更新

新增知识文档：

```text
.ai/L5#Knowledge/sft-log-chain-structure-and-summary-ordering.md
```

并更新：

```text
.ai/L5#Knowledge/README.md
```

## 后续要求

SFT 后续应把 `trace_parent_id` 和 `trace_sibling_seq` 真实写入，让数据库原始行可以严谨还原父子关系和同级顺序。

