# SFT 日志链路结构与 20/50 摘要排序说明

> 创建日期：2026-05-16
> 更新日期：2026-05-16
> 适用范围：`/console/log`、`/api/log/grouped`、`/api/log/trace`、SFT 容错重试日志

## 1. 数据库真实行

NACP 已将 SFT 日志升级为“原始事件行 + 物化摘要行”的结构。新 SFT 类型会真实写入 `logs.type`，不是只靠前端派生显示。

数据库仍然保持“一行一条独立日志”，但日志行分为两类：

| 类型 | DB `type` | DB `trace_role` | 行性质 |
|---|---:|---|---|
| `2：正常消费成功` | `2` | `consume` | 直接成功的普通消费日志 |
| `20：容错重试后成功` | `20` | `summary_success` | SFT 成功链路摘要行 |
| `21：容错重试成功` | `21` | `consume` | SFT 链路内最终成功事件 |
| `29：容错探测成功` | `29` | `probe_success` | SFT 轻量探测成功事件 |
| `50：容错重试后失败` | `50` | `summary_failed` | SFT 失败链路摘要行 |
| `51：容错重试已拦截` | `51` | `error_intercepted` | 正式请求错误被拦截事件 |
| `52：容错重试客户端可见` | `52` | `error_visible` | 最终返回给用户的错误事件 |
| `59：容错探测失败` | `59` | `probe_failed` | SFT 轻量探测失败事件 |

`20/50` 现在是真实摘要日志行，有自己的 `logs.id`。子行通过 `summary_log_id` 指向所属摘要行，摘要行通过 `terminal_log_id` 指向终态事件。

## 2. 同链路关联依据

同一链路靠这些字段还原：

| 字段 | 作用 |
|---|---|
| `request_id` | 同一次用户请求的请求 ID，也是 API 查询入口 |
| `trace_id` | SFT 链路 ID；当前通常等于 `request_id` |
| `trace_role` | 当前日志在链路中的角色 |
| `trace_seq` | 当前日志在链路中的线性步骤序号 |
| `trace_parent_id` | 父日志 ID，目标用于还原树形父子关系 |
| `trace_sibling_seq` | 同父级下的顺序，目标用于还原并发分支的稳定顺序 |
| `summary_log_id` | 子行归属到真实 `20/50` 摘要行 |
| `terminal_log_id` | 摘要行指向最终终态事件，成功为 `21`，失败为 `52` |
| `trace_version` | 新日志结构版本；旧日志为空或 `0`，新 SFT 为 `1` |

当前已稳定使用：

```text
trace_id/request_id 负责归组
trace_seq 负责线性排序
trace_role 负责事件语义
summary_log_id 负责子行归属到真实 20/50 摘要行
terminal_log_id 负责摘要行指向最终 21/52
```

当前尚需增强：

```text
trace_parent_id 需要真实指向触发当前步骤的父日志
trace_sibling_seq 需要真实记录同父级下的分支顺序
```

## 3. 链路内排序

单条链路展开时，后端按以下规则返回：

```text
trace_seq ASC
-> trace_sibling_seq ASC
-> id ASC
```

因此同一 `trace_id` 下：

```text
51 -> 29/59 -> 21/52
```

实际由 `trace_seq` 保证稳定顺序。后续补齐 `trace_parent_id/trace_sibling_seq` 后，可以进一步还原树形父子关系与并发分支。

## 4. 20/50 摘要行排序

`/api/log/grouped` 默认返回普通日志和真实 `20/50` 摘要日志，后端排序为：

```text
ORDER BY id DESC
```

默认主列表隐藏已经归属到摘要行的子日志：

```text
NOT (summary_log_id > 0 AND type NOT IN (20,50))
```

因此新 SFT 链路在 `/console/log` 主表中的位置由真实摘要行 `logs.id` 决定，不再由前端“当前页第一次遇到该链路某个子行”的位置推断。

前端仍保留过渡期兼容分组：如果后端返回了有 `trace_id/trace_role` 但还没有 `summary_log_id`/`20/50` 的旧 SFT 子行，前端可以临时聚合显示。显式筛选 `21/29/51/52/59` 时不启用兼容分组，避免筛选结果被前端隐藏。

## 5. 老日志排序

NewAPI 原生旧日志和当前非 SFT 普通日志在 `/console/log` 默认列表中的排序不是按时间字段排序，而是后端 SQL 明确按自增 ID 倒序：

```text
ORDER BY id DESC
```

涉及接口：

```text
/api/log/
/api/log/grouped
/api/log/self
```

`created_at` 当前主要用于：

1. 页面“时间”列展示。
2. 时间范围筛选。
3. 详情中的时间说明。

因此旧日志 `1/3/4/6` 以及历史或外部扩展的 `7/8/9`，只要走当前日志列表接口，默认主表排序就是：

```text
logs.id DESC
```

不是：

```text
logs.created_at DESC
```

通常 `id` 与写入时间同向增长，所以看起来接近按时间倒序；但严格依据是 `id`。

## 6. SFT 总超时起点

SFT 容错重试总时限从 NACP 接收到并开始处理用户 relay 请求的最早入口开始计算，而不是从分发器完成模型读取、初始渠道选择之后开始。

当前代码里已有的 `ContextKeyRequestStartTime` 设置位置在 `middleware.Distribute()` 内：

```text
getModelRequest
-> token/model/group/channel selection
-> Set ContextKeyRequestStartTime
-> SetupContextForSelectedChannel
-> Relay
```

这个位置偏晚，因为用户在进入 NACP relay 路由后，已经处于等待状态。

已新增更早的时间字段：

```text
ContextKeyRelayReceivedAt
```

设置位置：

```text
router 全局 request received middleware
-> TokenAuth
-> ModelRequestRateLimit
-> Distribute
-> Relay
```

该时间表示 NACP 服务器开始处理该 HTTP 请求的时间。`RelayInfo.StartTime` 优先使用 `relay_received_at`，再回退到 `request_start_time`，最后回退到 `time.Now()`。

SFT 总超时计算：

```text
now - relay_received_at >= sft_total_timeout
```

当前默认：

```text
sft_total_timeout = 60s
sft_first_byte_timeout = 20s
```

当前实现语义：

1. SFT 总时限从 `relay_received_at` 开始算，默认 60 秒。
2. 单渠道正式请求只等待上游首字/响应头，默认最多 20 秒。
3. 每次正式上游请求实际可等待时间为 `min(20s, total_remaining)`。
4. 如果总剩余时间只剩 1 秒，则本次渠道请求最多只等 1 秒首字；不会再出现“29 秒时启动下一跳，下一跳自己跑 60 秒”的情况。
5. 超时后记录最后一个真实正式请求错误，并进入 `52 -> 50` 收尾。
6. 超时只约束首字到达前的等待。一旦流式响应已经开始，完整流式输出不受 `sft_first_byte_timeout` / `sft_total_timeout` 限制。

与 `RELAY_TIMEOUT` 的边界：

1. `RELAY_TIMEOUT` 是 NewAPI 原生 HTTP client 生命周期超时，默认 `0`，表示关闭。
2. `RELAY_TIMEOUT` 覆盖完整 HTTP 请求生命周期，可能包含完整响应体读取和流式输出过程。
3. SFT 新增的 60 秒/20 秒是业务级首字限制，不依赖 `RELAY_TIMEOUT`，也不应该用短 `RELAY_TIMEOUT` 代替。
4. 如果未来为 `RELAY_TIMEOUT` 设置非零值，必须确认不会截断长流式输出。

## 7. 当前排序的边界

当前新方案已经解决了前端当前页推断造成的主表位置失真：新 SFT 主表排序由真实 `20/50` 的 `logs.id DESC` 保证。

仍需继续增强的边界：

1. 如果终态 `21/52` 已写入，但摘要 `20/50` 写入失败，会出现“有子行无摘要”，需要一致性复核补摘要。
2. 如果异步 probe 在摘要之后才写入，摘要 `other.summary.step_count/channel_path/platform_probe_quota` 可能过期，需要复核重算。
3. 如果未来要以“终态事件时间”而不是“摘要行生成时间”排序，可继续增加 `sort_at/source_log_id` 或让摘要 `created_at` 精确继承终态事件；当前统一保持 `logs.id DESC`，摘要行真实入库后自然排序。

## 8. 目标父子结构

SFT 后续应把 `trace_parent_id` 和 `trace_sibling_seq` 真实写入：

```text
A1 error_intercepted
├── A2 same-channel retry
├── A3 same-channel retry
├── B- probe_success
├── C- probe_failed
└── B2 consume/error_intercepted
```

字段规则：

| 场景 | `trace_parent_id` | `trace_sibling_seq` |
|---|---:|---:|
| 初始失败 A1 | `0` | `0` |
| A2/A3 同渠道重试 | A1 的 `log.id` | 1, 2 |
| B-/C- 轻量探测 | A1 的 `log.id` | 按候选渠道顺位递增 |
| B2 正式请求成功/失败 | A1 的 `log.id` 或对应探测节点 ID | 按执行决策顺序 |
| B2 失败后继续分支 | B2 失败日志的 `log.id` | 新一层从 1 递增 |

目标是让数据库原始行即可还原：

```text
同链路归属
父子关系
同级顺序
最终 20/50 摘要
展开顺序
```
