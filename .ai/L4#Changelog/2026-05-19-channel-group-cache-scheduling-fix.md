# 2026-05-19 渠道分组调度缓存修复

## 背景

NACP 已经把渠道调度从 NewAPI 原始的全局 `channels.priority` / `channels.weight` 扩展为“渠道 + 分组”的独立调度配置：

- `channel_group_configs` 保存每个渠道在每个分组内的配置。
- `abilities.priority` / `abilities.weight` 作为实际调度索引。
- `channels.priority` / `channels.weight` 只作为默认值和旧兼容字段。

排查容错重试系统时发现，本地和测试环境默认开启 `MEMORY_CACHE_ENABLED=true`，缓存路径仍从 `channels.group` / `channels.models` 重建可用渠道，并使用 `channels.priority` / `channels.weight` 排序。这会绕过分组级 `abilities`，导致同一个渠道在不同分组内的优先级和权重不生效。

## 修复

1. `model/channel_cache.go`
   - 内存缓存不再保存裸 `channel_id` 列表，改为保存 `channel_id + priority + weight`。
   - 缓存构建来源改为启用状态的 `abilities`，并校验对应渠道也处于启用状态。
   - 缓存排序改为 `abilities.priority DESC, abilities.weight DESC, channel_id ASC`。
   - `GetNextSatisfiedChannel()` 按缓存中的分组级调度顺序进行顺位选择。
   - `GetRandomSatisfiedChannel()` 按缓存中的分组级 `priority/weight` 做优先级层选择和同层权重选择。

2. `model/channel_satisfy.go`
   - 适配缓存条目结构，继续支持分组/模型可用性判断。

3. `model/channel_group_config_test.go`
   - 新增 `TestChannelCacheUsesPerGroupAbilityScheduling`。
   - 构造两个渠道的全局优先级与分组优先级相反，验证缓存路径下 `GetNextSatisfiedChannel()` 和 `GetRandomSatisfiedChannel()` 都按分组级 `abilities` 生效。

4. `controller/relay.go` / `service/channel_affinity.go`
   - 明确“渠道亲和 + 原始分组队列”的组合语义。
   - 亲和命中并实际选用渠道后，只把该渠道作为一次前置尝试，不提前写入 SFT 排除集。
   - 如果原始分组队列中仍包含该渠道，后续仍会按原始优先级/权重位置再给一次机会。
   - 例：原始队列 `C -> D -> A -> B`，亲和渠道 `A`，实际队列为 `A -> C -> D -> A -> B`。
   - Relay 第一轮固定使用 Distributor 已选渠道，避免刚进入 Relay 循环时被顺位队列覆盖首次随机结果。

5. `middleware/distributor.go` / `model/ability.go`
   - 无亲和的首次正式请求改为最高优先级层内按原始 `weight` 概率选择。
   - 首次请求失败后，SFT 重试队列仍按 `priority DESC, weight DESC, channel_id ASC` 确定排序。
   - 数据库直查路径移除旧的 `weight + 10` 平滑，避免 `8:3` 被错误稀释为 `18:13`。
   - 新增测试覆盖缓存路径和数据库路径的原始权重语义。

6. `service/channel_affinity.go` / `middleware/distributor.go`
   - `skip_retry_on_failure` 只在亲和渠道实际被选用后生效。
   - 规则匹配但缓存未命中、渠道禁用、渠道不可路由、分组/模型不匹配，都视为亲和失效，继续普通分组队列。
   - `using_group=auto` 的亲和命中会同时记录 auto 分组名称和分组下标，避免后续 SFT 从 auto 第一个分组错误重启。
   - Codex CLI / Claude CLI 默认亲和规则的 `skip_retry_on_failure` 调整为 `false`，后台显示为“重试”。

## 行为结论

开启内存缓存后，渠道调度与数据库直查路径保持一致：

1. 先按请求分组和模型匹配 `abilities`。
2. 只选择启用的 ability 和启用的 channel。
3. 无亲和首次请求只在最高优先级层内按原始权重随机。
4. SFT 容错重试按 `priority DESC, weight DESC, channel_id ASC` 确定队列逐个尝试。
5. 若存在渠道亲和命中，亲和渠道会前置一次；这一次不改变原始分组队列。
6. 亲和未生效时不会触发 `skip_retry_on_failure`，不会阻断普通容错链路。

## 验证

- `GOCACHE=/private/tmp/nacp-gocache go test ./middleware`
- `GOCACHE=/private/tmp/nacp-gocache go test ./model`
- `GOCACHE=/private/tmp/nacp-gocache go test ./controller ./service ./model`
- `GOCACHE=/private/tmp/nacp-gocache go test ./...`

以上均通过。
