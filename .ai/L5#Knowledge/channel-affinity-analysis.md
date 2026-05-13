# 渠道亲和性（Channel Affinity）与健康状态机的协同

> **分析日期**：2026-05-13
> **状态**：方案确认中

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
            SkipRetryOnFailure: true,  // 亲和渠道失败不重试！
            ...
        },
        {
            Name:               "claude cli trace",
            SkipRetryOnFailure: true,  // 亲和渠道失败不重试！
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
| SkipRetryOnFailure | 保持 true — 因为 Degraded 在亲和检查阶段就被跳过了 |

### 为什么不需要改 SkipRetryOnFailure

`SkipRetryOnFailure: true` 的语义是"如果亲和渠道失败，不要重试其他渠道"。

在新方案下：
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

**最后更新**：2026-05-13
