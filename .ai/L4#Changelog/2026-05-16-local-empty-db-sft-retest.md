# 2026-05-16 — 本地空库 SFT 容错重试复测

## 操作内容

1. 使用本地 `http://localhost:5173/` + `http://localhost:3000`，连接本地 Docker MySQL `nacp_dev`，重新创建 SFT 测试夹具。
2. 通过 API 创建普通用户、用户 token、测试模型、成功/失败渠道，并设置 `RetryTimes=2` 与测试模型倍率。
3. 通过 OpenAI-compatible mock upstream 执行三类真实 relay 调用：直接成功、容错成功、容错失败。
4. 验证 `/api/log/grouped`、`/api/log/trace` 与日志页 `/console/log` 的 summary 行和展开步骤。
5. 修复日志页前端列规则，使 `20/21/29/50/51/52/59` 显示完整请求字段。

## 关键发现

1. 旧后端进程如果未重启，可能仍带 `SKIP_DB_MIGRATION=true`，导致新日志字段未迁移；本地测试前必须确认后端已读取当前 `.env`。
2. 普通用户 token 默认不能访问临时自定义分组；SFT API 夹具应使用 `default`，或显式配置用户可访问分组。
3. 新模型如果未加入 `ModelRatio`，relay 会返回 `model_price_error`；SFT 夹具创建后必须配置模型倍率。
4. 本轮 API 数据符合 SFT 终端约束：`20` 以 `21` 收尾，`50` 以 `52` 收尾。
5. 前端原先只把 `2/5` 识别为请求类日志，导致 `20/50` summary 行数据存在但显示为空；已修复。

## 验证结果

| 项目 | 结果 |
|------|------|
| 直接成功 `2` | 通过 |
| 容错成功 `20 -> 21` | 通过 |
| 成功探测 `29` usage/quota | 通过 |
| 容错失败 `50 -> 52` | 通过 |
| 失败探测 `59` | 通过 |
| grouped 默认列表隐藏子步骤 | 通过 |
| trace 展开顺序按 `trace_seq` | 通过 |
| 日志页 20/50 字段完整显示 | 通过 |
| `bun run build` | 通过 |

## 证据文档

`.ai/L5#Knowledge/caf-local-isolated-db-sft-api-report.md`
