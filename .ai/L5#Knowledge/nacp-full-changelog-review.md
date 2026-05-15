# NACP 完整变更清单 — 供检测与测试计划使用

> **基线版本**：QuantumNous/new-api v0.13.2 (commit bee339d)
> **当前版本**：classic-plus-0.2.0-dev
> **整理日期**：2026-05-15
> **用途**：交给 Claude Opus 4.7 进行完整检测和测试计划制定

---

## 一、项目定位

NACP (NewAPI Classic Plus) 是基于 NewAPI v0.13.2 classic 版本线的增强分支，专注于核心中继能力增强。不跟 upstream v1 新 UI/新路线混合。

**核心目标**：上游供应商质量不一时，客户端不再看到报错，最多感知延迟稍高。

---

## 二、已完成功能（已部署/已实现）

### v0.1.0 — 智能重试与渠道健康管理 ✅

#### 2.1 渠道健康状态机

**五状态模型**：Healthy → Probing → Degraded → Recovering → Healthy + ManuallyDisabled

| 转换 | 触发条件 |
|------|---------|
| Healthy → Probing | 用户请求返回非 400/408/504/524 错误 |
| Probing → Healthy | 探测成功 |
| Probing → Degraded | 连续探测失败 ≥ 2 次 |
| Degraded → Recovering | 周期探测连续成功 ≥ 3 次 |
| Recovering → Healthy | 观察期（10 分钟）无错误 |
| Recovering → Degraded | 观察期内任何错误 |
| Any → Degraded | 401 + 禁用关键词（立即降级） |
| ManuallyDisabled | 不做任何自动转换 |

**新增文件**：
- `service/channel_health.go` (~280 行) — 状态机核心
- `service/channel_health_config.go` (~70 行) — 配置/阈值
- `service/channel_probe.go` (~230 行) — 轻量探测 + 并行预热

**数据库变更**（channels 表新增 4 字段，GORM AutoMigrate）：
- `health_status` VARCHAR(16) DEFAULT 'healthy'
- `health_updated_at` BIGINT DEFAULT 0
- `health_fail_count` INT DEFAULT 0
- `health_success_count` INT DEFAULT 0

**配置参数**（硬编码默认值）：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| SameChannelRetryCount | 2 | 同渠道重试次数 |
| ProbeFailThreshold | 2 | 连续探测失败几次确认降级 |
| DegradedProbeInterval | 5 分钟 | 降级渠道周期性探测间隔 |
| RecoverySuccessThreshold | 3 | 连续探测成功几次进入恢复期 |
| RecoveryObservationPeriod | 10 分钟 | 恢复观察期时长 |
| ProbeTimeout | 3 秒 | 单次探测超时 |
| MaxRetrySamePriority | 3 | 同优先级最多尝试几个渠道 |
| PreWarmChannelCount | 2 | 并行预热几个顺位渠道 |

#### 2.2 同渠道重试

- 发现错误后重试同渠道 2 次（应对偶发性错误）
- 400 错误直接切换渠道（参数问题，不是渠道问题）
- 401 + 禁用关键词直接降级（永久性问题）
- 504/524/408 不重试（超时类）

#### 2.3 并行预热

- 重试同渠道的同时，并行探测顺位渠道
- 最小请求体（max_tokens=1），3 秒超时
- 同渠道重试失败后，直接切到已验证健康的顺位渠道（零等待）

#### 2.4 同优先级渠道切换

- 同优先级内最多尝试 3 个渠道后才降级
- 排除已试过的渠道 ID
- 用完同优先级才降到低优先级

#### 2.5 渠道亲和性健康感知

- Degraded 渠道自动跳过亲和绑定
- 解决"开亲和后渠道故障持续报错"的痛点
- SwitchOnSuccess 自动更新亲和到新渠道
- SkipRetryOnFailure 与健康感知互补（健康感知在选渠道阶段生效，SkipRetry 在请求失败后生效）

#### 2.6 低渠道告警 + 安全兜底

- 可用渠道 ≤ 2 时 warning，全部降级时 critical
- 所有渠道被健康过滤后回退到不过滤（防止状态机 bug 导致服务中断）

#### 2.7 后台恢复探测

- 定时探测 Degraded 渠道（5 分钟间隔）
- 连续 3 次成功进入恢复期
- 仅 Master 节点运行

**修改的现有文件**：
- `common/constants.go` — +10 行（健康状态常量 + SameChannelRetryCount）
- `model/channel.go` — +5 行（Channel struct 新增 4 个健康字段）
- `model/channel_cache.go` — +30 行（GetRandomSatisfiedChannel 健康过滤 + 排除 + 兜底）
- `controller/relay.go` — +80 行（增强重试循环 + 并行预热 + processChannelError 集成状态机）
- `middleware/distributor.go` — +4 行（亲和性健康感知）
- `main.go` — +3 行（启动时初始化健康管理）

**集成测试结果**：12/12 全部通过（2026-05-14，生产服务器 143.198.87.200）

---

### v0.1.1 — 错误日志拆分 + 本地开发环境 ✅

#### 2.8 错误日志类型拆分

- 新增 `LogTypeErrorIntercepted = 51`（被重试系统拦截，客户端未感知）
- 新增 `LogTypeErrorClientVisible = 52`（所有重试失败后返回给客户端）
- 保留 `LogTypeError = 5`（历史数据兼容）
- `processChannelError` 默认记录 type=51，循环结束记录 type=52
- 前端日志页面新增两种类型的筛选和标签显示

**数据库迁移**：`UPDATE logs SET type = 52 WHERE type = 5;`

#### 2.9 本地开发环境

- `docker-compose.dev.yml` — 本地开发 MySQL（端口 3307）
- `.env` / `.env.example` — 环境变量模板
- `gogogo.sh` 选项 0 — 一键启动本地开发（Docker + Go + Vite HMR）
- `SKIP_DB_MIGRATION` 环境变量 — 跳过慢迁移加速启动

---

## 三、已实现但未部署的功能（v0.2.0-dev）

### v0.2.0 — 请求链路视图 + 日志分组展示

#### 3.1 请求链路视图（Request Trace View）

**后端 API**：
- `GET /api/log/traces` — 分页列出最近的请求链路摘要（按 request_id 分组聚合）
- `GET /api/log/trace?request_id=xxx` — 按 request_id 查询完整链路详情

**新增文件**：
- `service/trace.go` — TraceSummary/TraceStep/TraceDetail DTO + GetTraceList/GetTraceDetail 查询
- `controller/trace.go` — GetTraceList/GetTraceDetail 控制器
- `web/src/pages/Trace/index.jsx` — TracesPage 前端组件
- `web/src/pages/Trace/utils.js` — formatQuotaToUSD 工具函数
- `service/trace_test.go` — 6 个属性测试（pgregory.net/rapid）
- `web/src/pages/Trace/__tests__/format.test.js` — 前端属性测试（fast-check）

**路由注册**：`/api/log/traces`、`/api/log/trace`（AdminAuth）
**前端路由**：`/console/traces`（AdminRoute）
**侧边栏**：新增"请求链路"菜单项

#### 3.2 日志链路分组展示（Log Trace Grouping）

**核心改动**：现有日志页面 `/console/log` 按 request_id 分组折叠展示

**日志类型体系**：

| 类型 | 含义 | 存储 | 列表显示 |
|------|------|------|---------|
| 2 | 正常消费（无重试） | 真实记录 | 独立一行 |
| 20 | 重试后成功摘要 | **虚拟**（API 聚合） | 可展开行 |
| 50 | 重试后失败摘要 | **虚拟**（API 聚合） | 可展开行 |
| 5 | 旧系统遗留错误 | 真实记录 | 独立一行 |
| 51 | 拦截错误 | 真实记录 | 展开后可见 |
| 52 | 客户可见错误 | 真实记录 | 展开后可见 |
| 29 | 探测成功 | 真实记录 | 展开后可见 |
| 59 | 探测失败 | 真实记录 | 展开后可见 |

**后端新增**：
- `model/log.go` — LogTypeProbeSuccess=29, LogTypeProbeFailed=59 常量
- `service/log_grouped.go` — GetGroupedLogs 两阶段查询 + 应用层合并分页
- `controller/log.go` — GetGroupedLogs 控制器函数
- `router/api-router.go` — `GET /api/log/grouped`（AdminAuth）

**后端修复**：
- `controller/relay.go` — 同渠道重试循环内补全 type=51 日志记录（Other 含 `"retry_type": "same_channel"`）
- `service/channel_probe.go` — recordProbeLog 实际写入日志表（type=29/59），ProbeNextChannels 新增 requestId 参数

**后端扩展**：
- `service/trace.go` — GetTraceDetail 查询条件扩展支持 type=29/59，TraceStep 新增 Other 字段

**前端新增**：
- `web/src/components/table/usage-logs/TraceExpandRender.jsx` — 链路步骤时间线组件

**前端修改**：
- `UsageLogsColumnDefs.jsx` — renderType 新增 type=20/50，渠道列支持 channel_path 渲染，所有列支持 type=51/52 显示
- `UsageLogsTable.jsx` — 导入 TraceExpandRender，差异化展开行为（摘要行 vs 普通行）
- `useUsageLogsData.jsx` — 管理员 API 切换到 `/api/log/grouped`，摘要行 key 处理
- i18n — 新增翻译键（成功(重试)、失败(重试)、探测成功、探测失败等）

**前端修复**：
- `UsageLogsColumnDefs.jsx` — 所有列的 type 过滤条件加入 51/52（修复拦截日志字段显示为空的问题）

---

## 四、架构决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| 健康状态存储 | 内存优先 + 异步 DB 持久化 | 性能优先，重启后自动恢复 |
| 探测方式 | 直接 HTTP 调用，绕过 relay 管道 | 避免触发计费、限流等中间件 |
| 同渠道重试 | 同步执行，不异步 | 客户在等待，需要快速决策 |
| 并行预热 | goroutine 并行，结果缓存 | 切换渠道时零等待 |
| 日志分组 | 两阶段查询 + 应用层合并 | 避免 GROUP_CONCAT 跨库不兼容 |
| Channel_Path | 应用层生成 | 避免数据库特有函数 |
| 虚拟类型 20/50 | 仅 API 响应，不存 DB | 不污染日志表，向后兼容 |
| 探测日志 | gopool.Go 异步写入 | 不阻塞用户请求重试流程 |
| 前端展开详情 | 按需加载（Trace_Detail_API） | 列表接口仅返回摘要，性能优先 |

---

## 五、数据库变更汇总

### 新增字段（channels 表，GORM AutoMigrate）

```sql
ALTER TABLE channels ADD COLUMN health_status VARCHAR(16) DEFAULT 'healthy';
ALTER TABLE channels ADD COLUMN health_updated_at BIGINT DEFAULT 0;
ALTER TABLE channels ADD COLUMN health_fail_count INT DEFAULT 0;
ALTER TABLE channels ADD COLUMN health_success_count INT DEFAULT 0;
```

### 数据迁移（一次性）

```sql
UPDATE logs SET type = 52 WHERE type = 5;
```

### 新增日志类型值（logs 表 type 字段）

| 值 | 含义 | 新增版本 |
|----|------|---------|
| 29 | 探测成功 | v0.2.0 |
| 59 | 探测失败 | v0.2.0 |
| 51 | 拦截错误 | v0.1.1 |
| 52 | 客户可见错误 | v0.1.1 |

### 无新增索引

所有查询复用已有索引：`idx_logs_request_id`、`idx_created_at_id`、`idx_created_at_type`

---

## 六、新增 API 端点汇总

| 端点 | 方法 | 权限 | 用途 | 版本 |
|------|------|------|------|------|
| `/api/log/traces` | GET | AdminAuth | 链路摘要列表 | v0.2.0 |
| `/api/log/trace` | GET | AdminAuth | 链路详情 | v0.2.0 |
| `/api/log/grouped` | GET | AdminAuth | 分组日志列表 | v0.2.0 |

---

## 七、新增/修改文件清单

### 新增文件（后端）

| 文件 | 行数 | 用途 |
|------|------|------|
| `service/channel_health.go` | ~280 | 健康状态机 |
| `service/channel_health_config.go` | ~70 | 配置/阈值 |
| `service/channel_probe.go` | ~350 | 轻量探测 + 并行预热 + 探测日志 |
| `service/trace.go` | ~250 | 链路查询服务 |
| `service/trace_test.go` | ~350 | 属性测试 |
| `service/log_grouped.go` | ~350 | 分组日志查询 |
| `controller/trace.go` | ~100 | 链路 API 控制器 |

### 新增文件（前端）

| 文件 | 用途 |
|------|------|
| `web/src/pages/Trace/index.jsx` | 链路视图页面 |
| `web/src/pages/Trace/utils.js` | 工具函数 |
| `web/src/pages/Trace/__tests__/format.test.js` | 属性测试 |
| `web/src/components/table/usage-logs/TraceExpandRender.jsx` | 链路展开组件 |

### 修改文件（后端）

| 文件 | 变更内容 |
|------|---------|
| `common/constants.go` | 健康状态常量 + SameChannelRetryCount |
| `model/channel.go` | Channel struct 新增 4 个健康字段 |
| `model/channel_cache.go` | GetRandomSatisfiedChannel 健康过滤 + 排除 + 兜底 |
| `model/log.go` | LogTypeErrorIntercepted=51, LogTypeErrorClientVisible=52, LogTypeProbeSuccess=29, LogTypeProbeFailed=59 |
| `model/main.go` | SKIP_DB_MIGRATION + GetLogGroupCol() |
| `controller/relay.go` | 增强重试循环 + 同渠道重试日志 + 并行预热 requestId 传递 |
| `controller/log.go` | GetGroupedLogs 控制器 |
| `middleware/distributor.go` | 亲和性健康感知 |
| `router/api-router.go` | 新增 /traces, /trace, /grouped 路由 |
| `main.go` | 启动时初始化健康管理 |

### 修改文件（前端）

| 文件 | 变更内容 |
|------|---------|
| `UsageLogsColumnDefs.jsx` | type=20/50/51/52 渲染 + channel_path + 列过滤修复 |
| `UsageLogsTable.jsx` | TraceExpandRender 导入 + 差异化展开 |
| `useUsageLogsData.jsx` | API 切换到 /grouped + 摘要行 key |
| `SiderBar.jsx` | 请求链路菜单项 + routerMap |
| `App.jsx` | /console/traces 路由 |
| i18n locales (7 files) | 新翻译键 |

---

## 八、已知风险与待验证项

| 风险 | 影响 | 状态 |
|------|------|------|
| 分组日志 API 在大数据量下的分页性能 | 两阶段查询 + 应用层合并可能在百万级日志时变慢 | 待压测 |
| 同渠道重试日志记录可能增加 DB 写入量 | 每次同渠道重试失败多写一条 type=51 | 待监控 |
| 探测日志（type=29/59）可能产生大量记录 | 每次预热探测 2 个渠道 = 2 条日志 | 待监控 |
| 前端 /api/log/grouped 替换 /api/log/ 后的兼容性 | 非管理员仍用 /api/log/self/ 不受影响 | 待验证 |
| 虚拟类型 20/50 在前端筛选 type 下拉中的处理 | 用户选 type=20/50 时后端如何响应 | 待确认 |
| gogogo.sh 本地开发模式的 web/dist 占位策略 | 占位 index.html 可能影响 Go embed | 待验证 |
| 独立链路页面 /console/traces 是否保留 | 用户表示只需要改进现有日志页面 | 待决定是否清理 |

---

## 九、测试覆盖现状

### 已有测试

| 类型 | 文件 | 覆盖 |
|------|------|------|
| PBT (Go) | `service/trace_test.go` | 6 个属性（过滤排序、status_code 解析、结果分类、聚合、HAVING、Quota） |
| PBT (JS) | `web/src/pages/Trace/__tests__/format.test.js` | 1 个属性（Quota 格式化） |
| 集成测试 | `.analysis/run_integration_tests.sh` | v0.1.0 智能重试 12 个场景 |

### 缺失测试

| 缺失 | 优先级 |
|------|--------|
| log_grouped.go 的属性测试（Property 4-6） | 高 |
| 同渠道重试日志记录的单元测试 | 高 |
| 探测日志写入的单元测试 | 中 |
| 前端 TraceExpandRender 组件测试 | 中 |
| 前端 UsageLogsTable 分组展示集成测试 | 中 |
| 三数据库兼容性测试（分组查询） | 高 |
| 分页正确性测试（混合列表无遗漏无重复） | 高 |

---

## 十、部署信息

| 项目 | 值 |
|------|-----|
| 服务器 | 143.198.87.200 (DigitalOcean) |
| 域名 | nacp.m.srl |
| 镜像 | ghcr.io/al90slj23/nacp:main |
| 部署方式 | GitHub Actions → GHCR → docker pull |
| 数据库 | MySQL (远程 143.198.87.200:3306) |
| 本地开发 | gogogo.sh 选项 0（Go + Vite HMR） |

---

**文档结束。请基于以上信息制定完整的检测和测试计划。**
