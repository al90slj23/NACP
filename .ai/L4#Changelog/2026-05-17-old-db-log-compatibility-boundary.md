# 2026-05-17 — 老库日志兼容边界回流

## 操作内容

将 NACP 数据库演进边界写入 `.ai`：

1. 数据库允许通过补字段启用 NACP 新能力。
2. 补字段后，老 NewAPI 日志必须继续以旧日志形态正常显示。
3. 老日志不强行转换为 SFT 链路，不误判为 `20/50`，不污染计费、计量和统计。
4. 业务库和日志库分离是目标形态，迁移 SQL 必须明确区分 `SQL_DSN` 和 `LOG_SQL_DSN`。
5. 涉及日志 schema、日志 API、日志页面的变更必须执行老库日志兼容验收。

## 文件变更

- 更新 `.ai/L3#Standards/standards/02.backend-02.database-compatibility.md`
- 更新 `.ai/L3#Standards/standards/06.quality-04.api-driven-test-fixtures.md`

## 决策记录

数据库升级的合格标准不是单纯“字段加成功”，而是：

```text
旧 NewAPI 双库数据 + NACP 新字段
-> 后端启动正常
-> /api/log/grouped 正常返回老日志
-> /console/log 正常显示老日志
-> 新请求可以写入并展示 NACP SFT 链路日志
```

