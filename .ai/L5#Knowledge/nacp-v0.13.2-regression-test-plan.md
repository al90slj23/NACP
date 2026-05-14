# NACP v0.13.2 基线对比与全量回归测试计划

> 基线：`/Volumes/RuiRui4TB/CloudBackup/Mac/code/github/al90slj23/NACP/NewAPIv0.13.2`
> 基线提交：`bee339d279ccecbf8c8a89e14ddbbd902f78bd5d`
> 当前分支目标：classic-plus-0.2.0-dev
> 输入依据：`.ai/L5#Knowledge/nacp-full-changelog-for-review.md`、`.kiro/specs/*/{requirements,design}.md`、当前工作区与 NewAPI v0.13.2 目录 diff
> 日期：2026-05-15

## 1. 本次结论先行

本次变更不是单点 UI 或单点接口，而是改动了 NewAPI 的核心请求路径：渠道选择、relay 重试、错误拦截、渠道健康、探测、日志记录、日志聚合、链路展示、管理端日志列表。测试必须按“请求生命周期”做端到端验证，同时用系统维度做回归，避免只看新增页面而漏掉计费、统计、权限、缓存、三数据库兼容性和原有 provider 适配。

当前已完成的本地核验：

| 项目 | 结果 |
|---|---|
| 下载基线 | 已下载到 `NewAPIv0.13.2` |
| 基线版本 | `v0.13.2` tag 指向提交 `bee339d279ccecbf8c8a89e14ddbbd902f78bd5d` |
| changelog 阅读 | 已完整阅读 357 行 |
| 后端现有定向测试 | `go test ./service -run 'Test(Trace|Health|Property|Default|GetHealth)'` 通过 |
| 前端现有定向测试 | `bunx vitest run` 通过，1 个测试通过 |

必须重点验证的高风险点：

| 风险 | 为什么高风险 | 必测结论 |
|---|---|---|
| 渠道健康状态内存源不一致 | 健康状态机维护 `channelHealthStates`，但 `model.GetRandomSatisfiedChannel` 读取 `channelsIDM[channel].HealthStatus` 字段，状态迁移时未显式同步 channel cache 字段 | Degraded 后必须立即不被选中，不能等 DB/cache 周期刷新 |
| `GetTraceList` 的 `status` 过滤在应用层分页之后做 | total 和分页可能只统计当前页，状态筛选可能漏数据 | success/failed 筛选必须跨全量结果正确分页 |
| `/api/log/grouped` 先取所有 retry request_id 再做 `IN ?` | 大日志量可能出现巨大 IN 列表和内存压力 | 10 万、100 万日志下必须压测并 EXPLAIN |
| 探测直接 HTTP 未走各 provider adaptor | Anthropic、Azure、Gemini、AWS 等认证/路径差异可能导致探测误判 | 每类主流 provider 要单独探测验证 |
| 错误日志拆分影响统计 | 51/52/29/59 进入 logs 表，统计必须仍只按真实消费 type=2 计费/计量 | 账单、额度、RPM/TPM、数据导出不能被探测和拦截日志污染 |
| 同渠道重试可能重复触发健康状态 | `processChannelError` 已调用 `OnUserRequestError`，同渠道重试失败后又调用一次 | 状态迁移次数和日志数量必须符合预期 |
| 未跟踪文件 `model/trace.go` | 与 `service/trace.go` 有重复实现，controller 当前使用 service 层 | 确认是否删除/合并，避免未来误用分叉逻辑 |
| `.DS_Store` 出现在 service diff | 无业务意义，容易污染提交 | 发布前清理 |

## 2. 与 NewAPI v0.13.2 的差异归纳

目录级 diff 显示本次变更集中在以下区域：

| 区域 | 变更摘要 | 影响等级 |
|---|---|---|
| `controller/relay.go` | 增强 relay 重试、同渠道重试、并行预热、51/52 错误日志、健康状态触发 | P0 |
| `service/channel_health.go` | 新增渠道健康状态机、恢复探测、告警 | P0 |
| `service/channel_probe.go` | 新增轻量探测、并行预热、29/59 探测日志 | P0 |
| `model/channel.go` | `channels` 表新增健康状态字段 | P0 |
| `model/channel_cache.go` | 渠道选择加入健康过滤、排除列表、安全兜底 | P0 |
| `middleware/distributor.go` | 渠道亲和性健康感知 | P0 |
| `model/log.go` | 新增 51/52/29/59 类型与错误日志记录类型化 | P0 |
| `service/log_grouped.go` | 新增分组日志列表聚合 | P0 |
| `service/trace.go` | 新增/扩展链路列表与详情 | P0 |
| `controller/log.go`、`controller/trace.go`、`router/api-router.go` | 新增 `/api/log/grouped`、`/api/log/traces`、`/api/log/trace` | P0 |
| `web/src/components/table/usage-logs/*` | 管理员日志页改为分组接口、摘要行、展开链路 | P1 |
| `web/src/pages/Trace/*` | 独立请求链路页 | P1 |
| `web/src/i18n/locales/*` | 多语言新增标签 | P1 |
| `model/main.go`、`gogogo.sh` | 迁移跳过、本地开发便利性 | P1 |
| `go.mod`、`web/package.json` | 新增测试依赖 `rapid`、`vitest`、`fast-check` 等 | P1 |

## 3. NewAPI/NACP 系统地图

测试必须覆盖“系统”而不只是“文件”。建议把项目拆成这些系统：

| 系统编号 | 系统 | 核心路径 | 本次是否直接影响 | 主要风险 |
|---|---|---|---|---|
| S1 | 请求入口/路由/权限 | `router/`、`middleware/`、`controller/` | 是 | 新接口权限、路径冲突、限流/CORS 顺序 |
| S2 | Relay 主链路 | `controller/relay.go`、`relay/` | 是 | 透明重试、body 复用、流式响应、错误返回 |
| S3 | 渠道选择/缓存/优先级 | `service/channel_select.go`、`model/channel_cache.go` | 是 | 同优先级、权重、exclude、健康过滤、fallback |
| S4 | 渠道健康管理 | `service/channel_health.go`、`service/channel_probe.go` | 是 | 状态机、持久化、恢复、告警、并发 |
| S5 | 渠道亲和性 | `service/channel_affinity*`、`middleware/distributor.go` | 是 | Degraded 亲和跳过、成功后切换亲和 |
| S6 | 日志系统 | `model/log.go`、`controller/log.go`、`service/log_grouped.go` | 是 | 51/52/29/59 类型、分组分页、旧日志兼容 |
| S7 | 链路可观测性 | `service/trace.go`、`controller/trace.go`、前端 Trace | 是 | status、详情排序、status_code 解析、权限 |
| S8 | 计费/额度/退款 | `service/billing_session.go`、`relay/helper/price.go` | 间接 | 预扣/退款只执行一次，探测不计费 |
| S9 | 统计/报表/数据导出 | `model.SumUsedQuota`、`controller/data*`、`LogQuotaData` | 间接 | RPM/TPM/Quota 不受错误和探测日志污染 |
| S10 | 用户/令牌/分组 | `model/user*`、`model/token*`、`group` | 间接 | token model limit、group filter、auto group |
| S11 | Provider 适配 | `relay/channel/*` | 间接 | 探测绕过 adaptor 后 provider 差异 |
| S12 | 任务类接口 | `relay/channel/task/*`、MJ/Suno | 间接 | 是否误用 chat retry/probe，计费日志不变 |
| S13 | 前端日志页 | `web/src/components/table/usage-logs/*` | 是 | 管理员 grouped/self 分流、展开、筛选、列配置 |
| S14 | 前端路由/i18n | `web/src/App.jsx`、`SiderBar.jsx`、`i18n` | 是 | AdminRoute、菜单、语言 key |
| S15 | 数据库/迁移/跨库 | `model/main.go`、GORM schema | 是 | SQLite/MySQL/PostgreSQL 保留字、GROUP BY、AutoMigrate |
| S16 | 部署/本地开发 | `gogogo.sh`、Docker、CI | 是 | `SKIP_DB_MIGRATION`、embed dist、端口 |

## 4. 依据树

用这棵树追踪每个测试为什么存在，后续新增功能也挂到同一棵树上。

```text
NACP-v0.2.0 验证依据
├── A. 上游基线兼容性
│   ├── A1. NewAPI v0.13.2 原有 API 行为不回退
│   ├── A2. 原有用户、令牌、渠道、模型、任务、充值、统计模块不受影响
│   └── A3. 保护项：new-api / QuantumNous 标识不得被改动或删除
├── B. 智能重试与渠道健康
│   ├── B1. 五状态机：Healthy/Probing/Degraded/Recovering/Disabled
│   ├── B2. 同渠道重试：400/401/408/504/524/5xx/429 分类
│   ├── B3. 同优先级切换：exclude、max 3、优先级降级
│   ├── B4. 并行预热：顺位渠道、成功复用、失败跳过
│   ├── B5. 亲和性健康感知
│   └── B6. 低渠道告警与安全兜底
├── C. 日志与链路
│   ├── C1. 51/52 错误拆分
│   ├── C2. 29/59 探测日志
│   ├── C3. grouped 列表：虚拟 20/50 + 普通 2/5
│   ├── C4. trace detail：2/5/51/52/29/59
│   └── C5. 旧日志 type=5 兼容
├── D. 计费、计量、统计
│   ├── D1. 用户请求成功只结算一次
│   ├── D2. 所有失败路径正确退款或违约计费
│   ├── D3. 探测 user_id=0/quota=0，不影响用户账单
│   ├── D4. Usage stat 只统计 type=2
│   └── D5. tiered_expr、订阅、token quota 不被重试破坏
├── E. 数据库兼容与性能
│   ├── E1. SQLite
│   ├── E2. MySQL >= 5.7.8
│   ├── E3. PostgreSQL >= 9.6
│   ├── E4. 大表分页与索引
│   └── E5. GORM/SQL 保留字兼容
├── F. 前端与体验
│   ├── F1. 管理员日志页 grouped 展示
│   ├── F2. 普通用户 self 日志不变
│   ├── F3. 展开链路、空态、加载、错误
│   ├── F4. 筛选/分页/列配置/紧凑模式
│   └── F5. i18n 多语言
└── G. 交付门禁
    ├── G1. 单元测试/属性测试
    ├── G2. 三数据库集成测试
    ├── G3. Relay E2E + mock upstream
    ├── G4. 前端 build/test/浏览器冒烟
    ├── G5. 性能压测
    └── G6. 灰度与回滚
```

## 5. P0 测试任务总表

| ID | 测试任务 | 系统 | 方法 | 通过标准 |
|---|---|---|---|---|
| P0-01 | 渠道健康状态迁移全覆盖 | S4 | Go unit + PBT | 所有状态转换符合表，Disabled 无自动变化 |
| P0-02 | 健康状态和 channel cache 同步 | S3/S4 | Go integration | Degraded 后立即不参与选择 |
| P0-03 | 同渠道重试日志 | S2/S6 | mock upstream E2E | 每次失败都有 51，成功后无 52 |
| P0-04 | 最终失败日志 | S2/S6 | mock upstream E2E | 所有渠道耗尽后有且仅有 1 条 52 |
| P0-05 | 400/401/408/504/524 分类 | S2/S4 | table test + E2E | 状态变化、重试、日志均符合规则 |
| P0-06 | 并行预热探测日志 | S4/S6 | mock HTTP | 29/59 写入，request_id 关联，quota=0 |
| P0-07 | grouped 五种链路模式 | S6/S13 | DB fixture + API | A/B/C/D/E 模式返回正确 |
| P0-08 | grouped 筛选与分页 | S6 | PBT + 三 DB | 无遗漏、无重复、total 正确 |
| P0-09 | trace list/detail | S7 | API + PBT | 排序、聚合、status、limit、Other 解析正确 |
| P0-10 | 计费不重复 | S8 | E2E + DB assert | 预扣/退款/结算次数和额度正确 |
| P0-11 | 统计不污染 | S9 | API + DB assert | 51/52/29/59 不进入 quota/RPM/TPM |
| P0-12 | 三数据库兼容 | S15 | SQLite/MySQL/PostgreSQL | migration/query/test 全过 |
| P0-13 | AdminAuth 权限 | S1/S7 | API | 非管理员无法访问 grouped/traces/trace |
| P0-14 | 前端日志页主流程 | S13 | Vitest + browser | 管理员 grouped、展开、筛选、分页正常 |
| P0-15 | 原有用户日志不变 | S13/S6 | API + browser | `/api/log/self` 和普通用户页面不走 grouped |

## 6. 详细测试矩阵

### 6.1 静态/结构测试

| ID | 检查项 | 命令/方法 | 预期 |
|---|---|---|---|
| ST-01 | 与 v0.13.2 diff 范围确认 | `git diff --no-index --stat NewAPIv0.13.2/{controller,service,model,middleware,web/src} ...` | 差异只在预期模块 |
| ST-02 | 禁止直接 JSON marshal/unmarshal 新增违规 | `rg -n "json\\.(Marshal|Unmarshal|NewDecoder|NewEncoder)"` | 新增业务代码使用 `common.*`；保留类型引用可接受 |
| ST-03 | 保护标识未删除 | `rg -n "new-api|QuantumNous|quantumnous"` 对比基线 | README、license、包路径、版权不被移除 |
| ST-04 | 无无关系统文件 | `git status --short` | `.DS_Store` 不进入提交 |
| ST-05 | 重复实现检查 | 检查 `model/trace.go` 与 `service/trace.go` | 明确保留一个权威实现 |
| ST-06 | i18n key 完整 | `bun run i18n:lint` | zh/en/fr/ru/ja/vi/zh-TW 无缺 key |
| ST-07 | 前端依赖锁定 | `bun install --frozen-lockfile` | lock 与 package 一致 |
| ST-08 | 后端依赖可解析 | `go mod verify`、`go test ./...` | 依赖完整，无额外下载失败 |

### 6.2 数据库迁移与跨库兼容

| ID | 数据库 | 用例 | 预期 |
|---|---|---|---|
| DB-01 | SQLite | 空库启动 AutoMigrate | `channels` 有 4 个 health 字段，默认值正确 |
| DB-02 | MySQL 5.7.8+ | 旧库升级启动 | 只 ADD COLUMN，不破坏已有数据 |
| DB-03 | PostgreSQL 9.6+ | 旧库升级启动 | `"group"` 保留字引用正确 |
| DB-04 | 三库 | `logs` 聚合 GROUP BY | `group_val`、COUNT/SUM/MIN/MAX/CASE WHEN 可执行 |
| DB-05 | 三库 | `/api/log/grouped?group=default` | 不因 `group` 保留字报错 |
| DB-06 | 三库 | `/api/log/traces?status=success` | total 与分页正确 |
| DB-07 | 三库 | `SKIP_DB_MIGRATION=true` | 跳过迁移时旧库缺字段会被明确识别，不能静默部分失败 |
| DB-08 | 三库 | type=5 迁移到 52 的一次性 SQL | 迁移前备份，迁移后历史错误展示语义符合预期 |
| DB-09 | 三库 | 大 request_id 集合 | `IN ?` 参数量不超过数据库限制或有降级策略 |
| DB-10 | 三库 | logs request_id 空字符串 | 普通日志仍能查询，不进入链路摘要 |

### 6.3 渠道健康状态机

| ID | 初始状态 | 事件 | 预期状态 | 附加断言 |
|---|---|---|---|---|
| HSM-01 | unknown | 读取状态 | Healthy | 安全默认 |
| HSM-02 | Healthy | 500/502/503/429 | Probing | fail_count=1 |
| HSM-03 | Healthy | 400 | Healthy | 不改变健康 |
| HSM-04 | Healthy | 408/504/524 | Healthy | 不改变健康 |
| HSM-05 | Healthy | 401 + disable keyword | Degraded | 立即降级 |
| HSM-06 | Healthy | 401 无 disable keyword | Healthy | 不误降级 |
| HSM-07 | Probing | probe success | Healthy | fail/success 清零 |
| HSM-08 | Probing | 连续失败达到阈值 2 | Degraded | DB 异步持久化 |
| HSM-09 | Degraded | 1/2 次 probe success | Degraded | success_count 递增 |
| HSM-10 | Degraded | 第 3 次 probe success | Recovering | RecoveryStartedAt 设置 |
| HSM-11 | Recovering | 观察期内任何错误 | Degraded | success_count 清零 |
| HSM-12 | Recovering | 观察期过后 success/timer | Healthy | 只过期后恢复 |
| HSM-13 | Disabled | 任意 success/error/probe | Disabled | 无自动转换 |
| HSM-14 | 并发 | 100 goroutine 同时 error/success | 无 data race | `go test -race` 通过 |
| HSM-15 | Degraded | channel selector 立刻选渠 | 不选该渠道 | 验证 channel cache 同步 |
| HSM-16 | Degraded 全部渠道 | 选渠 fallback | 有安全兜底日志，服务不中断 | 但需告警 |
| HSM-17 | 降级后低渠道数 <=2 | 触发 warning | NotifyRootUser 被调用 |
| HSM-18 | 降级后健康数 0 | 触发 critical | 告警内容含 group/model/count |

### 6.4 Relay 重试与上游错误分类

| ID | 场景 | 上游响应脚本 | 预期客户端 | 预期日志 | 预期计费 |
|---|---|---|---|---|---|
| REL-01 | 无错误直接成功 | A:200 | 200 | 1 条 type=2 | 结算一次 |
| REL-02 | 首次 500，同渠道第 1 次成功 | A:500, A:200 | 200 | 1 条 51 + 1 条 2 | 只结算成功 |
| REL-03 | A 连续失败，预热 B 成功，B 成功 | A:500, A:500, B probe 200, B 200 | 200 | A 的 51、B 的 29、B 的 2 | 只结算 B 成功 |
| REL-04 | 所有渠道失败 | A/B/C 全 500 | 500/最后错误 | 多条 51 + 1 条 52 | 预扣全退或按违约规则 |
| REL-05 | 400 参数错误 | A:400, B:? | 按策略切换或最终错误 | 不改变健康 | 不把渠道降级 |
| REL-06 | 401 + 余额不足关键词 | A:401 keyword | 触发降级 | 51/52 视最终结果 | 不重复扣费 |
| REL-07 | 401 invalid key 无关键词 | A:401 | 不误健康降级 | 错误日志正确 | 不重复扣费 |
| REL-08 | 408/504/524 | A timeout code | 不健康降级 | 日志正确 | 超时退款正确 |
| REL-09 | RetryTimes=0 | A:500 | 立即客户端可见 | 1 条 52 | 退款 |
| REL-10 | specific_channel_id | 指定 A 失败 | 不切其他渠道 | 52 | 退款 |
| REL-11 | token specific channel | 指定渠道 disabled | 403 | 无 relay retry | 无预扣 |
| REL-12 | body too large | 请求体超限 | 413 | 不进入 retry | 无预扣 |
| REL-13 | stream 首包前失败 | A:500 before bytes | 可重试 | 51/2 或 52 | 正确 |
| REL-14 | stream 中途失败 | 已发数据后断开 | 不透明重试 | 流状态/错误日志保留 | 不重复 |
| REL-15 | realtime websocket | 握手/relay 错误 | websocket 错误格式 | 不破坏连接关闭 | 不重复 |
| REL-16 | Claude 格式 | Claude error response | Claude 错误格式 | request_id 注入 | 不重复 |
| REL-17 | Gemini 格式 | Gemini error response | Gemini 路径正常 | 日志正常 | 不重复 |

### 6.5 同优先级、权重和亲和性

| ID | 场景 | 预期 |
|---|---|---|
| SEL-01 | 同优先级 A/B/C，A 失败 | 优先尝试 B/C，不直接降到低优先级 |
| SEL-02 | 同优先级超过 `MaxRetrySamePriority=3` | 才降到下一优先级 |
| SEL-03 | excludeIDs 中已有 A/B | 不再选 A/B |
| SEL-04 | 同优先级只剩 excluded | fallback 行为可控，并有 warning |
| SEL-05 | 权重 0 全部渠道 | 保持原有平滑逻辑 |
| SEL-06 | 部分 Degraded | 正常不选 Degraded |
| SEL-07 | 全部 Degraded | 安全兜底可选，避免服务中断 |
| SEL-08 | 亲和渠道 Healthy | 正常使用亲和 |
| SEL-09 | 亲和渠道 Degraded | 跳过亲和，选健康渠道 |
| SEL-10 | 亲和渠道 disabled + SkipRetryOnFailure | 按原逻辑返回/跳过 |
| SEL-11 | SwitchOnSuccess | 成功切到新渠道后亲和更新 |
| SEL-12 | auto group | 亲和与 auto group 选择一致 |

### 6.6 探测系统

| ID | 场景 | 预期 |
|---|---|---|
| PROBE-01 | OpenAI-compatible 2xx | success=true，type=29 |
| PROBE-02 | OpenAI-compatible 500 | success=false，type=59 |
| PROBE-03 | timeout | 3 秒内失败，status_code=0，type=59 |
| PROBE-04 | request body | `model/messages/max_tokens=1/stream=false` |
| PROBE-05 | request_id | pre_warm 探测日志带主请求 request_id |
| PROBE-06 | degraded_probe | request_id 为空，probe_trigger=degraded_probe |
| PROBE-07 | billing | user_id=0，quota=0，不改变用户额度 |
| PROBE-08 | channel used quota | 不增加渠道真实 user used quota，或明确记录独立成本 |
| PROBE-09 | Anthropic | 认证 header/endpoint 正确，不误判 |
| PROBE-10 | Azure OpenAI | endpoint/version/deployment 正确，不误判 |
| PROBE-11 | Gemini | endpoint/auth 正确，不误判 |
| PROBE-12 | AWS Bedrock | 如果不支持直接 chat endpoint，应跳过或专用适配 |
| PROBE-13 | 多 key channel | 选 key 与 key 状态一致，不用 disabled key |
| PROBE-14 | 并发预热 2 个渠道 | goroutine 全部回收，无泄漏 |
| PROBE-15 | LOG_DB 写入失败 | 不阻塞用户请求，SysLog 可见 |

### 6.7 日志分组与链路 API

准备固定 fixture：

| 模式 | logs 数据 | grouped 预期 |
|---|---|---|
| A 无重试直接成功 | 1 条 type=2，request_id 可空或非空 | 普通 type=2 行 |
| B 重试后成功 | 51 + 29/59 + 2，同 request_id | 虚拟 type=20 摘要，可展开 |
| C 重试后失败 | 51 + 29/59 + 52 | 虚拟 type=50 摘要，可展开 |
| D 直接失败 | 52 单条 | 依据需求为 type=50 摘要；若实现不一致需决策 |
| E 旧错误 | type=5 | 普通 type=5 行或迁移后 type=52 |

| ID | 接口 | 场景 | 预期 |
|---|---|---|---|
| LOG-01 | `/api/log/grouped` | 默认列表 | 摘要行 + 普通行按时间倒序 |
| LOG-02 | grouped | type=2 | 只返回无重试 type=2 普通行 |
| LOG-03 | grouped | type=51 | 只返回包含 51 的摘要行 |
| LOG-04 | grouped | type=52 | 只返回包含 52 的摘要行 |
| LOG-05 | grouped | type=20/50 | 明确后端行为，不可让前端筛选失效 |
| LOG-06 | grouped | request_id 指定 | 平铺返回该 request_id 所有日志 |
| LOG-07 | grouped | model_name exact | 只返回该模型 |
| LOG-08 | grouped | username/token/channel/group | 过滤正确 |
| LOG-09 | grouped | start/end timestamp | 边界包含正确 |
| LOG-10 | grouped | channel_path 12,12,14,14,12 | `12→14→12`，不含 29/59 |
| LOG-11 | grouped | total_quota | 只 sum type=2 |
| LOG-12 | grouped | step_count | 包含 51/52/2/29/59 |
| LOG-13 | grouped | pagination page 1/2/3 | 无重复、无遗漏、total 不随页变化 |
| LOG-14 | grouped | 大量普通日志 + 少量摘要 | 合并分页稳定 |
| TRACE-01 | `/api/log/trace` | valid request_id | 返回 2/5/51/52/29/59，created_at ASC |
| TRACE-02 | trace detail | 不存在 request_id | steps=[]，success=true |
| TRACE-03 | trace detail | request_id 空 | 参数错误 |
| TRACE-04 | trace detail | request_id >64 | 参数错误 |
| TRACE-05 | trace detail | Other.admin_info.status_code int/float/string | 可解析或 nil |
| TRACE-06 | trace detail | invalid Other JSON | status_code=null，不报错 |
| TRACE-07 | `/api/log/traces` | status=success/failed | total 和分页跨全量正确 |
| TRACE-08 | traces | 单条 type=2 无错误 | 不出现在 trace list |
| TRACE-09 | traces | 单条 51/52 | 出现在 trace list |
| TRACE-10 | traces | limit 100 | detail 最多 100 条 |

### 6.8 计费、计量、统计、日志互斥

| ID | 场景 | 验证点 |
|---|---|---|
| BILL-01 | 直接成功 | user quota 扣一次，token used quota 一次，channel used quota 一次 |
| BILL-02 | 失败后同渠道成功 | 只有成功结果结算，失败预扣退款 |
| BILL-03 | 跨渠道成功 | 只按最终成功渠道/模型结算 |
| BILL-04 | 所有渠道失败 | 预扣全退，或 violation fee 按配置生效 |
| BILL-05 | tiered_expr | snapshot、expr_hash、tier、quota 与原逻辑一致 |
| BILL-06 | subscription billing | subscription pre/post consumed 不因重试重复 |
| BILL-07 | token quota insufficient | 不进入 probe/relay 后续路径 |
| BILL-08 | free model | 跳过预扣，探测仍不计用户账 |
| BILL-09 | log stat | `/api/log/stat` 只统计 type=2 |
| BILL-10 | self stat | `/api/log/self/stat` 只统计当前用户 type=2 |
| BILL-11 | data export | `LogQuotaData` 只由 type=2 consume 触发 |
| BILL-12 | probe logs | user_id=0 不被用户账单、用户日志、充值/消费统计误纳入 |

### 6.9 前端回归

| ID | 页面/组件 | 场景 | 预期 |
|---|---|---|---|
| FE-01 | `/console/log` admin | 首屏加载 | 请求 `/api/log/grouped` |
| FE-02 | `/console/log` user | 首屏加载 | 请求 `/api/log/self/`，不走 grouped |
| FE-03 | 日志表 | type=20 | 显示“成功(重试)”绿色标签 |
| FE-04 | 日志表 | type=50 | 显示“失败(重试)”红色标签 |
| FE-05 | 日志表 | type=51/52 | 字段不为空，标签正确 |
| FE-06 | 渠道列 | channel_path | 渲染 `12 → 14`，普通行仍显示 channel_name |
| FE-07 | 展开摘要行 | 加载成功 | 调 `/api/log/trace` 并显示树形步骤 |
| FE-08 | 展开摘要行 | API 失败 | loading 结束，有错误/空态，不挂死 |
| FE-09 | 普通行展开 | 仍显示原 Descriptions | 不被 TraceExpandRender 接管 |
| FE-10 | 筛选 type | 2/20/50/51/52/29/59 | 前后端语义一致 |
| FE-11 | request_id 筛选 | 指定 ID | 平铺展示 |
| FE-12 | 分页/页大小 | 10/20/50/100 | 行数、total、页码正确 |
| FE-13 | 列配置 | 隐藏/显示列 | 摘要行和普通行都不崩 |
| FE-14 | compact mode | 开/关 | 布局不重叠 |
| FE-15 | `/console/traces` | AdminRoute | 管理员可见，普通用户不可见 |
| FE-16 | 多语言 | zh/en/fr/ru/ja/vi/zh-TW | 新增 key 不显示中文 fallback 泄漏，除非有意 |
| FE-17 | build | `bun run build` | 无编译错误 |
| FE-18 | browser | 桌面/移动宽度 | 表格、展开区文本不溢出 |

### 6.10 原模块回归

| 系统 | 最少回归项 |
|---|---|
| Auth | 登录、JWT、OAuth、2FA/passkey preflight、权限守卫 |
| Token | 创建 token、模型限制、分组限制、token 日志 self 查询 |
| Channel CRUD | 新增/编辑/启停/多 key 状态/余额查询/测试 |
| Model/ratio | 模型同步、价格、倍率、分组倍率 |
| Provider | OpenAI、Claude、Gemini、Azure、Bedrock、OpenAI-compatible 各 1 个成功和失败 |
| Task/MJ/Suno | 任务创建、查询、回调、计费日志 |
| Billing | 充值、订阅、兑换码、退款、余额不足 |
| Settings | operation/performance/system setting 不被新常量破坏 |
| Logs | 旧 `/api/log/`、`/api/log/self`、`/api/log/token` 兼容 |
| Data | quota dates、dashboard billing usage |

## 7. 性能与容量测试

| ID | 数据规模 | 场景 | 指标 |
|---|---|---|---|
| PERF-01 | 1 万 logs | grouped 默认第一页 | P95 < 300ms |
| PERF-02 | 10 万 logs | grouped 多筛选 | P95 < 800ms |
| PERF-03 | 100 万 logs | grouped 默认第一页 | P95 < 2s，内存无尖峰 |
| PERF-04 | 10 万 retry request_id | `getRetryRequestIds` | 不触发 DB 参数上限 |
| PERF-05 | 100 并发请求 | relay 重试 + prewarm | goroutine 不泄漏，DB 写入可承受 |
| PERF-06 | 1000 degraded channels | degraded probe loop | 不形成突发上游/DB 压力 |
| PERF-07 | 前端 100 行每页 | 20 个摘要展开 | UI 不明显卡顿 |

必须对三库跑 `EXPLAIN`：

| 查询 | 期望索引 |
|---|---|
| `WHERE request_id = ? ORDER BY created_at ASC` | `idx_logs_request_id` + created_at 排序可接受 |
| `WHERE type IN (...) AND request_id != '' GROUP BY request_id...` | `idx_created_at_type` 或 request_id 索引 |
| `WHERE type = 51/52 AND request_id != ''` | type/time 索引 |
| normal rows 排除 retry request_id | 避免超大 NOT IN |

## 8. 建议补充的自动化测试文件

| 优先级 | 文件 | 覆盖 |
|---|---|---|
| P0 | `service/log_grouped_test.go` | 五种链路模式、筛选、分页、channel_path、total |
| P0 | `model/channel_cache_health_test.go` | Degraded 与 cache 同步、exclude、fallback |
| P0 | `controller/relay_retry_test.go` | mock upstream、51/52、预热、计费退款 |
| P0 | `service/channel_probe_test.go` | 29/59、request_id、quota=0、timeout |
| P0 | `tests/db_cross_compat/...` | SQLite/MySQL/PostgreSQL grouped/trace |
| P1 | `web/src/components/table/usage-logs/__tests__/TraceExpandRender.test.jsx` | loading、empty、steps 渲染 |
| P1 | `web/src/components/table/usage-logs/__tests__/UsageLogsTable.grouped.test.jsx` | 摘要/普通展开差异 |
| P1 | `web/src/hooks/usage-logs/__tests__/useUsageLogsData.test.jsx` | admin grouped/user self URL |
| P1 | `.analysis/run_nacp_regression.sh` | 一键串联测试 |

## 9. 推荐执行顺序

1. 基线静态检查：确认 diff 范围、清理 `.DS_Store`、处理 `model/trace.go` 重复实现。
2. 后端单元测试：健康状态机、probe、log grouped、trace、channel cache。
3. 三数据库集成：迁移、grouped/trace fixture、group 保留字、分页 total。
4. Relay mock E2E：构造 A/B/C 渠道，上游脚本化返回 200/400/401/408/429/500/502/503/504/524。
5. 计费一致性：每个 E2E 场景后断言 users/tokens/channels/logs/quota dates。
6. 前端自动化：vitest、build、浏览器检查日志页和 trace 页。
7. 性能压测：大 logs 表、并发 relay、degraded probe loop。
8. 生产灰度：小流量打开，观察 51/52/29/59 比例、客户端错误率、DB 写入、P95 latency、退款异常。

## 10. 发布门禁

必须全部满足：

| 门禁 | 标准 |
|---|---|
| 单测 | `go test ./...` 通过，新增 P0 测试通过 |
| 前端 | `bunx vitest run`、`bun run build`、`bun run i18n:lint` 通过 |
| 三库 | SQLite/MySQL/PostgreSQL migration + grouped/trace fixture 通过 |
| Relay | P0 Relay E2E 全通过 |
| 计费 | 所有重试场景无重复扣费、无探测计费 |
| 统计 | `/api/log/stat` 与实际 type=2 消费一致 |
| 性能 | 10 万 logs P95 < 800ms；100 万 logs 有可接受策略 |
| 安全 | 新接口 AdminAuth；普通用户只能看 self logs |
| 保护项 | 未删除/替换 new-api、QuantumNous 相关标识 |
| 清洁度 | 无 `.DS_Store`、无重复废弃实现、无无关文件 |

## 11. 灰度观察指标

| 指标 | 正常预期 | 异常含义 |
|---|---|---|
| 客户端 5xx/429 比例 | 下降 | 重试未生效或渠道耗尽 |
| type=51 数量 | 有上升 | 被拦截错误可观测 |
| type=52 数量 | 低于旧 type=5 | 最终失败减少 |
| type=29/59 数量 | 与重试量相关 | 过高说明预热太激进 |
| 平均/P95 latency | 小幅上升 | 过高说明同渠道重试/探测拖慢 |
| DB writes/sec | 可控上升 | 51/29/59 日志写入压力 |
| user quota 异常退款 | 0 | 计费流程被破坏 |
| channel health degraded 数 | 稳定波动 | 探测误判或 provider 故障 |
| grouped API P95 | 稳定 | 聚合查询性能问题 |

## 12. 回滚和应急策略

| 问题 | 快速缓解 |
|---|---|
| grouped API 慢 | 前端临时切回 `/api/log/`，保留 trace 接口 |
| 探测误判导致大量降级 | 临时关闭健康过滤或调大阈值，清空 health_status |
| 日志写入压力过高 | 降低/采样 probe log，或只记录失败探测 |
| 计费异常 | 立即关闭增强 retry，恢复 v0.13.2 relay 路径 |
| 前端日志页异常 | 隐藏 grouped 展开，普通列表兜底 |
| 三库某库 SQL 异常 | 为该库增加分支或回退平铺日志 |

## 13. 当前建议立即处理项

| 优先级 | 项 |
|---|---|
| P0 | 明确并修复健康状态机与 channel cache 的同步策略 |
| P0 | 为 `/api/log/grouped` 增加属性测试和三库 fixture |
| P0 | 修正/验证 trace list status 过滤的 total 和分页 |
| P0 | 确认 provider-specific probe 支持范围，不支持则跳过或走 adaptor |
| P1 | 处理 `model/trace.go` 与 `service/trace.go` 重复实现 |
| P1 | 清理 `service/.DS_Store` |
| P1 | 为前端 grouped table 增加组件/集成测试 |
| P1 | 将本测试计划做成 CI 分层 job：unit、db、relay-e2e、frontend、perf |
