# 2026-05-17 单位管理与 NACP统计调整

## 变更

- 单位管理编辑弹窗内整合账号统计源信息。
- 单位管理列表操作区新增 `详情` 按钮，位于 `编辑`、`停用/启用`、`删除` 后面。
- 单位详情弹窗会自动读取单位信息、挂载账号和账号统计源；具备账户访问令牌和账户 ID 的账号如果尚未创建统计源，会自动创建并立即检测。
- 支持在单位账号行内创建统计源并立即检测。
- 支持在单位账号行内刷新统计源、查看摘要详情和 JSON 原始信息。
- `/api/unit_monitor/` 增加 `unit_id` 与 `unit_account_id` 查询过滤，用于精确加载单位下账号统计源。
- NewAPI 兼容单位账号采集器新增 `/api/status` 读取，检测结果写入 `raw_json.platform_status`。
- 单位详情展示平台计价与充值信息，包括系统名称、版本、额度单位、`quota_per_unit`、`price`、`stripe_unit_price`、`usd_exchange_rate`、`custom_currency_symbol`、`custom_currency_exchange_rate`、`top_up_link`。
- `price` 字段按“充值倍率”展示；如果上游平台未暴露该字段，则显示 `-`，不做推断。
- `NACP统计` 页面新增顶部统计看板，展示统计源状态、余额汇总、采集能力和最近刷新时间。
- `NACP统计` 页面增加用户统计、模型统计、渠道健康、SFT 容错重试真实聚合模块。
- `NACP统计` 页面新增独立主题页 `流水&成本`，展示平台收入、上游成本、预计利润、毛利率和快照覆盖度。
- 新增 `unit_account_monitor_snapshots` 快照表，单位账号监控每次成功检测后写入余额、已用额度、单位、账号和检测时间。
- 新增 `/api/nacp_stats/overview` 专用概览接口。
- 新增 `/api/nacp_stats/models`、`/api/nacp_stats/users`、`/api/nacp_stats/channels` 三个分页排行接口。
- `/api/nacp_stats/overview` 支持 `hours` 时间范围参数，当前限制最大 30 天。
- 三个排行接口支持 `hours`、`p`、`page_size`、`sort` 参数，`page_size` 最大 100。
- 概览接口使用条件聚合和分组聚合，避免前端拉取大量统计源明细后聚合。
- 概览接口内部并发查询独立统计块，降低远程测试库延迟累加。
- `NACP统计` 页面增加 1 小时、6 小时、24 小时、7 天、30 天范围切换。
- `NACP统计` 页面增加 Top 用户、Top 渠道、Top 模型活动排行。
- `NACP统计` 的 `模型&性能`、`用户&消耗`、`单位&渠道` 三个主题页增加完整分页排行表。
- 完整排行表展示请求、成功/失败、`2/20/50/5` 类型拆分、成功率、流式占比、平均耗时、Token、额度。
- 渠道排行额外展示渠道名称、启用状态、健康状态、优先级、权重、单位 ID、账号 ID。
- `NACP统计` 页面固定为五个主题页：`综合`、`模型&性能`、`用户&消耗`、`流水&成本`、`单位&渠道`。
- 五个主题页分别组织关键卡片、趋势条形图和摘要面板，避免所有统计堆在一个长页面里。
- `/api/nacp_stats/overview` 增加趋势序列，按范围自动分桶返回请求、成功、错误、探测、Token、额度、成功率。
- 修复趋势分桶在 MySQL 下返回 decimal 字符串导致 `int64` 扫描失败的问题。
- 修复未配置 `LOG_SQL_DSN`、日志库复用业务库时，统计趋势误判日志库类型为 SQLite 的问题。
- 修复短时间范围没有排行数据时 `models/top_users/top_channels` 返回 `null` 的问题，统一返回空数组。
- 修复 `NACP统计` 顶部主题 Tab 被固定顶部导航栏遮挡的问题，页面外层增加 Header 安全间距。
- 前端趋势图不依赖 Semi 动态 CSS 变量名，使用明确色值，避免条形图颜色空白。
- `NACP统计` 页面增加请求质量和消耗统计，包括请求量、成功率、平均耗时、流式占比、Token、额度、探测量、拦截错误、用户可见错误。
- 调整 NACP统计口径：用户可见请求按 `2/20/50/5` 统计，Token 和额度只按 `2/20` 成功消费统计，避免 `20` 与 `21` 重复计算。
- 新增流水成本口径：平台收入按今日成功消费 `quota / QuotaPerUnit` 计算；上游成本按今日单位账号快照 `latest.used_amount - first.used_amount` 计算；快照不足时标记估算。
- 收紧请求质量 SQL 的日志类型过滤，避免只按时间范围扫描无关日志。
- 阅读 `ReferenceProjects` 下 `new-api-latest`、`one-api-latest`、`sub2api-latest`、`metapi-latest`、`NiceApiManager-latest` 五个项目的 dashboard / usage / monitor / stats 相关实现，提炼 NACP统计页面规则。
- `NACP统计` 五个主题页顶部增加主题说明、当前范围、生成时间、主题焦点标签，避免只靠 Tab 名称理解页面用途。
- KPI 卡片增加主题色顶边和比例进度条，用于成功率、统计源健康率、渠道健康率、SFT 成功率、毛利率等指标。
- `流水&成本` 主题页显示单位账号统计源表，用于检查上游成本快照覆盖、余额采集和估算原因。
- `.ai` 补充参考项目吸收原则和 NACP统计页面展示规则。
- 新增 `/api/nacp_stats/costs` 流水成本明细接口，支持 `dimension=unit|channel|model|user`。
- `dimension=unit` 使用单位账号统计源今日首尾快照差值计算上游成本，属于精确快照口径；快照不足时标记估算。
- `dimension=channel|model|user` 使用今日 `2/20` 成功消费额度进行上游成本分摊，接口返回 `cost_source=allocated_by_quota` 和 `estimated=true`，避免把分摊成本伪装成精确成本。
- `流水&成本` 页面新增明细表，可切换单位/账号、渠道、模型、用户四个维度，并按收入、成本、利润、毛利率、请求、Token、额度排序。
- 成本明细表展示收入、上游成本、预计利润、毛利率、成本口径、快照数量、快照时间和缺失原因。
- 继续研读 `ReferenceProjects` 五个项目后，新增 `/api/nacp_stats/dimensions` 多维度钻取接口，支持 `dimension=group|token|endpoint|ip`。
- 多维度钻取接口按统一口径统计请求、成功/失败、`2/20/50/5` 类型拆分、流式、Token、额度、平均耗时、成功率。
- `NACP统计` 页面新增 `多维度钻取排行`，可切换分组、令牌、端点、IP，并支持请求数、Token、额度、错误、耗时、成功数排序。
- 多维度钻取表默认随主题页切换：综合/单位&渠道默认分组，模型&性能默认端点，用户&消耗默认令牌。
- `NACP统计` 概览新增运营健康分，按请求成功率、单位账号统计源健康率、渠道健康率、今日成本快照覆盖率加权计算。
- 综合页新增 `运营健康分` KPI 卡片，展示健康/关注/危险状态，以及请求、渠道、成本覆盖拆分。
- 新增 `/api/nacp_stats/model_coverage` 模型覆盖与缺口矩阵接口，按 `abilities + channels + logs` 汇总模型覆盖分组、启用渠道、健康渠道、降级渠道、停用渠道、最近请求、错误、Token、额度和平均耗时。
- `模型&性能` 主题页新增 `模型覆盖与缺口矩阵` 表，支持按覆盖风险、健康率、启用渠道少、请求数、错误数、平均耗时、Token、额度排序。
- 模型覆盖风险明确区分 `ok`、`idle`、`warning`、`critical`，用于发现模型没有启用渠道、没有健康渠道、近期成功率偏低或当前范围无真实请求。

## 验证

- `GOCACHE=/private/tmp/nacp-gocache go test ./model ./controller ./service` 通过。
- `web/` 下 `bun run build` 通过。
- 本地开发环境可通过 `gogogo.sh` 重启到 `http://localhost:23901` / `http://localhost:23900`。
- 未登录访问 `/api/nacp_stats/overview` 返回 401，符合 AdminAuth 预期。
- 管理员登录后带 `New-Api-User: 1` 访问 `/api/nacp_stats/overview?hours=24` 返回 200，并包含 `trend`、`models`、`top_users`、`top_channels` 真实数据。
- 管理员登录后访问 `/api/nacp_stats/overview?hours=1` 返回 200，空排行字段返回 `[]`。
- 管理员登录后访问 `/api/nacp_stats/models?hours=24&p=1&page_size=5` 返回真实模型排行，包含 `claude-haiku-4-5-20251001`、`nacp-sft-model-044939` 等测试数据。
- 管理员登录后访问 `/api/nacp_stats/users?hours=24&p=1&page_size=5` 返回真实用户排行，包含 `nacpt044939` 等测试用户。
- 管理员登录后访问 `/api/nacp_stats/channels?hours=24&p=1&page_size=5` 返回真实渠道排行，包含 `MOCK-Controllable-P100`、`nacp-sft-local-044939-*` 等测试渠道。
- 管理员登录后访问 `/api/nacp_stats/overview?hours=24` 返回 `cost` 字段，包含 `platform_revenue_usd`、`upstream_cost_usd`、`estimated_profit_usd`、`gross_margin_perm`、`snapshot_count`。
- 管理员执行 `/api/unit_monitor/check_all` 后成功写入 `unit_account_monitor_snapshots`，再次读取 `overview` 时 `cost.snapshot_count` 从 0 变为 1。
- 管理员登录后访问 `/api/unit_monitor/?p=1&page_size=5` 返回 200，`流水&成本` 页面可复用统计源表检查成本快照来源。
- `GOCACHE=/private/tmp/nacp-gocache go test ./model ./controller ./service` 在新增成本明细接口后通过。
- `web/` 下 `bun run build` 在新增成本明细表后通过。
- `GOCACHE=/private/tmp/nacp-gocache go test ./service ./model ./controller` 在新增单位详情自动采集和 `/api/status` 采集后通过。
- `web/` 下 `bun run build` 在新增单位详情弹窗和平台计价字段后通过。
- `GOCACHE=/private/tmp/nacp-gocache go test ./model ./controller` 在新增多维度钻取接口后通过。
- `web/` 下 `bun run build` 在新增多维度钻取排行后通过。
- `GOCACHE=/private/tmp/nacp-gocache go test ./model ./controller` 在新增运营健康分后通过。
- `web/` 下 `bun run build` 在新增运营健康分卡片后通过。
- `GOCACHE=/private/tmp/nacp-gocache go test ./model ./controller` 在新增模型覆盖矩阵后通过。
- `web/` 下 `bun run build` 在新增模型覆盖矩阵后通过。

## 注意

- 当前 NACP统计概览已经由后端专用 overview API 聚合，前端只负责展示和触发时间范围切换。
- 当前测试库单位列表为空，但统计源存在历史记录，因此单位管理和 NACP统计显示数量可能不一致；页面已按兼容显示处理。
