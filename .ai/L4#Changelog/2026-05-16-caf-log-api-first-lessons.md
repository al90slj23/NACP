# 2026-05-16 — CAF 日志页 API 先行测试经验回流

## 背景

本地测试 `http://localhost:5173/console/log` 时，页面出现空表。

排查后确认并非无数据，也不是单纯前端渲染问题，而是：

1. `/api/log/grouped` 默认第一页被后台独立健康探测日志占满。
2. 这些 probe 行没有 `request_id/user/token`。
3. 前端为避免独立 probe 刷屏，会隐藏无链路归属的 `probe_success/probe_failed`。
4. 结果是 API 第一页有数据，但前端过滤后页面为空。

## 经验回流

已更新：

1. `.ai/L3#Standards/standards/06.quality-02.change-assurance-framework.md`
   - 明确 CAF 环境门禁：本地开发环境测试 -> 线上测试服务器测试 -> 正式服务器部署后观察。
   - 明确本地未通过不得进入测试服务器，测试服务器未通过不得部署正式服务器。
   - 新增 CAF Phase 5 的“API 先行原则”。
   - 明确日志、trace、计费、统计类问题先查 API 和数据字段，再验证浏览器展示。

2. `.ai/L5#Knowledge/caf-change-assurance-framework-playbook.md`
   - 更新固定流程为 12 步，加入本地、测试服务器、正式服观察三阶段。
   - 新增环境分阶段测试表模板：Stage A / Stage B / Stage C。
   - 新增“日志页与链路功能测试经验”。
   - 固化 `/api/log/grouped`、`/api/log/trace`、`/api/log/traces` 的检查顺序和字段清单。
   - 固化后台独立 probe 不得污染默认日志列表的规则。

3. `.ai/L5#Knowledge/README.md`
   - 更新时间。
   - 增加后续脚本化方向：CAF 日志页 API 先行测试脚本化。

## 对应代码修复

本轮同步修复：

1. `service/log_grouped.go`
   - 默认排除没有 `request_id` 的后台独立 probe 行。
   - 保留有 `request_id` 的链路内 probe，供 trace 展开使用。

2. `service/log_grouped_test.go`
   - 新增后台独立 probe 不污染默认日志列表的测试。

3. `service/trace.go`
   - 修复 `/api/log/traces` 按 `username/token_name` 拆分同一 `request_id` 摘要的问题。

4. `service/trace_test.go`
   - 新增同一 `request_id` 只能归并成一条摘要的测试。

## 验证

```bash
go test ./service -run 'TestGroupedLogs|TestTrace'
```

结果：PASS。

## 后续

建议把本次手工 API 检查固化为 smoke 脚本，至少覆盖：

1. 管理员登录。
2. `/api/log/grouped` 默认页非空且不被 standalone probe 占满。
3. `/api/log/traces` 同一 `request_id` 不重复。
4. `/api/log/trace` 成功/失败链路顺序与终态正确。
