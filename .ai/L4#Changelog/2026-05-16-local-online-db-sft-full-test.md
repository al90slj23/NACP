# 2026-05-16 — 本地代码直连测试站数据库完成 SFT 完整 API 复测

## 操作内容

1. 使用 `./gogogo.sh 0 restart` 重启本地开发环境，前端固定 `23901`，后端固定 `23900`。
2. 本地代码连接 `nacp.m.srl` 测试站 MySQL，通过 API 创建普通用户、token 和 6 个 OpenAI-compatible 测试渠道。
3. 使用普通用户 token 实际调用 `/v1/chat/completions`，覆盖直接成功、容错成功、容错失败。
4. 通过 `/api/log/grouped`、`/api/log/trace`、`/api/log/traces` 校验 SFT 摘要、子步骤、排序、筛选和 token/quota。
5. 发现并修复 `/api/log/traces` 的 `token_name` 过滤失效问题。
6. 发现并修复 `summary.channel_path` 混入 probe 渠道导致正式路径失真的问题。
7. 复跑完整脚本，最终通过。

## 文件变更

1. `controller/trace.go`：读取 `token_name` 查询参数。
2. `service/trace.go`：`TraceListParams` 增加 `TokenName`，链路列表查询按 token 名过滤。
3. `model/log.go`：`summary.channel_path` 只统计正式请求步骤，排除 `29/59` probe，并只压缩相邻重复渠道。
4. `.ai/L5#Knowledge/caf-local-online-db-sft-api-report.md`：新增本轮测试报告。
5. `.ai/L5#Knowledge/README.md`：增加本轮报告入口。

## 决策记录

本轮继续执行“真实 API 路径优先”的测试规范，不直接伪造日志数据。测试证明成功 probe `29` 在上游返回 usage 时会记录 token/quota；失败 probe `59` 因上游失败无 usage，可保持 0。

SFT 主链路展示必须以真实 `20/50` 摘要行为入口，展开顺序以 `trace_seq` 为准，正式渠道路径以 `51/21/52` 等正式请求步骤为准，不能由 probe 或日志 id 推断。

