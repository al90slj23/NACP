# 2026-05-18 NACP统计单位渠道与流水成本深化

## 背景

本次继续深化 `/console/nacp_stats` 的 `单位&渠道` 和 `流水&成本` 两个主题页。目标不是继续堆页面骨架，而是让这两个页面真正能回答运营问题：

- 哪些渠道在被调用，它们属于哪个单位和账号。
- 渠道是否启用、是否健康、优先级和权重是什么。
- 今日平台收入、上游成本和预计利润分别是多少。
- 成本来自精确快照差值，还是按成功消费额度分摊估算。
- 哪些单位账号缺少快照、统计源或成本基线。

## 后端调整

1. `NacpChannelRankStats` 新增：
   - `unit_name`
   - `unit_type`
   - `unit_account_name`

2. 渠道排行 `/api/nacp_stats/channels` 现在在聚合日志后，会回查渠道绑定的单位和账号，直接返回可读归属信息，不再只返回 `unit_id` / `unit_account_id`。

3. `NacpCostRankStats` 新增：
   - `unit_type`
   - `channel_status`
   - `channel_health_status`
   - `priority`
   - `weight`

4. 流水成本 `/api/nacp_stats/costs` 的 `dimension=unit` 和 `dimension=channel` 补齐单位类型、账号显示名、渠道状态、渠道健康、优先级、权重、余额、已用等字段。

5. 修复单位成本聚合中的 slice 指针风险：
   - 原逻辑使用 `map[key]*NacpCostRankStats` 指向 slice 元素。
   - 后续 `append` 可能导致 slice 底层数组迁移，旧指针可能写入非最终数组。
   - 当前改为 `map[key]int` 保存下标，每次写入都通过 `&items[index]` 取当前 slice 元素。

## 前端调整

1. `单位&渠道` 主题页的渠道排行表新增 `所属单位/账号` 列：
   - 显示单位名称。
   - 显示单位类型。
   - 显示账号名称或账号 ID。

2. `单位&渠道` 顶部摘要面板增强：
   - 渠道排行预览展示单位归属。
   - 新增渠道配置摘要：启用/总数、手动停用、自动停用、健康/降级/异常。
   - 新增当前钻取维度预览，展示分组/令牌/端点/IP 的前几名。

3. `流水&成本` 明细表增强：
   - 对象列显示对象本身。
   - 新增归属列，显示单位、账号、渠道优先级、权重。
   - 新增余额/已用列。
   - 成本口径列显示快照差值、额度分摊、缺少统计源，并显示精确/估算和平台状态。

4. `流水&成本` 顶部摘要面板增强：
   - 新增当前成本排行预览。
   - 继续保留收入、成本、利润三块口径说明。

## 验证

已执行：

```bash
GOCACHE=/private/tmp/nacp-gocache go test ./model ./controller
bun run build
```

本地开发环境通过 `gogogo.sh` 启动后，使用测试库验证：

- `/api/nacp_stats/overview?hours=24` 成功返回。
- `/api/nacp_stats/channels?hours=24&p=1&page_size=3&sort=requests` 成功返回空列表，当前测试库 24 小时内无用户可见渠道日志。
- `/api/nacp_stats/costs?dimension=unit&p=1&page_size=3&sort=revenue` 成功返回单位成本数据，包含 `unit_name`、`unit_type`、`account`、`snapshot_count`、`cost_source=snapshot_delta`。
- `/api/nacp_stats/costs?dimension=channel&p=1&page_size=3&sort=revenue` 成功返回空列表，当前测试库今日无渠道成功消费日志。
- `/api/nacp_stats/dimensions?hours=24&dimension=group&p=1&page_size=3&sort=requests` 成功返回空列表，当前测试库 24 小时内无分组请求日志。

