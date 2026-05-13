# NACP — NewAPI Classic Plus

> The final enhanced edition of NewAPI Classic.

基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) v0.13.2 经典版本线的最终增强版。

---

## 定位

NACP 是为了解决 NewAPI classic 版本线在实际运营中遇到的上游质量问题而创建的增强分支。不跟 upstream v1 新 UI/新路线混合，专注于核心中继能力的增强。

## 核心增强

### v0.1.0 — 智能重试与渠道健康管理

**解决的问题：** 上游供应商质量不一，客户频繁看到报错信息。

**增强内容：**

1. **渠道健康状态机** — 五状态模型（Healthy → Probing → Degraded → Recovering → Healthy），替代简单的开/关
2. **同渠道重试** — 发现错误后不立即返回客户，先重试同渠道 2 次（可能是偶发）
3. **并行预热** — 重试同渠道的同时，并行探测顺位渠道的健康状态，建立就绪队列
4. **同优先级渠道切换** — 同优先级内最多尝试 3 个渠道后才降级到低优先级
5. **渠道亲和性健康感知** — Degraded 渠道自动跳过亲和绑定，避免持续发到问题渠道
6. **低渠道告警** — 可用渠道不足时自动告警
7. **安全兜底** — 所有渠道被过滤时回退到不过滤（宁可用 degraded 的也不能完全没有）

**对客户的效果：** 错误在本层拦截，客户最多感知到延迟稍高，不再看到报错信息（除非所有渠道都不可用）。

---

## 版本线

| 项目 | 值 |
|------|-----|
| 基线 | QuantumNous/new-api v0.13.2 |
| 版本格式 | `v0.x.y` |
| 当前版本 | v0.1.0 |

---

## 部署

与原版 NewAPI 相同，Docker 部署，MySQL 数据库。

首次部署时 GORM AutoMigrate 会自动添加新的数据库字段（`health_status`, `health_updated_at`, `health_fail_count`, `health_success_count`），无需手动迁移。

---

## 配置

### 新增配置项（硬编码默认值，后续可配置）

| 参数 | 默认值 | 说明 |
|------|--------|------|
| SameChannelRetryCount | 2 | 同渠道重试次数 |
| ProbeFailThreshold | 2 | 连续探测失败几次确认降级 |
| DegradedProbeInterval | 5 分钟 | 降级渠道周期性探测间隔 |
| RecoverySuccessThreshold | 3 | 连续探测成功几次进入恢复期 |
| RecoveryObservationPeriod | 10 分钟 | 恢复观察期时长 |
| ProbeTimeout | 3 秒 | 单次探测超时 |
| MaxRetrySamePriority | 3 | 同优先级最多尝试几个渠道 |
| PreWarmChannelCount | 2 | 并行预热几个顺位渠道 |

---

## 文件变更清单

### 新增文件
- `service/channel_health.go` — 健康状态机核心
- `service/channel_health_config.go` — 配置/阈值
- `service/channel_probe.go` — 轻量探测 + 并行预热

### 修改文件
- `common/constants.go` — 新增健康状态常量
- `model/channel.go` — Channel struct 新增 4 个健康字段
- `model/channel_cache.go` — GetRandomSatisfiedChannel 加入健康过滤 + 排除
- `controller/relay.go` — 增强重试循环 + processChannelError 集成状态机
- `middleware/distributor.go` — 亲和性健康感知
- `main.go` — 启动时初始化健康管理

---

## 与原版的兼容性

- 所有渠道默认 `health_status = "healthy"`，不触发任何新逻辑
- 现有的 `Status` 字段（开/关/自动禁用）不受影响
- 现有的自动禁用机制继续工作（共存策略）
- 如果 `RetryTimes = 0`（默认），增强重试循环仍然会执行同渠道重试和预热
