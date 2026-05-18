# 2026-05-18 渠道分组级优先级与权重

## 背景

同一个渠道可以同时分配到多个分组，但原有 `channels.priority` / `channels.weight` 是渠道全局字段，无法表达“同一渠道在不同分组中有不同调度顺位”的需求。

## 本次调整

- 新增 `channel_group_configs` 持久化表，记录 `channel_id + group` 维度的 `priority` 和 `weight`。
- 新增渠道 API 字段 `group_configs`，用于新增/编辑渠道时传入每个分组的调度配置。
- 渠道新增、批量新增、编辑、复制后，根据 `channel_group_configs` 生成 `abilities.priority/weight`。
- 能力表修复时从 `channel_group_configs` 重建，避免丢失分组级配置。
- 渠道详情接口兼容旧数据：优先读 `channel_group_configs`，没有时从 `abilities` 推导，再没有时回退 `channels.priority/weight`。
- 渠道新增/编辑弹窗新增“分组调度配置”，每个已选分组可单独设置优先级和权重。
- 渠道列表中的全局字段改为“默认优先级 / 默认权重”，避免和分组级调度值混淆。
- 收口旧的列表内联和标签批量优先级/权重入口：这些入口只修改默认值；无显式分组配置时生成默认分组配置并重建 `abilities`，已有显式分组配置时不覆盖。

## 验证

- `GOCACHE=/private/tmp/nacp-gocache go test ./model`
- `GOCACHE=/private/tmp/nacp-gocache go test ./controller ./service`
- `bun run build`
- 本地浏览器打开 `http://localhost:23901/console/channel`，添加渠道弹窗可看到“分组调度配置”及分组级优先级/权重输入框。

## 注意

`channels.priority` / `channels.weight` 仍保留，用作旧数据兼容和新分组默认值；实际调度读取的是 `abilities`。
