# NewAPI 中继重试机制深度分析

> **分析日期**：2026-05-13
> **基线版本**：v0.13.2 (bee339d)
> **分析目的**：为 NACP 重试增强提供决策依据

---

## 〇、2026-05-15 精读纯化结论

> 本节基于本地 `NewAPIv0.13.2` 源码再次精读，目的是把 NewAPI 原生“重试机制”和 NACP 容错增强边界说清楚。

### 0.1 一句话结论

NewAPI v0.13.2 原生确实有重试机制，但它是**失败后串行重选渠道并重发完整请求**；不是 NACP 目标里的**同渠道重试 + 候选渠道提前探测 + 链路结构化 + 最终状态可观测**。

### 0.2 NewAPI 原生重试到底是什么

主入口在 `NewAPIv0.13.2/controller/relay.go`：

```go
for ; retryParam.GetRetry() <= common.RetryTimes; retryParam.IncreaseRetry() {
    channel, channelErr := getChannel(c, relayInfo, retryParam)
    // restore request body
    newAPIError = relayHandler(...)
    if newAPIError == nil { return }
    processChannelError(...)
    if !shouldRetry(...) { break }
}
```

它的实际行为：

1. 第一次请求使用 `retry=0` 选中的渠道。
2. 上游失败后，先记录错误、可能自动禁用渠道。
3. `shouldRetry` 判断状态码/错误类型是否允许继续。
4. 如果允许，下一轮用 `retry=1/2/...` 再选渠道。
5. 每一轮都是完整真实请求，不存在轻量探测请求。
6. 成功则直接返回客户端；最终失败才返回错误。

### 0.3 默认情况下为什么看起来“没用”

默认重试次数是：

```go
var RetryTimes = 0
```

位置：`NewAPIv0.13.2/common/constants.go`

所以默认只会跑：

```text
retry = 0
```

如果后台没有把“失败重试次数”配置为大于 0，原生机制不会发生第二次真实重试。

### 0.4 它怎么换渠道

渠道选择由 `service.CacheGetRandomSatisfiedChannel` 和 `model.GetRandomSatisfiedChannel` 完成。

普通分组：

```text
retry=0 -> 最高 priority
retry=1 -> 第二档 priority
retry=2 -> 第三档 priority
retry 超过 priority 数量 -> 使用最低 priority
```

同一 priority 内是按 weight 随机选一个渠道。

关键限制：

- 不记录“本次请求已经试过哪些具体渠道”来做严格排除。
- 同 priority 下再次选择时，理论上可能随机到已失败渠道。
- 没有“先探测候选渠道健康度，再决定下一个真实请求发给谁”。
- 没有把 A1/A2/B1-/C1- 这类步骤结构化落到链路字段。

### 0.5 auto 分组与跨分组重试

`auto` 分组下，NewAPI 有跨分组重试设计：

```text
GroupA priority0
GroupA priority1
GroupA exhausted -> GroupB priority0
GroupB priority1
```

但它仍是串行尝试：

```text
A 失败后，下一轮才考虑 B
```

不是：

```text
A 失败时，同时轻量探测 B/C，提前判断哪个健康
```

### 0.6 shouldRetry 的边界

`shouldRetry` 主要看：

- 是否还有剩余 retry 次数
- 是否指定了 specific channel
- 是否是 SkipRetry 错误
- 是否是 2xx
- 是否是非标准状态码
- 是否命中永不重试错误码
- 是否命中自动重试状态码范围

默认自动重试状态码：

```text
100-199
300-399
401-407
409-499
500-503
505-523
525-599
```

默认永不重试：

```text
504
524
ErrorCodeBadResponseBody
```

默认不包含：

```text
400
408
```

其中 `429` 默认会重试。

### 0.7 原生日志能看到什么、看不到什么

NewAPI 原生失败时会写错误日志，并在 `other.admin_info.use_channel` 里保存用过的渠道列表。

能看到：

```text
这次请求尝试过哪些 channel id
某一条错误日志的 status_code / channel / error_code
```

看不到：

```text
哪条错误是“已拦截，不返回用户”
哪条错误是“最终客户端可见”
哪条是轻量探测成功/失败
完整链路的严格父子关系
同级步骤顺序
容错最终成功/失败的结构化状态
```

所以旧 NewAPI 日志不应强行推断 NACP 容错链路。老日志只需要按旧类型正常展示兼容。

### 0.8 NACP 增强不是重复造轮子

NewAPI 原生重试解决的是：

```text
失败后，要不要串行换一个渠道再完整重发？
```

NACP 增强要解决的是：

```text
A 失败后，如何拦截错误不立刻返回用户？
A 是否要做同渠道重试？
B/C 是否可以提前轻量探测健康状态？
当 A2/A3 和 B1-/C1- 回包先后不同，如何决策？
每个步骤如何结构化还原成一个严谨链路？
如何避免污染 logs.type，同时保留可搜索/可展示语义？
```

### 0.9 当前设计原则

1. `logs.type` 保持 NewAPI 老类型，用于兼容、计费、统计、旧日志显示。
2. NACP 容错语义进入 `trace_*` 字段，例如 `trace_id / trace_seq / trace_role`。
3. 老日志不做容错链路推断，只按原始日志正常显示。
4. 新日志一行仍是一条独立真实日志，链路由结构字段还原。
5. `20/50/51/52/21/29/59` 可以作为“语义筛选/展示概念”，但不应作为 `logs.type` 的真实落库值。

---

## 一、请求生命周期总览

```
客户端请求
    ↓
[Router] → 路由匹配 (router/)
    ↓
[Auth Middleware] → Token 验证、用户识别
    ↓
[Distribute Middleware] → 首次渠道选择（按优先级+权重）
    ↓
[Relay Controller] → 核心中继循环（含重试）
    │
    ├── [getChannel] → 获取渠道（首次用 Distribute 选的，重试时重新选）
    ├── [relayHandler] → 转发请求到上游
    ├── [成功] → 返回客户端 ✅
    └── [失败] → processChannelError → shouldRetry 判断
            ├── [可重试] → 循环继续，选下一个渠道
            └── [不可重试] → 返回错误给客户端 ❌
```

---

## 二、渠道选择机制

### 2.1 数据模型

**Channel 表核心字段：**
| 字段 | 说明 |
|------|------|
| `priority` | 优先级（bigint），越大越优先 |
| `weight` | 权重（uint），同优先级内按权重随机 |
| `status` | 状态：1=启用, 2=手动禁用, 3=自动禁用 |
| `auto_ban` | 是否允许自动禁用（1=是, 0=否） |
| `group` | 所属分组（逗号分隔） |
| `models` | 支持的模型（逗号分隔） |

**Ability 表（渠道能力索引）：**
- 组合主键：`(group, model, channel_id)`
- 字段：`enabled`, `priority`, `weight`, `tag`
- 用于快速查找某个 group+model 下的可用渠道

### 2.2 选择算法

```
1. 按 group + model 查找所有 enabled 的 ability 记录
2. 提取所有不同的 priority 值，降序排列
3. retry=0 → 取最高优先级的渠道集合
   retry=1 → 取第二高优先级的渠道集合
   retry=N → 取第 N+1 高优先级（超出则取最低）
4. 在同优先级渠道集合内，按 weight 加权随机选择一个
```

**权重随机算法：**
- 每个渠道有效权重 = `weight * smoothingFactor + smoothingAdjustment`
- 当所有权重为 0 时：每个渠道有效权重 = 100（均匀分布）
- 当平均权重 < 10 时：smoothingFactor = 100（放大差异）

### 2.3 Auto Group（跨分组重试）

当 token 的 group 为 `"auto"` 时：
- 按 autoGroups 列表顺序尝试每个分组
- 每个分组内用完所有优先级后，才切换到下一个分组
- 支持 `crossGroupRetry` 配置

---

## 三、现有重试机制

### 3.1 重试循环（controller/relay.go → Relay 函数）

```go
for ; retryParam.GetRetry() <= common.RetryTimes; retryParam.IncreaseRetry() {
    channel = getChannel(...)      // 选渠道（retry 递增 → 降低优先级）
    newAPIError = relayHandler(...)  // 转发请求
    
    if newAPIError == nil { return }  // 成功，直接返回
    
    processChannelError(...)         // 处理错误（可能禁用渠道）
    
    if !shouldRetry(...) { break }   // 判断是否继续重试
}
```

### 3.2 重试次数

- **全局配置**：`common.RetryTimes`（默认 0 = 不重试）
- **可通过管理后台设置**：`RetryTimes` option
- **含义**：最多额外重试 N 次（总共尝试 N+1 次）

### 3.3 shouldRetry 判断逻辑

```
优先级从高到低：
1. error == nil → 不重试（成功了）
2. Channel Affinity 失败 → 不重试
3. IsChannelError (errorCode 以 "channel:" 开头) → 重试
4. IsSkipRetryError (显式标记跳过) → 不重试
5. retryTimes <= 0 → 不重试（次数用完）
6. specific_channel_id 存在 → 不重试（指定渠道不切换）
7. 2xx → 不重试
8. <100 或 >599 → 重试
9. AlwaysSkipRetryCode (ErrorCodeBadResponseBody) → 不重试
10. ShouldRetryByStatusCode(code) → 按配置的状态码范围判断
```

### 3.4 可重试的状态码（默认配置）

| 范围 | 说明 |
|------|------|
| 100-199 | 信息性响应（异常） |
| 300-399 | 重定向（异常） |
| 401-407 | 认证/授权/代理错误 |
| 409-499 | 客户端错误（排除 400/408） |
| 500-503 | 服务器错误 |
| 505-523 | 服务器错误 |
| 525-599 | 服务器错误 |

**永远不重试：**
- 400 Bad Request（客户端参数错误）
- 408 Request Timeout
- 504 Gateway Timeout
- 524 A Timeout Occurred (Cloudflare)

### 3.5 重试时的渠道切换

重试时 `retry` 参数递增，导致选择**更低优先级**的渠道：
- retry=0 → 最高优先级渠道
- retry=1 → 第二优先级渠道
- retry=N → 第 N+1 优先级渠道

**关键限制：同一优先级内不会换渠道重试。**

---

## 四、自动禁用机制

### 4.1 触发条件（ShouldDisableChannel）

```
1. AutomaticDisableChannelEnabled == false → 不禁用
2. IsChannelError → 禁用
3. IsSkipRetryError → 不禁用
4. ShouldDisableByStatusCode(code) → 默认只有 401 触发禁用
5. 错误消息包含关键词 → 禁用
```

**自动禁用关键词（默认）：**
- "Your credit balance is too low"
- "This organization has been disabled."
- "You exceeded your current quota"
- "Permission denied"
- "The security token included in the request is invalid"
- "Operation not allowed"
- "Your account is not authorized"

### 4.2 禁用流程

```go
processChannelError() {
    // 1. 记录错误日志
    // 2. 判断是否应该禁用
    if ShouldDisableChannel(err) && channelError.AutoBan {
        gopool.Go(func() {
            DisableChannel(...)  // 异步禁用
        })
    }
    // 3. 记录错误日志到数据库
}
```

**DisableChannel 操作：**
- 更新 channel status → `ChannelStatusAutoDisabled` (3)
- 更新 ability enabled → false
- 发送通知给 root 用户
- 支持 multi-key 模式（单个 key 禁用，不影响其他 key）

### 4.3 自动启用机制

**仅在 TestAllChannels 中触发：**
```go
// 测试所有渠道时
if !isChannelEnabled && ShouldEnableChannel(newAPIError, channel.Status) {
    EnableChannel(...)
}
```

**ShouldEnableChannel 条件：**
- `AutomaticEnableChannelEnabled == true`
- 测试无错误 (`newAPIError == nil`)
- 渠道状态为 `ChannelStatusAutoDisabled` (3)（手动禁用的不自动启用）

---

## 五、定时测试机制

### 5.1 AutomaticallyTestChannels

- **触发**：后台 goroutine，仅 Master 节点运行
- **频率**：`AutoTestChannelMinutes`（默认 10 分钟，可配置）
- **开关**：`AutoTestChannelEnabled`（默认 false）
- **环境变量**：`CHANNEL_TEST_FREQUENCY`（分钟数，设置后自动启用）

### 5.2 测试逻辑

```
遍历所有渠道：
  - 跳过手动禁用的渠道
  - 发送测试请求（用渠道的 test_model 或第一个模型）
  - 检查结果：
    - 有错误 + 应该禁用 + auto_ban → 禁用渠道
    - 响应时间超阈值 → 禁用渠道
    - 无错误 + 渠道是自动禁用状态 → 启用渠道
  - 更新响应时间
  - 间隔 RequestInterval 后测试下一个
```

### 5.3 禁用阈值

- `ChannelDisableThreshold`：响应时间阈值（秒），默认 5.0
- 超过此阈值的渠道在全量测试时会被禁用

---

## 六、错误日志记录

### 6.1 记录条件

- `constant.ErrorLogEnabled == true`
- `types.IsRecordErrorLog(err) == true`

### 6.2 记录内容

```go
model.RecordErrorLog(c, userId, channelId, modelName, tokenName, 
    err.MaskSensitiveErrorWithStatusCode(), tokenId, useTimeSeconds, 
    isStream, userGroup, other)
```

**other 字段包含：**
- `request_path`
- `error_type`
- `error_code`
- `status_code`
- `channel_id`
- `channel_name`
- `channel_type`
- `admin_info`（使用的渠道列表、multi-key 信息）

---

## 七、现有机制的不足（对应业务需求）

### 7.1 重试机制不足

| 问题 | 现状 | 期望 |
|------|------|------|
| 重试次数 | 全局统一 `RetryTimes`（默认 0） | 按渠道/模型可配置 |
| 重试策略 | 立即重试，无退避 | 支持指数退避 + 抖动 |
| 已尝试渠道排除 | 无严格排除；同优先级随机选择可能再次命中已失败渠道 | 同一链路内应有 tried-channel 集合，避免无意义重复 |
| 候选健康预判 | 无；下一轮才完整请求下一个渠道 | A 失败后并发轻量探测 B/C，提前选择更健康候选 |
| 错误分类 | 粗粒度（状态码范围） | 细粒度（区分临时性/永久性） |
| 链路结构化 | 只有 `admin_info.use_channel` 这种弱痕迹 | 需要 `trace_id/trace_seq/trace_role` 严格还原 |
| 流式请求 | 中断后无法重试 | 需要特殊处理 |
| 客户感知 | 重试失败后返回错误 | 应尽量透明，最多延迟 |

### 7.2 自动禁用/启用不足

| 问题 | 现状 | 期望 |
|------|------|------|
| 禁用触发 | 单次错误即禁用（如果匹配关键词/状态码） | 应有容错阈值（N次失败才禁用） |
| 启用恢复 | 仅在定时全量测试时恢复 | 应持续探测，确认稳定后恢复 |
| 错误监控 | 被动（等客户报告） | 主动（日志出现错误立即核验） |
| 恢复策略 | 测试通过即启用 | 应谨慎恢复（持续测试 N 次通过才启用） |
| 通知 | 仅通知 root 用户 | 应有更灵活的告警机制 |

### 7.3 请求透传不足

| 问题 | 现状 | 期望 |
|------|------|------|
| 错误过滤 | 上游错误直接返回客户端 | 应在本层拦截，尝试其他渠道 |
| 延迟感知 | 无 | 可接受假性延迟换取成功率 |
| 降级策略 | 按优先级降级 | 应更灵活（同优先级换渠道 → 降优先级） |

---

## 八、关键代码位置索引

| 功能 | 文件 | 函数/位置 |
|------|------|---------|
| 主重试循环 | `controller/relay.go` | `Relay()` L67-240 |
| 重试判断 | `controller/relay.go` | `shouldRetry()` L318-348 |
| 渠道选择 | `service/channel_select.go` | `CacheGetRandomSatisfiedChannel()` |
| 缓存渠道选择 | `model/channel_cache.go` | `GetRandomSatisfiedChannel()` |
| 优先级+权重 | `model/ability.go` | `GetChannel()`, `getPriority()` |
| 自动禁用判断 | `service/channel.go` | `ShouldDisableChannel()` |
| 执行禁用 | `service/channel.go` | `DisableChannel()` |
| 自动启用判断 | `service/channel.go` | `ShouldEnableChannel()` |
| 定时测试 | `controller/channel-test.go` | `AutomaticallyTestChannels()` |
| 全量测试 | `controller/channel-test.go` | `testAllChannels()` |
| 错误处理 | `controller/relay.go` | `processChannelError()` L350-395 |
| 状态码配置 | `setting/operation_setting/status_code_ranges.go` | 全文件 |
| 禁用关键词 | `setting/operation_setting/operation_setting.go` | `AutomaticDisableKeywords` |
| 监控设置 | `setting/operation_setting/monitor_setting.go` | `MonitorSetting` |
| 重试次数 | `common/constants.go` | `RetryTimes` (默认 0) |
| 渠道状态常量 | `common/constants.go` | `ChannelStatus*` |

---

## 九、总结：现有系统的设计哲学

1. **优先级分层** — 高优先级渠道优先使用，失败后降级到低优先级
2. **权重随机** — 同优先级内按权重随机分配流量
3. **快速失败** — 单次错误即可触发禁用（匹配条件时）
4. **被动恢复** — 依赖定时测试来恢复被禁用的渠道
5. **全局配置** — 重试次数、禁用条件等为全局统一配置
6. **简单直接** — 没有复杂的退避策略、熔断器、健康检查等

这套设计在上游质量稳定时工作良好，但在上游质量波动大的场景下，存在明显的不足。

---

**最后更新**：2026-05-15
