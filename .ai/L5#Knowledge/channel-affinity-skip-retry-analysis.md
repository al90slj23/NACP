# 渠道亲和 SkipRetryOnFailure 与健康感知的关系

> **分析日期**：2026-05-14
> **状态**：已确认

---

## 功能说明

### "失败后是否重试"（SkipRetryOnFailure）

- UI 列标题："失败后是否重试"
- 值："不重试" = `SkipRetryOnFailure: true`
- 值："重试" = `SkipRetryOnFailure: false`
- 默认预置规则（codex/claude）= `true`（不重试）

### 代码逻辑链

```
规则 SkipRetryOnFailure: true
  → meta.SkipRetry = true
  → MarkChannelAffinityUsed() 存入 context
  → 请求失败后 shouldRetry() 调用 ShouldSkipRetryAfterChannelAffinityFailure()
  → 返回 true → shouldRetry() 返回 false → 不重试 → 报错给客户端
```

### 为什么 Codex/Claude 设为"不重试"

- CLI 长会话依赖 prompt_cache_key / metadata.user_id 做缓存绑定
- 切换渠道 = 丢失缓存 = 所有 token 重新计算 = 成本翻倍
- 宁可偶尔报错让用户重试，也不换渠道丢缓存

---

## 与 NACP 健康感知亲和的关系

### 不冲突，互补

| 阶段 | SkipRetryOnFailure | NACP 健康感知 |
|------|-------------------|--------------|
| 作用时机 | 请求已发出并失败后 | 请求还没发出，选渠道阶段 |
| 判断依据 | 配置开关 | 渠道 health_status |
| 行为 | 决定是否切换渠道重试 | 决定是否跳过该渠道 |

### 两种设置下的行为

**"不重试"（SkipRetryOnFailure=true）+ 健康感知：**
```
A(Healthy) → 发请求 → 偶发失败 → 不切换，报错（保护缓存）
A(Degraded) → 健康感知跳过 A → 选 B → 正常发送（不触发 SkipRetry）
```

**"重试"（SkipRetryOnFailure=false）+ 健康感知：**
```
A(Healthy) → 发请求 → 偶发失败 → 切换渠道重试（牺牲缓存）
A(Degraded) → 健康感知跳过 A → 选 B → 正常发送（不触发 SkipRetry）
```

### 结论

- 健康感知在 SkipRetry 判断之前生效
- Degraded 渠道被跳过时，SkipRetry 标记不会被设置
- 建议保持"不重试" — 健康感知已解决持续故障问题，"不重试"保护偶发失败时的缓存命中率

---

**最后更新**：2026-05-14
