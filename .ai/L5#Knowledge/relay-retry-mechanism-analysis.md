# NewAPI 中继重试机制深度分析

> **分析日期**：2026-05-13
> **基线版本**：v0.13.2 (bee339d)
> **分析目的**：为 NACP 重试增强提供决策依据

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
| 同优先级重试 | 不支持（同优先级只选一次） | 同优先级内也应尝试其他渠道 |
| 错误分类 | 粗粒度（状态码范围） | 细粒度（区分临时性/永久性） |
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

**最后更新**：2026-05-13
