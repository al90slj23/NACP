# NACP 运营监控大屏参考源码研读

> 日期：2026-05-17
> 目的：为 NACP 的“NACP统计 / 运营监控大屏”建立源码级参考依据，覆盖用户统计、模型统计、渠道统计、单位/账号监控、容错重试链路、成本与健康度。

---

## 参考源码位置

参考源码统一放在项目根目录：

```text
ReferenceProjects/
├── new-api-latest/           # QuantumNous/new-api 最新版
├── sub2api-latest/           # Wei-Shaw/sub2api 最新版
├── metapi-latest/            # cita-777/metapi 最新版
├── NiceApiManager-latest/    # NiceAIGC/NiceApiManager 最新版
└── one-api-latest/           # songquanpeng/one-api 最新版，作为谱系基线
```

该目录已加入 `.gitignore`，作为本地长期源码参考库，不进入 NACP 业务提交。

当前记录的 commit：

| 项目 | commit |
|---|---|
| new-api-latest | `f69ceb6` |
| sub2api-latest | `f5bd25b` |
| metapi-latest | `c308a3e` |
| NiceApiManager-latest | `5cb4753` |
| one-api-latest | `8df4a26` |

---

## 已发现的根目录完整项目

| 路径 | 性质 | 处理原则 |
|---|---|---|
| `./` | NACP 主项目 | 正常开发 |
| `web/` | NACP 前端子项目 | 正常开发 |
| `electron/` | NACP Electron 子项目 | 按现有结构保留 |
| `NewAPIv0.13.2/` | NewAPI v0.13.2 基线源码 | 保留作为固定版本对照 |
| `ZERO/` | ZERO 框架规范源码 | 保留在根目录 |
| `ReferenceProjects/` | 外部参考源码库 | 只读学习，不混入业务代码 |

---

## 各项目值得学习的部分

### NewAPI 最新版

关键文件：

- `model/usedata.go`
- `controller/usedata.go`
- `model/usedata_rankings.go`
- `controller/rankings.go`
- `model/log.go`

特点：

- 用 `quota_data` 做小时级聚合。
- 聚合维度主要是：`user_id`、`username`、`model_name`、`created_at`、`token_used`、`quota`、`count`。
- 提供基础模型用量、用户用量、排行榜能力。
- 适合参考“NewAPI 原生统计基础”，但不足以支撑 NACP 运营大屏。

NACP 吸收点：

- 保留“小时桶 + 用户 + 模型”的基础用量统计思路。
- 排行榜可以参考其 `model_name + token_used` 的简单模型。

NACP 不应照搬：

- 不应只做 `quota_data` 单表聚合，因为缺少渠道、单位、账号、容错重试、平台成本、失败原因、延迟、SLA 等维度。

### OneAPI

关键文件：

- `controller/log.go`
- `model/log.go`
- `monitor/channel.go`
- `monitor/metric.go`
- `monitor/manage.go`

特点：

- 代表更早期的 API 网关统计/日志/渠道监控模型。
- 监控逻辑更轻，重点在错误后禁用渠道、日志查询、基础统计。

NACP 吸收点：

- 用作“原始谱系基线”，判断哪些行为是 NewAPI 继承来的，哪些是后续增强。

NACP 不应照搬：

- 监控维度太粗，不能满足单位、模型、用户、容错链路、成本分摊。

### Sub2API

关键文件：

- `backend/internal/pkg/usagestats/usage_log_types.go`
- `backend/internal/pkg/usagestats/account_stats.go`
- `backend/internal/service/dashboard_service.go`
- `backend/internal/service/ops_dashboard.go`
- `backend/internal/service/ops_dashboard_models.go`
- `backend/internal/service/ops_realtime_traffic.go`
- `backend/internal/service/ops_metrics_collector.go`
- `backend/internal/repository/ops_repo_*.go`
- `backend/internal/repository/dashboard_aggregation_repo.go`
- `frontend/src/views/admin/ops/OpsDashboard.vue`
- `frontend/src/api/admin/ops.ts`

特点：

- 运营监控体系最完整。
- 同时覆盖 Dashboard 和 Ops 两套统计：
  - Dashboard：用户、API key、账号、请求量、token、成本、今日/累计、RPM/TPM。
  - Ops：SLA、错误率、上游错误率、429/529、QPS/TPS、延迟分位数、系统指标、后台任务心跳、告警。
- 支持 `raw / preagg / auto` 查询模式。
- 有缓存新鲜度：fresh TTL、cache TTL、聚合水位、stale 标记。
- 支持用户消费排行、用户趋势、API Key 趋势、模型统计、分组统计、账号统计。

NACP 吸收点：

- 运营大屏的主框架应参考 Sub2API：
  - 顶部健康分数。
  - SLA / 成功率 / 错误率 / 上游错误率。
  - QPS / TPS / RPM / TPM。
  - 延迟分位数 P50/P90/P95/P99。
  - 用户统计、模型统计、分组统计、API key 统计。
  - 模型统计需要支持 `requested/upstream/mapping` 三种模型来源；NACP 当前日志只有 `model_name`，后续如果要做上游模型和映射模型，需要在日志结构中继续补字段。
  - 排行不应只停留在模型、用户、渠道，还要覆盖分组、令牌、端点、IP 等维度，方便定位某个分组、某个令牌或某类入口路径造成的异常。
  - 系统任务心跳和聚合水位。
  - 错误分布、错误趋势、请求详情弹窗。
- API 设计应保留 `query_mode=auto|raw|preagg`，便于第一版先 raw，后续平滑切预聚合。

NACP 不应照搬：

- Sub2API 是 Ent/Vue 架构，NACP 是 GORM/React/Semi UI。
- 不应直接复制 SQL / ORM 结构，应按 NACP 的 `logs`、`channels`、`tokens`、`users`、`units`、`unit_accounts`、SFT trace 字段重建。

### Metapi

关键文件：

- `drizzle/0023_usage_aggregates.sql`
- `src/server/services/usageAggregationService.ts`
- `src/server/services/dashboardSnapshotService.ts`
- `src/server/services/siteStatsSnapshotService.ts`
- `src/server/routes/api/stats.ts`
- `src/web/pages/Dashboard.tsx`
- `src/web/pages/ProxyLogs.tsx`
- `src/web/pages/TokenRoutes.tsx`

特点：

- 聚合层设计非常值得学习。
- 使用 `analytics_projection_checkpoints` 记录投影器水位。
- 从 `proxy_logs` 增量投影到：
  - `site_day_usage`
  - `site_hour_usage`
  - `model_day_usage`
- 支持：
  - lease 防止多实例重复聚合。
  - checkpoint 水位。
  - recompute_from_id 触发重算。
  - 快照缓存。
- 前端注重站点可用性、站点分布、模型分析、日志跳转。

NACP 吸收点：

- NACP 后续预聚合层应参考 Metapi 的“投影器 + 水位 + 可重算”设计。
- 对 NACP 来说，这比直接每次扫 `logs` 表更稳：
  - 第一版 raw 查询。
  - 第二版增量投影。
  - 第三版支持从某个 `log_id` 重算。
- `site` 概念可映射到 NACP 的 `unit`。
- `proxy-logs` 的 meta/query/full 三层返回方式值得吸收：NACP 后续日志页和统计页都应区分“只取筛选元数据”“只取摘要”“取完整明细”，避免页面每次都拉重数据。
- `model marketplace / model-by-site / token-candidates` 的思想可以映射为 NACP 的“模型覆盖矩阵”和“模型缺口检测”：哪些模型没有可用渠道、哪些模型只有单一单位覆盖、哪些分组缺模型。
- 已落地第一版“模型覆盖与缺口矩阵”：以后端 `abilities + channels + logs` 聚合为准，展示模型覆盖分组、启用渠道、健康渠道、降级渠道、停用渠道、最近真实请求、错误、Token、额度、平均耗时和覆盖风险；前端只展示后端结果，不从普通排行里二次猜测覆盖关系。

NACP 不应照搬：

- Metapi 是 TypeScript/Drizzle，NACP 要用 GORM 并兼容 SQLite/MySQL/PostgreSQL。
- Metapi 聚合维度偏 `site/account/proxy_log`，NACP 需要额外加入用户、令牌、渠道、分组、SFT trace。

### NiceApiManager

关键文件：

- `app/services/dashboard_service.py`
- `app/schemas/dashboard.py`
- `app/models/daily_usage_stat.py`
- `app/models/instance.py`
- `web/src/pages/DashboardPage.tsx`
- `web/src/api/dashboard.ts`

特点：

- 管理对象是 `Instance`，很接近 NACP 的“单位/账号”。
- Dashboard 聚合重点是：
  - 实例数量。
  - 启用实例数量。
  - 健康/异常实例数量。
  - 预付费/后付费。
  - 总额度、已用额度、今日请求数。
  - 实例趋势和实例拆分。
- 前端提供按标签、计费模式、启用状态、健康状态过滤。

NACP 吸收点：

- NACP统计中的“单位/账号余额、健康、同步状态、趋势”可以参考它。
- NACP 单位管理可以加入：
  - 单位标签。
  - 单位健康状态。
  - 余额同步时间。
  - 预付费/后付费。
  - 预计可用天数。
- Dashboard 过滤器要能按单位标签、计费模式、启用状态、健康状态做筛选；NACP 当前先落单位/账号统计和健康状态，后续需要给 `units` 补标签与计费模式字段。

NACP 不应照搬：

- 它更偏资源实例管理，不是 API 网关完整运营监控。
- 用户统计、模型统计、容错链路统计不足。

---

## NACP 运营监控大屏目标

当前“NACP统计”不应只是单位账号详情页。建议最终拆成：

1. `单位管理`
   - 维护单位、账号、账号密钥、账号余额、账号探测详情。
   - 当前已经做的账号详情、JSON、余额探测结果，应并入这里。

2. `运营监控大屏`
   - 面向实时运营。
   - 统计用户、模型、渠道、单位、账号、分组、令牌、容错链路、成本、健康度。

---

## 大屏模块树

```text
运营监控大屏
├── 全局态势
│   ├── 健康分数
│   ├── 请求总数 / 成功 / 失败
│   ├── SLA / 错误率 / 上游错误率
│   ├── RPM / TPM / QPS / TPS
│   ├── 平均耗时 / P50 / P90 / P95 / P99
│   ├── 用户扣费
│   ├── 平台成本
│   └── 毛利估算
├── 用户统计
│   ├── 总用户
│   ├── 今日新增
│   ├── 今日活跃
│   ├── 小时活跃
│   ├── Top 用户消费排行
│   ├── Top 用户请求排行
│   ├── 用户失败率排行
│   └── 用户异常增长
├── 模型统计
│   ├── 请求模型排行
│   ├── 上游模型排行（需要日志补字段）
│   ├── 映射模型排行（需要日志补字段）
│   ├── 模型请求 / Token / 额度趋势
│   ├── 模型成功率 / 错误率 / 流式占比
│   ├── 模型 P50 / P90 / P95 / P99（需要延迟分位聚合）
│   ├── 模型-渠道覆盖矩阵（已落第一版）
│   └── 模型缺口检测（已落第一版）
├── 分组与令牌统计
│   ├── 分组请求排行
│   ├── 分组成本/收入排行
│   ├── 令牌请求排行
│   ├── 令牌失败率排行
│   ├── 端点路径排行
│   └── IP 来源排行
├── 单位与账号
│   ├── 单位总数 / 启用单位
│   ├── 账号总数 / 正常 / 异常 / 限流 / 禁用
│   ├── 余额汇总
│   ├── 今日消耗
│   ├── 预计可用天数
│   ├── 单位健康趋势
│   └── 余额低告警
├── 渠道统计
│   ├── 渠道成功率
│   ├── 渠道失败率
│   ├── 渠道平均耗时
│   ├── 分组/优先级/权重命中情况
│   └── 渠道错误码分布
├── SFT 容错重试统计
│   ├── 2 正常消费成功
│   ├── 20 容错重试后成功
│   ├── 50 容错重试后失败
│   ├── 21 容错重试成功步骤
│   ├── 51 容错重试已拦截步骤
│   ├── 52 容错重试最终失败步骤
│   ├── 29 容错探测成功
│   ├── 59 容错探测失败
│   ├── 平均尝试渠道数
│   ├── 最大尝试渠道数
│   └── 最常见失败链路
├── 分组与令牌
│   ├── 分组请求量
│   ├── 分组成本
│   ├── 分组失败率
│   ├── API Key 请求量
│   └── API Key 消费排行
├── 实时事件流
│   ├── 最近成功请求
│   ├── 最近失败请求
│   ├── 最近容错链路
│   └── 最近告警事件
└── 系统与聚合状态
    ├── 数据库连接状态
    ├── Redis 状态
    ├── 后台任务心跳
    ├── 聚合水位
    ├── 聚合延迟
    └── 统计数据 stale 标记
```

---

## 推荐 API 结构

第一阶段使用 raw logs 查询，第二阶段增加预聚合，接口先保持可扩展：

```text
GET /api/ops/overview
GET /api/ops/users
GET /api/ops/models
GET /api/ops/units
GET /api/ops/accounts
GET /api/ops/channels
GET /api/ops/groups
GET /api/ops/tokens
GET /api/ops/sft
GET /api/ops/errors
GET /api/ops/realtime
GET /api/ops/aggregation/status
```

通用查询参数：

```text
start_time
end_time
time_range=5m|30m|1h|6h|24h|7d|30d|custom
query_mode=auto|raw|preagg
group
model
user_id
token_id
channel_id
unit_id
unit_account_id
trace_id
```

---

## 预聚合表建议

第一版不强制上预聚合。等 raw 查询验证指标定义正确后，再增加。

建议表：

```text
ops_aggregation_checkpoints
ops_usage_hourly
ops_usage_daily
ops_model_hourly
ops_model_daily
ops_user_hourly
ops_user_daily
ops_channel_hourly
ops_channel_daily
ops_unit_hourly
ops_unit_daily
ops_sft_hourly
ops_sft_daily
ops_error_hourly
ops_realtime_windows
ops_job_heartbeats
ops_alert_events
```

关键原则：

- 原始 `logs` 永远是事实来源。
- 聚合表是可删除、可重算的派生数据。
- 聚合表必须记录水位，至少包含 `last_log_id` 和 `watermark_created_at`。
- 支持从某个 `log_id` 触发重算。
- 查询默认 `auto`：短时间 raw，长时间 preagg，preagg 不可用时 fallback raw。

---

## NACP 特有增强

NACP 不能只学习外部项目，必须加入自己的 SFT 能力：

1. `logs` 中 20/50 是最终用户可见摘要。
2. 21/51/52/29/59 是链路步骤。
3. 统计时要区分：
   - 用户计费：2、20。
   - 用户失败：50。
   - 平台运营成本：29/59 探测、51 拦截、52 最终失败是否实际打到上游。
4. 用户统计、模型统计、渠道统计都要能回答：
   - 直接成功多少？
   - 容错后成功多少？
   - 容错后失败多少？
   - 因哪个渠道失败？
   - 最终切到哪个渠道成功？
   - 哪些模型最容易触发容错？

---

## 实施建议

### P0：先完成页面定位调整

- `单位管理`：承载单位、账号、余额探测详情。
- `NACP统计 / 运营监控`：改成大屏，不再只展示账号详情。
- 第一版固定四个顶部主题页：
  - `综合`：全局态势、请求质量、消耗、SFT、Top 摘要。
  - `模型&性能`：模型活动、请求/成功/错误趋势、平均耗时、流式占比。
  - `用户&消耗`：活跃用户、Top 用户、Token 趋势、额度趋势、用户可见错误。
  - `单位&渠道`：统计源、余额、采集能力、渠道健康、探测趋势。

### P1：先做 raw 查询版大屏

- 不先建复杂聚合表。
- 直接从当前 logs、users、tokens、channels、units、unit_accounts 统计。
- 先验证指标定义、展示结构、筛选逻辑。
- 趋势先由 `/api/nacp_stats/overview` 返回 raw logs 分桶结果；后续再迁移到小时级预聚合，不改变前端主题页结构。

### P2：补模型统计、用户统计

- 用户统计：
  - 请求数、tokens、扣费、失败率、SFT 触发率。
- 模型统计：
  - requested/upstream/mapping 三种口径。
  - 输入/输出/cache tokens。
  - 成本、毛利、失败率、延迟。

### P3：补 SFT 专属统计

- 20/50 摘要数。
- 21/51/52/29/59 步骤数。
- trace 平均长度。
- 失败链路排行。
- 渠道切换成功率。

### P4：加预聚合

- 参考 Metapi 的投影器设计。
- 参考 Sub2API 的 query_mode / stale 标记。
- 保证 SQLite/MySQL/PostgreSQL 兼容。

### P5：加告警

- 单位余额过低。
- 渠道失败率突增。
- 模型失败率突增。
- SFT 50 比例突增。
- 聚合水位落后。
- 日志写入异常。

---

## 当前结论

如果目标是“把 NACP 做厉害”，不能只做 NewAPI 式基础数据看板。

推荐组合：

- **NewAPI / OneAPI**：作为原生日志、用量、排行榜基线。
- **Sub2API**：作为运营大屏、SLA、错误、延迟、用户/模型统计主参考。
- **Metapi**：作为预聚合投影器、水位、可重算架构主参考。
- **NiceApiManager**：作为单位/账号余额、健康状态、趋势和实例筛选主参考。

NACP 的最终优势应该是：

- 既有 API 网关基础统计；
- 又有单位/账号运营视角；
- 还有 SFT 容错链路独有统计；
- 同时能把用户扣费、平台成本、模型消耗、渠道健康全部串起来。
