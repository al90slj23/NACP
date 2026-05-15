# Implementation Plan: 请求链路视图 (Request Trace View)

## Overview

按照 Router → Controller → Service → Model 分层架构，实现请求链路视图功能。后端使用 Go (Gin + GORM)，前端使用 React + Semi Design。任务按依赖关系编排，支持并行执行。

## Tasks

- [x] 1. 后端：DTO 定义与 Service 层实现
  - [x] 1.1 创建 service/trace.go — 定义 DTO 结构体与查询函数
    - 创建 `service/trace.go` 文件
    - 定义 `TraceListParams` 结构体（Page, PageSize, StartTimestamp, EndTimestamp, ModelName, Username, Status）
    - 定义 `TraceSummary` 结构体（RequestId, CreatedAt, ModelName, Username, TokenName, Status, ChannelCount, TotalDuration, TotalQuota, TotalPromptTokens, TotalCompletionTokens）
    - 定义 `TraceStep` 结构体（Id, ChannelId, ChannelName, Type, StatusCode *int, UseTime, ModelName, Quota, CreatedAt）
    - 定义 `TraceDetail` 结构体（RequestId, CreatedAt, ModelName, Username, TokenName, TotalQuota, TotalPromptTokens, TotalCompletionTokens, Steps []TraceStep）
    - 实现 `GetTraceList(params TraceListParams) ([]TraceSummary, int64, error)` 函数：使用 LOG_DB 执行 GROUP BY 聚合查询，仅使用标准 SQL 聚合函数（COUNT, SUM, MIN, MAX, CASE WHEN），使用 `logGroupCol` 引用保留字列，HAVING 过滤条件确保 log_count >= 2 OR has_error = 1，支持 status 筛选在应用层过滤
    - 实现 `GetTraceDetail(requestId string) (*TraceDetail, error)` 函数：按 request_id 查询 type IN (2, 5, 51, 52) 的日志，按 created_at ASC 排序，LIMIT 100，从 Other 字段 JSON 解析 status_code（使用 `common.StrToMap`），批量查询渠道名称（复用 CacheGetChannel 模式）
    - 确保所有 JSON 操作使用 `common.Marshal`/`common.Unmarshal`/`common.StrToMap` 等包装函数
    - 确保 SQL 兼容 SQLite、MySQL、PostgreSQL（不使用 GROUP_CONCAT，不使用数据库特有 JSON 函数）
    - _需求: 1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.8, 5.1, 5.2, 5.3, 5.4, 6.1, 6.2, 6.3, 6.5_

  - [x]* 1.2 编写 service/trace_test.go — Property 1: 链路详情查询过滤与排序
    - **Property 1: 链路详情查询过滤与排序**
    - **验证: 需求 1.1**
    - 使用 `pgregory.net/rapid` 库
    - 使用内存 SQLite 作为测试数据库
    - 生成器：随机生成一组共享同一 request_id 的 Log 记录（type 随机覆盖 1/2/3/4/5/51/52）
    - 断言：返回的 steps 仅包含 type ∈ {2, 5, 51, 52}，按 created_at 升序，数量 ≤ 100

  - [x]* 1.3 编写 service/trace_test.go — Property 2: Other 字段 status_code 解析
    - **Property 2: Other 字段 status_code 解析**
    - **验证: 需求 1.3**
    - 生成器：随机生成 Other 字段内容（有效 JSON 含 admin_info.status_code、有效 JSON 不含该字段、空字符串、非法 JSON）
    - 断言：status_code 正确解析或为 nil

  - [x]* 1.4 编写 service/trace_test.go — Property 3: 链路最终结果分类
    - **Property 3: 链路最终结果分类**
    - **验证: 需求 2.4, 2.5**
    - 生成器：随机生成一组日志（type 混合），部分含 type=2，部分不含
    - 断言：存在 type=2 时 status="success"，否则 status="failed"

  - [x]* 1.5 编写 service/trace_test.go — Property 5: 链路摘要聚合正确性
    - **Property 5: 链路摘要聚合正确性**
    - **验证: 需求 2.3**
    - 生成器：随机生成一组共享 request_id 的日志，channel_id 和 created_at 随机
    - 断言：channel_count = 去重 channel_id 数，total_duration = max(created_at) - min(created_at)，created_at = min(created_at)

  - [x]* 1.6 编写 service/trace_test.go — Property 6: HAVING 过滤条件
    - **Property 6: HAVING 过滤条件**
    - **验证: 需求 2.8**
    - 生成器：随机生成多个 request_id 的日志集合（部分仅 1 条且无错误，部分 >= 2 条，部分含错误日志）
    - 断言：返回列表中每条记录的 request_id 满足 log_count >= 2 OR has_error

  - [x]* 1.7 编写 service/trace_test.go — Property 8: Quota 与 Token 聚合
    - **Property 8: Quota 与 Token 聚合**
    - **验证: 需求 6.1, 6.2, 6.3**
    - 生成器：随机生成一组日志（type=2 的记录含随机 quota/prompt_tokens/completion_tokens）
    - 断言：total_quota = sum(type=2 的 quota)，total_prompt_tokens = sum(type=2 的 prompt_tokens)，total_completion_tokens = sum(type=2 的 completion_tokens)

- [x] 2. 后端：Controller 与路由注册
  - [x] 2.1 创建 controller/trace.go — 实现 GetTraceList 和 GetTraceDetail
    - 创建 `controller/trace.go` 文件
    - 实现 `GetTraceList(c *gin.Context)`：解析 query 参数（p, page_size, start_timestamp, end_timestamp, model_name, username, status），参数校验（page >= 1, page_size 1-100，默认 20），调用 `service.GetTraceList`，返回标准分页响应
    - 实现 `GetTraceDetail(c *gin.Context)`：解析 request_id 参数，校验非空且长度 ≤ 64，调用 `service.GetTraceDetail`，返回标准响应
    - 错误处理：参数缺失返回 `{"success": false, "message": "..."}`，数据库错误返回通用错误信息
    - _需求: 1.4, 1.5, 2.2, 2.7_

  - [x] 2.2 在 router/api-router.go 注册链路路由
    - 在 `logRoute` 组中添加两条路由：
      - `logRoute.GET("/traces", middleware.AdminAuth(), controller.GetTraceList)`
      - `logRoute.GET("/trace", middleware.AdminAuth(), controller.GetTraceDetail)`
    - 放置在现有 logRoute 定义之后、`logRoute.Use(middleware.CORS()...)` 之前
    - _需求: 1.4, 2.7_

- [x] 3. 检查点 — 后端编译与测试
  - 确保 `go build ./...` 编译通过
  - 确保所有测试通过，如有问题请询问用户

- [x] 4. 前端：TracesPage 组件实现
  - [x] 4.1 创建 web/src/pages/Trace/index.jsx — 链路列表页面
    - 创建 `web/src/pages/Trace/index.jsx` 文件
    - 实现 TracesPage 组件，包含：
      - 筛选栏：Semi DatePicker (dateTimeRange 类型) + Input (模型名称, maxLength=100) + Input (用户名, maxLength=100) + Button (查询) + Button (重置)
      - 数据表格：Semi Table 展示链路摘要列表，列包括 request_id、请求时间（YYYY-MM-DD HH:mm:ss）、模型名称、用户名/Token 名称、最终结果（Tag green/red）、尝试渠道数、总耗时、总消耗额度
      - 行展开：expandedRowRender 展示链路详情时间线，使用树形连接线（├── / └──）展示步骤
      - 步骤状态图标：type=51 红色 ❌ "已拦截"、type=52 红色 ❌ "客户端错误"、type=2 绿色 ✅ "成功" + 额度显示
      - 分页：默认 20 条/页，可选 10/20/50/100
      - 加载状态：Table loading prop
      - 空状态：Table empty prop 显示"搜索无结果"
      - 错误提示：Semi Toast
      - Quota 格式化：quota × 0.000001 转美元，显示 `$X.XXXXXX`（6 位小数）
    - 使用 `useTranslation()` hook，所有文案使用 `t('中文key')` 格式
    - API 调用：GET /api/log/traces（列表）、GET /api/log/trace（详情）
    - _需求: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.8, 3.9, 3.10, 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7, 6.4_

  - [x]* 4.2 编写 web/src/pages/Trace/__tests__/format.test.js — Property 9: Quota 金额格式化
    - **Property 9: Quota 金额格式化**
    - **验证: 需求 6.4**
    - 使用 `fast-check` 库
    - 生成器：fc.nat() 生成随机非负整数 quota
    - 断言：格式化结果 === `$` + (quota * 0.000001).toFixed(6)

- [x] 5. 前端：路由与导航集成
  - [x] 5.1 在 SiderBar.jsx 添加"请求链路"菜单项
    - 在 `adminItems` 数组中添加：`{ text: t('请求链路'), itemKey: 'traces', to: '/traces', className: isAdmin() ? '' : 'tableHiddle' }`
    - 在 `routerMap` 对象中添加：`traces: '/console/traces'`
    - _需求: 3.7_

  - [x] 5.2 在 App.jsx 注册 /console/traces 路由
    - 导入 TracesPage 组件（lazy import）
    - 添加路由：`<Route path='/console/traces' element={<AdminRoute><TracesPage /></AdminRoute>} />`
    - _需求: 3.1_

- [x] 6. 前端：i18n 翻译键
  - [x] 6.1 在 web/src/i18n/locales/ 各语言文件中添加翻译键
    - 在 zh.json 中添加中文键（作为 key 本身，value 为中文）
    - 在 en.json 中添加对应英文翻译
    - 翻译键包括：请求链路、请求时间、尝试渠道数、总耗时、总消耗额度、最终结果、成功、失败、已拦截、客户端错误、无链路数据、搜索无结果、查询、重置、渠道ID、渠道名称、HTTP状态码、耗时、链路详情 等
    - _需求: 3.2, 3.7, 4.2, 4.3, 4.4, 4.5_

- [x] 7. 最终检查点 — 全栈编译验证
  - 确保 `go build ./...` 编译通过
  - 确保 `cd web && bun run build` 前端构建通过
  - 确保所有测试通过，如有问题请询问用户

## Notes

- 标记 `*` 的子任务为可选测试任务，可跳过以加速 MVP 交付
- 每个任务引用具体需求编号以确保可追溯性
- 属性测试验证系统的通用正确性属性，单元测试验证具体示例和边界情况
- 后端 JSON 操作必须使用 `common/json.go` 包装函数，禁止直接导入 `encoding/json`
- 数据库查询必须兼容 SQLite、MySQL、PostgreSQL 三种数据库
- 前端使用 Bun 作为包管理器和脚本运行器

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "4.1", "6.1"] },
    { "id": 1, "tasks": ["1.2", "1.3", "1.4", "1.5", "1.6", "1.7", "2.1", "4.2"] },
    { "id": 2, "tasks": ["2.2", "5.1", "5.2"] }
  ]
}
```
