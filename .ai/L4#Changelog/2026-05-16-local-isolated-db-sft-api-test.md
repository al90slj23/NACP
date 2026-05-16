# 2026-05-16 — 本地隔离数据库 SFT API 测试

## 操作内容

1. 将 CAF 本地测试规则从“可共用 `nacp.m.srl` 测试库”调整为“本地 Stage A 使用本地隔离数据库”。
2. 将本地 `.env` 数据库连接切换到 Docker MySQL `nacp-mysql-dev`。
3. 使用本地临时后端 `http://127.0.0.1:3001` 和本地 mock upstream 执行 API 驱动测试。
4. 通过 API 创建普通用户、用户 token、失败渠道、成功渠道。
5. 验证直接成功 `2`、容错成功 `20/21/29/51/59`、容错失败 `50/52/51/59`。

## 关键发现

1. `RetryTimes=0` 时不会进入增强重试链；SFT 测试必须设置 `RetryTimes >= 1`，建议为 `2`。
2. 成功 probe `29` 可以记录 usage/quota，显示层应展示为平台运营消耗而非用户扣费。
3. 真实 summary 行 `20/50` 和子步骤 `21/29/51/52/59` 已能通过 API 验证。

## 证据文档

`.ai/L5#Knowledge/caf-local-isolated-db-sft-api-report.md`

