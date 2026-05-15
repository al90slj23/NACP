# 2026-05-16 — classic-plus-0.2.0 本地 API 测试计划

## 背景

按 CAF 新增环境门禁，`classic-plus-0.2.0-dev` 必须先完成本地开发环境测试，再进入线上测试服务器测试，最后才允许正式服务器部署后观察。

本轮用户要求针对 `http://localhost:5173/` 制定本地 API 优先的完整测试计划。

## 新增文档

- `.ai/L5#Knowledge/caf-classic-plus-0-2-0-local-api-plan.md`

## 计划范围

该计划确认：

1. NewAPI v0.13.2 是上游基础版本。
2. `classic-plus-0.1.0` 是 NACP 的第一个增强版本，核心为智能重试与渠道健康管理。
3. `classic-plus-0.2.0-dev` 在 0.1.x 基础上补齐 SFT 结构化链路、日志兼容、CAF 与版本治理。

本地计划覆盖：

1. `http://localhost:5173/api/status` 和管理员登录。
2. `/api/log/grouped`、`/api/log/traces`、`/api/log/trace`。
3. `logs.type` 不写入 `20/21/29/50/51/52/59`。
4. standalone probe 不污染默认日志列表。
5. 同一 `request_id` 不重复摘要。
6. 成功/失败链路终态。
7. 普通用户 token 调 Claude/Codex。
8. 计费、计量、统计不被 probe/intercepted error 污染。
9. 旧日志兼容。
10. 最后少量 `/console/log` 显示层抽查。

## 索引更新

- `.ai/L5#Knowledge/README.md` 增加本地 API 测试计划入口。

## 后续

本地 Stage A 通过后，才进入已有线上测试站计划：

- `.ai/L5#Knowledge/caf-classic-plus-0-2-0-plan.md`
- `.ai/L5#Knowledge/test-nacp-online-execution-plan.md`
