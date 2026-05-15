# Requirements Document

## Introduction

为 NACP v0.2.0 新增"请求链路"功能，按 `request_id` 分组展示每次 API 请求的完整链路信息，包括哪些渠道被尝试、哪些失败、最终结果。该功能帮助管理员快速定位请求的重试路径和失败原因，提升系统可观测性。

## Glossary

- **Trace_Service**：后端请求链路查询服务，负责按 request_id 聚合日志并返回结构化链路数据
- **Trace_List_API**：`GET /api/log/traces` 接口，分页列出最近的请求链路摘要
- **Trace_Detail_API**：`GET /api/log/trace` 接口，按 request_id 查询单条请求的完整链路详情
- **Trace_View**：前端请求链路视图页面组件，展示链路列表和详情
- **Request_Id**：每次 API 请求的唯一标识符，已存储在 Log 表的 `request_id` 字段中
- **链路步骤（Trace_Step）**：一条请求链路中的单个渠道尝试记录，对应一条 Log 记录
- **拦截日志（Intercepted_Log）**：类型为 51 的日志，表示错误被重试系统拦截，客户端未感知
- **客户端可见日志（Client_Visible_Log）**：类型为 52 的日志，表示错误最终返回给客户端
- **消费日志（Consume_Log）**：类型为 2 的日志，表示请求成功完成并计费

## Requirements

### 需求 1：按 request_id 查询完整链路

**用户故事：** 作为管理员，我想通过 request_id 查询一次请求的完整链路，以便了解该请求经历了哪些渠道尝试和最终结果。

#### 验收标准

1. WHEN 管理员提供有效的 request_id 参数（非空字符串，最大长度 64 字符）调用 Trace_Detail_API，THE Trace_Service SHALL 返回该 request_id 关联的所有日志记录（日志类型包括 Consume=2、ErrorIntercepted=51、ErrorClientVisible=52），按创建时间升序排列，最多返回 100 条记录
2. WHEN 管理员提供的 request_id 在数据库中不存在，THE Trace_Service SHALL 返回空的链路步骤列表和 HTTP 200 状态码
3. THE Trace_Detail_API SHALL 对每条链路步骤返回以下字段：日志 ID、渠道 ID、渠道名称、日志类型、HTTP 状态码（从 Other 字段 JSON 解析，若 Other 为空或不含该字段则返回 null）、耗时（use_time，单位为秒）、模型名称、创建时间（Unix 时间戳）
4. IF 非管理员权限的用户调用 Trace_Detail_API，THEN THE Trace_Service SHALL 拒绝请求并返回 HTTP 403 状态码及权限不足的错误提示
5. IF 管理员调用 Trace_Detail_API 时未提供 request_id 参数或参数为空字符串，THEN THE Trace_Service SHALL 返回 HTTP 400 状态码及参数缺失的错误提示

### 需求 2：分页列出最近的请求链路摘要

**用户故事：** 作为管理员，我想分页浏览最近的请求链路列表，以便快速发现异常请求和重试情况。

#### 验收标准

1. WHEN 管理员调用 Trace_List_API 并提供分页参数（page、page_size），THE Trace_Service SHALL 返回按最早请求时间倒序排列的请求链路摘要列表，同时返回符合条件的链路总数（total）以支持前端分页控件
2. IF page 小于 1 或 page_size 不在 1–100 范围内，THEN THE Trace_List_API SHALL 拒绝请求并返回参数错误提示；未提供 page_size 时默认值为 20
3. THE Trace_Service SHALL 对每条链路摘要聚合以下信息：request_id、最早请求时间（该 request_id 下最小 CreatedAt）、模型名称（ModelName）、用户名（Username）、Token 名称（TokenName）、最终结果（成功或失败）、尝试渠道总数（该 request_id 下不同 ChannelId 的去重计数）、总耗时（该 request_id 下最大 CreatedAt 减去最小 CreatedAt，单位为秒）
4. IF 一个 request_id 下存在 Type=2（Consume）的日志记录，THEN THE Trace_Service SHALL 将该链路最终结果标记为"成功"
5. IF 一个 request_id 下不存在 Type=2（Consume）的日志记录，THEN THE Trace_Service SHALL 将该链路最终结果标记为"失败"
6. THE Trace_List_API SHALL 支持按时间范围（start_timestamp、end_timestamp，Unix 秒级时间戳）、模型名称（精确匹配）、用户名（精确匹配）、最终结果状态（成功/失败）进行筛选，所有筛选条件均为可选
7. IF 不具有管理员权限的用户调用 Trace_List_API，THEN THE Trace_List_API SHALL 拒绝请求并返回权限不足的错误提示
8. THE Trace_List_API SHALL 仅返回包含 2 条及以上日志记录的链路，或包含错误日志（Type=5 或 Type=51 或 Type=52）的链路，以过滤无重试的正常请求

### 需求 3：前端链路列表视图

**用户故事：** 作为管理员，我想在管理后台看到请求链路列表页面，以便快速浏览和筛选请求链路。

#### 验收标准

1. THE Trace_View SHALL 在路由 `/admin/traces` 下注册为仅管理员可访问的页面（使用 AdminRoute 权限守卫）
2. THE Trace_View SHALL 以表格形式展示链路列表，每行显示：request_id、请求时间（格式：YYYY-MM-DD HH:mm:ss）、模型名称、用户名/Token 名称、最终结果（成功/失败标签）、尝试渠道数，默认按请求时间降序排列
3. WHEN 最终结果为成功，THE Trace_View SHALL 使用 Semi Design Tag 组件的 green 类型显示"成功"
4. WHEN 最终结果为失败，THE Trace_View SHALL 使用 Semi Design Tag 组件的 red 类型显示"失败"
5. THE Trace_View SHALL 提供时间范围选择器（DateTimeRange 类型）、模型名称输入框（最大长度 100 字符）、用户名输入框（最大长度 100 字符）作为筛选条件，并提供"查询"按钮提交筛选和"重置"按钮清空所有筛选条件
6. THE Trace_View SHALL 支持分页浏览，默认每页 20 条记录，可选每页条数为 10、20、50、100
7. THE Trace_View SHALL 在侧边栏导航菜单的管理员区域中添加"请求链路"入口，itemKey 为 `traces`
8. WHILE 数据正在加载，THE Trace_View SHALL 在表格区域显示加载状态指示器
9. IF 查询结果为空，THEN THE Trace_View SHALL 在表格区域显示空状态占位图和"搜索无结果"提示文案
10. IF 数据加载失败，THEN THE Trace_View SHALL 通过 Toast 通知显示错误提示信息

### 需求 4：前端链路详情展开视图

**用户故事：** 作为管理员，我想点击某条链路后查看完整的时间线详情，以便了解每个渠道尝试的具体情况。

#### 验收标准

1. WHEN 管理员点击链路列表中的某一行，THE Trace_View SHALL 在该行下方内联展开显示该请求的完整链路时间线，展开区域顶部显示请求摘要（请求时间、请求模型、用户 Token 名称），再次点击同一行时收起该展开区域
2. THE Trace_View SHALL 通过 RequestId 字段将同一请求的所有日志（类型 2、51、52）聚合为一条链路，并对每个链路步骤显示：渠道 ID、渠道名称、HTTP 状态码、耗时（整数秒，取自 UseTime 字段）、结果状态图标
3. WHEN 链路步骤为 Intercepted_Log（类型 51），THE Trace_View SHALL 显示红色 ❌ 图标和"已拦截"标签
4. WHEN 链路步骤为 Client_Visible_Log（类型 52），THE Trace_View SHALL 显示红色 ❌ 图标和"客户端错误"标签
5. WHEN 链路步骤为 Consume_Log（类型 2），THE Trace_View SHALL 显示绿色 ✅ 图标和"成功"标签，并以系统标准额度格式显示消耗额度（与日志列表页额度列格式一致）
6. THE Trace_View SHALL 按 CreatedAt 时间戳升序从上到下排列链路步骤，使用树形连接线（├── 和 └──）展示同一 RequestId 下各步骤与请求根节点的父子关系
7. IF 某个 RequestId 下查询到的链路步骤数为 0，THEN THE Trace_View SHALL 在展开区域内显示"无链路数据"的空状态提示

### 需求 5：数据库查询兼容性

**用户故事：** 作为开发者，我想确保链路查询功能在所有支持的数据库上正常工作，以便不同部署环境的用户都能使用该功能。

#### 验收标准

1. THE Trace_Service SHALL 使用 GORM 抽象方法（Create、Find、Where、Order、Limit 等）或结合项目已有的跨数据库辅助变量（commonGroupCol、commonTrueVal、commonFalseVal、common.UsingPostgreSQL 等）进行数据库查询，确保兼容 SQLite、MySQL 和 PostgreSQL
2. THE Trace_Service SHALL 利用已有的 `idx_logs_request_id` 索引进行 request_id 查询，无需新增数据库索引或表结构变更
3. WHEN 执行链路列表聚合查询时，THE Trace_Service SHALL 仅使用三种数据库均支持的标准聚合函数（COUNT、SUM、MIN、MAX）和 GROUP BY 子句，且 SELECT 中的非聚合列必须出现在 GROUP BY 中，并使用 commonGroupCol 变量引用保留字列名
4. IF 按 request_id 查询链路时该 request_id 在日志表中不存在，THEN THE Trace_Service SHALL 返回空结果集（空数组），不返回错误

### 需求 6：链路摘要中的额度与 Token 统计

**用户故事：** 作为管理员，我想在链路摘要中看到该请求的总消耗额度和 Token 用量，以便评估重试带来的成本影响。

#### 验收标准

1. WHEN 管理员查看某 request_id 的链路摘要，THE Trace_Service SHALL 返回该 request_id 下所有 Consume_Log（type=2）记录的 Quota 字段之和，作为总消耗额度
2. WHEN 管理员查看某 request_id 的链路摘要，THE Trace_Service SHALL 返回该 request_id 下所有 Consume_Log（type=2）记录的 prompt_tokens 总和与 completion_tokens 总和，分别作为独立字段
3. IF 该 request_id 下不存在任何 Consume_Log（type=2）记录，THEN THE Trace_Service SHALL 返回总消耗额度为 0、prompt_tokens 总和为 0、completion_tokens 总和为 0
4. WHEN Trace_View 显示链路详情中某个成功步骤的消耗额度，THE Trace_View SHALL 将 Quota 值按 1 单位 = $0.000001 的换算比例转换为美元金额，显示精度为小数点后 6 位（例如 "$0.001234"）
5. IF 请求的 request_id 在日志中不存在，THEN THE Trace_Service SHALL 返回空的链路摘要，总消耗额度和 Token 总和均为 0
