# NACP 渠道分组调度配置说明

> 最后更新：2026-05-18

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
3. 同优先级内按 `weight` 做权重选择或顺序兜底。
4. SFT 顺位容错中的 `GetNextChannel` 使用 `priority DESC, weight DESC, channel_id ASC`。

这意味着 `channel_group_configs` 是配置源，`abilities` 是调度索引。

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
