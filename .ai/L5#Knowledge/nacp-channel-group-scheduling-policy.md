# NACP 渠道分组调度配置说明

> 最后更新：2026-05-19

## 背景

NewAPI 原始渠道表中 `channels.priority` 和 `channels.weight` 是渠道级字段。NACP 允许一个渠道同时挂载到多个分组后，这种全局优先级/权重不再足够：同一个渠道在 `default`、`vip`、`test` 等不同分组中，应该可以有不同的调度顺位。

因此 NACP 将调度语义调整为：

- `channels.priority` / `channels.weight`：全局默认值、旧数据兼容 fallback。
- `channel_group_configs`：渠道在某个分组里的持久化调度配置。
- `abilities.priority` / `abilities.weight`：实际调度索引，由渠道分组配置生成。

## 数据结构

新增业务库表：

| 表 | 字段 | 说明 |
|---|---|---|
| `channel_group_configs` | `channel_id` | 渠道 ID |
| `channel_group_configs` | `group` | 分组名称 |
| `channel_group_configs` | `priority` | 该渠道在该分组内的优先级 |
| `channel_group_configs` | `weight` | 该渠道在该分组内的权重 |

主键语义为 `channel_id + group`。同一个渠道可以对不同分组保存不同 priority / weight。

## 读写规则

新增/编辑渠道时，前端传入：

```json
{
  "group": "default,vip",
  "priority": 0,
  "weight": 0,
  "group_configs": [
    { "group": "default", "priority": 10, "weight": 50 },
    { "group": "vip", "priority": 100, "weight": 20 }
  ]
}
```

保存规则：

1. 如果请求中包含 `group_configs`，后端按当前 `channel.group` 里的分组重写 `channel_group_configs`。
2. 如果请求没有包含 `group_configs`，后端保留原有分组配置，避免老 API 客户端编辑渠道时误删配置。
3. 如果某个分组没有独立配置，则回退到 `channels.priority` / `channels.weight`。
4. 新增、更新、复制、批量新增渠道后，都会重新生成 `abilities`。
5. 旧的列表内联修改或标签批量修改 `priority/weight` 只修改“默认优先级 / 默认权重”。如果该渠道尚未有 `channel_group_configs`，后端会为当前分组生成默认配置并同步重建 `abilities`；如果已经有明确的分组配置，则不会覆盖这些显式配置。

## 调度规则

实际分发仍然读取 `abilities`：

1. 先按请求分组和模型筛选 `abilities`。
2. 按 `priority DESC` 选择优先级层。
3. 无亲和的首次正式请求：只在最高优先级层内按原始 `weight` 做概率选择。
4. SFT 顺位容错队列：使用 `priority DESC, weight DESC, channel_id ASC`，此时 `weight` 是同优先级内的子排序，不再是概率。

这意味着 `channel_group_configs` 是配置源，`abilities` 是调度索引。

## 2026-05-19 缓存路径修正

本地和测试环境默认 `MEMORY_CACHE_ENABLED=true`。修正前，内存缓存会从 `channels.group` / `channels.models` 重建 `group -> model -> channel_id`，并使用 `channels.priority` / `channels.weight` 排序。这会导致同一个渠道在不同分组的独立调度配置被绕过。

修正后：

1. `InitChannelCache()` 直接读取 `abilities` 构建缓存。
2. 缓存项保存 `channel_id + abilities.priority + abilities.weight`。
3. `GetNextSatisfiedChannel()` 在缓存路径下按 `abilities.priority DESC, abilities.weight DESC, channel_id ASC` 顺序选择。
4. `GetRandomSatisfiedChannel()` 在缓存路径下按 `abilities.priority` 选择优先级层，并用 `abilities.weight` 做同优先级内权重选择。
5. `IsChannelEnabledForGroupModel()` 仍通过缓存判断渠道是否属于指定分组/模型，只是缓存列表元素从裸 `channel_id` 变成带调度信息的条目。

这个边界很重要：任何新调度逻辑都必须保证数据库路径和内存缓存路径读取同一套调度语义，即 `abilities`，不能重新回退到 `channels.priority` / `channels.weight`。

## 2026-05-19 首次选择与 SFT 重试队列

NACP 当前采用“首次概率，失败后确定队列”的策略：

1. 没有渠道亲和时，首次正式请求只在最高优先级层内按原始 `weight` 随机。
2. 如果最高优先级层所有渠道 `weight=0`，则在该层内等概率随机。
3. 一旦首次请求失败，进入 SFT 顺位容错队列。
4. SFT 队列不再按概率随机，而是按 `priority DESC, weight DESC, channel_id ASC` 确定排序。
5. 非最高优先级层的权重不会影响首次概率，只会影响后续队列中的同层顺序。
6. Relay 第一轮必须使用 Distributor 已经选好的渠道，不能在进入 relay 循环后再次用顺位队列覆盖首次随机结果。

例子：

```text
A priority=3 weight=1
B priority=2 weight=1
C priority=5 weight=8
D priority=5 weight=3
E priority=1 weight=6
F priority=1 weight=9
```

无亲和时：

```text
8/11 概率：C -> D -> A -> B -> F -> E
3/11 概率：D -> C -> A -> B -> F -> E
```

如果亲和渠道为 `A`：

```text
A -> C -> D -> A -> B -> F -> E
```

如果亲和渠道为 `C`：

```text
C -> C -> D -> A -> B -> F -> E
```

注意：亲和前置机会不改变原始分组队列，因此亲和渠道在原始队列中仍保留一次按自身位置重试的机会。

`using_group=auto` 的亲和命中需要额外记录命中的 auto 分组下标。后续 SFT 重试必须从该下标对应分组继续，而不是回到 auto 的第一个分组重新选路。

## 2026-05-19 SFT 首字超时与总预算

SFT 顺位容错的时间限制已经从“下一跳启动前软检查”调整为“首字硬预算”：

1. 总预算从 NACP 接收到 relay 请求的 `relay_received_at` 开始计算，默认 `60s`。
2. 单渠道正式请求等待上游首字/响应头的上限默认 `20s`。
3. 每次正式请求实际等待首字的时间为 `min(20s, 总剩余时间)`。
4. 如果 A/B/C 已经耗时 59 秒，D 最多只能等待 1 秒首字，不允许 D 再单独跑 20 秒或更久。
5. 首字到达后，流式完整输出不受 SFT 60 秒/20 秒限制，继续走原有流式扫描、保活和流式超时机制。
6. `RELAY_TIMEOUT` 保留为 NewAPI 原生全局 HTTP client 生命周期超时，默认 `0` 等于关闭。它可能覆盖完整响应体读取和完整流式输出，因此不能用来表达 SFT 首字限制。

这条规则和渠道队列顺序共同决定最终行为：队列顺序负责“下一个是谁”，首字预算负责“每个尝试最多等多久、整条链最多等多久”。

## 兼容规则

老数据库没有 `channel_group_configs` 时：

1. 页面读取渠道详情会先查 `channel_group_configs`。
2. 如果没有配置，则从旧 `abilities` 中按分组推导显示值。
3. 如果连 `abilities` 也没有，则使用 `channels.priority` / `channels.weight` 作为每个分组的默认值。

这样不会要求旧日志或旧渠道做强制迁移，也不会破坏 NewAPI 老数据。

## 修复能力表规则

管理员触发能力表修复或系统重建 `abilities` 时，必须从 `channel_group_configs` 重新生成能力表，而不是只读 `channels.priority` / `channels.weight`。

这条是关键边界：`abilities` 可以被删除重建，`channel_group_configs` 不能因此丢失。

## 前端规则

渠道新增/编辑弹窗：

1. 保留原有全局 `优先级` / `权重`，作为默认值和旧兼容。
2. 在分组选择后显示“分组调度配置”。
3. 每个已选分组展示独立 `优先级` 和 `权重`。
4. 修改分组列表时，已有分组配置保留；新增分组使用全局默认值填充。
5. 渠道列表里的全局字段必须标注为“默认优先级 / 默认权重”，避免误解为最终调度值。
6. 标签聚合模式下的批量修改文案也必须标注为“默认优先级 / 默认权重”。

## 测试要求

每次修改这块逻辑至少覆盖：

1. 新建渠道：同渠道多分组生成不同 `abilities.priority/weight`。
2. 编辑渠道模型：不传 `group_configs` 时保留既有分组配置。
3. 修复能力表：删除重建 `abilities` 后仍从 `channel_group_configs` 恢复。
4. 前端构建：渠道弹窗可正常渲染“分组调度配置”。
5. 三数据库迁移：`channel_group_configs` 必须通过 GORM AutoMigrate 支持 SQLite、MySQL、PostgreSQL。
6. 旧批量入口：无显式分组配置的渠道修改默认值后会生成分组配置；已有显式分组配置的渠道不会被默认值覆盖。
7. 内存缓存路径：开启 `MEMORY_CACHE_ENABLED=true`，构造两个渠道的全局优先级和分组优先级相反，确认 `GetNextSatisfiedChannel()` / `GetRandomSatisfiedChannel()` 都按分组内 `abilities` 排序。
