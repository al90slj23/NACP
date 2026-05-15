# Implementation Plan: 日志链路分组展示 (Log Trace Grouping)

## Overview

按照 Router → Controller → Service → Model 分层架构，实现日志链路分组展示功能。后端修复同渠道重试日志缺失、新增探测日志记录、实现分组日志列表接口；前端改造 UsageLogsTable 支持摘要行折叠展开。任务按依赖关系编排，支持并行执行。

## Tasks

- [x] 1. 后端：常量定义与基础修复
  - [x] 1.1 在 model/log.go 新增探测日志类型常量
    - 在现有常量区域（`LogTypeErrorClientVisible = 52` 之后）新增：
      - `LogTypeProbeSuccess = 29`（探测成功）
      - `LogTypeProbeFailed = 59`（探测失败）
    - 不新增 type=20/21/50 常量（虚拟类型，仅 API 响应中出现）
    - _需求: 10.1, 10.2, 10.3_

  - [x] 1.2 修复 controller/relay.go 同渠道重试日志记录缺失
    - 在同渠道重试循环内（约第 310 行 `relayInfo.LastError = newAPIError` 之后），当 `newAPIError != nil` 时插入 type=51 日志记录
    - 使用 `model.RecordErrorLogWithType(c, model.LogTypeErrorIntercepted, ...)` 记录
    - Other 字段包含 `"retry_type": "same_channel"` 和 `"retry_index"` 标记
    - 条件守卫：`constant.ErrorLogEnabled && types.IsRecordErrorLog(newAPIError)`
    - 确保 request_id 通过 gin context 自动携带（RecordErrorLogWithType 内部已处理）
    - _需求: 1.1, 1.2, 1.3, 1.4_

  - [x] 1.3 实现 service/channel_probe.go 探测日志写入
    - 替换现有 `recordProbeLog` 函数的 TODO 实现为实际日志写入
    - 根据 `probeLog.Success` 判定 type=29 或 type=59
    - 使用 `gopool.Go` 异步写入 `model.LOG_DB.Create(log)`
    - 设置 user_id=0, quota=0（探测不计费）
    - Other 字段包含 `"probe_trigger"` 标记（"pre_warm" 或 "degraded_probe"）
    - 修改 `ProbeNextChannels` 函数签名，新增 `requestId string` 参数
    - 修改 `probeDegradedChannels` 调用点，传入空字符串 requestId
    - 修改 controller/relay.go 中 `ProbeNextChannels` 调用点，传入 `c.GetString(common.RequestIdKey)`
    - _需求: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 9.6_

- [x] 2. 检查点 — 后端基础修复编译验证
  - 确保 `go build ./...` 编译通过，如有问题请询问用户

- [x] 3. 后端：分组日志列表接口实现
  - [x] 3.1 创建 service/log_grouped.go — 数据结构与辅助函数
    - 创建 `service/log_grouped.go` 文件
    - 定义 `GroupedLogParams` 结构体（Page, PageSize, StartTimestamp, EndTimestamp, LogType, ModelName, Username, TokenName, Channel, Group, RequestId）
    - 定义 `GroupedLogItem` 结构体（统一的列表项，包含通用字段和摘要行专用字段 ChannelPath, TotalDuration, StepCount, IsSummary）
    - 实现 `buildChannelPath(channelIds []int) string` 函数：去除连续重复 channel_id，以 "→" 连接
    - 实现 `applyCommonFilters(tx *gorm.DB, params GroupedLogParams) *gorm.DB` 函数：应用时间范围、模型名称、用户名、Token 名称、渠道 ID、分组等筛选条件
    - 使用 `model/main.go` 中的 `logGroupCol` 变量引用 `group` 保留字列
    - _需求: 3.6, 4.1, 4.2, 4.3, 9.1, 9.3_

  - [x] 3.2 实现 service/log_grouped.go — GetGroupedLogs 核心查询逻辑
    - 实现 `GetGroupedLogs(params GroupedLogParams) ([]GroupedLogItem, int64, error)` 函数
    - 特殊分支：RequestId 非空时切换平铺模式（直接查询该 request_id 下所有日志）
    - 特殊分支：LogType=2 时仅返回无重试的普通消费行
    - 特殊分支：LogType=51/52 时仅返回含该类型的摘要行
    - 通用分支：Phase 1 查询有重试活动的 request_id 摘要（GROUP BY + 聚合），Phase 2 查询无重试的普通行，应用层合并排序分页
    - 实现 `getChannelPaths(requestIds []string) (map[string]string, error)` 批量查询渠道路径
    - 实现 `traceSummaryQuery` 子查询：找出有 type∈{51,52,29,59} 的 request_id，对其做 GROUP BY 聚合
    - 实现 `normalLogsQuery`：查询 type∈{2,5} 且 request_id 不在摘要集合中的普通行
    - 实现 `mergeAndPaginate`：合并两类数据按 created_at DESC 排序，截取当前页
    - 批量查询渠道名称（复用 CacheGetChannel 模式）
    - 确保所有 SQL 使用 GORM 抽象 + 标准 SQL 函数（COUNT, SUM, MIN, MAX, CASE WHEN），兼容三种数据库
    - _需求: 3.1, 3.2, 3.3, 3.4, 3.5, 3.7, 3.8, 3.9, 9.1, 9.2, 9.3, 9.4, 9.5_

  - [ ]* 3.3 编写 service/log_grouped_test.go — Property 4: 日志分组分类正确性
    - **Property 4: 日志分组分类正确性**
    - **验证: 需求 3.1, 3.3, 3.4**
    - 使用 `pgregory.net/rapid` 库
    - 使用内存 SQLite 作为测试数据库
    - 生成器：随机生成多个 request_id 的日志集合（type 随机覆盖 2/5/51/52/29/59，部分 request_id 有重试活动，部分无）
    - 断言：(a) 有 type∈{51,52,29,59} 的 request_id 归类为摘要行；(b) 摘要行有 type=2 则 type=20，否则 type=50；(c) 无重试的 type=2 作为普通行；(d) 结果按 created_at DESC 排列

  - [ ]* 3.4 编写 service/log_grouped_test.go — Property 5: 摘要行聚合字段正确性
    - **Property 5: 摘要行聚合字段正确性**
    - **验证: 需求 3.2**
    - 使用 `pgregory.net/rapid` 库
    - 生成器：随机生成一组共享 request_id 的日志，quota/prompt_tokens/completion_tokens 随机
    - 断言：total_quota = SUM(type=2 的 quota)，total_prompt_tokens = SUM(type=2 的 prompt_tokens)，total_completion_tokens = SUM(type=2 的 completion_tokens)，step_count = COUNT(*)，total_duration = MAX(created_at) - MIN(created_at)

  - [ ]* 3.5 编写 service/log_grouped_test.go — Property 6: Channel_Path 生成正确性
    - **Property 6: Channel_Path 生成正确性**
    - **验证: 需求 4.1, 4.2, 4.3**
    - 使用 `pgregory.net/rapid` 库
    - 生成器：随机生成 channel_id 序列（[]int，长度 0~20，值 1~100）
    - 断言：(a) 结果中无连续重复 channel_id；(b) 仅包含数字和 "→" 字符；(c) 空输入返回空字符串；(d) 单元素输入返回该数字字符串

  - [x] 3.6 实现 controller/log.go — GetGroupedLogs 控制器函数
    - 在 `controller/log.go` 中新增 `GetGroupedLogs(c *gin.Context)` 函数
    - 解析 query 参数：p, page_size, start_timestamp, end_timestamp, type, model_name, username, token_name, channel, group, request_id
    - 参数校验：page >= 1（默认 1），page_size 1-100（默认 20）
    - 调用 `service.GetGroupedLogs(params)`
    - 返回标准分页响应 `{"success": true, "data": {"page", "page_size", "total", "items"}}`
    - _需求: 3.1, 3.6_

  - [x] 3.7 在 router/api-router.go 注册分组日志路由
    - 在 `logRoute` 组中添加：`logRoute.GET("/grouped", middleware.AdminAuth(), controller.GetGroupedLogs)`
    - 放置在现有 logRoute 路由定义附近
    - _需求: 3.1_

- [x] 4. 后端：Trace_Detail_API 扩展
  - [x] 4.1 扩展 service/trace.go GetTraceDetail 支持 type=29/59
    - 修改 `GetTraceDetail` 函数中的 WHERE 条件：`type IN (2, 5, 51, 52)` → `type IN (2, 5, 51, 52, 29, 59)`
    - 在 `TraceStep` 结构体中新增 `Other string` 字段（`json:"other,omitempty"`），以便前端获取 retry_type/probe_trigger 标记
    - 在构建 step 时赋值 `Other: row.Other`
    - _需求: 8.1, 8.2, 8.3_

  - [ ]* 4.2 编写 service/trace_test.go — Property 7: 链路详情包含所有日志类型
    - **Property 7: 链路详情包含所有日志类型**
    - **验证: 需求 8.1, 8.2, 8.3**
    - 使用 `pgregory.net/rapid` 库
    - 使用内存 SQLite 作为测试数据库
    - 生成器：随机生成一组共享 request_id 的 Log 记录（type 随机覆盖 1/2/3/4/5/29/51/52/59）
    - 断言：返回的 steps 仅包含 type∈{2,5,51,52,29,59}，按 created_at 升序，每条 step 包含 Other 字段

- [x] 5. 检查点 — 后端全部功能编译与测试
  - 确保 `go build ./...` 编译通过
  - 确保所有测试通过，如有问题请询问用户

- [x] 6. 前端：TraceExpandRender 组件与列渲染适配
  - [x] 6.1 创建 web/src/components/table/usage-logs/TraceExpandRender.jsx
    - 新建 TraceExpandRender 组件
    - 接收 `requestId` prop，调用 `GET /api/log/trace?request_id=xxx` 获取链路详情
    - 加载中显示 Semi Spin 组件
    - 步骤为空时显示 Semi Empty "无链路数据"
    - 使用树形连接线样式渲染步骤列表（├── / └──）
    - 步骤类型渲染：type=51 红色 ❌ "已拦截"、type=52 红色 ❌ "客户端错误"、type=2/21 绿色 ✅ "成功"、type=29 蓝色 🔍 "探测成功"、type=59 灰色 🔍 "探测失败"
    - 每步显示：渠道 ID 和名称、HTTP 状态码（Tag）、耗时（秒）、type=2 时额度（renderQuota）
    - 使用 `useTranslation()` hook，文案使用 `t('中文key')` 格式
    - _需求: 5.6, 5.7, 5.8, 5.9, 5.10, 5.11, 5.12, 5.13, 5.14_

  - [x] 6.2 修改 web/src/components/table/usage-logs/UsageLogsColumnDefs.jsx — 新增类型渲染
    - 在 `renderType` 函数中新增 type=20 和 type=50 的渲染：
      - type=20：绿色 Tag "成功(重试)"
      - type=50：红色 Tag "失败(重试)"
    - 在渠道列 render 函数中，对摘要行（`record.is_summary && record.channel_path`）渲染 Channel_Path：将 "12→14" 拆分为多个彩色 Tag，中间以 "→" 连接
    - channel_path 为空时渠道列显示 "-"
    - _需求: 5.2, 5.3, 6.6_

  - [x] 6.3 修改 UsageLogsTable — 切换 API 与差异化展开行为
    - 将日志列表 API 端点从 `/api/log/` 切换为 `/api/log/grouped`
    - 实现行类型判断：`record.is_summary === true` 为摘要行
    - 摘要行展开：渲染 `<TraceExpandRender requestId={record.request_id} />`
    - 普通行展开：保持现有 Descriptions 展开行为不变
    - 摘要行始终可展开（显示展开箭头），普通行保持现有 expandable 逻辑
    - 摘要行的时间列、模型列、用户列、花费列、用时列、输入/输出列使用现有渲染函数（数据字段名一致）
    - _需求: 5.1, 5.4, 5.5, 5.6, 6.1, 6.2, 6.3, 6.4, 6.5, 6.7, 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 7. 前端：i18n 翻译键
  - [x] 7.1 在 web/src/i18n/locales/ 各语言文件中添加翻译键
    - 新增翻译键：
      - "成功(重试)" → "Success (Retry)"
      - "失败(重试)" → "Failed (Retry)"
      - "探测成功" → "Probe Success"
      - "探测失败" → "Probe Failed"
      - "已拦截" → "Intercepted"
      - "客户端错误" → "Client Error"
      - "无链路数据" → "No trace data"
    - 在 zh.json、en.json 中添加（其他语言文件按需同步）
    - _需求: 5.2, 5.3, 5.8, 5.9, 5.10, 5.11, 5.12_

- [x] 8. 最终检查点 — 全栈编译验证
  - 确保 `go build ./...` 编译通过
  - 确保 `cd web && bun run build` 前端构建通过
  - 确保所有测试通过，如有问题请询问用户

## Notes

- 标记 `*` 的子任务为可选测试任务，可跳过以加速 MVP 交付
- 每个任务引用具体需求编号以确保可追溯性
- 属性测试使用 `pgregory.net/rapid` 库（项目已有依赖），验证系统的通用正确性属性
- 后端 JSON 操作必须使用 `common/json.go` 包装函数，禁止直接导入 `encoding/json`
- 数据库查询必须兼容 SQLite、MySQL、PostgreSQL 三种数据库
- 前端使用 Bun 作为包管理器和脚本运行器
- 分组日志接口采用两阶段查询 + 应用层合并策略，避免 N+1 查询
- Channel_Path 在应用层生成（避免 GROUP_CONCAT 跨数据库不兼容问题）
- 探测日志使用 gopool.Go 异步写入，避免阻塞用户请求重试流程

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "6.1", "7.1"] },
    { "id": 1, "tasks": ["1.2", "1.3", "3.1", "6.2"] },
    { "id": 2, "tasks": ["3.2", "4.1"] },
    { "id": 3, "tasks": ["3.3", "3.4", "3.5", "3.6", "4.2"] },
    { "id": 4, "tasks": ["3.7", "6.3"] }
  ]
}
```
