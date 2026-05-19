# 渠道亲和性（Channel Affinity）与健康状态机的协同

> **分析日期**：2026-05-13
> **最后更新**：2026-05-19
> **状态**：已纳入 SFT 容错调度语义

---

## 一、现有亲和机制

### 工作原理

1. 按规则匹配请求（模型正则、路径正则、UserAgent）
2. 从请求体提取亲和 key（如 Codex 的 `prompt_cache_key`，Claude 的 `metadata.user_id`）
3. 缓存绑定：`亲和key → channel_id`（LRU + Redis，TTL 默认 1 小时）
4. 下次同 key 请求优先路由到同一渠道
5. `SwitchOnSuccess: true` — 如果请求最终成功（可能经过重试换了渠道），更新亲和缓存到成功的渠道

### 当前配置

```go
ChannelAffinitySetting{
    Enabled:           true,
    SwitchOnSuccess:   true,      // 成功后更新亲和到实际使用的渠道
    MaxEntries:        100_000,
    DefaultTTLSeconds: 3600,      // 1 小时 TTL
    Rules: []ChannelAffinityRule{
        {
            Name:               "codex cli trace",
            SkipRetryOnFailure: false, // 亲和渠道失败后继续 SFT 容错
            ...
        },
        {
            Name:               "claude cli trace",
            SkipRetryOnFailure: false, // 亲和渠道失败后继续 SFT 容错
            ...
        },
    },
}
```

### 关键代码位置

- `service/channel_affinity.go` — 亲和逻辑核心
- `setting/operation_setting/channel_affinity_setting.go` — 配置定义
- `middleware/distributor.go` L104-108 — 亲和渠道检查入口
- `controller/relay.go` L320 — `ShouldSkipRetryAfterChannelAffinityFailure` 阻止重试

---

## 二、痛点分析

| 场景 | 不开亲和 | 开亲和（当前） |
|------|---------|-------------|
| 缓存命中 | ❌ 低（请求分散） | ✅ 高（固定渠道） |
| 成本 | 高（多次建缓存） | 低（复用缓存） |
| 渠道故障时 | ✅ 自动切换 | ❌ 持续报错（SkipRetry=true） |
| 用户体验 | 一般 | 正常时好，故障时差 |

**核心矛盾：** 亲和性为了缓存命中率牺牲了容错能力。

---

## 三、解决方案：健康状态感知的亲和

### 改动点

在 `middleware/distributor.go` 的亲和检查处（L104-108），加入健康状态判断：

```go
// 现有逻辑
if preferredChannelID, found := service.GetPreferredChannelByAffinity(...); found {
    preferred, err := model.CacheGetChannel(preferredChannelID)
    if err == nil && preferred != nil {
        if preferred.Status != common.ChannelStatusEnabled {
            // 现有：渠道被禁用时的处理
        }
        // ★ 新增：检查健康状态
        if service.GetChannelHealthStatus(preferred.Id) == service.HealthStatusDegraded {
            // 亲和渠道已降级 → 跳过亲和，走正常渠道选择
            // 不设置 SkipRetry 标记
            channel = nil  // 让后续逻辑走正常选择
        }
    }
}
```

### 流程

```
请求进来 → 匹配亲和规则 → 找到绑定的渠道 A
    ↓
检查渠道 A 的 health_status
    ├── Healthy/Recovering/Probing → 正常发送到 A（享受缓存命中）
    └── Degraded → 跳过亲和，走正常渠道选择
                    ↓
                选到渠道 B → 发送请求
                    ↓
                成功 → SwitchOnSuccess 更新亲和缓存 key → B
                （下次请求直接到 B，开始在 B 上建立缓存）
```

### 效果

| 场景 | 行为 |
|------|------|
| 亲和渠道健康 | 正常亲和，享受缓存命中 |
| 亲和渠道降级 | 自动跳过，切到其他渠道，亲和缓存更新 |
| 亲和渠道恢复 | 不主动切回（等 TTL 过期后自然重新分配） |
| SkipRetryOnFailure | 建议 false — 亲和渠道失败后仍进入 SFT 容错队列 |

### SkipRetryOnFailure 的当前默认值

`SkipRetryOnFailure: true` 的语义是"如果亲和渠道失败，不要重试其他渠道"。

在当前 SFT 容错语义下：

- 默认 `skip_retry_on_failure=false`，也就是后台显示“重试”。
- 如果亲和渠道是 Degraded → 亲和检查阶段就跳过了 → 不会走到 SkipRetry 逻辑。
- 如果亲和渠道 Healthy 但本次请求失败 → 继续进入 SFT 容错队列，保留用户成功率。
- 如果业务明确要求强亲和，才设置 `skip_retry_on_failure=true`，失败后直接返回错误。
- `skip_retry_on_failure` 只在“亲和生效”之后生效：规则匹配、提取到亲和 key、但缓存未命中或渠道不可用，都只算亲和失效，不会阻断普通分组队列。

旧方案记录：

- 如果亲和渠道是 Degraded → 亲和检查阶段就跳过了 → 不会走到 SkipRetry 逻辑
- 如果亲和渠道是 Healthy 但本次请求失败 → 触发健康状态机 → 可能进入 Probing
  - 此时 SkipRetry=true 会阻止重试 → 用户看到错误
  - 但下一次请求时，如果渠道已经变成 Degraded → 自动跳过亲和
  - **第一次错误无法避免**（因为 SkipRetry=true），但后续请求不会再发到问题渠道

### 进一步优化（可选，后续迭代）

如果想连第一次错误都避免，可以：
- 在健康状态机中，当亲和渠道从 Healthy → Probing 时，临时清除该渠道的亲和缓存
- 这样下一个请求就不会被亲和到这个正在探测的渠道
- 但这会牺牲一些缓存命中率（可能是误判的偶发错误）

---

## 四、轻量探测的未来价值（记录）

轻量探测虽然当前版本暂不实现，但对亲和性有额外价值：

1. **响应时间监控** — 探测可以获取渠道延迟信息
2. **预警机制** — 延迟升高时提前进入"警备状态"
3. **亲和切换决策** — 如果亲和渠道延迟过高，主动切换到更快的渠道
4. **缓存成本优化** — 结合缓存命中率统计，判断是否值得维持亲和

---

## 五、2026-05-19 亲和前置队列语义

渠道亲和不改变分组原始重试队列，只在队列最前面追加一次“亲和首选机会”。

定义：

1. 已建立亲和：写入亲和缓存成功，也就是创建了 `亲和 Key -> channel_id`。
2. 亲和命中：读取亲和缓存命中。
3. 亲和生效：命中的渠道通过启用、健康、分组、模型校验，并实际被选用。
4. 亲和更新：亲和渠道失败后，其他渠道成功，缓存切换到成功渠道。
5. 亲和失效：缓存不存在、过期、渠道禁用、渠道不可路由、分组/模型不匹配。

队列规则：

- 无亲和：使用原始分组队列。
- 有亲和：`亲和渠道 + 原始分组队列`。
- 亲和渠道在原始分组队列中仍保留自己的位置，不因为前置试过一次就被移除。
- 亲和规则的 `skip_retry_on_failure=false` 时，亲和渠道失败后继续 SFT 容错。
- 亲和规则的 `skip_retry_on_failure=true` 时，亲和渠道失败后不继续切换。
- `skip_retry_on_failure` 必须以 `MarkChannelAffinityUsed()` 已记录“亲和渠道实际被选用”为前提；仅匹配亲和规则或仅读到亲和缓存，不应影响正常容错。

例子：

```text
原始队列：C -> D -> A -> B
亲和渠道：A
实际队列：A -> C -> D -> A -> B
```

如果后续 `C` 成功并写入亲和缓存，则下次亲和渠道变为 `C`：

```text
亲和渠道：C
实际队列：C -> C -> D -> A -> B
```

这个语义的原因是：亲和渠道已经建立过上游缓存，先给一次机会可以保留缓存收益；如果失败，再完整执行分组原始队列，原始队列中的该渠道仍然有一次按自身优先级/权重位置重试的机会。

实现约束：

- `middleware/distributor.go` 命中亲和并实际选用渠道后，调用 `MarkChannelAffinityUsed()` 记录前置亲和渠道 ID。
- `controller/relay.go` 中第一次亲和前置尝试不写入 `realTriedChannelIDs`，因此不会污染原始分组队列。
- 当同一渠道之后在原始分组队列中再次被选中时，再按普通尝试写入排除集，避免原始队列内部无限重复。
- `using_group=auto` 且亲和命中落在后续 auto 分组时，必须同时写入 `ContextKeyAutoGroup` 和 `ContextKeyAutoGroupIndex`，否则后续 SFT 队列可能从 auto 的第一个分组重新开始，导致链路跳回错误分组。

**最后更新**：2026-05-19
