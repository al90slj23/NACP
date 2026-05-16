# SFT 物化摘要日志实施计划

> 创建日期：2026-05-16
> 状态：第一阶段已实现
> 目标：让 `20/50` 成为有真实 `logs.id` 的链路摘要日志，并让 `21/29/51/52/59` 成为可持久化的新日志类型。

## 1. 目标语义

NACP 日志从“单纯原始事件日志”升级为：

```text
原始事件日志 + 链路摘要日志
```

新标准：

| 类型 | 含义 | 入库 | 主表展示 | 展开展示 | 用户消费统计 | 平台探测统计 |
|---:|---|---|---|---|---|---|
| 2 | 正常消费成功 | 是 | 是 | 否 | 是 | 否 |
| 20 | 容错重试后成功摘要 | 是 | 是 | 否 | 是 | 否 |
| 21 | 容错链路内最终成功事件 | 是 | 否 | 是 | 否 | 否 |
| 29 | 容错轻量探测成功 | 是 | 否 | 是 | 否 | 是 |
| 50 | 容错重试后失败摘要 | 是 | 是 | 否 | 失败统计 | 否 |
| 51 | 容错重试已拦截事件 | 是 | 否 | 是 | 否 | 否 |
| 52 | 容错重试最终客户端可见错误 | 是 | 否 | 是 | 否 | 否 |
| 59 | 容错轻量探测失败 | 是 | 否 | 是 | 否 | 可计失败 |

## 2. 新字段

在 `logs` 上增加：

| 字段 | 用途 |
|---|---|
| `summary_log_id` | 子行指向所属 `20/50` 摘要行；摘要行可指向自己 |
| `terminal_log_id` | 摘要行指向最终终态事件，成功为 `21`，失败为 `52` |
| `trace_version` | 新日志结构版本；旧日志为 `0` 或空，新 SFT 为 `1` |

旧日志不迁移、不补数据。

## 3. 主表排序

统一规则：

```text
主列表 = logs.id DESC
链路展开 = trace_seq ASC -> trace_sibling_seq ASC -> id ASC
```

`20/50` 真实入库后有自己的 `logs.id`，因此主表排序直接按摘要行 id。

## 4. 时间规则

`created_at` 仍表示该日志行生成时间：

| 类型 | created_at 触发点 |
|---:|---|
| 2 | 直接成功计费日志生成时间 |
| 20 | 终态 `21` 后摘要生成时间；详情记录 started_at/ended_at |
| 21 | 容错链路最终成功计费日志生成时间 |
| 29 | 探测成功结果记录时间 |
| 50 | 终态 `52` 后摘要生成时间；详情记录 started_at/ended_at |
| 51 | 正式请求错误被拦截时 |
| 52 | 最终错误决定返回用户时 |
| 59 | 探测失败/超时结果记录时间 |

## 5. SFT 总超时

总超时从 NACP 接收到并开始处理用户 relay 请求开始，而不是从分发器完成选渠道后开始。

目标新增：

```text
ContextKeyRelayReceivedAt
```

用于：

```text
now - relay_received_at >= sft_total_timeout
```

超时后返回最后一个真实正式请求错误，不返回轻量探测错误。

当前实现边界：

```text
默认 TotalRetryTimeout = 30s
已限制后续同渠道重试、预热探测等待、备用渠道正式请求调度
暂不强行中断已经发出的正式上游请求
```

## 6. 生成流程

成功链：

```text
51/29/59...
-> 21
-> 生成 20 summary
-> 回填所有子行 summary_log_id
```

失败链：

```text
51/29/59...
-> 52
-> 生成 50 summary
-> 回填所有子行 summary_log_id
```

直接成功：

```text
2
```

不生成 `20`。

## 7. 一致性复核

查询或后台任务可做轻量修复：

1. 有 `21/52` 但没有 `20/50`：补 summary。
2. 有 summary 但 `terminal_log_id` 错：重算。
3. 子行缺 `summary_log_id`：回填。
4. summary `other.summary.step_count/channel_path/platform_probe_quota` 过期：重算。

## 8. 老日志兼容

| 形态 | 判断 | 显示 |
|---|---|---|
| 老 NewAPI 日志 | 无 `trace_id/trace_role/summary_log_id` | 单行显示 |
| NACP 过渡期 SFT | 有 `trace_id`，无 `summary_log_id`，无 `20/50` | 前端兼容聚合 |
| 新 SFT | 有 `20/50` 或 `summary_log_id` | 主表显示 summary，展开子行 |

## 9. API 调整

`/api/log/grouped`：

```text
默认返回：普通日志 + 20/50 summary
默认隐藏：summary_log_id > 0 的子行
request_id 搜索：优先返回 summary；没有 summary 时返回过渡期子行
```

`/api/log/trace`：

```text
返回子行，不返回 summary 行
按 trace_seq/sibling/id 正序
```

## 10. 统计调整

用户消费：

```text
type IN (2, 20)
```

平台探测：

```text
type = 29
```

链路失败：

```text
type = 50
```

子行 `21/51/52/59` 不进入用户消费统计，避免与 `20/50` 重复。

## 11. 第一阶段落地范围

已完成：

| 模块 | 调整 |
|---|---|
| `model.Log` | 增加 `summary_log_id`、`terminal_log_id`、`trace_version` |
| 日志类型 | `20/21/29/50/51/52/59` 真实写入 `logs.type` |
| 摘要生成 | `21/52` 终态后 upsert `20/50`，并回填子行 `summary_log_id` |
| 主列表 | 默认返回普通日志 + `20/50`，隐藏已归属摘要的子行 |
| 展开接口 | `/api/log/trace` 返回子行，不返回 `20/50` |
| 展开排序 | `trace_seq ASC -> trace_sibling_seq ASC -> id ASC` |
| 统计 | 默认消费统计改为 `type IN (2,20)` |
| 前端筛选 | 增加 `20/21/29/50/51/52/59` 类型入口 |
| 前端兼容 | 仅在“全部类型”时保留过渡期前端聚合；显式筛选子类型时直接展示子行 |
| 计时起点 | 新增 `ContextKeyRelayReceivedAt`，`RelayInfo.StartTime` 优先使用入口时间 |
| SFT 总窗口 | 默认 30 秒，超时后停止后续调度并用最后一次正式请求错误收尾 |

待继续增强：

| 项 | 原因 |
|---|---|
| `trace_parent_id/trace_sibling_seq` 精准父子结构 | 当前仍主要靠 `trace_seq` 线性顺序 |
| 摘要一致性后台复核 | 防止终态已写但摘要写失败、异步 probe 后到导致摘要统计过期 |
| 总超时强中断上游请求 | 需要把 deadline 注入请求 context，并单独验证流式和长响应 |
| 平台探测运营消耗统计页 | 已有 `29` 数据基础，统计入口仍需单独设计 |
