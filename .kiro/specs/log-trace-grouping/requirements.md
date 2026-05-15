# Requirements Document

## Introduction

改进现有日志页面（`/console/log`），将平铺显示的日志按 `request_id` 分组折叠展示为链路视图。核心变更包括：

1. **后端修复**：补全同渠道重试（same-channel retry）的 type=51 日志记录缺失问题
2. **后端新增**：将并行探测（probe/pre-warm）结果写入日志表（type=29 探测成功、type=59 探测失败）
3. **后端新增**：分组日志列表接口，返回混合数据（虚拟摘要行 + 普通行）
4. **前端改造**：日志列表支持分组折叠展示，摘要行可展开查看完整链路步骤

### 日志类型体系

| 类型 | 含义 | 存储 | 列表显示 |
|------|------|------|---------|
| 2 | 正常消费（无重试直接成功） | 真实记录 | 列表中独立一行 |
| 20 | 重试后成功的链路摘要 | **虚拟**（后端聚合生成） | 列表中一行，可展开 |
| 50 | 重试后失败的链路摘要 | **虚拟**（后端聚合生成） | 列表中一行，可展开 |
| 5 | 旧系统遗留错误日志 | 真实记录（历史数据） | 列表中独立一行，不展开 |
| 21 | 链路内的成功消费步骤 | 真实记录（数据库中为 type=2） | 仅在展开 20 时显示 |
| 51 | 拦截错误（客户端未感知） | 真实记录 | 仅在展开 20/50 时显示 |
| 52 | 客户可见错误（所有重试失败） | 真实记录 | 仅在展开 50 时显示 |
| 29 | 探测成功（轻量 probe 请求成功） | 真实记录 | 仅在展开 20/50 时显示 |
| 59 | 探测失败（轻量 probe 请求失败） | 真实记录 | 仅在展开 20/50 时显示 |

### 链路模式

```
模式 A — 无重试直接成功：
  列表显示：type=2（一行）

模式 B — 重试后成功：
  列表显示：type=20（摘要行，可展开）
  展开后：
    ├── 51  拦截错误（1~N 条，含同渠道重试失败）
    ├── 29  探测成功（0~N 条）
    ├── 59  探测失败（0~N 条）
    └── 21  成功消费（1 条，实际为 type=2）

模式 C — 重试后失败（有拦截）：
  列表显示：type=50（摘要行，可展开）
  展开后：
    ├── 51  拦截错误（1~N 条）
    ├── 29  探测成功（0~N 条）
    ├── 59  探测失败（0~N 条）
    └── 52  客户可见错误（1 条）

模式 D — 直接失败（无拦截，如流式中断）：
  列表显示：type=50（摘要行，可展开）
  展开后：
    └── 52  客户可见错误（1 条）

模式 E — 旧系统遗留：
  列表显示：type=5（一行，不展开）
```

## Glossary

- **Log_Page**：现有日志页面组件（路由 `/console/log`），包含 UsageLogsTable 及其筛选、分页功能
- **Grouped_Log_API**：后端新增的分组日志列表接口，返回按 request_id 分组后的混合数据
- **Trace_Summary_Row**：链路摘要行（虚拟），代表一次有重试活动的请求。type=20 表示最终成功，type=50 表示最终失败
- **Normal_Log_Row**：无重试的普通日志行（type=2 消费、type=5 旧错误等），直接显示为单行
- **Trace_Step**：链路内的单个步骤，展开摘要行后可见，包括 type=51/52/21/29/59
- **Channel_Path**：渠道路径字符串，表示请求经过的渠道序列（如 "12→14"）
- **Expand_Area**：摘要行展开后的内联区域，以树形连接线展示链路步骤详情
- **Request_Id**：每次 API 请求的唯一标识符，存储在 Log 表的 `request_id` 字段中
- **Trace_Detail_API**：已有的 `GET /api/log/trace` 接口，按 request_id 查询单条请求的完整链路详情
- **Same_Channel_Retry**：同渠道重试，对同一个渠道连续发送多次请求（A1、A2、A3、A4）
- **Probe**：轻量探测请求（max_tokens=1），用于预热顺位渠道、验证渠道健康状态
- **Pre_Warm**：并行预热，在重试主渠道的同时对顺位渠道发送探测请求

## Requirements

### 需求 1：修复同渠道重试的日志记录缺失

**用户故事：** 作为管理员，我想看到每次同渠道重试的失败记录，以便完整了解请求在单个渠道上的重试情况。

#### 验收标准

1. WHEN 同渠道重试（Same_Channel_Retry）中某次请求失败时，THE Relay_Controller SHALL 为该次失败记录一条 type=51（ErrorIntercepted）日志，包含 channel_id、model_name、use_time、error 信息、HTTP 状态码
2. THE Relay_Controller SHALL 确保同渠道重试产生的每条 type=51 日志都携带与主请求相同的 request_id，以便后续按 request_id 聚合
3. THE Relay_Controller SHALL 在同渠道重试的 type=51 日志的 Other 字段中标记 `"retry_type": "same_channel"`，以区分同渠道重试和跨渠道重试
4. IF 同渠道重试中某次请求成功，THEN THE Relay_Controller SHALL 不记录该次的错误日志，仅记录最终的 type=2 消费日志

### 需求 2：探测记录写入日志表

**用户故事：** 作为管理员，我想看到每次探测（probe）的结果记录，以便了解探测产生的运营成本和渠道健康状况。

#### 验收标准

1. WHEN 并行预热（Pre_Warm）过程中对某个渠道发送探测请求并成功（HTTP 2xx）时，THE Probe_Service SHALL 在日志表中记录一条 type=29 日志，包含 channel_id、channel_name、model_name、耗时（use_time，毫秒转秒取整）、HTTP 状态码
2. WHEN 并行预热过程中对某个渠道发送探测请求并失败（非 HTTP 2xx 或超时）时，THE Probe_Service SHALL 在日志表中记录一条 type=59 日志，包含 channel_id、channel_name、model_name、耗时、HTTP 状态码（超时时为 0）、错误信息
3. THE Probe_Service SHALL 确保探测日志的 request_id 与触发该探测的用户请求的 request_id 相同，以便在链路展开时关联显示
4. THE Probe_Service SHALL 将探测日志的 user_id 设为 0，quota 设为 0，以确保探测成本不计入任何用户账单
5. THE Probe_Service SHALL 在探测日志的 Other 字段中记录 `"probe_trigger": "pre_warm"` 标记，以区分预热探测和定时降级探测
6. IF 探测是由定时降级探测循环（StartDegradedProbeLoop）触发的，THEN THE Probe_Service SHALL 记录探测日志但 request_id 为空字符串（因为不关联任何用户请求）

### 需求 3：后端分组日志列表接口

**用户故事：** 作为前端开发者，我想通过一个接口获取已按 request_id 分组的日志数据，以便在日志列表中直接渲染折叠视图。

#### 验收标准

1. WHEN 管理员调用 Grouped_Log_API 并提供分页参数（page、page_size），THE Grouped_Log_API SHALL 返回混合列表，其中有重试活动的请求以 Trace_Summary_Row 形式出现（type=20 或 type=50），无重试的正常请求以 Normal_Log_Row 形式出现（type=2），旧系统错误以 Normal_Log_Row 形式出现（type=5），整体按时间倒序排列
2. THE Grouped_Log_API SHALL 对每条 Trace_Summary_Row 返回以下字段：request_id、type（20 或 50）、最早请求时间（created_at）、模型名称（model_name）、用户名（username）、Token 名称（token_name）、Channel_Path（渠道路径字符串）、总耗时（total_duration，单位秒）、总消耗额度（total_quota，仅统计 type=2 的 quota）、总 prompt_tokens、总 completion_tokens、步骤数量（step_count，包含 51/52/2/29/59 的总条数）
3. THE Grouped_Log_API SHALL 判定 Trace_Summary_Row 的 type 值：IF 该 request_id 下存在 type=2 的日志记录，THEN type=20（成功）；IF 不存在 type=2 的日志记录，THEN type=50（失败）
4. THE Grouped_Log_API SHALL 判定"有重试活动"的条件为：同一 request_id 下存在 type=51 或 type=52 或 type=29 或 type=59 的日志记录
5. THE Grouped_Log_API SHALL 对 Normal_Log_Row 返回与现有 GET /api/log/ 接口相同的字段结构，确保前端可以复用现有列定义渲染
6. THE Grouped_Log_API SHALL 支持现有日志列表的所有筛选条件：时间范围（start_timestamp、end_timestamp）、日志类型（type）、模型名称（model_name）、用户名（username）、Token 名称（token_name）、渠道 ID（channel）、分组（group）、request_id
7. IF 筛选条件中指定了具体的 request_id，THEN THE Grouped_Log_API SHALL 返回该 request_id 下的所有日志记录（平铺模式，不折叠），以便查看单个请求的完整详情
8. IF 筛选条件中指定了 type=2，THEN THE Grouped_Log_API SHALL 仅返回 Normal_Log_Row（无重试的 type=2 日志），不返回 Trace_Summary_Row
9. IF 筛选条件中指定了 type=51 或 type=52，THEN THE Grouped_Log_API SHALL 仅返回包含该类型日志的 Trace_Summary_Row，不返回 Normal_Log_Row

### 需求 4：链路摘要行的渠道路径生成

**用户故事：** 作为管理员，我想在折叠的摘要行中看到请求经过的渠道路径（如 "12→14"），以便快速了解重试路径。

#### 验收标准

1. WHEN 生成 Trace_Summary_Row 的 Channel_Path 字段时，THE Grouped_Log_API SHALL 按链路步骤（type=51/52/2，不含 29/59 探测记录）的 created_at 时间升序提取各步骤的 channel_id，去除连续重复后以 "→" 连接（如步骤依次经过渠道 12、12、14，则 Channel_Path 为 "12→14"）
2. IF 链路中仅有一个渠道（所有非探测步骤 channel_id 相同），THEN THE Grouped_Log_API SHALL 返回该单个 channel_id 作为 Channel_Path（如 "12"）
3. THE Grouped_Log_API SHALL 在 Channel_Path 中使用 channel_id 数字而非渠道名称，以保持简洁

### 需求 5：前端日志列表分组折叠展示

**用户故事：** 作为管理员，我想在日志列表中看到按 request_id 分组折叠的视图，以便减少重试日志的视觉噪音并快速定位问题请求。

#### 验收标准

1. WHEN Log_Page 加载日志数据时，THE Log_Page SHALL 调用 Grouped_Log_API 获取分组后的混合列表，替代原有的平铺日志列表
2. WHEN 列表中出现 type=20 的 Trace_Summary_Row 时，THE Log_Page SHALL 以绿色 Tag 显示"成功(重试)"类型标签，并在渠道列显示 Channel_Path
3. WHEN 列表中出现 type=50 的 Trace_Summary_Row 时，THE Log_Page SHALL 以红色 Tag 显示"失败(重试)"类型标签，并在渠道列显示 Channel_Path
4. WHEN 列表中出现 Normal_Log_Row（type=2 或 type=5）时，THE Log_Page SHALL 以现有样式渲染该行，保持与当前日志列表完全一致的外观
5. THE Log_Page SHALL 在 Trace_Summary_Row 行显示可展开指示器（展开箭头图标），Normal_Log_Row 保持现有的展开行为（展开显示 Descriptions 详情）
6. WHEN 管理员点击 Trace_Summary_Row 的展开指示器或行本身时，THE Log_Page SHALL 调用 Trace_Detail_API 获取该 request_id 的链路详情，并在该行下方内联展开显示链路步骤时间线
7. THE Log_Page SHALL 在展开区域使用树形连接线样式展示链路步骤，非末尾步骤使用 "├──" 前缀，末尾步骤使用 "└──" 前缀
8. WHEN 链路步骤为 type=51（拦截错误），THE Log_Page SHALL 显示红色 ❌ 图标、"已拦截" 标签、渠道 ID 和名称、HTTP 状态码、耗时（秒）
9. WHEN 链路步骤为 type=52（客户可见错误），THE Log_Page SHALL 显示红色 ❌ 图标、"客户端错误" 标签、渠道 ID 和名称、HTTP 状态码、耗时（秒）
10. WHEN 链路步骤为 type=2/21（成功消费），THE Log_Page SHALL 显示绿色 ✅ 图标、"成功" 标签、渠道 ID 和名称、HTTP 状态码、耗时（秒）、消耗额度（使用系统标准 renderQuota 格式）
11. WHEN 链路步骤为 type=29（探测成功），THE Log_Page SHALL 显示蓝色 🔍 图标、"探测成功" 标签、渠道 ID 和名称、HTTP 状态码、耗时（秒）
12. WHEN 链路步骤为 type=59（探测失败），THE Log_Page SHALL 显示灰色 🔍 图标、"探测失败" 标签、渠道 ID 和名称、HTTP 状态码、耗时（秒）
13. WHILE 链路详情数据正在加载，THE Log_Page SHALL 在展开区域显示 Spin 加载指示器
14. IF 链路详情返回的步骤数为 0，THEN THE Log_Page SHALL 在展开区域显示"无链路数据"的空状态提示

### 需求 6：Trace_Summary_Row 的摘要信息展示

**用户故事：** 作为管理员，我想在折叠的摘要行中看到该请求的关键信息，以便无需展开即可快速判断请求状态。

#### 验收标准

1. THE Log_Page SHALL 在 Trace_Summary_Row 的时间列显示最早请求时间，格式与现有日志列表一致（YYYY-MM-DD HH:mm:ss）
2. THE Log_Page SHALL 在 Trace_Summary_Row 的模型列显示请求的模型名称，使用现有 renderModelTag 组件渲染
3. THE Log_Page SHALL 在 Trace_Summary_Row 的用户列显示用户名和 Avatar，使用现有样式渲染
4. THE Log_Page SHALL 在 Trace_Summary_Row 的花费列显示总消耗额度（仅 type=2 的 quota 之和），使用现有 renderQuota 函数格式化
5. THE Log_Page SHALL 在 Trace_Summary_Row 的用时列显示总耗时（total_duration 秒），使用现有 renderUseTime 组件渲染
6. THE Log_Page SHALL 在 Trace_Summary_Row 的渠道列显示 Channel_Path（如 "12→14"），使用带颜色的 Tag 组件渲染各渠道 ID，中间以 "→" 连接
7. THE Log_Page SHALL 在 Trace_Summary_Row 的输入/输出列显示总 prompt_tokens 和总 completion_tokens

### 需求 7：分组模式与现有功能的兼容性

**用户故事：** 作为管理员，我想在分组视图下仍然能使用所有现有的筛选、分页和列配置功能。

#### 验收标准

1. THE Log_Page SHALL 保持现有的所有筛选条件（时间范围、类型、模型名称、用户名、Token 名称、渠道、分组、request_id）在分组模式下正常工作
2. THE Log_Page SHALL 保持现有的分页功能在分组模式下正常工作，每页条数（10/20/50/100）和页码切换行为不变
3. THE Log_Page SHALL 保持现有的列可见性配置功能在分组模式下正常工作，Trace_Summary_Row 和 Normal_Log_Row 共用相同的列配置
4. WHEN 筛选条件中指定了 request_id 时，THE Log_Page SHALL 切换为平铺模式（不折叠），直接显示该 request_id 下的所有日志
5. THE Log_Page SHALL 保持现有的紧凑模式（compactMode）在分组模式下正常工作

### 需求 8：Trace_Detail_API 扩展支持新日志类型

**用户故事：** 作为前端开发者，我想在调用链路详情接口时获取到探测记录（type=29/59），以便在展开视图中完整展示链路步骤。

#### 验收标准

1. WHEN 调用 Trace_Detail_API 查询某 request_id 的链路详情时，THE Trace_Detail_API SHALL 返回该 request_id 下所有类型为 2、51、52、29、59 的日志记录，按 created_at 升序排列
2. THE Trace_Detail_API SHALL 对 type=29 和 type=59 的步骤返回与其他步骤相同的字段结构（id、channel_id、channel_name、type、status_code、use_time、model_name、quota、created_at）
3. THE Trace_Detail_API SHALL 在返回的步骤中保留 Other 字段中的 `retry_type` 和 `probe_trigger` 标记，以便前端区分步骤来源

### 需求 9：数据库兼容性与性能

**用户故事：** 作为开发者，我想确保所有新增查询在三种数据库上正常工作且性能可接受。

#### 验收标准

1. THE Grouped_Log_API SHALL 使用 GORM 抽象方法或结合项目已有的跨数据库辅助变量（commonGroupCol、commonTrueVal、commonFalseVal）进行数据库查询，确保兼容 SQLite、MySQL 和 PostgreSQL
2. THE Grouped_Log_API SHALL 利用已有的 `idx_logs_request_id` 索引和 `idx_created_at_id` 索引进行查询，无需新增数据库索引或表结构变更
3. THE Grouped_Log_API SHALL 仅使用三种数据库均支持的标准 SQL 功能（COUNT、SUM、MIN、MAX、GROUP BY、CASE WHEN、子查询）
4. THE Grouped_Log_API SHALL 在单次查询中完成分组聚合，避免 N+1 查询问题
5. THE Grouped_Log_API SHALL 将链路步骤详情的查询延迟到用户展开摘要行时（通过 Trace_Detail_API 按需加载），列表接口仅返回摘要信息
6. THE Probe_Service SHALL 使用异步写入（gopool.Go）记录探测日志，避免阻塞用户请求的重试流程

### 需求 10：新日志类型常量定义

**用户故事：** 作为开发者，我想在代码中有清晰的常量定义来表示新的日志类型，以便代码可读性和维护性。

#### 验收标准

1. THE Log_Model SHALL 新增以下常量定义：LogTypeProbeSuccess = 29（探测成功）、LogTypeProbeFailed = 59（探测失败）
2. THE Log_Model SHALL 保持现有常量不变：LogTypeConsume = 2、LogTypeError = 5、LogTypeErrorIntercepted = 51、LogTypeErrorClientVisible = 52
3. THE Log_Model SHALL 不新增 type=20、type=21、type=50 的常量，因为这些是后端聚合生成的虚拟类型，不存储在数据库中

